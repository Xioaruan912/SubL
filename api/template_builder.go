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

// BuilderRequest 构建模板请求
type BuilderRequest struct {
	Filename  string          `json:"filename"`
	Port      int             `json:"port"`
	SocksPort int             `json:"socks_port"`
	AllowLan  bool            `json:"allow_lan"`
	Mode      string          `json:"mode"`
	TestURL   string          `json:"test_url"`
	Interval  int             `json:"interval"`
	Groups    []BuilderGroup  `json:"groups"`
}

// clashBuilder 用于 yaml 序列化的 clash 配置
type clashBuilder struct {
	Port           int                   `yaml:"port"`
	SocksPort      int                   `yaml:"socks-port"`
	AllowLan       bool                  `yaml:"allow-lan"`
	Mode           string                `yaml:"mode"`
	LogLevel       string                `yaml:"log-level"`
	Proxies        interface{}           `yaml:"proxies"` // ~ 占位，订阅时填充
	ProxyGroups    []map[string]any      `yaml:"proxy-groups"`
	Rules          []string              `yaml:"rules"`
}

// 默认规则（与 template/clash.yaml 保持一致的精简版）
var defaultClashRules = []string{
	"DOMAIN-SUFFIX,local,🎯 全球直连",
	"IP-CIDR,192.168.0.0/16,🎯 全球直连,no-resolve",
	"IP-CIDR,10.0.0.0/8,🎯 全球直连,no-resolve",
	"IP-CIDR,172.16.0.0/12,🎯 全球直连,no-resolve",
	"IP-CIDR,127.0.0.0/8,🎯 全球直连,no-resolve",
	"IP-CIDR,100.64.0.0/10,🎯 全球直连,no-resolve",
	"GEOIP,CN,🎯 全球直连",
	"MATCH,🐟 漏网之鱼",
}

// 常用测速 URL
var defaultTestURL = "http://www.gstatic.com/generate_204"

// TemplateBuild 通过表单输入生成 clash 配置并保存到模板目录。
// POST /api/v1/template/build
func TemplateBuild(c *gin.Context) {
	req := BuilderRequest{}
	// 支持 JSON body 或表单
	ct := c.GetHeader("Content-Type")
	if strings.Contains(ct, "application/json") {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"code": "40000", "msg": "请求体解析失败: " + err.Error()})
			return
		}
	} else {
		req.Filename = c.PostForm("filename")
		req.Port, _ = strconv.Atoi(c.PostForm("port"))
		req.SocksPort, _ = strconv.Atoi(c.PostForm("socks_port"))
		req.AllowLan = c.PostForm("allow_lan") == "true" || c.PostForm("allow_lan") == "1"
		req.Mode = c.PostForm("mode")
		req.TestURL = c.PostForm("test_url")
		req.Interval, _ = strconv.Atoi(c.PostForm("interval"))
		if g := c.PostForm("groups"); g != "" {
			_ = json.Unmarshal([]byte(g), &req.Groups)
		}
	}

	// 校验
	if req.Filename == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "文件名不能为空"})
		return
	}
	if !strings.HasSuffix(req.Filename, ".yaml") {
		req.Filename += ".yaml"
	}
	if req.Port == 0 {
		req.Port = 7890
	}
	if req.SocksPort == 0 {
		req.SocksPort = 7891
	}
	if req.Mode == "" {
		req.Mode = "Rule"
	}
	if req.TestURL == "" {
		req.TestURL = defaultTestURL
	}
	if req.Interval == 0 {
		req.Interval = 300
	}

	yamlText, err := buildClashYAML(req)
	if err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "生成配置失败: " + err.Error()})
		return
	}

	// 保存到模板目录（复用 safeFilePath 防目录遍历）
	fullPath, err := safeFilePath(req.Filename)
	if err != nil {
		c.JSON(400, gin.H{"code": "40000", "msg": "文件名非法: " + err.Error()})
		return
	}
	if err := os.WriteFile(fullPath, []byte(yamlText), 0666); err != nil {
		log.Println("写入模板失败:", err)
		c.JSON(500, gin.H{"code": "50000", "msg": "保存模板失败"})
		return
	}

	c.JSON(200, gin.H{
		"code": "00000",
		"msg":  "模板已保存",
		"data": gin.H{
			"filename": req.Filename,
			"yaml":     yamlText,
		},
	})
}

// buildClashYAML 根据表单输入生成 clash YAML
func buildClashYAML(req BuilderRequest) (string, error) {
	cfg := clashBuilder{
		Port:      req.Port,
		SocksPort: req.SocksPort,
		AllowLan:  req.AllowLan,
		Mode:      req.Mode,
		LogLevel:  "info",
		Proxies:   "~",
		Rules:     defaultClashRules,
	}

	// 构造分组
	if len(req.Groups) == 0 {
		// 默认一个节点选择组
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
		}
		if len(g.Proxies) > 0 {
			group["proxies"] = g.Proxies
		}
		groups = append(groups, group)
	}
	cfg.ProxyGroups = groups

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	// 将 proxies 占位输出为 ~（与现有模板格式一致）
	out := strings.Replace(string(data), "proxies: \"~\"", "proxies: ~", 1)
	return out, nil
}