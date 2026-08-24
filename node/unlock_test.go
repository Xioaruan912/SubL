package node

import "testing"

func TestBuildOutboundVless(t *testing.T) {
	link := "vless://19d40b33-c63f-4cf6-bde7-9e6a53a02036@154.31.116.16:43804?type=tcp&security=reality&pbk=Fnu3wR5hEeonakgRDrgG9yRG9XyM9KScbZlmPzrUXwM&fp=random&sni=music.apple.com&sid=0892831900b76d85&spx=%2F&flow=xtls-rprx-vision#DMUT_JP"
	cfg, name, err := buildOutboundConfig(link)
	if err != nil {
		t.Fatalf("buildOutboundConfig error: %v", err)
	}
	t.Logf("name=%s config=%s", name, cfg)
	if name != "DMUT_JP" {
		t.Errorf("name=%s want DMUT_JP", name)
	}
}

func TestBuildOutboundHy2(t *testing.T) {
	link := "hysteria2://8ae98cad@216.23.84.93:29810/?insecure=1&sni=www.bing.com#Hysteria2_JP"
	cfg, name, err := buildOutboundConfig(link)
	if err != nil {
		t.Fatalf("buildOutboundConfig error: %v", err)
	}
	t.Logf("name=%s config=%s", name, cfg)
}
