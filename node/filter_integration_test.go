package node

import (
	"os"
	"strings"
	"testing"
)

// 通过完整 EncodeClash 流程验证 filter 正则匹配
func TestEncodeClashFilterIntegration(t *testing.T) {
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

	urls := []string{
		"ss://YWVzLTI1Ni1nY206cGFzc3dvcmQxMjM0NTY3ODk@1.2.3.4:8388#US-节点01",
		"ss://YWVzLTI1Ni1nY206cGFzc3dvcmQxMjM0NTY3ODk@5.6.7.8:8388#HK-节点02",
		"ss://YWVzLTI1Ni1nY206cGFzc3dvcmQxMjM0NTY3ODk@9.10.11.12:8388#JP-节点03",
	}

	out, err := EncodeClash(urls, SqlConfig{Clash: path})
	if err != nil {
		t.Fatalf("EncodeClash error: %v", err)
	}
	s := string(out)

	aiIdx := strings.Index(s, "name: AI")
	if aiIdx < 0 {
		t.Fatalf("找不到 AI 组:\n%s", s)
	}
	aiSec := s[aiIdx:]
	if next := strings.Index(aiSec[1:], "name:"); next >= 0 {
		aiSec = aiSec[:next+1]
	}

	if !strings.Contains(aiSec, "US-节点01") {
		t.Errorf("AI 组应包含 US-节点01，AI组:\n%s", aiSec)
	}
	if strings.Contains(aiSec, "HK-节点02") || strings.Contains(aiSec, "JP-节点03") {
		t.Errorf("AI 组不应包含 HK/JP 节点，AI组:\n%s", aiSec)
	}
}
