package api

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"ppeelink/rulecenter"
)

type templatePreflightIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Line     int    `json:"line,omitempty"`
}

type templatePreflightRoute struct {
	Domain      string   `json:"domain"`
	Status      string   `json:"status"`
	MatchedRule string   `json:"matchedRule,omitempty"`
	RuleIndex   int      `json:"ruleIndex,omitempty"`
	Policy      string   `json:"policy,omitempty"`
	Chain       []string `json:"chain"`
	Candidates  []string `json:"candidates"`
	Notes       []string `json:"notes"`
}

type templateProtocolStat struct {
	Type    string `json:"type"`
	Network string `json:"network,omitempty"`
	Count   int    `json:"count"`
}

type templateCompatibility struct {
	Target string `json:"target"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type templatePreflightSummary struct {
	Errors        int `json:"errors"`
	Warnings      int `json:"warnings"`
	Infos         int `json:"infos"`
	Proxies       int `json:"proxies"`
	Groups        int `json:"groups"`
	Providers     int `json:"providers"`
	RuleProviders int `json:"ruleProviders"`
	Rules         int `json:"rules"`
}

type templatePreflightReport struct {
	Valid         bool                     `json:"valid"`
	Format        string                   `json:"format"`
	Coverage      string                   `json:"coverage"`
	Summary       templatePreflightSummary `json:"summary"`
	Issues        []templatePreflightIssue `json:"issues"`
	Routes        []templatePreflightRoute `json:"routes"`
	Protocols     []templateProtocolStat   `json:"protocols"`
	Compatibility []templateCompatibility  `json:"compatibility"`
}

type clashPreflightGroup struct {
	Name                string   `yaml:"name"`
	Type                string   `yaml:"type"`
	Proxies             []string `yaml:"proxies"`
	Use                 []string `yaml:"use"`
	Filter              string   `yaml:"filter"`
	IncludeAll          bool     `yaml:"include-all"`
	IncludeAllProxies   bool     `yaml:"include-all-proxies"`
	IncludeAllProviders bool     `yaml:"include-all-providers"`
}

type clashPreflightConfig struct {
	Proxies        []map[string]interface{}                 `yaml:"proxies"`
	ProxyGroups    []clashPreflightGroup                    `yaml:"proxy-groups"`
	ProxyProviders map[string]map[string]interface{}        `yaml:"proxy-providers"`
	RuleProviders  map[string]rulecenter.ProviderDefinition `yaml:"rule-providers"`
	Rules          []string                                 `yaml:"rules"`
}

type preflightPolicyGroup struct {
	Name       string
	Type       string
	Members    []string
	Dynamic    []string
	SourceLine int
}

var preflightBuiltins = map[string]bool{
	"DIRECT": true, "REJECT": true, "REJECT-DROP": true, "PASS": true,
	"COMPATIBLE": true, "GLOBAL": true,
}

var preflightRuleTypes = map[string]bool{
	"DOMAIN": true, "DOMAIN-SUFFIX": true, "DOMAIN-KEYWORD": true,
	"DOMAIN-WILDCARD": true, "DOMAIN-REGEX": true, "GEOSITE": true,
	"IP-CIDR": true, "IP-CIDR6": true, "IP-SUFFIX": true, "IP-ASN": true,
	"GEOIP": true, "SRC-GEOIP": true, "SRC-IP-ASN": true, "SRC-IP-CIDR": true,
	"DST-PORT": true, "SRC-PORT": true, "IN-PORT": true, "IN-TYPE": true,
	"IN-USER": true, "IN-NAME": true, "PROCESS-PATH": true,
	"PROCESS-PATH-REGEX": true, "PROCESS-NAME": true,
	"PROCESS-NAME-REGEX": true, "UID": true, "NETWORK": true, "DSCP": true,
	"RULE-SET": true, "AND": true, "OR": true, "NOT": true,
	"SUB-RULE": true, "MATCH": true, "FINAL": true,
}

var preflightNetworks = map[string]bool{
	"": true, "tcp": true, "ws": true, "http": true, "h2": true,
	"grpc": true, "httpupgrade": true,
}

var preflightProxyTypes = map[string]bool{
	"direct": true, "reject": true, "ss": true, "ssr": true,
	"vmess": true, "vless": true, "trojan": true, "snell": true,
	"socks5": true, "http": true, "hysteria": true, "hysteria2": true,
	"tuic": true, "wireguard": true,
}

func (report *templatePreflightReport) addIssue(severity, code, message string, line int) {
	for _, issue := range report.Issues {
		if issue.Severity == severity && issue.Code == code && issue.Message == message && issue.Line == line {
			return
		}
	}
	report.Issues = append(report.Issues, templatePreflightIssue{Severity: severity, Code: code, Message: message, Line: line})
	switch severity {
	case "error":
		report.Summary.Errors++
	case "warning":
		report.Summary.Warnings++
	default:
		report.Summary.Infos++
	}
}

func templateFormat(filename, text string) string {
	lowerName, lowerText := strings.ToLower(filepath.Base(filename)), strings.ToLower(text)
	if strings.HasSuffix(lowerName, ".yaml") || strings.HasSuffix(lowerName, ".yml") {
		return "clash"
	}
	if strings.Contains(lowerName, "loon") || strings.Contains(lowerText, "[remote rule]") || strings.Contains(lowerText, "[plugin]") {
		return "loon"
	}
	if strings.Contains(lowerName, "surge") {
		return "surge"
	}
	if strings.HasSuffix(lowerName, ".conf") {
		return "surge/loon"
	}
	return "generic"
}

func normalizeTemplateYAML(text string) string {
	return strings.ReplaceAll(text, "{{ nodes }}", "[]")
}

func preflightDomains(raw string) []string {
	values := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ';' || r == ' ' || r == '\t'
	})
	if len(values) == 0 {
		for _, target := range planTargets {
			values = append(values, target.domain)
		}
	}
	result, seen := make([]string, 0, len(values)), map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if parsed, err := url.Parse(value); err == nil && parsed.Hostname() != "" {
			value = strings.ToLower(parsed.Hostname())
		}
		value = strings.TrimSuffix(strings.TrimPrefix(value, "www."), ".")
		if value == "" || len(value) > 253 || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
		if len(result) == 20 {
			break
		}
	}
	return result
}

// PreflightTemp performs a read-only, publication-grade validation. It never
// writes the template, so the editor can safely call it before every save.
func PreflightTemp(c *gin.Context) {
	filename, text := strings.TrimSpace(c.PostForm("filename")), c.PostForm("text")
	if filename == "" || strings.TrimSpace(text) == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "模板文件名和内容不能为空"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
	defer cancel()
	report := buildTemplatePreflight(ctx, filename, text, preflightDomains(c.PostForm("domains")))
	c.JSON(200, gin.H{"code": "00000", "data": report, "msg": "发布前预检完成"})
}

func buildTemplatePreflight(ctx context.Context, filename, text string, domains []string) templatePreflightReport {
	format := templateFormat(filename, text)
	report := templatePreflightReport{
		Format: format, Coverage: "basic", Issues: []templatePreflightIssue{},
		Routes: []templatePreflightRoute{}, Protocols: []templateProtocolStat{},
		Compatibility: []templateCompatibility{},
	}
	if strings.TrimSpace(text) == "" {
		report.addIssue("error", "empty_template", "模板内容不能为空", 0)
		report.Valid = false
		return report
	}
	switch format {
	case "clash":
		report.Coverage = "full"
		analyzeClashPreflight(ctx, text, domains, &report)
	case "surge", "loon", "surge/loon":
		analyzeINIClientPreflight(text, domains, &report)
	default:
		report.addIssue("warning", "unsupported_format", "当前文件类型仅执行基础非空检查，建议使用 .yaml、.yml 或 .conf", 0)
		report.Compatibility = append(report.Compatibility, templateCompatibility{Target: "配置格式", Status: "warning", Detail: "未识别目标客户端，无法执行语义检查"})
	}
	report.Valid = report.Summary.Errors == 0
	return report
}

func analyzeClashPreflight(ctx context.Context, text string, domains []string, report *templatePreflightReport) {
	normalized := normalizeTemplateYAML(text)
	var cfg clashPreflightConfig
	if err := yaml.Unmarshal([]byte(normalized), &cfg); err != nil {
		report.addIssue("error", "yaml_syntax", err.Error(), 0)
		report.Compatibility = append(report.Compatibility, templateCompatibility{Target: "Mihomo", Status: "error", Detail: "YAML 无法解析"})
		return
	}
	report.Summary.Proxies = len(cfg.Proxies)
	report.Summary.Groups = len(cfg.ProxyGroups)
	report.Summary.Providers = len(cfg.ProxyProviders)
	report.Summary.RuleProviders = len(cfg.RuleProviders)
	report.Summary.Rules = len(cfg.Rules)

	proxyNames := map[string]bool{}
	protocolCounts := map[string]int{}
	for index, proxy := range cfg.Proxies {
		name := strings.TrimSpace(fmt.Sprint(proxy["name"]))
		proxyType := strings.ToLower(strings.TrimSpace(fmt.Sprint(proxy["type"])))
		network := strings.ToLower(strings.TrimSpace(fmt.Sprint(proxy["network"])))
		if name == "<nil>" {
			name = ""
		}
		if proxyType == "<nil>" {
			proxyType = ""
		}
		if network == "<nil>" {
			network = ""
		}
		if name == "" {
			report.addIssue("error", "proxy_name_missing", fmt.Sprintf("第 %d 个代理缺少 name", index+1), 0)
		} else if proxyNames[name] {
			report.addIssue("error", "proxy_name_duplicate", "代理名称重复: "+name, 0)
		} else {
			proxyNames[name] = true
		}
		if proxyType == "" {
			report.addIssue("error", "proxy_type_missing", fmt.Sprintf("代理 %s 缺少 type", displayName(name, index+1)), 0)
		} else if !preflightProxyTypes[proxyType] {
			report.addIssue("warning", "proxy_type_unknown", fmt.Sprintf("代理 %s 使用未识别类型 %s，请确认目标 Mihomo 版本支持", displayName(name, index+1), proxyType), 0)
		}
		if !preflightNetworks[network] {
			report.addIssue("error", "network_unknown", fmt.Sprintf("代理 %s 的 network=%s 不受当前生成/测试链支持", displayName(name, index+1), network), 0)
		}
		if network == "ws" {
			if _, ok := proxy["ws-opts"]; !ok {
				report.addIssue("warning", "ws_options_missing", fmt.Sprintf("代理 %s 使用 WS 但没有 ws-opts", displayName(name, index+1)), 0)
			}
		}
		if network == "grpc" {
			if _, ok := proxy["grpc-opts"]; !ok {
				report.addIssue("warning", "grpc_options_missing", fmt.Sprintf("代理 %s 使用 gRPC 但没有 grpc-opts", displayName(name, index+1)), 0)
			}
		}
		protocolCounts[proxyType+"\x00"+network]++
	}
	report.Protocols = protocolStats(protocolCounts)

	groups := map[string]preflightPolicyGroup{}
	for index, group := range cfg.ProxyGroups {
		name := strings.TrimSpace(group.Name)
		if name == "" {
			report.addIssue("error", "group_name_missing", fmt.Sprintf("第 %d 个策略组缺少 name", index+1), 0)
			continue
		}
		if _, exists := groups[name]; exists {
			report.addIssue("error", "group_name_duplicate", "策略组名称重复: "+name, 0)
			continue
		}
		dynamic := make([]string, 0, 3)
		for _, provider := range group.Use {
			if _, ok := cfg.ProxyProviders[provider]; !ok {
				report.addIssue("error", "proxy_provider_missing", fmt.Sprintf("策略组 %s 引用了不存在的 proxy-provider: %s", name, provider), 0)
			}
			dynamic = append(dynamic, "provider:"+provider)
		}
		if group.Filter != "" {
			if _, err := regexp.Compile(group.Filter); err != nil {
				report.addIssue("error", "group_filter_invalid", fmt.Sprintf("策略组 %s 的 filter 正则无效: %v", name, err), 0)
			} else {
				dynamic = append(dynamic, "filter 匹配节点")
			}
		} else if !strings.EqualFold(group.Type, "relay") {
			// DecodeClash appends subscription nodes to every non-relay group
			// without a filter, even when the template already has fixed members.
			dynamic = append(dynamic, "订阅构建时注入全部节点")
		}
		if group.IncludeAll || group.IncludeAllProxies || group.IncludeAllProviders {
			dynamic = append(dynamic, "动态包含节点")
		}
		if len(group.Proxies) == 0 && len(dynamic) == 0 && !strings.EqualFold(group.Type, "relay") {
			// SubLinkX fills subscription nodes into empty Clash groups at build
			// time. Report this explicitly instead of producing a false error.
			dynamic = append(dynamic, "订阅构建时注入节点")
		}
		groups[name] = preflightPolicyGroup{Name: name, Type: group.Type, Members: append([]string(nil), group.Proxies...), Dynamic: dynamic}
	}

	for _, group := range groups {
		for _, member := range group.Members {
			if isBuiltinPolicy(member) || proxyNames[member] {
				continue
			}
			if _, ok := groups[member]; !ok {
				report.addIssue("warning", "group_member_dynamic", fmt.Sprintf("策略组 %s 的成员 %s 不在静态模板中；需在订阅生成后复核", group.Name, member), 0)
			}
		}
	}
	checkPolicyCycles(groups, report)

	rules := parseSplitRules(normalized)
	if len(cfg.Rules) == 0 {
		report.addIssue("warning", "rules_empty", "模板没有 rules，流量不会按模板规则分流", 0)
	}
	for index, raw := range cfg.Rules {
		parts := splitTopLevelRule(strings.TrimSpace(raw))
		if len(parts) < 2 {
			report.addIssue("error", "rule_invalid", fmt.Sprintf("第 %d 条规则格式无效: %s", index+1, raw), 0)
			continue
		}
		kind := strings.ToUpper(strings.TrimSpace(parts[0]))
		if !preflightRuleTypes[kind] {
			report.addIssue("warning", "rule_type_unknown", fmt.Sprintf("第 %d 条规则使用未识别类型 %s", index+1, kind), 0)
		}
		if (kind == "MATCH" || kind == "FINAL") && index != len(cfg.Rules)-1 {
			report.addIssue("warning", "rules_after_match", fmt.Sprintf("MATCH/FINAL 位于第 %d 条，后续 %d 条规则永远不会命中", index+1, len(cfg.Rules)-index-1), 0)
		}
	}
	for _, rule := range rules {
		if rule.Policy == "" {
			report.addIssue("error", "rule_policy_missing", fmt.Sprintf("第 %d 条规则缺少策略: %s", rule.Index, rule.Raw), 0)
			continue
		}
		if rule.Kind == "RULE-SET" {
			if _, ok := cfg.RuleProviders[rule.Domain]; !ok {
				report.addIssue("error", "rule_provider_missing", fmt.Sprintf("第 %d 条规则引用了不存在的 rule-provider: %s", rule.Index, rule.Domain), 0)
			}
		}
		if !isBuiltinPolicy(rule.Policy) && !proxyNames[rule.Policy] {
			if _, ok := groups[rule.Policy]; !ok {
				report.addIssue("warning", "rule_policy_dynamic", fmt.Sprintf("规则策略 %s 不在静态模板中；若它是订阅节点，请在生成后复核", rule.Policy), 0)
			}
		}
	}
	report.Routes = explainClashDomains(ctx, normalized, rules, cfg.RuleProviders, groups, proxyNames, domains)

	status, detail := "pass", "Clash YAML、策略引用和可识别传输检查通过"
	if report.Summary.Errors > 0 {
		status, detail = "error", "存在会阻止安全发布的结构或引用错误"
	} else if report.Summary.Warnings > 0 {
		status, detail = "warning", "可生成，但存在需要确认的动态引用或兼容提示"
	}
	report.Compatibility = append(report.Compatibility,
		templateCompatibility{Target: "Mihomo", Status: status, Detail: detail},
		templateCompatibility{Target: "SublinkX 实测引擎", Status: runtimeTestCompatibility(cfg.Proxies), Detail: runtimeTestCompatibilityDetail(cfg.Proxies)},
	)
}

func displayName(name string, index int) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("#%d", index)
}

func protocolStats(counts map[string]int) []templateProtocolStat {
	result := make([]templateProtocolStat, 0, len(counts))
	for key, count := range counts {
		parts := strings.SplitN(key, "\x00", 2)
		item := templateProtocolStat{Type: parts[0], Count: count}
		if len(parts) == 2 {
			item.Network = parts[1]
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Type == result[j].Type {
			return result[i].Network < result[j].Network
		}
		return result[i].Type < result[j].Type
	})
	return result
}

func runtimeTestCompatibility(proxies []map[string]interface{}) string {
	for _, proxy := range proxies {
		t := strings.ToLower(strings.TrimSpace(fmt.Sprint(proxy["type"])))
		if t == "ssr" || t == "hysteria" {
			return "warning"
		}
	}
	return "pass"
}

func runtimeTestCompatibilityDetail(proxies []map[string]interface{}) string {
	unsupported := map[string]bool{}
	for _, proxy := range proxies {
		t := strings.ToLower(strings.TrimSpace(fmt.Sprint(proxy["type"])))
		if t == "ssr" || t == "hysteria" {
			unsupported[t] = true
		}
	}
	if len(unsupported) == 0 {
		return "已识别代理未发现与当前 sing-box 实测转换层冲突的协议"
	}
	values := make([]string, 0, len(unsupported))
	for value := range unsupported {
		values = append(values, value)
	}
	sort.Strings(values)
	return "Mihomo 可使用，但当前节点实测转换层不支持: " + strings.Join(values, ", ")
}

func isBuiltinPolicy(value string) bool {
	return preflightBuiltins[strings.ToUpper(strings.TrimSpace(value))]
}

func checkPolicyCycles(groups map[string]preflightPolicyGroup, report *templatePreflightReport) {
	state, stack := map[string]int{}, []string{}
	var visit func(string)
	visit = func(name string) {
		if state[name] == 2 {
			return
		}
		if state[name] == 1 {
			start := 0
			for index, item := range stack {
				if item == name {
					start = index
					break
				}
			}
			cycle := append(append([]string(nil), stack[start:]...), name)
			report.addIssue("error", "group_cycle", "策略组循环引用: "+strings.Join(cycle, " → "), 0)
			return
		}
		state[name] = 1
		stack = append(stack, name)
		for _, member := range groups[name].Members {
			if _, ok := groups[member]; ok {
				visit(member)
			}
		}
		stack = stack[:len(stack)-1]
		state[name] = 2
	}
	for name := range groups {
		visit(name)
	}
}

type providerPreflightCache struct {
	rules []rulecenter.NormalizedRule
	err   error
}

func explainClashDomains(ctx context.Context, content string, rules []splitRule, definitions map[string]rulecenter.ProviderDefinition, groups map[string]preflightPolicyGroup, proxies map[string]bool, domains []string) []templatePreflightRoute {
	results := make([]templatePreflightRoute, 0, len(domains))
	cache := map[string]providerPreflightCache{}
	for _, domain := range domains {
		route := templatePreflightRoute{Domain: domain, Status: "unmatched", Chain: []string{}, Candidates: []string{}, Notes: []string{}}
		partial := false
		for _, rule := range rules {
			matched := false
			switch rule.Kind {
			case "DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD", "DOMAIN-WILDCARD", "DOMAIN-REGEX":
				matched = inlineDomainRuleMatch(rule, domain)
			case "RULE-SET":
				definition, exists := definitions[rule.Domain]
				if !exists {
					partial = true
					continue
				}
				if strings.EqualFold(definition.Behavior, "ipcidr") {
					continue
				}
				entry, ok := cache[rule.Domain]
				if !ok {
					entry.rules, _, entry.err = rulecenter.ResolveProviderRules(ctx, content, rule.Domain)
					cache[rule.Domain] = entry
				}
				if entry.err != nil {
					partial = true
					route.Notes = appendUnique(route.Notes, fmt.Sprintf("rule-provider %s 无法验证: %v", rule.Domain, entry.err))
					continue
				}
				matched = rulecenter.MatchDomain(entry.rules, domain)
			case "AND", "OR", "NOT", "SUB-RULE", "GEOSITE":
				partial = true
				route.Notes = appendUnique(route.Notes, fmt.Sprintf("第 %d 条 %s 需要更多请求上下文，域名模式未计算", rule.Index, rule.Kind))
			case "MATCH", "FINAL":
				matched = true
			}
			if !matched {
				continue
			}
			route.Status = "matched"
			if partial {
				route.Status = "partial"
			}
			route.MatchedRule, route.RuleIndex, route.Policy = rule.Raw, rule.Index, rule.Policy
			route.Chain, route.Candidates = tracePolicy(rule.Policy, groups, proxies)
			break
		}
		if route.MatchedRule == "" && partial {
			route.Status = "partial"
		}
		results = append(results, route)
	}
	return results
}

func inlineDomainRuleMatch(rule splitRule, domain string) bool {
	value := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(rule.Domain, ".")))
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	switch rule.Kind {
	case "DOMAIN":
		return domain == value
	case "DOMAIN-SUFFIX":
		return domain == value || strings.HasSuffix(domain, "."+value)
	case "DOMAIN-KEYWORD":
		return value != "" && strings.Contains(domain, value)
	case "DOMAIN-WILDCARD":
		pattern := strings.ReplaceAll(regexp.QuoteMeta(value), `\*`, ".*")
		matched, _ := regexp.MatchString("^"+pattern+"$", domain)
		return matched
	case "DOMAIN-REGEX":
		matched, _ := regexp.MatchString(rule.Domain, domain)
		return matched
	}
	return false
}

func tracePolicy(policy string, groups map[string]preflightPolicyGroup, proxies map[string]bool) ([]string, []string) {
	chain, candidates, seen := []string{}, []string{}, map[string]bool{}
	current := strings.TrimSpace(policy)
	for current != "" {
		chain = append(chain, current)
		if isBuiltinPolicy(current) || proxies[current] {
			break
		}
		group, ok := groups[current]
		if !ok {
			chain = append(chain, "生成后解析")
			break
		}
		if seen[current] {
			chain = append(chain, "循环引用")
			break
		}
		seen[current] = true
		candidates = append(candidates, group.Members...)
		candidates = append(candidates, group.Dynamic...)
		if len(group.Members) == 0 {
			if len(group.Dynamic) > 0 {
				chain = append(chain, group.Dynamic[0])
			}
			break
		}
		current = group.Members[0]
	}
	return chain, uniqueStrings(candidates)
}

type iniPreflightLine struct {
	Text string
	Line int
}

func analyzeINIClientPreflight(text string, domains []string, report *templatePreflightReport) {
	sections := map[string][]iniPreflightLine{}
	current := ""
	for index, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			match := regexp.MustCompile(`^\[([^]]+)\]$`).FindStringSubmatch(line)
			if match == nil {
				report.addIssue("error", "section_invalid", fmt.Sprintf("第 %d 行段落标题无效: %s", index+1, line), index+1)
				continue
			}
			current = strings.ToLower(strings.TrimSpace(match[1]))
			if _, ok := sections[current]; !ok {
				sections[current] = []iniPreflightLine{}
			}
			continue
		}
		if current == "" {
			report.addIssue("warning", "content_outside_section", fmt.Sprintf("第 %d 行不属于任何段落", index+1), index+1)
			continue
		}
		sections[current] = append(sections[current], iniPreflightLine{Text: line, Line: index + 1})
	}
	if _, ok := sections["proxy group"]; !ok {
		report.addIssue("error", "proxy_group_section_missing", "缺少 [Proxy Group] 段落", 0)
	}
	if _, ok := sections["rule"]; !ok {
		report.addIssue("error", "rule_section_missing", "缺少 [Rule] 段落", 0)
	}

	proxyNames, protocolCounts := map[string]bool{}, map[string]int{}
	for _, line := range sections["proxy"] {
		name, value, ok := splitAssignment(line.Text)
		if !ok {
			report.addIssue("error", "proxy_invalid", fmt.Sprintf("第 %d 行代理格式无效", line.Line), line.Line)
			continue
		}
		if proxyNames[name] {
			report.addIssue("error", "proxy_name_duplicate", "代理名称重复: "+name, line.Line)
		}
		proxyNames[name] = true
		parts := splitTopLevelRule(value)
		proxyType := "unknown"
		if len(parts) > 0 {
			proxyType = strings.ToLower(strings.TrimSpace(parts[0]))
		}
		network := ""
		for _, part := range parts {
			lower := strings.ToLower(strings.TrimSpace(part))
			if strings.HasPrefix(lower, "transport=") {
				network = strings.TrimSpace(strings.TrimPrefix(lower, "transport="))
			} else if lower == "ws=true" {
				network = "ws"
			}
		}
		protocolCounts[proxyType+"\x00"+network]++
	}
	report.Summary.Proxies = len(proxyNames)
	report.Protocols = protocolStats(protocolCounts)

	groups := map[string]preflightPolicyGroup{}
	for _, line := range sections["proxy group"] {
		name, value, ok := splitAssignment(line.Text)
		if !ok {
			report.addIssue("error", "group_invalid", fmt.Sprintf("第 %d 行策略组格式无效", line.Line), line.Line)
			continue
		}
		if _, exists := groups[name]; exists {
			report.addIssue("error", "group_name_duplicate", "策略组名称重复: "+name, line.Line)
			continue
		}
		parts := splitTopLevelRule(value)
		group := preflightPolicyGroup{Name: name, SourceLine: line.Line}
		if len(parts) > 0 {
			group.Type = strings.TrimSpace(parts[0])
		}
		for _, candidate := range parts[1:] {
			candidate = strings.TrimSpace(candidate)
			lower := strings.ToLower(candidate)
			if candidate == "" || strings.Contains(candidate, "=") || strings.HasPrefix(lower, "filter(") {
				continue
			}
			group.Members = append(group.Members, candidate)
		}
		if len(group.Members) == 0 {
			group.Dynamic = []string{"订阅构建时注入节点"}
		}
		groups[name] = group
	}
	report.Summary.Groups = len(groups)
	checkPolicyCycles(groups, report)

	rules := make([]splitRule, 0, len(sections["rule"]))
	for index, line := range sections["rule"] {
		parts := splitTopLevelRule(line.Text)
		if len(parts) < 2 {
			report.addIssue("error", "rule_invalid", fmt.Sprintf("第 %d 行规则格式无效", line.Line), line.Line)
			continue
		}
		kind := strings.ToUpper(strings.TrimSpace(parts[0]))
		policyIndex := 2
		if kind == "MATCH" || kind == "FINAL" {
			policyIndex = 1
		}
		if policyIndex >= len(parts) {
			report.addIssue("error", "rule_policy_missing", fmt.Sprintf("第 %d 行规则缺少策略", line.Line), line.Line)
			continue
		}
		domain := ""
		if len(parts) > 2 {
			domain = strings.TrimSpace(parts[1])
		}
		rule := splitRule{Kind: kind, Domain: domain, Policy: strings.TrimSpace(parts[policyIndex]), Raw: line.Text, Index: index + 1}
		rules = append(rules, rule)
		if (kind == "MATCH" || kind == "FINAL") && index != len(sections["rule"])-1 {
			report.addIssue("warning", "rules_after_match", fmt.Sprintf("第 %d 行 FINAL/MATCH 后仍有规则，后续内容不会命中", line.Line), line.Line)
		}
	}
	report.Summary.Rules = len(rules)
	for _, rule := range rules {
		if !isBuiltinPolicy(rule.Policy) && !proxyNames[rule.Policy] {
			if _, ok := groups[rule.Policy]; !ok {
				report.addIssue("warning", "rule_policy_dynamic", fmt.Sprintf("规则策略 %s 不在静态模板中；需在订阅生成后复核", rule.Policy), 0)
			}
		}
	}
	for _, line := range sections["remote rule"] {
		lower := strings.ToLower(line.Text)
		position := strings.Index(lower, "policy=")
		if position < 0 {
			report.addIssue("warning", "remote_rule_policy_missing", fmt.Sprintf("第 %d 行远程规则没有 policy", line.Line), line.Line)
			continue
		}
		policy := strings.TrimSpace(strings.SplitN(line.Text[position+len("policy="):], ",", 2)[0])
		if !isBuiltinPolicy(policy) && !proxyNames[policy] {
			if _, ok := groups[policy]; !ok {
				report.addIssue("warning", "remote_rule_policy_dynamic", fmt.Sprintf("远程规则策略 %s 不在静态模板中", policy), line.Line)
			}
		}
	}
	report.Summary.RuleProviders = len(sections["remote rule"])
	report.Routes = explainInlineDomains(rules, groups, proxyNames, domains, len(sections["remote rule"]) > 0)

	status, detail := "pass", "段落、策略组和本地规则结构检查通过"
	if report.Summary.Errors > 0 {
		status, detail = "error", "存在会阻止安全发布的段落或规则错误"
	} else if report.Summary.Warnings > 0 {
		status, detail = "warning", "基础结构可用，但动态或远程引用需要生成后复核"
	}
	report.Compatibility = append(report.Compatibility, templateCompatibility{Target: strings.ToUpper(report.Format), Status: status, Detail: detail})
}

func splitAssignment(line string) (string, string, bool) {
	position := strings.Index(line, "=")
	if position <= 0 || position == len(line)-1 {
		return "", "", false
	}
	name, value := strings.TrimSpace(line[:position]), strings.TrimSpace(line[position+1:])
	return name, value, name != "" && value != ""
}

func explainInlineDomains(rules []splitRule, groups map[string]preflightPolicyGroup, proxies map[string]bool, domains []string, hasRemote bool) []templatePreflightRoute {
	result := make([]templatePreflightRoute, 0, len(domains))
	for _, domain := range domains {
		route := templatePreflightRoute{Domain: domain, Status: "unmatched", Chain: []string{}, Candidates: []string{}, Notes: []string{}}
		for _, rule := range rules {
			matched := inlineDomainRuleMatch(rule, domain)
			if rule.Kind == "MATCH" || rule.Kind == "FINAL" {
				matched = true
			}
			if !matched {
				continue
			}
			route.Status, route.MatchedRule, route.RuleIndex, route.Policy = "matched", rule.Raw, rule.Index, rule.Policy
			route.Chain, route.Candidates = tracePolicy(rule.Policy, groups, proxies)
			if hasRemote && (rule.Kind == "MATCH" || rule.Kind == "FINAL") {
				route.Status = "partial"
				route.Notes = append(route.Notes, "模板含远程规则；当前仅能确认本地规则与最终兜底，真实命中需下载远程规则后验证")
			}
			break
		}
		result = append(result, route)
	}
	return result
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
