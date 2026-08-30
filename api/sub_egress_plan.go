package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"ppeelink/models"
	"ppeelink/node"
	"ppeelink/rulecenter"
)

type splitRule struct {
	Kind   string
	Domain string
	Policy string
	Raw    string
	Index  int
}
type planNode struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	CountryCode   string  `json:"countryCode"`
	Score         int     `json:"score"`
	AverageRtt    int     `json:"averageRtt"`
	Availability  float64 `json:"availability"`
	Confidence    int     `json:"confidence"`
	QualitySource string  `json:"qualitySource"`
	RankScore     int     `json:"rankScore"`
	Link          string  `json:"-"`
}
type egressPlanItem struct {
	Key             string                  `json:"key"`
	Name            string                  `json:"name"`
	Domain          string                  `json:"domain"`
	Group           string                  `json:"group"`
	MatchedRule     string                  `json:"matchedRule"`
	Policy          string                  `json:"policy"`
	ExpectedCountry string                  `json:"expectedCountry"`
	SelectedNode    *planNode               `json:"selectedNode"`
	CandidateCount  int                     `json:"candidateCount"`
	Fallback        bool                    `json:"fallback"`
	Result          *node.EgressCheckResult `json:"result,omitempty"`
}
type egressPlanResponse struct {
	SubscriptionID   int              `json:"subscriptionId"`
	SubscriptionName string           `json:"subscriptionName"`
	Template         string           `json:"template"`
	Items            []egressPlanItem `json:"items"`
	Warnings         []string         `json:"warnings"`
}

type egressTestRunner func(context.Context, string, time.Duration, []node.EgressTarget) (*node.EgressResult, error)

func templateContent(sub *models.Subcription) (string, string, error) {
	var cfg models.SubscriptionConfig
	if err := json.Unmarshal([]byte(sub.Config), &cfg); err != nil {
		return "", "", err
	}
	for _, name := range []string{cfg.Clash, cfg.Surge, cfg.Loon} {
		if name == "" {
			continue
		}
		clean := strings.TrimPrefix(filepath.ToSlash(name), "./template/")
		if strings.Contains(clean, "..") || filepath.IsAbs(clean) {
			continue
		}
		path, err := safeFilePath(clean)
		if err != nil {
			continue
		}
		body, err := os.ReadFile(path)
		if err == nil {
			return filepath.Base(path), string(body), nil
		}
	}
	return "", "", fmt.Errorf("未找到可读取的本地订阅模板")
}

func parseSplitRules(content string) []splitRule {
	var root map[string]interface{}
	if yaml.Unmarshal([]byte(content), &root) != nil {
		return nil
	}
	raw, ok := root["rules"].([]interface{})
	if !ok {
		return nil
	}
	result := make([]splitRule, 0, len(raw))
	for index, value := range raw {
		line := strings.TrimSpace(fmt.Sprint(value))
		parts := splitTopLevelRule(line)
		if len(parts) < 2 {
			continue
		}
		kind := strings.ToUpper(strings.TrimSpace(parts[0]))
		policyIndex := 2
		if kind == "MATCH" || kind == "FINAL" {
			policyIndex = 1
		} else if kind == "AND" || kind == "OR" || kind == "NOT" {
			policyIndex = len(parts) - 1
		}
		if policyIndex >= len(parts) {
			continue
		}
		policy := strings.TrimSpace(parts[policyIndex])
		domain := ""
		if len(parts) > 2 {
			domain = strings.TrimSpace(parts[1])
		}
		result = append(result, splitRule{Kind: kind, Domain: domain, Policy: policy, Raw: line, Index: index + 1})
	}
	return result
}

// splitTopLevelRule keeps logical Clash rule expressions intact. A plain
// strings.Split breaks AND/OR rules because nested predicates also use commas.
func splitTopLevelRule(line string) []string {
	parts := make([]string, 0, 4)
	start, depth := 0, 0
	var quote rune
	for index, char := range line {
		switch {
		case quote != 0:
			if char == quote {
				quote = 0
			}
		case char == '\'' || char == '"':
			quote = char
		case char == '(':
			depth++
		case char == ')' && depth > 0:
			depth--
		case char == ',' && depth == 0:
			parts = append(parts, strings.TrimSpace(line[start:index]))
			start = index + 1
		}
	}
	parts = append(parts, strings.TrimSpace(line[start:]))
	return parts
}

