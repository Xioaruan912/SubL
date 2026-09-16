package node

import (
	"os"
	"strings"
	"testing"
)

func TestEncodeClashWithPerNodeFlags(t *testing.T) {
	tmpl := `
proxies:
proxy-groups:
  - name: 全部
    type: select
    proxies:
      - DIRECT
`
	path := writeTempYaml(t, tmpl)
	defer os.Remove(path)

	flagged := "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQxMjM0NTY3ODk@1.2.3.4:8388#FLAGGED"
	plain := "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQxMjM0NTY3ODk@5.6.7.8:8388#PLAIN"
	out, err := EncodeClashWithFlags([]string{flagged, plain}, map[string]NodeFlags{
		flagged: {UDP: true, TFO: true, SkipCertVerify: true},
	}, SqlConfig{Clash: path})
	if err != nil {
		t.Fatalf("EncodeClashWithFlags error: %v", err)
	}
	s := string(out)
	for _, want := range []string{"udp: true", "tfo: true", "skip-cert-verify: true"} {
		if !strings.Contains(s, want) {
			t.Fatalf("per-node flag %q missing from clash output:\n%s", want, s)
		}
	}
	if !strings.Contains(s, "FLAGGED") || !strings.Contains(s, "PLAIN") {
		t.Fatalf("nodes missing from clash output:\n%s", s)
	}
}
