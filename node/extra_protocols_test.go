package node

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtraProtocolRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		link string
		host string
	}{
		{"anytls", EncodeAnyTLSURL(AnyTLS{Password: "pw", Host: "1.1.1.1", Port: 443, Sni: "a.com", Name: "anytls-node"}), "1.1.1.1"},
		{"snell", EncodeSnellURL(Snell{Psk: "psk", Host: "2.2.2.2", Port: 443, Version: "4", Name: "snell-node"}), "2.2.2.2"},
		{"socks5", EncodeSocksURL(Socks{Type: "socks5", Username: "u", Password: "p", Host: "3.3.3.3", Port: 1080, Name: "socks-node"}), "3.3.3.3"},
		{"http", EncodeSocksURL(Socks{Type: "http", Host: "4.4.4.4", Port: 8080, Name: "http-node"}), "4.4.4.4"},
		{"ssh", EncodeSSHURL(SSH{User: "root", Password: "pw", Host: "5.5.5.5", Port: 22, Name: "ssh-node"}), "5.5.5.5"},
		{"wireguard", EncodeWireGuardURL(WireGuard{PrivateKey: "priv", PublicKey: "pub", Host: "6.6.6.6", Port: 51820, Address: "10.0.0.2/32", Name: "wg-node"}), "6.6.6.6"},
	}
	for _, tc := range cases {
		p, err := ParseOutbound(tc.link)
		if err != nil {
			t.Fatalf("%s parse: %v", tc.name, err)
		}
		if p.Server != tc.host {
			t.Fatalf("%s server=%q want %q", tc.name, p.Server, tc.host)
		}
		if p.Port == 0 {
			t.Fatalf("%s port not parsed", tc.name)
		}
		reparsed, err := ParseOutbound(p.ToLink())
		if err != nil {
			t.Fatalf("%s re-parse: %v", tc.name, err)
		}
		if reparsed.Server != p.Server || reparsed.Name != p.Name {
			t.Fatalf("%s round trip changed: %#v -> %#v", tc.name, p, reparsed)
		}
	}
}

func TestEncodeSingbox(t *testing.T) {
	urls := []string{
		EncodeSSURL(Ss{Param: Param{Cipher: "aes-256-gcm", Password: "pw"}, Server: "1.2.3.4", Port: 8388, Name: "ss-1"}),
		EncodeAnyTLSURL(AnyTLS{Password: "pw", Host: "5.6.7.8", Port: 443, Name: "any-1"}),
		EncodeWireGuardURL(WireGuard{PrivateKey: "priv", PublicKey: "pub", Host: "9.9.9.9", Port: 51820, Address: "10.0.0.2/32", Name: "wg-1"}),
	}
	out, err := EncodeSingbox(urls, SqlConfig{})
	if err != nil {
		t.Fatalf("EncodeSingbox: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	outbounds, ok := cfg["outbounds"].([]interface{})
	if !ok || len(outbounds) < 4 {
		t.Fatalf("unexpected outbounds: %#v", cfg["outbounds"])
	}
	if !strings.Contains(out, "\"type\": \"selector\"") || !strings.Contains(out, "\"type\": \"anytls\"") || !strings.Contains(out, "\"type\": \"wireguard\"") {
		t.Fatalf("missing expected outbounds:\n%s", out)
	}
}

func TestEncodeQX(t *testing.T) {
	urls := []string{
		EncodeSSURL(Ss{Param: Param{Cipher: "aes-256-gcm", Password: "pw"}, Server: "1.2.3.4", Port: 8388, Name: "ss-1"}),
		EncodeTrojanURL(Trojan{Password: "tpw", Hostname: "2.2.2.2", Port: 443, Name: "trojan-1", Type: "trojan"}),
	}
	out, err := EncodeQX(urls, SqlConfig{})
	if err != nil {
		t.Fatalf("EncodeQX: %v", err)
	}
	if !strings.Contains(out, "shadowsocks=1.2.3.4:8388") {
		t.Fatalf("missing ss line:\n%s", out)
	}
	if !strings.Contains(out, "trojan=2.2.2.2:443") {
		t.Fatalf("missing trojan line:\n%s", out)
	}
}
