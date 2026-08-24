package node

import (
	"os"
	"strings"
	"testing"
)

// 验证构建器生成的模板（含 filter 分组）能被订阅生成正确使用
func TestBuilderTemplateWithFilter(t *testing.T) {
	// 用与 api 包 buildClashYAML 相同的结构生成模板
	tmpl := `
port: 7890
socks-port: 7891
allow-lan: true
mode: Rule
log-level: info
proxies: ~
proxy-groups:
  - name: 节点选择
    proxies:
      - DIRECT
    type: select
  - name: AI
    filter: (?i)US|USA|美国
    include-all-providers: true
    proxies:
      - DIRECT
    type: select
  - name: 自动
    interval: 300
    proxies:
      - DIRECT
    type: url-test
    url: http://www.gstatic.com/generate_204
rules:
  - MATCH,🐟 漏网之鱼
`
	f, err := os.CreateTemp("", "builder_tmpl_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(tmpl)
	f.Close()

	proxys := []Proxy{
		{Name: "US-01", Type: "ss"},
		{Name: "USA-02", Type: "ss"},
		{Name: "HK-01", Type: "ss"},
	}
	out, err := DecodeClash(proxys, f.Name())
	if err != nil {
		t.Fatalf("DecodeClash error: %v", err)
	}
	s := string(out)

	// AI 组（filter）只包含 US 节点
	aiIdx := strings.Index(s, "name: AI")
	if aiIdx < 0 {
		t.Fatalf("找不到 AI 组")
	}
	aiSec := s[aiIdx:]
	if next := strings.Index(aiSec[1:], "name:"); next >= 0 {
		aiSec = aiSec[:next+1]
	}
	for _, want := range []string{"US-01", "USA-02"} {
		if !strings.Contains(aiSec, want) {
			t.Errorf("AI 组应包含 %s", want)
		}
	}
	if strings.Contains(aiSec, "HK-01") {
		t.Errorf("AI 组不应包含 HK-01")
	}

	// 节点选择组（无 filter）包含全部
	allIdx := strings.Index(s, "name: 节点选择")
	if allIdx < 0 {
		t.Fatalf("找不到 节点选择 组")
	}
	allSec := s[allIdx:]
	if next := strings.Index(allSec[1:], "name:"); next >= 0 {
		allSec = allSec[:next+1]
	}
	for _, p := range proxys {
		if !strings.Contains(allSec, p.Name) {
			t.Errorf("节点选择组应包含 %s", p.Name)
		}
	}
}