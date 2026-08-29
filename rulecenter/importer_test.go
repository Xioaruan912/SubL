package rulecenter

import (
	"strings"
	"testing"
)

func TestImportClashProviderBeforeMatch(t *testing.T) {
	input := "port: 7890\nrules:\n  - DOMAIN-SUFFIX,example.com,DIRECT\n  - MATCH,Proxy\n"
	res, err := ImportClashProvider(input, ImportOptions{ProviderName:"OpenAI", URL:"https://example.com/OpenAI.yaml", Policy:"AI", ConflictPolicy:"keep"})
	if err != nil { t.Fatal(err) }
	if !strings.Contains(res.Text, "rule-providers:") { t.Fatal("provider block missing") }
	idxRule := strings.Index(res.Text, "RULE-SET,OpenAI,AI")
	idxMatch := strings.Index(res.Text, "MATCH,Proxy")
	if idxRule < 0 || idxMatch < 0 || idxRule > idxMatch { t.Fatalf("RULE-SET must be before MATCH:\n%s", res.Text) }
}

func TestImportClashProviderDoesNotDuplicateRule(t *testing.T) {
	input := "rule-providers:\n  OpenAI:\n    type: http\n    behavior: classical\n    url: https://example.com/OpenAI.yaml\nrules:\n  - RULE-SET,OpenAI,AI\n  - MATCH,Proxy\n"
	res, err := ImportClashProvider(input, ImportOptions{ProviderName:"OpenAI", URL:"https://example.com/OpenAI.yaml", Policy:"AI"})
	if err != nil { t.Fatal(err) }
	if strings.Count(res.Text, "RULE-SET,OpenAI,AI") != 1 { t.Fatalf("rule duplicated:\n%s", res.Text) }
}

func TestImportClashProviderWritesProxyAndProviderOptions(t *testing.T) {
	input := "proxy-groups:\n  - name: 日常使用\n    type: select\nrules:\n  - MATCH,日常使用\n"
	res, err := ImportClashProvider(input, ImportOptions{ProviderName:"ESET_China", URL:"https://kelee.one/Tool/Clash/Rule/ESET_China.yaml", Policy:"DIRECT", Proxy:"日常使用"})
	if err != nil { t.Fatal(err) }
	for _, want := range []string{"interval: 3600", "format: yaml", "proxy: 日常使用", "path: ./rules/ESET_China.yaml", "url: https://kelee.one/Tool/Clash/Rule/ESET_China.yaml"} {
		if !strings.Contains(res.Text, want) { t.Fatalf("missing %q in:\n%s", want, res.Text) }
	}
}

func TestImportClashProviderAllowsEmptyProxy(t *testing.T) {
	input := "rules:\n  - MATCH,Proxy\n"
	res, err := ImportClashProvider(input, ImportOptions{ProviderName:"OpenAI", URL:"https://example.com/OpenAI.yaml", Policy:"AI", Proxy:""})
	if err != nil { t.Fatal(err) }
	if !strings.Contains(res.Text, "proxy: \"\"") { t.Fatalf("empty proxy must be emitted explicitly:\n%s", res.Text) }
}

func TestImportClashProviderConflictUpdateURL(t *testing.T) {
	input := "rule-providers:\n  OpenAI:\n    type: http\n    behavior: classical\n    url: https://old.example/OpenAI.yaml\nrules:\n  - MATCH,Proxy\n"
	res, err := ImportClashProvider(input, ImportOptions{ProviderName:"OpenAI", URL:"https://new.example/OpenAI.yaml", Policy:"AI", ConflictPolicy:"update-url"})
	if err != nil { t.Fatal(err) }
	if !res.Conflict || !strings.Contains(res.Text, "https://new.example/OpenAI.yaml") { t.Fatalf("conflict update failed:\n%s", res.Text) }
}
