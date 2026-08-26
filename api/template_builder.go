package api

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// BuilderGroup 表单中一个分组的配置
type BuilderGroup struct {
	Name                string   `json:"name"`
	Type                string   `json:"type"` // select / url-test / fallback
	Filter              string   `json:"filter"`
	IncludeAllProviders bool     `json:"include_all_providers"`
	Proxies             []string `json:"proxies"`
}

// RuleProvider 规则集
type RuleProvider struct {
	Name     string `json:"name"`
	Type     string `json:"type"`     // http / file
	Behavior string `json:"behavior"` // classical / ipcidr / domain
	Format   string `json:"format"`   // yaml / text
	URL      string `json:"url"`
	Path     string `json:"path"`
	Proxy    string `json:"proxy"`
	Interval int    `json:"interval"`
}

// BuilderRequest 构建模板请求
type BuilderRequest struct {
	Filename  string `json:"filename"`
	EditOldname string `json:"edit_oldname"` // 编辑现有模板时的旧文件名
	Source    string `json:"source"`          // default / custom
	Target    string `json:"target"`          // clash / loon
	LoonText  string `json:"loon_text"`       // Loon 文本模板模式

	// 基础端口（用户可改，其他端口基于主端口偏移联动）
	Port          int  `json:"port"`
	SocksPort     int  `json:"socks_port"`
	MixedPort     int  `json:"mixed_port"`
	RedirPort     int  `json:"redir_port"`
	TproxyPort    int  `json:"tproxy_port"`
	PortOffset    bool `json:"port_offset"` // 是否联动偏移关联端口

	// 基础配置
	AllowLan bool   `json:"allow_lan"`
	Mode     string `json:"mode"`
	LogLevel string `json:"log_level"`
	IPv6     bool   `json:"ipv6"`
	UDP      bool   `json:"udp"`
	ExternalController string `json:"external_controller"`

	// 高级配置
	UnifiedDelay            bool   `json:"unified_delay"`
	GeodataMode             bool   `json:"geodata_mode"`
	GeodataLoader           string `json:"geodata_loader"`
	GeoAutoUpdate           bool   `json:"geo_auto_update"`
	GeoUpdateInterval       int    `json:"geo_update_interval"`
	TcpConcurrent           bool   `json:"tcp_concurrent"`
	FindProcessMode         string `json:"find_process_mode"`
	GlobalClientFingerprint string `json:"global_client_fingerprint"`
	SnifferEnable           bool   `json:"sniffer_enable"`
	TunEnable               bool   `json:"tun_enable"`
	TunStack                string `json:"tun_stack"`
	DNSEnable               bool   `json:"dns_enable"`
	DNSEnhancedMode         string `json:"dns_enhanced_mode"`
	DNSListenPort           int    `json:"dns_listen_port"`

	// 分组
	Groups   []BuilderGroup `json:"groups"`
	TestURL  string         `json:"test_url"`
	Interval int            `json:"interval"`

	// 规则集
	RuleProviders []RuleProvider `json:"rule_providers"`
	// 规则（每行一条）
	Rules []string `json:"rules"`
}

// 默认测速 URL
var defaultTestURL = "http://www.gstatic.com/generate_204"

// 默认 geox-url
const defaultMMDBURL = "https://geodata.kelee.one/Country-Masaiki.mmdb"
const defaultASNURL  = "https://geodata.kelee.one/GeoLite2-ASN-P3TERX.mmdb"

// defaultGroups 默认策略分组
func defaultGroups() []BuilderGroup {
	return []BuilderGroup{
		{Name: "日常使用", Type: "select", IncludeAllProviders: true, Proxies: []string{"DIRECT"}},
		{Name: "AI", Type: "select", IncludeAllProviders: true, Filter: "(?i)US|USA|United States|美国|JP"},
	}
}

