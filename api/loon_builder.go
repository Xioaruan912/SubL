package api

import (
	"fmt"
	"strings"
)

// LoonGeneral Loon [General] 常用配置（表单填空）
type LoonGeneral struct {
	IPMode            string `json:"ip_mode"`            // ipv4-only / dual / ipv4-preferred / ipv6-preferred
	DNSServer         string `json:"dns_server"`
	SniSniffing       bool   `json:"sni_sniffing"`
	DisableStun       bool   `json:"disable_stun"`
	AllowWifiAccess   bool   `json:"allow_wifi_access"`
	WifiHTTPPort      int    `json:"wifi_http_port"`
	WifiSocksPort     int    `json:"wifi_socks_port"`
	TestTimeout       int    `json:"test_timeout"`
	SwitchNodeAfterFail int  `json:"switch_node_after_failure"`
	InternetTestURL   string `json:"internet_test_url"`
	ProxyTestURL      string `json:"proxy_test_url"`
	ResourceParser    string `json:"resource_parser"`
	GeoipURL          string `json:"geoip_url"`
	IpasnURL          string `json:"ipasn_url"`
	UDPFallbackMode   string `json:"udp_fallback_mode"`  // DIRECT / REJECT
	DomainRejectMode  string `json:"domain_reject_mode"` // DNS / Request
	DNSRejectMode     string `json:"dns_reject_mode"`    // LoopbackIP / NOANSWER / NXDOMAIN
	BypassTun         string `json:"bypass_tun"`
	SkipProxy         string `json:"skip_proxy"`
}

// LoonFilter Remote Filter 节点筛选
type LoonFilter struct {
	Name  string `json:"name"`
	Regex string `json:"regex"`
}

// LoonGroup Proxy Group 策略组
type LoonGroup struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // select / url-test / fallback
	Policies string `json:"policies"`
	URL      string `json:"url"`
	ImgURL   string `json:"img_url"`
}

// LoonBuilder Loon 模板构建请求
type LoonBuilder struct {
	Filename    string        `json:"filename"`
	EditOldname string        `json:"edit_oldname"`
	General     LoonGeneral   `json:"general"`
	Filters     []LoonFilter  `json:"filters"`
	Groups      []LoonGroup   `json:"groups"`
	RemoteRules string        `json:"remote_rules"`
	Plugins     string        `json:"plugins"`
	Rules       string        `json:"rules"`
}

// defaultLoonGeneral 默认 [General]（基于常用基础配置）
func defaultLoonGeneral() LoonGeneral {
	return LoonGeneral{
		IPMode:              "v4-only",
		DNSServer:           "system",
		SniSniffing:         true,
		DisableStun:         true,
		AllowWifiAccess:     false,
		WifiHTTPPort:        7222,
		WifiSocksPort:       7221,
		TestTimeout:         5,
		SwitchNodeAfterFail: 3,
		InternetTestURL:     "https://www.youtube.com",
		ProxyTestURL:        "https://www.youtube.com",
		ResourceParser:      "https://github.com/sub-store-org/Sub-Store/releases/latest/download/sub-store-parser.loon.min.js",
		GeoipURL:            "https://github.com/Masaiki/GeoIP2-CN/raw/release/Country.mmdb",
		IpasnURL:            "https://raw.githubusercontent.com/P3TERX/GeoLite.mmdb/download/GeoLite2-ASN.mmdb",
		UDPFallbackMode:     "REJECT",
		DomainRejectMode:    "DNS",
		DNSRejectMode:       "LoopbackIP",
		BypassTun:           "10.0.0.0/8,100.64.0.0/10,127.0.0.0/8,169.254.0.0/16,172.16.0.0/12,192.0.0.0/24,192.0.2.0/24,192.88.99.0/24,192.168.0.0/16,198.51.100.0/24,203.0.113.0/24,224.0.0.0/4,255.255.255.255/32",
		SkipProxy:           "192.168.0.0/16,10.0.0.0/8,172.16.0.0/12,localhost,*.local,e.crashlynatics.com",
	}
}

