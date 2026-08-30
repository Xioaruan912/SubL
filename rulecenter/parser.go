package rulecenter

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var supportedRuleTypes = map[string]bool{
	"DOMAIN": true, "DOMAIN-SUFFIX": true, "DOMAIN-KEYWORD": true,
	"IP-CIDR": true, "IP-CIDR6": true, "IP-ASN": true, "GEOIP": true,
	"PROCESS-NAME": true, "USER-AGENT": true,
}

func ParseRules(data []byte, format string) ([]NormalizedRule, []string, error) {
	return ParseRulesWithBehavior(data, format, "classical")
}

// ParseRulesWithBehavior normalizes all Clash rule-provider behaviours into
// one representation. Domain and IPCIDR providers contain bare payload values
// instead of classical "TYPE,value" lines, so treating every provider as
// classical silently drops valid rules such as "+.example.com".
func ParseRulesWithBehavior(data []byte, format, behavior string) ([]NormalizedRule, []string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	behavior = strings.ToLower(strings.TrimSpace(behavior))
	if behavior == "" {
		behavior = "classical"
	}
	if format == "mrs" {
		return nil, nil, fmt.Errorf("暂不支持解析 MRS 二进制规则集")
	}
	if format == "yaml" || format == "yml" {
		var root struct {
			Payload []string `yaml:"payload"`
		}
		if err := yaml.Unmarshal(data, &root); err != nil {
			return nil, nil, err
		}
		return normalizeProviderLines(root.Payload, behavior), warningsForProviderLines(root.Payload, behavior), nil
	}
	lines := make([]string, 0)
	s := bufio.NewScanner(bytes.NewReader(data))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		lines = append(lines, line)
	}
	if err := s.Err(); err != nil {
		return nil, nil, err
	}
	return normalizeProviderLines(lines, behavior), warningsForProviderLines(lines, behavior), nil
}

func normalizeProviderLines(lines []string, behavior string) []NormalizedRule {
	switch behavior {
	case "domain":
		out := make([]NormalizedRule, 0, len(lines))
		for _, raw := range lines {
			value := strings.TrimSpace(raw)
			if value == "" {
				continue
			}
			ruleType := "DOMAIN"
			if strings.HasPrefix(value, "+.") {
				ruleType, value = "DOMAIN-SUFFIX", strings.TrimPrefix(value, "+.")
			} else if strings.HasPrefix(value, ".") {
				ruleType, value = "DOMAIN-SUFFIX", strings.TrimPrefix(value, ".")
			} else if strings.Contains(value, "*") {
				ruleType = "DOMAIN-WILDCARD"
			}
			if value != "" {
				out = append(out, NormalizedRule{Type: ruleType, Value: value})
			}
		}
		return out
	case "ipcidr":
		out := make([]NormalizedRule, 0, len(lines))
		for _, raw := range lines {
			value := strings.TrimSpace(raw)
			if value == "" {
				continue
			}
			ruleType := "IP-CIDR"
			if strings.Contains(value, ":") {
				ruleType = "IP-CIDR6"
			}
			out = append(out, NormalizedRule{Type: ruleType, Value: value})
		}
		return out
	default:
		return normalizeLines(lines)
	}
}

func warningsForProviderLines(lines []string, behavior string) []string {
	if behavior == "domain" || behavior == "ipcidr" {
		return nil
	}
	return warningsForLines(lines)
}

func normalizeLines(lines []string) []NormalizedRule {
	out := make([]NormalizedRule, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		t := strings.ToUpper(strings.TrimSpace(parts[0]))
		v := strings.TrimSpace(parts[1])
		if t == "" || v == "" {
			continue
		}
		opts := make([]string, 0, len(parts)-2)
		for _, p := range parts[2:] {
			if p = strings.TrimSpace(p); p != "" {
				opts = append(opts, p)
			}
		}
		out = append(out, NormalizedRule{Type: t, Value: v, Options: opts})
	}
	return out
}

func warningsForLines(lines []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		t := strings.ToUpper(strings.TrimSpace(parts[0]))
		if !supportedRuleTypes[t] && !seen[t] {
			seen[t] = true
			out = append(out, fmt.Sprintf("暂未标准化规则类型: %s", t))
		}
	}
	return out
}

func CountTypes(rules []NormalizedRule) map[string]int {
	m := map[string]int{}
	for _, r := range rules {
		m[r.Type]++
	}
	return m
}

func MatchDomain(rules []NormalizedRule, domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(domain, ".")))
	for _, r := range rules {
		v := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(r.Value, ".")))
		switch r.Type {
		case "DOMAIN":
			if domain == v {
				return true
			}
		case "DOMAIN-SUFFIX":
			if domain == v || strings.HasSuffix(domain, "."+v) {
				return true
			}
		case "DOMAIN-KEYWORD":
			if v != "" && strings.Contains(domain, v) {
				return true
			}
		case "DOMAIN-WILDCARD":
			pattern := regexp.QuoteMeta(v)
			pattern = strings.ReplaceAll(pattern, `\*`, ".*")
			if matched, _ := regexp.MatchString("^"+pattern+"$", domain); matched {
				return true
			}
		}
	}
	return false
}