// defaultRuleProviders 默认规则集
func defaultRuleProviders() []RuleProvider {
	return []RuleProvider{
		{Name: "LAN", Type: "http", Behavior: "classical", Interval: 3600, Format: "yaml", Proxy: "DIRECT", Path: "./rules/Lan.yaml", URL: "https://kelee.one/Tool/Clash/Rule/LAN_SPLITTER.yaml"},
		{Name: "Direct", Type: "http", Behavior: "classical", Interval: 3600, Format: "yaml", Proxy: "DIRECT", Path: "./rules/Direct.yaml", URL: "https://kelee.one/Tool/Clash/Rule/Direct.yaml"},
		{Name: "Proxy", Type: "http", Behavior: "classical", Interval: 3600, Format: "yaml", Proxy: "DIRECT", Path: "./rules/Proxy.yaml", URL: "https://kelee.one/Tool/Clash/Rule/Proxy.yaml"},
		{Name: "AI", Type: "http", Behavior: "classical", Interval: 3600, Format: "yaml", Proxy: "DIRECT", Path: "./rules/AI.yaml", URL: "https://kelee.one/Tool/Clash/Rule/AI.yaml"},
		{Name: "ESET_China", Type: "http", Behavior: "classical", Interval: 3600, Format: "yaml", Proxy: "DIRECT", Path: "./rules/ESET_China.yaml", URL: "https://kelee.one/Tool/Clash/Rule/ESET_China.yaml"},
	}
}

// defaultRules 默认规则
func defaultRules() []string {
	return []string{
		"RULE-SET,LAN,DIRECT",
		"RULE-SET,Direct,DIRECT",
		"RULE-SET,ESET_China,DIRECT",
		"RULE-SET,AI,AI",
		"RULE-SET,Proxy,日常使用",
		"GEOIP,CN,DIRECT",
		"MATCH,日常使用",
	}
}

// TemplateBuild 通过表单输入生成 clash/loon 配置并保存到模板目录。
// POST /api/v1/template/build
func TemplateBuild(c *gin.Context) {
	req := BuilderRequest{}
	var lb LoonBuilder
	target := "clash"
	ct := c.GetHeader("Content-Type")
	if strings.Contains(ct, "application/json") {
		raw, err := c.GetRawData()
		if err != nil {
			c.JSON(400, gin.H{"code": "40000", "msg": "请求体读取失败"})
			return
		}
		var probe struct {
			Target string `json:"target"`
		}
		_ = json.Unmarshal(raw, &probe)
		if probe.Target == "loon" {
			target = "loon"
			if err := json.Unmarshal(raw, &lb); err != nil {
				c.JSON(400, gin.H{"code": "40000", "msg": "Loon 配置解析失败: " + err.Error()})
				return
			}
		} else {
			if err := json.Unmarshal(raw, &req); err != nil {
				c.JSON(400, gin.H{"code": "40000", "msg": "请求体解析失败: " + err.Error()})
				return
			}
			target = req.Target
		}
	} else {
		bindFormBuilder(&req, c)
		target = req.Target
		if target == "loon" {
			lb.Filename = c.PostForm("filename")
			lb.EditOldname = c.PostForm("edit_oldname")
			lb.RemoteRules = c.PostForm("remote_rules")
			lb.Plugins = c.PostForm("plugins")
			lb.Rules = c.PostForm("rules")
			if s := c.PostForm("general"); s != "" {
				_ = json.Unmarshal([]byte(s), &lb.General)
			}
			if s := c.PostForm("filters"); s != "" {
				_ = json.Unmarshal([]byte(s), &lb.Filters)
			}
			if s := c.PostForm("groups"); s != "" {
				_ = json.Unmarshal([]byte(s), &lb.Groups)
			}
		}
	}
	if target == "" {
		target = "clash"
	}

	var out string
	if target == "loon" {
		if lb.Filename == "" {
			c.JSON(400, gin.H{"code": "40000", "msg": "文件名不能为空"})
			return
		}
		if !strings.HasSuffix(lb.Filename, ".conf") {
			lb.Filename += ".conf"
		}
		req.Filename = lb.Filename
		req.EditOldname = lb.EditOldname
		out = buildLoonConfig(&lb)
	} else {
		if req.Filename == "" {
			c.JSON(400, gin.H{"code": "40000", "msg": "文件名不能为空"})
			return
		}
		if !strings.HasSuffix(req.Filename, ".yaml") {
			req.Filename += ".yaml"
		}
		applyBuilderDefaults(&req)

		yamlText, err := buildClashYAML(req)
		if err != nil {
			c.JSON(500, gin.H{"code": "50000", "msg": "生成配置失败: " + err.Error()})
			return
		}
		out = yamlText
	}

	// 编辑现有模板时，先删除旧文件（改名）
	if req.EditOldname != "" && req.EditOldname != req.Filename {
		if oldPath, err := safeFilePath(req.EditOldname); err == nil {
			os.Remove(oldPath)
		}
	}

	fullPath, err := safeFilePath(req.Filename)
	if err != nil {
		c.JSON(400, gin.H{"code": "40000", "msg": "文件名非法: " + err.Error()})
		return
	}
	if err := os.WriteFile(fullPath, []byte(out), 0666); err != nil {
		log.Println("写入模板失败:", err)
		c.JSON(500, gin.H{"code": "50000", "msg": "保存模板失败"})
		return
	}

	c.JSON(200, gin.H{
		"code": "00000",
		"msg":  "模板已保存",
		"data": gin.H{
			"filename": req.Filename,
			"yaml":     out,
		},
	})
}