// defaultLoonFilters 默认 Remote Filter（国家筛选）
func defaultLoonFilters() []LoonFilter {
	return []LoonFilter{
		{Name: "美国", Regex: `(?i)(美国|波特兰|达拉斯|俄勒冈|凤凰城|费利蒙|硅谷|拉斯维加斯|洛杉矶|圣何塞|圣克拉拉|西雅图|芝加哥|🇺🇸|US|USA)`},
		{Name: "香港", Regex: `(?i)(香港|hong|HK|HKG|🇭🇰)`},
		{Name: "日本", Regex: `(?i)(日本|东京|大阪|泉日|埼玉|🇯🇵|JP|Japan)`},
		{Name: "新加坡", Regex: `(?i)(狮城|新加坡|🇸🇬|SG|Singapore)`},
		{Name: "台湾", Regex: `(?i)(台湾|台灣|🇹🇼|TW|Taiwan)`},
	}
}

// defaultLoonGroups 默认 Proxy Group
func defaultLoonGroups() []LoonGroup {
	testURL := "http://cp.cloudflare.com/generate_204"
	icon := "https://raw.githubusercontent.com/erdongchanyo/icon/main/"
	return []LoonGroup{
		{Name: "Global", Type: "select", Policies: "US, HK, JP, SG, DIRECT", URL: testURL, ImgURL: icon + "Policy-Filter/Outside.png"},
		{Name: "US", Type: "select", Policies: "美国", URL: testURL, ImgURL: icon + "Policy-Country/US.png"},
		{Name: "HK", Type: "select", Policies: "香港", URL: testURL, ImgURL: icon + "Policy-Country/HK02.png"},
		{Name: "JP", Type: "select", Policies: "日本", URL: testURL, ImgURL: icon + "Policy-Country/JP.png"},
		{Name: "SG", Type: "select", Policies: "新加坡", URL: testURL, ImgURL: icon + "Policy-Country/SG.png"},
		{Name: "TW", Type: "select", Policies: "台湾", URL: testURL, ImgURL: icon + "Policy-Country/TW.png"},
		{Name: "Netflix", Type: "select", Policies: "Global,TW, US, HK, JP, SG, DIRECT", URL: testURL, ImgURL: icon + "Policy-Filter/Netflix.png"},
		{Name: "Youtube", Type: "select", Policies: "Global, TW,HK, US, JP, SG, DIRECT", URL: testURL, ImgURL: icon + "Policy-Filter/Youtube.png"},
		{Name: "Mainland", Type: "select", Policies: "DIRECT", URL: testURL, ImgURL: icon + "Policy-Filter/Mainland.png"},
		{Name: "Advertising", Type: "select", Policies: "REJECT", URL: testURL, ImgURL: icon + "Policy-Filter/AdBlock.png"},
		{Name: "FINAL", Type: "select", Policies: "Global, DIRECT", URL: testURL, ImgURL: icon + "Policy-Filter/Final01.png"},
	}
}

// applyLoonDefaults 应用默认值（字段为空时兜底）
func applyLoonDefaults(req *LoonBuilder) {
	if req.General.IPMode == "" {
		req.General = defaultLoonGeneral()
	} else {
		d := defaultLoonGeneral()
		g := &req.General
		if g.DNSServer == "" {
			g.DNSServer = d.DNSServer
		}
		if g.WifiHTTPPort == 0 {
			g.WifiHTTPPort = d.WifiHTTPPort
		}
		if g.WifiSocksPort == 0 {
			g.WifiSocksPort = d.WifiSocksPort
		}
		if g.TestTimeout == 0 {
			g.TestTimeout = d.TestTimeout
		}
		if g.InternetTestURL == "" {
			g.InternetTestURL = d.InternetTestURL
		}
		if g.ProxyTestURL == "" {
			g.ProxyTestURL = d.ProxyTestURL
		}
		if g.ResourceParser == "" {
			g.ResourceParser = d.ResourceParser
		}
		if g.GeoipURL == "" {
			g.GeoipURL = d.GeoipURL
		}
		if g.IpasnURL == "" {
			g.IpasnURL = d.IpasnURL
		}
		if g.UDPFallbackMode == "" {
			g.UDPFallbackMode = d.UDPFallbackMode
		}
		if g.DomainRejectMode == "" {
			g.DomainRejectMode = d.DomainRejectMode
		}
		if g.DNSRejectMode == "" {
			g.DNSRejectMode = d.DNSRejectMode
		}
		if g.BypassTun == "" {
			g.BypassTun = d.BypassTun
		}
		if g.SkipProxy == "" {
			g.SkipProxy = d.SkipProxy
		}
	}
	if len(req.Filters) == 0 {
		req.Filters = defaultLoonFilters()
	}
	if len(req.Groups) == 0 {
		req.Groups = defaultLoonGroups()
	}
}