func ruleForDomain(rules []splitRule, domain string) (splitRule, bool) {
	key := strings.ToLower(strings.TrimPrefix(domain, "www."))
	tokens := strings.FieldsFunc(key, func(r rune) bool { return r == '.' || r == '-' || r == '_' })
	for _, r := range rules {
		d := strings.TrimPrefix(strings.ToLower(r.Domain), ".")
		switch r.Kind {
		case "DOMAIN":
			if strings.EqualFold(d, domain) {
				return r, true
			}
		case "DOMAIN-SUFFIX":
			if strings.HasSuffix(domain, d) {
				return r, true
			}
		case "DOMAIN-KEYWORD":
			if strings.Contains(domain, d) {
				return r, true
			}
		case "RULE-SET":
			// Clash rule providers carry matching sites in the provider
			// itself. When the provider is named after a domain/brand,
			// associate it generically instead of hard-coding one template.
			for _, token := range tokens {
				if len(token) >= 4 && strings.Contains(d, token) {
					return r, true
				}
			}
		}
	}
	for _, r := range rules {
		if r.Kind == "MATCH" || r.Kind == "FINAL" {
			return r, true
		}
	}
	return splitRule{}, false
}

func ruleForDomainResolved(ctx context.Context, rules []splitRule, content, domain string) (splitRule, bool, string) {
	key := strings.ToLower(strings.TrimPrefix(domain, "www."))
	warning := ""
	for _, r := range rules {
		d := strings.TrimPrefix(strings.ToLower(r.Domain), ".")
		switch r.Kind {
		case "DOMAIN":
			if strings.EqualFold(d, key) {
				return r, true, ""
			}
		case "DOMAIN-SUFFIX":
			if key == d || strings.HasSuffix(key, "."+d) {
				return r, true, ""
			}
		case "DOMAIN-KEYWORD":
			if d != "" && strings.Contains(key, d) {
				return r, true, ""
			}
		case "RULE-SET":
			providerRules, _, err := rulecenter.ResolveProviderRules(ctx, content, r.Domain)
			if err != nil {
				if warning == "" {
					warning = "rule-provider " + r.Domain + " 解析失败: " + err.Error()
				}
				continue
			}
			if rulecenter.MatchDomain(providerRules, key) {
				return r, true, warning
			}
		case "MATCH", "FINAL":
			return r, true, warning
		}
	}
	return splitRule{}, false, warning
}

func policyFilterCountry(content, policy string) string {
	var root map[string]interface{}
	if yaml.Unmarshal([]byte(content), &root) != nil {
		return ""
	}
	groups, _ := root["proxy-groups"].([]interface{})
	for _, raw := range groups {
		group, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if strings.EqualFold(fmt.Sprint(group["name"]), policy) {
			filter := strings.ToUpper(fmt.Sprint(group["filter"]))
			codes := []string{}
			for _, code := range []string{"JP", "US", "SG", "HK", "TW", "GB"} {
				if strings.Contains(filter, code) {
					codes = append(codes, code)
				}
			}
			if len(codes) == 1 {
				return codes[0]
			}
			return ""
		}
	}
	return ""
}

func countryFromText(value string) string {
	v := strings.ToUpper(value)
	for _, pair := range []struct {
		code  string
		words []string
	}{{"JP", []string{"JP", "日本", "JAPAN"}}, {"US", []string{"US", "美国", "USA", "UNITED STATES"}}, {"SG", []string{"SG", "新加坡", "SINGAPORE"}}, {"HK", []string{"HK", "香港"}}, {"TW", []string{"TW", "台湾", "TAIWAN"}}, {"GB", []string{"GB", "英国", "UK"}}} {
		for _, word := range pair.words {
			if strings.Contains(v, word) {
				return pair.code
			}
		}
	}
	return ""
}

