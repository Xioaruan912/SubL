package api

import (
	"context"
	"testing"
)

func TestEvaluateExplainRuleDomainPortNetwork(t *testing.T) {
	req := ruleExplainRequest{Target: "gemini.google.com", IP: "8.8.8.8", Port: 443, Protocol: "tcp"}
	cases := []struct {
		rule splitRule
		want bool
	}{
		{splitRule{Kind: "DOMAIN-SUFFIX", Domain: "google.com"}, true},
		{splitRule{Kind: "DOMAIN", Domain: "openai.com"}, false},
		{splitRule{Kind: "DST-PORT", Domain: "80/443/8443"}, true},
		{splitRule{Kind: "DST-PORT", Domain: "1000-2000"}, false},
		{splitRule{Kind: "NETWORK", Domain: "tcp"}, true},
		{splitRule{Kind: "NETWORK", Domain: "udp"}, false},
		{splitRule{Kind: "IP-CIDR", Domain: "8.8.8.0/24"}, true},
		{splitRule{Kind: "MATCH"}, true},
	}
	for _, tc := range cases {
		got, _, _ := evaluateExplainRule(context.Background(), tc.rule, "", req)
		if got != tc.want {
			t.Fatalf("%s %s: got %v want %v", tc.rule.Kind, tc.rule.Domain, got, tc.want)
		}
	}
}

func TestClashExplainPolicyChain(t *testing.T) {
	content := `
proxies:
  - name: JP-01
    type: ss
    server: 127.0.0.1
    port: 443
    cipher: aes-128-gcm
    password: x
proxy-groups:
  - name: Gemini
    type: select
    proxies: [JP Auto]
  - name: JP Auto
    type: select
    proxies: [JP-01]
rules: []
`
	chain, candidates := clashExplainPolicy(content, "Gemini")
	if len(chain) != 3 || chain[0] != "Gemini" || chain[1] != "JP Auto" || chain[2] != "JP-01" {
		t.Fatalf("unexpected chain: %#v", chain)
	}
	if len(candidates) != 2 || candidates[0] != "JP Auto" || candidates[1] != "JP-01" {
		t.Fatalf("unexpected candidates: %#v", candidates)
	}
}