// boolStr 布尔转 INI 值
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// buildLoonConfig 生成 Loon 配置文本（INI）
func buildLoonConfig(req *LoonBuilder) string {
	applyLoonDefaults(req)
	var b strings.Builder

	// [General]
	b.WriteString("[General]\n")
	g := req.General
	b.WriteString(fmt.Sprintf("ip-mode = %s\n", g.IPMode))
	b.WriteString(fmt.Sprintf("dns-server = %s\n", g.DNSServer))
	b.WriteString(fmt.Sprintf("sni-sniffing = %s\n", boolStr(g.SniSniffing)))
	b.WriteString(fmt.Sprintf("disable-stun = %s\n", boolStr(g.DisableStun)))
	b.WriteString(fmt.Sprintf("udp-fallback-mode = %s\n", g.UDPFallbackMode))
	b.WriteString(fmt.Sprintf("domain-reject-mode = %s\n", g.DomainRejectMode))
	b.WriteString(fmt.Sprintf("dns-reject-mode = %s\n", g.DNSRejectMode))
	b.WriteString(fmt.Sprintf("wifi-access-http-port = %d\n", g.WifiHTTPPort))
	b.WriteString(fmt.Sprintf("wifi-access-socks5-port = %d\n", g.WifiSocksPort))
	b.WriteString(fmt.Sprintf("allow-wifi-access = %s\n", boolStr(g.AllowWifiAccess)))
	b.WriteString("interface-mode = auto\n")
	b.WriteString(fmt.Sprintf("test-timeout = %d\n", g.TestTimeout))
	b.WriteString(fmt.Sprintf("switch-node-after-failure-times = %d\n", g.SwitchNodeAfterFail))
	b.WriteString(fmt.Sprintf("internet-test-url = %s\n", g.InternetTestURL))
	b.WriteString(fmt.Sprintf("proxy-test-url = %s\n", g.ProxyTestURL))
	b.WriteString(fmt.Sprintf("resource-parser = %s\n", g.ResourceParser))
	b.WriteString(fmt.Sprintf("geoip-url = %s\n", g.GeoipURL))
	b.WriteString(fmt.Sprintf("ipasn-url = %s\n", g.IpasnURL))
	b.WriteString(fmt.Sprintf("skip-proxy = %s\n", g.SkipProxy))
	b.WriteString(fmt.Sprintf("bypass-tun = %s\n", g.BypassTun))

	b.WriteString("\n[Proxy]\n\n[Remote Proxy]\n\n")

	// [Remote Filter]
	b.WriteString("[Remote Filter]\n")
	b.WriteString("# 远程节点订阅正则筛选\n")
	for _, f := range req.Filters {
		if f.Name == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("%s = NameRegex, FilterKey = \"%s\"\n", f.Name, f.Regex))
	}

	// [Proxy Group]
	b.WriteString("\n[Proxy Group]\n\n")
	for _, gr := range req.Groups {
		if gr.Name == "" {
			continue
		}
		line := fmt.Sprintf("%s = %s, %s", gr.Name, gr.Type, gr.Policies)
		if gr.URL != "" {
			line += fmt.Sprintf(", url = %s", gr.URL)
		}
		if gr.ImgURL != "" {
			line += fmt.Sprintf(", img-url = %s", gr.ImgURL)
		}
		b.WriteString(line + "\n")
	}

	// [Remote Rule]
	b.WriteString("\n[Remote Rule]\n")
	if strings.TrimSpace(req.RemoteRules) != "" {
		b.WriteString(strings.TrimSpace(req.RemoteRules) + "\n")
	}

	b.WriteString("\n[Proxy Chain]\n")

	// [Rule]
	b.WriteString("[Rule]\n")
	if strings.TrimSpace(req.Rules) != "" {
		b.WriteString(strings.TrimSpace(req.Rules) + "\n")
	}
	b.WriteString("DOMAIN-SUFFIX,local,DIRECT\nGEOIP,CN,DIRECT\nFINAL,FINAL\n")

	b.WriteString("\n[Host]\n\n[Rewrite]\n\n[Script]\n\n")

	// [Plugin]
	b.WriteString("[Plugin]\n")
	if strings.TrimSpace(req.Plugins) != "" {
		b.WriteString(strings.TrimSpace(req.Plugins) + "\n")
	}

	b.WriteString("\n[Mitm]\nhostname =\nskip-server-cert-verify = false\n")
	return b.String()
}