// bindFormBuilder 从表单解析
func bindFormBuilder(req *BuilderRequest, c *gin.Context) {
	req.Filename = c.PostForm("filename")
	req.EditOldname = c.PostForm("edit_oldname")
	req.Source = c.PostForm("source")
	req.Target = c.PostForm("target")
	req.LoonText = c.PostForm("loon_text")
	req.Port, _ = strconv.Atoi(c.PostForm("port"))
	req.SocksPort, _ = strconv.Atoi(c.PostForm("socks_port"))
	req.MixedPort, _ = strconv.Atoi(c.PostForm("mixed_port"))
	req.RedirPort, _ = strconv.Atoi(c.PostForm("redir_port"))
	req.TproxyPort, _ = strconv.Atoi(c.PostForm("tproxy_port"))
	req.PortOffset = c.PostForm("port_offset") == "true"
	req.AllowLan = c.PostForm("allow_lan") == "true"
	req.Mode = c.PostForm("mode")
	req.LogLevel = c.PostForm("log_level")
	req.IPv6 = c.PostForm("ipv6") == "true"
	req.UDP = c.PostForm("udp") == "true"
	req.ExternalController = c.PostForm("external_controller")
	req.UnifiedDelay = c.PostForm("unified_delay") == "true"
	req.GeodataMode = c.PostForm("geodata_mode") == "true"
	req.GeodataLoader = c.PostForm("geodata_loader")
	req.GeoAutoUpdate = c.PostForm("geo_auto_update") == "true"
	req.GeoUpdateInterval, _ = strconv.Atoi(c.PostForm("geo_update_interval"))
	req.TcpConcurrent = c.PostForm("tcp_concurrent") == "true"
	req.FindProcessMode = c.PostForm("find_process_mode")
	req.GlobalClientFingerprint = c.PostForm("global_client_fingerprint")
	req.SnifferEnable = c.PostForm("sniffer_enable") == "true"
	req.TunEnable = c.PostForm("tun_enable") == "true"
	req.TunStack = c.PostForm("tun_stack")
	req.DNSEnable = c.PostForm("dns_enable") == "true"
	req.DNSEnhancedMode = c.PostForm("dns_enhanced_mode")
	req.DNSListenPort, _ = strconv.Atoi(c.PostForm("dns_listen_port"))
	req.TestURL = c.PostForm("test_url")
	req.Interval, _ = strconv.Atoi(c.PostForm("interval"))
	if g := c.PostForm("groups"); g != "" {
		_ = json.Unmarshal([]byte(g), &req.Groups)
	}
	if rp := c.PostForm("rule_providers"); rp != "" {
		_ = json.Unmarshal([]byte(rp), &req.RuleProviders)
	}
	if r := c.PostForm("rules"); r != "" {
		for _, line := range strings.Split(r, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				req.Rules = append(req.Rules, line)
			}
		}
	}
}

