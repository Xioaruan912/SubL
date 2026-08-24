package api

import (
	"strings"
	"testing"
)

func TestBuildClashYAML(t *testing.T) {
	req := BuilderRequest{
		Filename:  "test_clash.yaml",
		Port:      7890,
		SocksPort: 7891,
		AllowLan:  true,
		Mode:      "Rule",
		TestURL:   "http://www.gstatic.com/generate_204",
		Interval:  300,
		Groups: []BuilderGroup{
			{Name: "🔰 节点选择", Type: "select"},
			{Name: "AI", Type: "select", Filter: "(?i)US|USA|美国", IncludeAllProviders: true},
			{Name: "自动选择", Type: "url-test"},
			{Name: "故障转移", Type: "fallback"},
		},
	}
	out, err := buildClashYAML(req)
	if err != nil {
		t.Fatalf("buildClashYAML error: %v", err)
	}
	t.Logf("yaml:\n%s", out)

	// 校验关键字段
	for _, s := range []string{"port: 7890", "socks-port: 7891", "proxies: ~", "name: AI", "filter: (?i)US|USA|美国", "include-all-providers: true", "type: url-test", "type: fallback", "url: http://www.gstatic.com/generate_204"} {
		if !strings.Contains(out, s) {
			t.Errorf("yaml 缺少: %s\n%s", s, out)
		}
	}
}