package node

import (
	"os"
	"strings"
	"testing"
)

func writeTempYaml(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "clash_filter_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestDecodeClashFilter(t *testing.T) {
	proxys := []Proxy{
		{Name: "🇺🇸 US-01", Type: "ss"},
		{Name: "🇺🇸 USA-02", Type: "ss"},
		{Name: "🇭🇰 HK-01", Type: "ss"},
		{Name: "🇭🇰 香港-02", Type: "ss"},
		{Name: "🇯🇵 JP-01", Type: "ss"},
	}

	tmpl := `
proxies:
proxy-groups:
  - name: AI
    type: select
    include-all-providers: true
    filter: "(?i)US|USA|United States|美国"
    proxies:
      - DIRECT
  - name: 全部
    type: select
    proxies:
      - DIRECT
`
	path := writeTempYaml(t, tmpl)
	defer os.Remove(path)

	out, err := DecodeClash(proxys, path)
	if err != nil {
		t.Fatalf("DecodeClash error: %v", err)
	}
	s := string(out)

	// 定位 AI 组的 proxies 列表段，只对该段做断言
	aiIdx := strings.Index(s, "name: AI")
	if aiIdx < 0 {
		t.Fatalf("输出中找不到 AI 组:\n%s", s)
	}
	// AI 组后到下一个 name: 或文件末尾之间即为 AI 组的 proxies
	aiSec := s[aiIdx:]
	if next := strings.Index(aiSec[1:], "name:"); next >= 0 {
		aiSec = aiSec[:next+1]
	}

	contains := func(sub string) bool { return strings.Contains(aiSec, sub) }

	// AI 组应只包含匹配 US/USA/美国 的节点
	if !contains("US-01") || !contains("USA-02") {
		t.Errorf("AI 组应包含 US-01/USA-02，AI组实际输出:\n%s", aiSec)
	}
	if contains("HK-01") || contains("JP-01") {
		t.Errorf("AI 组不应包含未匹配节点（HK-01/JP-01），AI组实际输出:\n%s", aiSec)
	}

	// 无 filter 的"全部"组应包含所有节点（全量填充兼容）
	allIdx := strings.Index(s, "name: 全部")
	if allIdx < 0 {
		t.Fatalf("输出中找不到 全部 组:\n%s", s)
	}
	allSec := s[allIdx:]
	if next := strings.Index(allSec[1:], "name:"); next >= 0 {
		allSec = allSec[:next+1]
	}
	for _, p := range proxys {
		name := p.Name
		idx := strings.LastIndexAny(name, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-")
		key := name[idx:]
		if !strings.Contains(allSec, key) {
			t.Errorf("无 filter 全部组应包含 %s (key=%s)，全部组实际输出:\n%s", name, key, allSec)
		}
	}
}
