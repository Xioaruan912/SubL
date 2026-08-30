package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"ppeelink/models"
	"ppeelink/node"
)

func TestPlanNodeJSONFields(t *testing.T) {
	b, err := json.Marshal(planNode{ID: 7, Name: "Yunyoo_USA", CountryCode: "US", Score: 92, AverageRtt: 180})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"name": "Yunyoo_USA", "countryCode": "US"} {
		if got[key] != want {
			t.Fatalf("%s = %v, want %s", key, got[key], want)
		}
	}
	if got["id"] != float64(7) || got["score"] != float64(92) || got["averageRtt"] != float64(180) {
		t.Fatalf("quality fields serialized incorrectly: %s", b)
	}
}

func TestRuleSetTargetMatching(t *testing.T) {
	rules := parseSplitRules("rules:\n  - RULE-SET, Google, Gemini\n  - MATCH, 日常使用\n")
	rule, ok := ruleForDomain(rules, "gemini.google.com")
	if !ok || rule.Policy != "Gemini" {
		t.Fatalf("unexpected rule: %#v", rule)
	}
}

func TestSplitRulePolicyIgnoresOptionsAndNestedCommas(t *testing.T) {
	rules := parseSplitRules("rules:\n  - IP-CIDR,10.0.0.0/8,DIRECT,no-resolve\n  - AND,((DOMAIN,example.com),(NETWORK,TCP)),Proxy\n  - MATCH,Fallback\n")
	if len(rules) != 3 {
		t.Fatalf("unexpected rules: %#v", rules)
	}
	if rules[0].Policy != "DIRECT" || rules[1].Policy != "Proxy" || rules[2].Policy != "Fallback" {
		t.Fatalf("policy parsing failed: %#v", rules)
	}
}

var testPlanTargets = []node.EgressTarget{
	{Key: "gemini", Name: "Gemini", Domain: "gemini.google.com", Group: "ai"},
	{Key: "chatgpt", Name: "ChatGPT", Domain: "chatgpt.com", Group: "ai"},
	{Key: "openai", Name: "OpenAI", Domain: "openai.com", Group: "ai"},
	{Key: "claude", Name: "Claude", Domain: "claude.ai", Group: "ai"},
}

func successfulPlanRunner(_ context.Context, _ string, _ time.Duration, targets []node.EgressTarget) (*node.EgressResult, error) {
	results := make([]node.EgressCheckResult, 0, len(targets))
	for _, target := range targets {
		results = append(results, node.EgressCheckResult{EgressTarget: target, Status: "available", CountryCode: "US"})
	}
	return &node.EgressResult{Results: results}, nil
}

func planItemByKey(t *testing.T, response egressPlanResponse, key string) egressPlanItem {
	t.Helper()
	for _, item := range response.Items {
		if item.Key == key {
			return item
		}
	}
	t.Fatalf("plan item %q not found", key)
	return egressPlanItem{}
}

