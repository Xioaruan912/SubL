package api

import (
	"encoding/json"
	"testing"
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
