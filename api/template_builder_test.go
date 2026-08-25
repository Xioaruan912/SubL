package api

import (
	"strings"
	"testing"
)

func TestBuildClashYAMLFull(t *testing.T) {
	req := BuilderRequest{
		Filename:               "mihomo.yaml",
		Port:                   7890,
		SocksPort:              7891,
		MixedPort:              7892,
		RedirPort:              7893,
		TproxyPort:             7894,
		PortOffset:             true,
		AllowLan:               true,
		Mode:                   "rule",
		LogLevel:               "info",
		IPv6:                   true,
		UDP:                    true,
		ExternalController:     "0.0.0.0:9090",
		UnifiedDelay:           true,
		GeodataLoader:          "standard",
		GeoAutoUpdate:          true,
		GeoUpdateInterval:      24,
		TcpConcurrent:          true,
		FindProcessMode:        "strict",
		GlobalClientFingerprint: "chrome",
		SnifferEnable:          true,
		TunEnable:              true,
		TunStack:               "system",
		DNSEnable:              true,
		DNSEnhancedMode:        "fake-ip",
		DNSListenPort:          1053,
		TestURL:                "http://www.gstatic.com/generate_204",
		Interval:               300,
		Groups:                 defaultGroups(),
		RuleProviders:          defaultRuleProviders(),
		Rules:                  defaultRules(),
	}
	out, err := buildClashYAML(req)
	if err != nil {
		t.Fatalf("buildClashYAML error: %v", err)
	}
	t.Logf("yaml:\n%s", out)

	// 基础配置
	for _, s := range []string{"port: 7890", "mixed-port: 7892", "tproxy-port: 7894", "proxies: ~", "unified-delay: true", "global-client-fingerprint: chrome", "external-controller: 0.0.0.0:9090"} {
		if !strings.Contains(out, s) {
			t.Errorf("缺少: %s", s)
		}
	}
	// tun / dns / sniffer
	for _, s := range []string{"tun:", "dns:", "sniffer:", "enhanced-mode: fake-ip", "0.0.0.0:1053"} {
		if !strings.Contains(out, s) {
			t.Errorf("缺少: %s", s)
		}
	}
	// 分组（含 filter）
	for _, s := range []string{"name: AI", "filter: (?i)US|USA|United States|美国|JP", "include-all-providers: true"} {
		if !strings.Contains(out, s) {
			t.Errorf("缺少: %s", s)
		}
	}
	// 规则集
	for _, s := range []string{"rule-providers:", "RULE-SET,LAN,DIRECT", "RULE-SET,AI,AI", "MATCH,日常使用"} {
		if !strings.Contains(out, s) {
			t.Errorf("缺少: %s", s)
		}
	}
}

func TestPortOffsetLinkage(t *testing.T) {
	req := BuilderRequest{
		Port:       10000,
		SocksPort:  0, // 触发默认
		MixedPort:  7892,
		RedirPort:  7893,
		TproxyPort: 7894,
		PortOffset: true,
		Mode:       "rule",
		TestURL:    defaultTestURL,
		Interval:   300,
		Groups:     []BuilderGroup{{Name: "节点选择", Type: "select"}},
	}
	applyBuilderDefaults(&req)
	applyPortOffsets(&req, req.Port)
	// 主端口 10000，偏移 delta=2110
	if req.MixedPort != 7892+2110 {
		t.Errorf("mixed-port 未联动: %d", req.MixedPort)
	}
	// DNS listen / external-controller 是独立端口，不随主端口偏移
	if req.DNSListenPort != 1053 {
		t.Errorf("dns listen 不应偏移: %d", req.DNSListenPort)
	}
	if !strings.Contains(req.ExternalController, ":") {
		t.Errorf("external-controller 异常: %s", req.ExternalController)
	}
	t.Logf("port=%d mixed=%d socks=%d dns=%d ec=%s", req.Port, req.MixedPort, req.SocksPort, req.DNSListenPort, req.ExternalController)
}