type planQualityMatrix struct {
	Targets map[int]map[string]models.TargetQualityStats
	Scenes  map[int]map[string]models.TargetQualityStats
}

func choosePlanNode(nodes []models.Node, expected string, stats map[int]models.NodeQualityStats, matrix planQualityMatrix, targetKey, scene string) ([]planNode, int) {
	all := make([]planNode, 0, len(nodes))
	filtered := make([]planNode, 0, len(nodes))
	for _, n := range nodes {
		host, _ := node.ExtractServerHost(n.Link)
		cc := countryFromText(n.Name)
		if cc == "" {
			cc = strings.ToUpper(node.LookupCountry(host))
		}
		st := stats[n.ID]
		score, averageRtt, availability, confidence, source := st.Score, st.AverageRtt, st.Availability, st.Confidence, "tcp"
		if perNode := matrix.Targets[n.ID]; perNode != nil {
			if targetStat, ok := perNode[targetKey]; ok && targetStat.SampleCount > 0 {
				score, averageRtt, availability, confidence, source = targetStat.Score, targetStat.AverageRtt, targetStat.Availability, targetStat.Confidence, "target"
			}
		}
		if source == "tcp" {
			if perNode := matrix.Scenes[n.ID]; perNode != nil {
				if sceneStat, ok := perNode[scene]; ok && sceneStat.SampleCount > 0 {
					score, averageRtt, availability, confidence, source = sceneStat.Score, sceneStat.AverageRtt, sceneStat.Availability, sceneStat.Confidence, "scene"
				}
			}
		}
		rankScore := (score*8 + confidence*2) / 10
		item := planNode{ID: n.ID, Name: n.Name, CountryCode: cc, Score: score, AverageRtt: averageRtt, Availability: availability, Confidence: confidence, QualitySource: source, RankScore: rankScore, Link: n.Link}
		all = append(all, item)
		if expected == "" || cc == expected {
			filtered = append(filtered, item)
		}
	}
	if expected == "" && len(filtered) == 0 {
		filtered = all
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].RankScore != filtered[j].RankScore {
			return filtered[i].RankScore > filtered[j].RankScore
		}
		if filtered[i].Availability != filtered[j].Availability {
			return filtered[i].Availability > filtered[j].Availability
		}
		if filtered[i].AverageRtt < 0 {
			return false
		}
		return filtered[i].AverageRtt < filtered[j].AverageRtt
	})
	return filtered, len(filtered)
}

func buildEgressPlan(ctx context.Context, sub *models.Subcription, templateName, content string, stats map[int]models.NodeQualityStats, matrix planQualityMatrix, targets []node.EgressTarget, run egressTestRunner) egressPlanResponse {
	response := egressPlanResponse{
		SubscriptionID:   sub.ID,
		SubscriptionName: sub.Name,
		Template:         templateName,
		Items:            make([]egressPlanItem, 0, len(targets)),
	}
	rules := parseSplitRules(content)
	chosen := make(map[int][]node.EgressTarget)
	for _, target := range targets {
		item := egressPlanItem{Key: target.Key, Name: target.Name, Domain: target.Domain, Group: target.Group}
		if rule, ok, resolveWarning := ruleForDomainResolved(ctx, rules, content, target.Domain); ok {
			item.MatchedRule, item.Policy = rule.Raw, rule.Policy
			item.ExpectedCountry = countryFromText(rule.Policy)
			if item.ExpectedCountry == "" {
				item.ExpectedCountry = policyFilterCountry(content, rule.Policy)
			}
		} else if resolveWarning != "" {
			response.Warnings = append(response.Warnings, target.Name+": "+resolveWarning)
		}
		if len(rules) == 0 {
			item.MatchedRule = "未读取模板，未应用模板规则"
		}
		candidates, count := choosePlanNode(sub.Nodes, item.ExpectedCountry, stats, matrix, target.Key, target.Group)
		item.CandidateCount = count
		if len(candidates) == 0 {
			response.Warnings = append(response.Warnings, target.Name+" 没有可用节点")
			response.Items = append(response.Items, item)
			continue
		}
		item.SelectedNode = &candidates[0]
		item.Fallback = item.ExpectedCountry != "" && item.SelectedNode.CountryCode != item.ExpectedCountry
		chosen[item.SelectedNode.ID] = append(chosen[item.SelectedNode.ID], target)
		response.Items = append(response.Items, item)
	}

	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cache := make(map[int]*node.EgressResult)
	for i := range response.Items {
		if response.Items[i].SelectedNode == nil {
			continue
		}
		id := response.Items[i].SelectedNode.ID
		result := cache[id]
		if result == nil {
			var runErr error
			result, runErr = run(testCtx, response.Items[i].SelectedNode.Link, 7*time.Second, chosen[id])
			if runErr != nil {
				response.Warnings = append(response.Warnings, runErr.Error())
				continue
			}
			cache[id] = result
		}
		for _, check := range result.Results {
			if check.Key == response.Items[i].Key {
				response.Items[i].Result = &check
				break
			}
		}
	}
	return response
}

