package rulecenter

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

var supportedRuleTypes = map[string]bool{
	"DOMAIN": true, "DOMAIN-SUFFIX": true, "DOMAIN-KEYWORD": true,
	"IP-CIDR": true, "IP-CIDR6": true, "IP-ASN": true, "GEOIP": true,
	"PROCESS-NAME": true, "USER-AGENT": true,
}

func ParseRules(data []byte, format string) ([]NormalizedRule, []string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "yaml" || format == "yml" {
		var root struct { Payload []string `yaml:"payload"` }
		if err := yaml.Unmarshal(data, &root); err != nil {
			return nil, nil, err
		}
		return normalizeLines(root.Payload), warningsForLines(root.Payload), nil
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
	if err := s.Err(); err != nil { return nil, nil, err }
	return normalizeLines(lines), warningsForLines(lines), nil
}

func normalizeLines(lines []string) []NormalizedRule {
	out := make([]NormalizedRule, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) < 2 { continue }
		t := strings.ToUpper(strings.TrimSpace(parts[0]))
		v := strings.TrimSpace(parts[1])
		if t == "" || v == "" { continue }
		opts := make([]string, 0, len(parts)-2)
		for _, p := range parts[2:] {
			if p = strings.TrimSpace(p); p != "" { opts = append(opts, p) }
		}
		out = append(out, NormalizedRule{Type:t, Value:v, Options:opts})
	}
	return out
}

func warningsForLines(lines []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) < 2 { continue }
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
	for _, r := range rules { m[r.Type]++ }
	return m
}

func MatchDomain(rules []NormalizedRule, domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(domain, ".")))
	for _, r := range rules {
		v := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(r.Value, ".")))
		switch r.Type {
		case "DOMAIN":
			if domain == v { return true }
		case "DOMAIN-SUFFIX":
			if domain == v || strings.HasSuffix(domain, "."+v) { return true }
		case "DOMAIN-KEYWORD":
			if v != "" && strings.Contains(domain, v) { return true }
		}
	}
	return false
}