// applyBuilderDefaults 应用默认值
func applyBuilderDefaults(req *BuilderRequest) {
	// 各字段默认值：仅当未提供时才填充
	if req.Port == 0 {
		req.Port = 7890
	}
	if req.SocksPort == 0 {
		req.SocksPort = req.Port + 1
	}
	if req.MixedPort == 0 {
		req.MixedPort = req.Port + 2
	}
	if req.RedirPort == 0 {
		req.RedirPort = req.Port + 3
	}
	if req.TproxyPort == 0 {
		req.TproxyPort = req.Port + 4
	}
	if req.Mode == "" {
		req.Mode = "rule"
	}
	if req.LogLevel == "" {
		req.LogLevel = "info"
	}
	if req.ExternalController == "" {
		req.ExternalController = "0.0.0.0:9090"
	}
	if req.TestURL == "" {
		req.TestURL = defaultTestURL
	}
	if req.Interval == 0 {
		req.Interval = 300
	}
	if req.GeodataLoader == "" {
		req.GeodataLoader = "standard"
	}
	if req.GeoUpdateInterval == 0 {
		req.GeoUpdateInterval = 24
	}
	if req.FindProcessMode == "" {
		req.FindProcessMode = "strict"
	}
	if req.GlobalClientFingerprint == "" {
		req.GlobalClientFingerprint = "chrome"
	}
	if req.DNSListenPort == 0 {
		req.DNSListenPort = 1053
	}
	if req.TunStack == "" {
		req.TunStack = "system"
	}
	if req.DNSEnhancedMode == "" {
		req.DNSEnhancedMode = "fake-ip"
	}
	if req.Source == "default" {
		// 默认模板：填充默认分组/规则集/规则
		req.PortOffset = true
		req.AllowLan = true
		req.IPv6 = true
		req.UDP = true
		req.UnifiedDelay = true
		req.GeoAutoUpdate = true
		req.TcpConcurrent = true
		req.SnifferEnable = true
		req.TunEnable = true
		req.DNSEnable = true
		if len(req.Groups) == 0 {
			req.Groups = defaultGroups()
		}
		if len(req.RuleProviders) == 0 {
			req.RuleProviders = defaultRuleProviders()
		}
		if len(req.Rules) == 0 {
			req.Rules = defaultRules()
		}
		return
	}
	if len(req.Groups) == 0 {
		req.Groups = []BuilderGroup{{Name: "🔰 节点选择", Type: "select"}}
	}
}

// TemplateGetDefault 返回默认的 mihomo 配置（供前端预填表单）。
// GET /api/v1/template/default
func TemplateGetDefault(c *gin.Context) {
	req := BuilderRequest{
		Filename:               "mihomo_default.yaml",
		Source:                 "default",
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
		GeodataMode:            false,
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
		TestURL:                defaultTestURL,
		Interval:               300,
		Groups:                 defaultGroups(),
		RuleProviders:          defaultRuleProviders(),
		Rules:                  defaultRules(),
	}
	c.JSON(200, gin.H{"code": "00000", "data": req, "msg": "默认配置"})
}