// SubscriptionEgressPlan validates the effective template split by selecting
// a quality-ranked node for each AI destination and testing through it.
func SubscriptionEgressPlan(c *gin.Context) {
	id := c.Query("id")
	subID, parseErr := strconv.Atoi(id)
	if parseErr != nil || subID <= 0 {
		c.JSON(400, gin.H{"code": "40000", "msg": "订阅 id 格式错误"})
		return
	}
	task, trackedCtx, taskErr := createTaskRun(c.Request.Context(), "subscription-egress", "订阅模板分流验证", subscriptionEgressTaskRequest{SubscriptionID: subID}, nil)
	if taskErr != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "创建任务记录失败: " + taskErr.Error()})
		return
	}
	updateTaskProgress(task.ID, 20, "正在解析模板并选择目标节点")
	response, err := runSubscriptionEgressPlanTask(trackedCtx, subID)
	if err != nil {
		if trackedCtx.Err() == context.Canceled {
			markTaskCancelled(task.ID)
		} else {
			finishTaskRun(task.ID, err, nil)
		}
		c.JSON(500, gin.H{"code": "50000", "msg": err.Error()})
		return
	}
	finishTaskRun(task.ID, nil, response)
	c.JSON(200, gin.H{"code": "00000", "data": response, "msg": "模板分流验证完成"})
}

func runSubscriptionEgressPlanTask(ctx context.Context, subID int) (egressPlanResponse, error) {
	var sub models.Subcription
	if err := models.DB.First(&sub, subID).Error; err != nil {
		return egressPlanResponse{}, fmt.Errorf("订阅不存在: %w", err)
	}
	if err := mergeGroupNodes(&sub); err != nil {
		return egressPlanResponse{}, err
	}
	filename, content, err := templateContent(&sub)
	templateName := filename
	initialWarnings := []string{}
	if err != nil {
		initialWarnings = append(initialWarnings, err.Error())
		templateName = "未读取到本地模板"
	}
	stats, _ := models.GetNodeQualityStats(time.Now().Add(-24 * time.Hour))
	targetStats, _ := models.GetNodeTargetQualityStats(time.Now().Add(-24 * time.Hour))
	sceneStats, _ := models.GetNodeSceneQualityStats(time.Now().Add(-24 * time.Hour))
	targets, targetErr := enabledNodeEgressTargets()
	if targetErr != nil {
		return egressPlanResponse{}, fmt.Errorf("读取分流检测目标失败: %w", targetErr)
	}
	response := buildEgressPlan(ctx, &sub, templateName, content, stats, planQualityMatrix{Targets: targetStats, Scenes: sceneStats}, targets, node.RunEgressTestTargets)
	for _, item := range response.Items {
		if item.SelectedNode != nil && item.Result != nil {
			_ = recordEgressQuality(item.SelectedNode.ID, &node.EgressResult{Results: []node.EgressCheckResult{*item.Result}})
		}
	}
	response.Warnings = append(initialWarnings, response.Warnings...)
	return response, nil
}
