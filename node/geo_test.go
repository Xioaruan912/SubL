package node

import "testing"

func TestLookupCountry(t *testing.T) {
	// GeoLite 数据库真实结果（数据库版本相关，仅验证能查询并返回非空）
	cases := []string{
		"154.31.116.16", "103.117.120.98", "209.248.45.208",
		"154.9.224.200", "216.23.84.93", "209.248.57.97",
	}
	for _, host := range cases {
		got := LookupCountry(host)
		if got == "" {
			t.Errorf("host %s: 查询国家失败，应返回非空", host)
		}
	}
}

func TestExtractServerHost(t *testing.T) {
	links := []string{
		"vless://19d40b33-c63f-4cf6-bde7-9e6a53a02036@154.31.116.16:43804?type=tcp&security=reality#DMUT_JP",
		"hysteria2://8ae98cad@216.23.84.93:29810/?insecure=1&sni=www.bing.com#Hysteria2_JP",
		"ss://YWVzLTI1Ni1nY206cGFzc3dvcmQxMjM0NTY3ODk@1.2.3.4:8388#US-节点01",
	}
	wants := []struct {
		host string
		port int
	}{
		{"154.31.116.16", 43804},
		{"216.23.84.93", 29810},
		{"1.2.3.4", 8388},
	}
	for i, l := range links {
		host, port := ExtractServerHost(l)
		if host != wants[i].host || port != wants[i].port {
			t.Errorf("link %d: got %s:%d want %s:%d", i, host, port, wants[i].host, wants[i].port)
		}
	}
}