// buildClashYAML 根据表单输入生成完整 mihomo/clash YAML
func buildClashYAML(req BuilderRequest) (string, error) {
	// 端口偏移联动
	basePort := req.Port
	applyPortOffsets(&req, basePort)

	cfg := make(map[string]any)

	// 基础端口
	cfg["port"] = req.Port
	cfg["socks-port"] = req.SocksPort
	cfg["mixed-port"] = req.MixedPort
	cfg["redir-port"] = req.RedirPort
	cfg["tproxy-port"] = req.TproxyPort

	// 基础配置
	cfg["allow-lan"] = req.AllowLan
	cfg["mode"] = req.Mode
	cfg["log-level"] = req.LogLevel
	cfg["ipv6"] = req.IPv6
	cfg["udp"] = req.UDP
	cfg["external-controller"] = req.ExternalController

	// 高级配置
	cfg["unified-delay"] = req.UnifiedDelay
	cfg["geodata-mode"] = req.GeodataMode
	cfg["geodata-loader"] = req.GeodataLoader
	cfg["geo-auto-update"] = req.GeoAutoUpdate
	cfg["geo-update-interval"] = req.GeoUpdateInterval
	cfg["tcp-concurrent"] = req.TcpConcurrent
	cfg["find-process-mode"] = req.FindProcessMode
	cfg["global-client-fingerprint"] = req.GlobalClientFingerprint

	// geox-url
	cfg["geox-url"] = map[string]string{
		"mmdb": defaultMMDBURL,
		"asn":  defaultASNURL,
	}

	// profile
	cfg["profile"] = map[string]bool{
		"store-selected": true,
		"store-fake-ip":  true,
	}

	// sniffer
	if req.SnifferEnable {
		cfg["sniffer"] = map[string]any{
			"enable":                 true,
			"force-dns-mapping":      true,
			"parse-pure-ip":          true,
			"override-destination":   true,
			"force-domain":           []string{"+.v2ex.com"},
			"skip-domain":            []string{"Mijia Cloud"},
			"sniff": map[string]any{
				"HTTP": map[string]any{"ports": []int{80, 8080}, "override-destination": true},
				"TLS":  map[string]any{"ports": []int{443, 8443}},
				"QUIC": map[string]any{"ports": []int{443, 8443}},
			},
		}
	}

	// tun
	if req.TunEnable {
		cfg["tun"] = map[string]any{
			"enable":               true,
			"stack":                req.TunStack,
			"auto-route":           true,
			"auto-detect-interface": true,
			"dns-hijack":           []string{"any:53"},
		}
	}

	// dns
	if req.DNSEnable {
		cfg["dns"] = map[string]any{
			"enable":              true,
			"prefer-h3":           true,
			"listen":              "0.0.0.0:" + strconv.Itoa(req.DNSListenPort),
			"ipv6":                false,
			"enhanced-mode":       req.DNSEnhancedMode,
			"fake-ip-range":       "28.0.0.1/8",
			"fake-ip-filter":      []string{"*", "+.lan"},
			"default-nameserver":  []string{"223.5.5.5", "223.6.6.6"},
			"nameserver":          []string{"https://223.5.5.5/dns-query", "https://223.6.6.6/dns-query"},
		}
	}

	// proxies 占位（订阅时填充）
	cfg["proxies"] = "~"

	// proxy-groups
	if len(req.Groups) == 0 {
		req.Groups = []BuilderGroup{{Name: "🔰 节点选择", Type: "select"}}
	}
	groups := make([]map[string]any, 0, len(req.Groups))
	for _, g := range req.Groups {
		if g.Name == "" {
			continue
		}
		group := map[string]any{
			"name":    g.Name,
			"type":    g.Type,
			"proxies": []string{"DIRECT"},
		}
		switch g.Type {
		case "url-test", "fallback":
			group["url"] = req.TestURL
			group["interval"] = req.Interval
		}
		if g.Filter != "" {
			group["filter"] = g.Filter
			group["include-all-providers"] = g.IncludeAllProviders
		} else if g.IncludeAllProviders {
			group["include-all-providers"] = true
		}
		if len(g.Proxies) > 0 {
			group["proxies"] = g.Proxies
		}
		groups = append(groups, group)
	}
	cfg["proxy-groups"] = groups

	// rule-providers
	if len(req.RuleProviders) > 0 {
		rps := make(map[string]any)
		for _, rp := range req.RuleProviders {
			if rp.Name == "" {
				continue
			}
			m := map[string]any{
				"type":     rp.Type,
				"behavior": rp.Behavior,
			}
			if rp.Interval > 0 {
				m["interval"] = rp.Interval
			}
			if rp.Format != "" {
				m["format"] = rp.Format
			}
			if rp.Proxy != "" {
				m["proxy"] = rp.Proxy
			}
			if rp.Path != "" {
				m["path"] = rp.Path
			}
			if rp.URL != "" {
				m["url"] = rp.URL
			}
			rps[rp.Name] = m
		}
		if len(rps) > 0 {
			cfg["rule-providers"] = rps
		}
	}

	// rules
	if len(req.Rules) == 0 {
		req.Rules = defaultRules()
	}
	cfg["rules"] = req.Rules

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	// 将 proxies 占位输出为 ~
	out := strings.Replace(string(data), "proxies: \"~\"", "proxies: ~", 1)
	out = strings.Replace(out, "proxies: '~'", "proxies: ~", 1)
	return out, nil
}

// applyPortOffsets 端口偏移联动：当主端口修改时，同步偏移其它端口。
func applyPortOffsets(req *BuilderRequest, basePort int) {
	// 基准端口（默认配置下的主端口）
	const baseDefaultPort = 7890
	if !req.PortOffset || basePort == 0 {
		return
	}
	delta := basePort - baseDefaultPort
	// 仅当关联端口仍是默认相对值时进行偏移
	if req.MixedPort == 7892 {
		req.MixedPort = 7892 + delta
	}
	if req.RedirPort == 7893 {
		req.RedirPort = 7893 + delta
	}
	if req.TproxyPort == 7894 {
		req.TproxyPort = 7894 + delta
	}
	if req.SocksPort == 7891 {
		req.SocksPort = 7891 + delta
	}
}