package node

import (
	"strings"
	"testing"
)

const sampleClashYAML = `mixed-port: 7890
allow-lan: false
mode: rule
proxies:
  - name: ss-node
    type: ss
    server: 1.2.3.4
    port: 8388
    cipher: aes-256-gcm
    password: pass123
  - name: vmess-node
    type: vmess
    server: 5.6.7.8
    port: 443
    uuid: 11111111-2222-3333-4444-555555555555
    alterId: 0
    cipher: auto
    network: ws
    tls: true
    servername: example.com
    ws-opts:
      path: /ws
      headers:
        Host: example.com
  - name: vless-reality
    type: vless
    server: 9.9.9.9
    port: 8443
    uuid: aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee
    flow: xtls-rprx-vision
    tls: true
    servername: www.example.com
    client-fingerprint: chrome
    reality-opts:
      public-key: pubkey123
      short-id: abcd
  - name: hy2-node
    type: hysteria2
    server: 2.2.2.2
    port: 443
    password: hy2pass
    sni: bing.com
    skip-cert-verify: true
  - name: tuic-node
    type: tuic
    server: 3.3.3.3
    port: 443
    uuid: 99999999-8888-7777-6666-555555555555
    password: tuicpass
    sni: tuic.example.com
proxy-groups:
  - name: Proxy
    type: select
    proxies: [ss-node]
rules:
  - MATCH,Proxy
`

func TestIsClashConfig(t *testing.T) {
	if !IsClashConfig([]byte(sampleClashYAML)) {
		t.Fatal("expected Clash YAML to be detected")
	}
	if IsClashConfig([]byte("vless://uuid@host:443?type=tcp#node\nss://abc@host:8388#x")) {
		t.Fatal("plain link list must not be detected as Clash YAML")
	}
	if IsClashConfig([]byte(Base64Encode("vmess://abc"))) {
		t.Fatal("base64 link list must not be detected as Clash YAML")
	}
}

func TestParseClashToNodes(t *testing.T) {
	nodes, err := ParseClashToNodes([]byte(sampleClashYAML))
	if err != nil {
		t.Fatalf("ParseClashToNodes error: %v", err)
	}
	if len(nodes) != 5 {
		t.Fatalf("expected 5 nodes, got %d", len(nodes))
	}
	wantScheme := map[string]string{
		"ss-node":       "ss://",
		"vmess-node":    "vmess://",
		"vless-reality": "vless://",
		"hy2-node":      "hy2://",
		"tuic-node":     "tuic://",
	}
	for _, n := range nodes {
		prefix, ok := wantScheme[n.Name]
		if !ok {
			t.Fatalf("unexpected node %q", n.Name)
		}
		if !strings.HasPrefix(n.Link, prefix) {
			t.Fatalf("node %q link %q does not start with %q", n.Name, n.Link, prefix)
		}
	}
}

func TestClashSSRoundTrip(t *testing.T) {
	nodes, err := ParseClashToNodes([]byte(sampleClashYAML))
	if err != nil {
		t.Fatal(err)
	}
	var link string
	for _, n := range nodes {
		if n.Name == "ss-node" {
			link = n.Link
		}
	}
	ss, err := DecodeSSURL(link)
	if err != nil {
		t.Fatalf("decode ss link: %v", err)
	}
	if ss.Server != "1.2.3.4" || ss.Port != 8388 {
		t.Fatalf("unexpected ss server/port: %s:%d", ss.Server, ss.Port)
	}
	if ss.Param.Cipher != "aes-256-gcm" || ss.Param.Password != "pass123" {
		t.Fatalf("unexpected ss cipher/password: %s/%s", ss.Param.Cipher, ss.Param.Password)
	}
}

func TestClashVlessRealityFields(t *testing.T) {
	nodes, err := ParseClashToNodes([]byte(sampleClashYAML))
	if err != nil {
		t.Fatal(err)
	}
	var link string
	for _, n := range nodes {
		if n.Name == "vless-reality" {
			link = n.Link
		}
	}
	v, err := DecodeVLESSURL(link)
	if err != nil {
		t.Fatalf("decode vless link: %v", err)
	}
	if v.Query.Security != "reality" {
		t.Fatalf("expected reality security, got %q", v.Query.Security)
	}
	if v.Query.Pbk != "pubkey123" || v.Query.Sid != "abcd" {
		t.Fatalf("unexpected reality opts: pbk=%q sid=%q", v.Query.Pbk, v.Query.Sid)
	}
	if v.Query.Flow != "xtls-rprx-vision" {
		t.Fatalf("unexpected flow: %q", v.Query.Flow)
	}
}

func TestClashSocks5ToLink(t *testing.T) {
	p, err := ClashProxyToLink(map[string]interface{}{
		"type": "socks5", "name": "s", "server": "1.2.3.4", "port": 1080, "username": "u", "password": "p",
	})
	if err != nil {
		t.Fatalf("socks5 convert: %v", err)
	}
	if !strings.HasPrefix(p.Link, "socks5://") {
		t.Fatalf("unexpected link: %s", p.Link)
	}
}

func TestParseClashNoProxies(t *testing.T) {
	if _, err := ParseClashToNodes([]byte("proxies: []\n")); err == nil {
		t.Fatal("expected error for empty proxies")
	}
}
