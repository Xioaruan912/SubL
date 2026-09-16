package node

import (
	"strings"
	"testing"
	"time"
)

func scriptFixtures(t *testing.T) []Outbound {
	t.Helper()
	links := []string{
		EncodeSSURL(Ss{Param: Param{Cipher: "aes-256-gcm", Password: "pw"}, Server: "1.2.3.4", Port: 8388, Name: "HK-01"}),
		EncodeSSURL(Ss{Param: Param{Cipher: "aes-256-gcm", Password: "pw"}, Server: "5.6.7.8", Port: 8388, Name: "JP-01"}),
	}
	out := make([]Outbound, 0, len(links))
	for _, l := range links {
		p, err := ParseOutbound(l)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

func TestRunProxyScriptRename(t *testing.T) {
	proxies := scriptFixtures(t)
	code := `function operator(proxies) {
		return proxies.map(function(p){ p.name = "X-" + p.name; return p; });
	}`
	out, err := RunProxyScript(code, proxies, time.Second)
	if err != nil {
		t.Fatalf("script error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(out))
	}
	for _, p := range out {
		if !strings.HasPrefix(p.Name, "X-") {
			t.Fatalf("rename not applied: %q", p.Name)
		}
		if p.Link == "" || !strings.Contains(p.Link, "://") {
			t.Fatalf("link not rebuilt: %q", p.Link)
		}
	}
}

func TestRunProxyScriptFilter(t *testing.T) {
	proxies := scriptFixtures(t)
	code := `function operator(proxies) {
		return proxies.filter(function(p){ return p.name.indexOf("HK") >= 0; });
	}`
	out, err := RunProxyScript(code, proxies, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Name != "HK-01" {
		t.Fatalf("unexpected filter result: %#v", out)
	}
}

func TestRunProxyScriptNewNode(t *testing.T) {
	proxies := scriptFixtures(t)
	newLink := EncodeSSURL(Ss{Param: Param{Cipher: "aes-256-gcm", Password: "pw"}, Server: "9.9.9.9", Port: 8388, Name: "NEW"})
	code := `function operator(proxies) {
		proxies.push({ name: "NEW", link: "` + newLink + `" });
		return proxies;
	}`
	out, err := RunProxyScript(code, proxies, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(out))
	}
	if out[2].Server != "9.9.9.9" {
		t.Fatalf("new node not parsed: %#v", out[2])
	}
}

func TestRunProxyScriptTimeout(t *testing.T) {
	proxies := scriptFixtures(t)
	code := `function operator(proxies) { while (true) {} }`
	start := time.Now()
	if _, err := RunProxyScript(code, proxies, 100*time.Millisecond); err == nil {
		t.Fatal("expected timeout error")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("script did not stop promptly: %v", time.Since(start))
	}
}

func TestRunProxyScriptRequiresOperator(t *testing.T) {
	if _, err := RunProxyScript(`var x = 1;`, scriptFixtures(t), time.Second); err == nil {
		t.Fatal("expected missing operator error")
	}
}