func TestBuildEgressPlanIntegration(t *testing.T) {
	stats := map[int]models.NodeQualityStats{
		1: {NodeID: 1, Score: 70, AverageRtt: 80},
		2: {NodeID: 2, Score: 95, AverageRtt: 45},
		3: {NodeID: 3, Score: 90, AverageRtt: 35},
	}
	nodes := []models.Node{
		{ID: 1, Name: "Tokyo JP", Link: "test://jp"},
		{ID: 2, Name: "Los Angeles USA", Link: "test://us"},
		{ID: 3, Name: "Singapore SG", Link: "test://sg"},
	}

	t.Run("node name region and explicit region rule", func(t *testing.T) {
		content := "rules:\n  - DOMAIN,chatgpt.com,US\n  - MATCH,Proxy\n"
		response := buildEgressPlan(context.Background(), &models.Subcription{ID: 10, Name: "real-sub", Nodes: nodes}, "clash.yaml", content, stats, planQualityMatrix{}, testPlanTargets, successfulPlanRunner)
		item := planItemByKey(t, response, "chatgpt")
		if item.ExpectedCountry != "US" || item.SelectedNode == nil || item.SelectedNode.ID != 2 {
			t.Fatalf("explicit US rule selected %#v", item)
		}
		if item.Result == nil || item.Result.Status != "available" {
			t.Fatalf("expected successful real-node result, got %#v", item.Result)
		}
	})

	t.Run("multi country filter does not force one region", func(t *testing.T) {
		content := "proxy-groups:\n  - name: AI\n    type: select\n    filter: '(?i)(US|JP)'\nrules:\n  - DOMAIN,chatgpt.com,AI\n  - MATCH,Proxy\n"
		response := buildEgressPlan(context.Background(), &models.Subcription{Nodes: nodes}, "clash.yaml", content, stats, planQualityMatrix{}, testPlanTargets, successfulPlanRunner)
		item := planItemByKey(t, response, "chatgpt")
		if item.ExpectedCountry != "" || item.CandidateCount != 3 || item.SelectedNode == nil || item.SelectedNode.ID != 2 {
			t.Fatalf("multi-country filter should rank all nodes, got %#v", item)
		}
	})

	t.Run("match fallback", func(t *testing.T) {
		content := "rules:\n  - DOMAIN,example.com,DIRECT\n  - MATCH,Proxy\n"
		response := buildEgressPlan(context.Background(), &models.Subcription{Nodes: nodes}, "clash.yaml", content, stats, planQualityMatrix{}, testPlanTargets, successfulPlanRunner)
		item := planItemByKey(t, response, "gemini")
		if item.Policy != "Proxy" || !strings.HasPrefix(item.MatchedRule, "MATCH,") {
			t.Fatalf("expected MATCH fallback, got %#v", item)
		}
	})

	t.Run("no nodes", func(t *testing.T) {
		response := buildEgressPlan(context.Background(), &models.Subcription{}, "clash.yaml", "rules:\n  - MATCH,Proxy\n", nil, planQualityMatrix{}, testPlanTargets, successfulPlanRunner)
		item := planItemByKey(t, response, "chatgpt")
		if item.SelectedNode != nil || item.CandidateCount != 0 {
			t.Fatalf("no-node plan unexpectedly selected node: %#v", item)
		}
		if len(response.Warnings) == 0 {
			t.Fatal("no-node plan should contain warning")
		}
	})

	t.Run("detection failure stays failed", func(t *testing.T) {
		runner := func(context.Context, string, time.Duration, []node.EgressTarget) (*node.EgressResult, error) {
			return nil, errors.New("synthetic egress failure")
		}
		response := buildEgressPlan(context.Background(), &models.Subcription{Nodes: nodes}, "clash.yaml", "rules:\n  - MATCH,Proxy\n", stats, planQualityMatrix{}, testPlanTargets, runner)
		item := planItemByKey(t, response, "openai")
		if item.SelectedNode == nil || item.Result != nil {
			t.Fatalf("failed detection must keep node but no result: %#v", item)
		}
		joined := strings.Join(response.Warnings, " | ")
		if !strings.Contains(joined, "synthetic egress failure") {
			t.Fatalf("missing detection failure warning: %s", joined)
		}
	})

	t.Run("target quality overrides generic tcp ranking", func(t *testing.T) {
		matrix := planQualityMatrix{Targets: map[int]map[string]models.TargetQualityStats{
			1: {"chatgpt": {NodeID: 1, TargetKey: "chatgpt", Score: 96, Availability: 100, AverageRtt: 55, Confidence: 80, SampleCount: 8}},
			2: {"chatgpt": {NodeID: 2, TargetKey: "chatgpt", Score: 25, Availability: 30, AverageRtt: 240, Confidence: 100, SampleCount: 12}},
		}}
		content := "rules:\n  - DOMAIN,chatgpt.com,Proxy\n  - MATCH,Proxy\n"
		response := buildEgressPlan(context.Background(), &models.Subcription{Nodes: nodes[:2]}, "clash.yaml", content, stats, matrix, testPlanTargets, successfulPlanRunner)
		item := planItemByKey(t, response, "chatgpt")
		if item.SelectedNode == nil || item.SelectedNode.ID != 1 || item.SelectedNode.QualitySource != "target" {
			t.Fatalf("expected target history to prefer node 1, got %#v", item.SelectedNode)
		}
	})
}
