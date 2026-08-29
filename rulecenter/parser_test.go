package rulecenter

import "testing"

func TestParseRulesYAMLAndMatch(t *testing.T) {
	data := []byte("payload:\n  - DOMAIN-SUFFIX,openai.com\n  - DOMAIN,chatgpt.com\n  - DOMAIN-KEYWORD,anthropic\n")
	rules, warnings, err := ParseRules(data, "yaml")
	if err != nil { t.Fatal(err) }
	if len(warnings) != 0 { t.Fatalf("unexpected warnings: %v", warnings) }
	if len(rules) != 3 { t.Fatalf("want 3 rules, got %d", len(rules)) }
	for _, domain := range []string{"openai.com", "api.openai.com", "chatgpt.com", "cdn.anthropic-assets.com"} {
		if !MatchDomain(rules, domain) { t.Fatalf("expected match for %s", domain) }
	}
	if MatchDomain(rules, "example.com") { t.Fatal("example.com should not match") }
}

func TestRuleUserAgentForKelee(t *testing.T) {
	if got := ruleUserAgent("https://rule.kelee.one/Clash/OpenAI.yaml"); got != "clash.meta" {
		t.Fatalf("unexpected kelee user-agent: %s", got)
	}
	if got := ruleUserAgent("https://raw.githubusercontent.com/a/b/main/x.yaml"); got == "clash.meta" {
		t.Fatalf("non-kelee URL should keep the generic user-agent")
	}
}

func TestParseRulesUnknownTypeWarning(t *testing.T) {
	rules, warnings, err := ParseRules([]byte("FOO,bar.example\nDOMAIN-SUFFIX,example.com\n"), "list")
	if err != nil { t.Fatal(err) }
	if len(rules) != 2 { t.Fatalf("want 2 normalized rows, got %d", len(rules)) }
	if len(warnings) != 1 { t.Fatalf("want one warning, got %v", warnings) }
}
