package api

import (
	"context"
	"fmt"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"ppeelink/models"
	"ppeelink/rulecenter"
)

type ruleExplainRequest struct {
	SubscriptionID int    `json:"subscriptionId"`
	Target         string `json:"target"`
	IP             string `json:"ip"`
	Port           int    `json:"port"`
	Protocol       string `json:"protocol"`
}

type ruleExplainEvaluation struct {
	Index  int    `json:"index"`
	Rule   string `json:"rule"`
	Status string `json:"status"`
	Reason string `json:"reason"`
	Source string `json:"source,omitempty"`
}

type ruleExplainResponse struct {
	SubscriptionID  int                     `json:"subscriptionId"`
	Template        string                  `json:"template"`
	Format          string                  `json:"format"`
	Target          string                  `json:"target"`
	IP              string                  `json:"ip,omitempty"`
	Port            int                     `json:"port,omitempty"`
	Protocol        string                  `json:"protocol,omitempty"`
	MatchedRule     string                  `json:"matchedRule,omitempty"`
	RuleIndex       int                     `json:"ruleIndex,omitempty"`
	Policy          string                  `json:"policy,omitempty"`
	Chain           []string                `json:"chain"`
	Candidates      []string                `json:"candidates"`
	SelectedNode    *planNode               `json:"selectedNode,omitempty"`
	CandidateCount  int                     `json:"candidateCount"`
	ExpectedCountry string                  `json:"expectedCountry,omitempty"`
	EvaluatedCount  int                     `json:"evaluatedCount"`
	Previous        []ruleExplainEvaluation `json:"previous"`
	Matched         *ruleExplainEvaluation  `json:"matched,omitempty"`
	Warnings        []string                `json:"warnings"`
}

func normalizeExplainTarget(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if parsed, err := url.Parse(value); err == nil && parsed.Hostname() != "" {
		value = strings.ToLower(parsed.Hostname())
	}
	return strings.TrimSuffix(strings.TrimPrefix(value, "www."), ".")
}

func explainPortMatch(expr string, port int) bool {
	if port <= 0 {
		return false
	}
	for _, part := range strings.Split(expr, "/") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			lo, e1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
			hi, e2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if e1 == nil && e2 == nil && port >= lo && port <= hi {
				return true
			}
			continue
		}
		value, err := strconv.Atoi(part)
		if err == nil && value == port {
			return true
		}
	}
	return false
}

func evaluateExplainRule(ctx context.Context, rule splitRule, content string, req ruleExplainRequest) (bool, string, string) {
	target := normalizeExplainTarget(req.Target)
	switch rule.Kind {
	case "DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD", "DOMAIN-WILDCARD", "DOMAIN-REGEX":
		if target == "" {
			return false, "未提供域名", ""
		}
		if inlineDomainRuleMatch(rule, target) {
			return true, "域名条件命中", ""
		}
		return false, "域名条件未命中", ""
	case "RULE-SET":
		if target == "" {
			return false, "未提供域名，未计算 domain/classical provider", rule.Domain
		}
		rules, _, err := rulecenter.ResolveProviderRules(ctx, content, rule.Domain)
		if err != nil {
			return false, "rule-provider 无法解析: " + err.Error(), rule.Domain
		}
		if rulecenter.MatchDomain(rules, target) {
			return true, "命中 rule-provider 内容", rule.Domain
		}
		return false, "rule-provider 内容未命中", rule.Domain
	case "IP-CIDR", "IP-CIDR6":
		ip, err := netip.ParseAddr(strings.TrimSpace(req.IP))
		if err != nil {
			return false, "未提供有效目标 IP", ""
		}
		prefix, err := netip.ParsePrefix(strings.TrimSpace(rule.Domain))
		if err != nil {
			return false, "CIDR 格式无效", ""
		}
		if prefix.Contains(ip) {
			return true, "目标 IP 位于 CIDR", ""
		}
		return false, "目标 IP 不在 CIDR", ""
	case "DST-PORT":
		if explainPortMatch(rule.Domain, req.Port) {
			return true, "目标端口命中", ""
		}
		return false, "目标端口未命中", ""
	case "NETWORK":
		if req.Protocol != "" && strings.EqualFold(strings.TrimSpace(rule.Domain), strings.TrimSpace(req.Protocol)) {
			return true, "协议条件命中", ""
		}
		return false, "协议条件未命中", ""
	case "MATCH", "FINAL":
		return true, "兜底规则", ""
	default:
		return false, fmt.Sprintf("%s 需要更多运行时上下文，当前未模拟", rule.Kind), ""
	}
}

func clashExplainPolicy(content, policy string) ([]string, []string) {
	var cfg clashPreflightConfig
	if yaml.Unmarshal([]byte(normalizeTemplateYAML(content)), &cfg) != nil {
		return []string{policy}, nil
	}
	proxyNames := map[string]bool{}
	for _, proxy := range cfg.Proxies {
		if name := strings.TrimSpace(fmt.Sprint(proxy["name"])); name != "" {
			proxyNames[name] = true
		}
	}
	groups := map[string]preflightPolicyGroup{}
	for _, group := range cfg.ProxyGroups {
		dynamic := append([]string(nil), group.Use...)
		if group.IncludeAll || group.IncludeAllProxies || group.IncludeAllProviders || strings.TrimSpace(group.Filter) != "" {
			dynamic = append(dynamic, "动态节点(filter/provider)")
		}
		groups[group.Name] = preflightPolicyGroup{Name: group.Name, Type: group.Type, Members: append([]string(nil), group.Proxies...), Dynamic: dynamic}
	}
	return tracePolicy(policy, groups, proxyNames)
}

// SubscriptionRuleExplain explains the first-match path for one concrete
// request context without changing templates, subscriptions, or node state.
func SubscriptionRuleExplain(c *gin.Context) {
	var req ruleExplainRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.SubscriptionID <= 0 {
		c.JSON(400, gin.H{"code": "40000", "msg": "订阅或请求参数无效"})
		return
	}
	req.Target = normalizeExplainTarget(req.Target)
	req.Protocol = strings.ToLower(strings.TrimSpace(req.Protocol))
	if req.Target == "" && strings.TrimSpace(req.IP) == "" && req.Port <= 0 {
		c.JSON(400, gin.H{"code": "40000", "msg": "至少提供域名、IP 或端口之一"})
		return
	}
	var sub models.Subcription
	if err := models.DB.Preload("Nodes").First(&sub, req.SubscriptionID).Error; err != nil {
		c.JSON(404, gin.H{"code": "40400", "msg": "订阅不存在"})
		return
	}
	if err := mergeGroupNodes(&sub); err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "合并可见节点失败"})
		return
	}
	templateName, content, err := templateContent(&sub)
	if err != nil {
		c.JSON(400, gin.H{"code": "40000", "msg": err.Error()})
		return
	}
	format := templateFormat(templateName, content)
	if format != "clash" {
		c.JSON(400, gin.H{"code": "40000", "msg": "当前独立解释器先支持 Clash/Mihomo；Surge/Loon 仍可在模板发布前预检中查看域名命中链"})
		return
	}
	rules := parseSplitRules(content)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	response := ruleExplainResponse{SubscriptionID: sub.ID, Template: templateName, Format: format, Target: req.Target, IP: req.IP, Port: req.Port, Protocol: req.Protocol, Chain: []string{}, Candidates: []string{}, Previous: []ruleExplainEvaluation{}, Warnings: []string{}}
	const previousLimit = 100
	for _, rule := range rules {
		matched, reason, source := evaluateExplainRule(ctx, rule, content, req)
		response.EvaluatedCount++
		evaluation := ruleExplainEvaluation{Index: rule.Index, Rule: rule.Raw, Status: "miss", Reason: reason, Source: source}
		if matched {
			evaluation.Status = "matched"
			response.Matched = &evaluation
			response.MatchedRule, response.RuleIndex, response.Policy = rule.Raw, rule.Index, rule.Policy
			response.Chain, response.Candidates = clashExplainPolicy(content, rule.Policy)
			break
		}
		if len(response.Previous) >= previousLimit {
			response.Previous = response.Previous[1:]
		}
		response.Previous = append(response.Previous, evaluation)
	}
	if response.Matched == nil {
		response.Warnings = append(response.Warnings, "没有找到可确定命中的规则")
	}
	if response.EvaluatedCount > len(response.Previous)+1 {
		response.Warnings = append(response.Warnings, fmt.Sprintf("仅展示命中前最近 %d 条未命中规则", previousLimit))
	}
	if response.Policy != "" {
		expected := countryFromText(response.MatchedRule)
		if expected == "" {
			expected = countryFromText(response.Policy)
		}
		if expected == "" {
			expected = policyFilterCountry(content, response.Policy)
		}
		response.ExpectedCountry = expected
		stats, _ := models.GetNodeQualityStats(time.Now().Add(-24 * time.Hour))
		targetStats, _ := models.GetNodeTargetQualityStats(time.Now().Add(-24 * time.Hour))
		sceneStats, _ := models.GetNodeSceneQualityStats(time.Now().Add(-24 * time.Hour))
		targetKey, scene := "", ""
		if configured, err := models.EnabledEgressTargets(); err == nil {
			for _, item := range configured {
				if strings.EqualFold(strings.TrimPrefix(req.Target, "www."), strings.TrimPrefix(item.Domain, "www.")) {
					targetKey, scene = item.Key, item.Group
					break
				}
			}
		}
		candidates, count := choosePlanNode(sub.Nodes, expected, stats, planQualityMatrix{Targets: targetStats, Scenes: sceneStats}, targetKey, scene)
		response.CandidateCount = count
		if len(candidates) > 0 {
			selected := candidates[0]
			response.SelectedNode = &selected
		}
	}
	c.JSON(200, gin.H{"code": "00000", "data": response, "msg": "规则解释完成"})
}
