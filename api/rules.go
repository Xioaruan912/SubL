package api

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"ppeelink/models"
	"ppeelink/rulecenter"
)

type ruleImportRequest struct {
	RuleIDs        []string `json:"ruleIds"`
	Template       string   `json:"template"`
	Policy         string   `json:"policy"`
	Mode           string   `json:"mode"`
	Position       string   `json:"position"`
	ConflictPolicy string   `json:"conflictPolicy"`
	Proxy          string   `json:"proxy"`
}

func RuleSources(c *gin.Context) {
	sources, err := rulecenter.Sources()
	if err != nil {
		c.JSON(500, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": sources, "msg": "ok"})
}

func RuleCatalog(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "48"))
	items, total, err := rulecenter.ListCatalog(c.Query("source"), c.Query("platform"), c.Query("category"), c.Query("keyword"), page, pageSize)
	if err != nil {
		c.JSON(500, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize}, "msg": "ok"})
}

func RulePreview(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(400, gin.H{"msg": "规则 id 不能为空"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 75*time.Second)
	defer cancel()
	item, _, err := rulecenter.LoadItem(ctx, id)
	if err != nil {
		c.JSON(502, gin.H{"msg": "读取规则失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": item, "msg": "ok"})
}

func RuleTemplateGroups(c *gin.Context) {
	filename := strings.TrimSpace(c.Query("template"))
	if filename == "" {
		c.JSON(400, gin.H{"msg": "模板不能为空"})
		return
	}
	path, err := safeFilePath(filename)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	body, err := os.ReadFile(path)
	if err != nil {
		c.JSON(404, gin.H{"msg": "模板不存在"})
		return
	}
	var root struct {
		ProxyGroups []struct {
			Name string `yaml:"name"`
		} `yaml:"proxy-groups"`
	}
	if err := yaml.Unmarshal(body, &root); err != nil {
		c.JSON(400, gin.H{"msg": "模板解析失败: " + err.Error()})
		return
	}
	groups := make([]string, 0, len(root.ProxyGroups))
	seen := map[string]bool{}
	for _, group := range root.ProxyGroups {
		name := strings.TrimSpace(group.Name)
		if name != "" && !seen[name] {
			seen[name] = true
			groups = append(groups, name)
		}
	}
	c.JSON(200, gin.H{"code": "00000", "data": groups, "msg": "ok"})
}

func RuleSync(c *gin.Context) {
	source := strings.TrimSpace(c.PostForm("source"))
	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()
	var err error
	if source == "" {
		err = rulecenter.SyncAll(ctx)
	} else {
		err = rulecenter.SyncSource(ctx, source)
	}
	if err != nil {
		c.JSON(502, gin.H{"msg": "同步失败: " + err.Error()})
		return
	}
	sources, _ := rulecenter.Sources()
	c.JSON(200, gin.H{"code": "00000", "data": sources, "msg": "同步完成"})
}

func RuleImport(c *gin.Context) {
	var req ruleImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"msg": "请求格式错误: " + err.Error()})
		return
	}
	if len(req.RuleIDs) == 0 || req.Template == "" || req.Policy == "" {
		c.JSON(400, gin.H{"msg": "规则、模板和策略组不能为空"})
		return
	}
	if req.Mode != "" && req.Mode != "provider" {
		c.JSON(400, gin.H{"msg": "V1 当前仅支持 Rule Provider 导入"})
		return
	}
	path, err := safeFilePath(req.Template)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	body, err := os.ReadFile(path)
	if err != nil {
		c.JSON(404, gin.H{"msg": "模板不存在"})
		return
	}
	text := string(body)
	results := make([]rulecenter.ImportResult, 0, len(req.RuleIDs))
	warnings := []string{}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	for _, id := range req.RuleIDs {
		item, _, loadErr := rulecenter.LoadItem(ctx, id)
		if loadErr != nil {
			c.JSON(502, gin.H{"msg": "读取规则失败: " + loadErr.Error()})
			return
		}
		if !strings.EqualFold(item.Platform, "Clash") {
			c.JSON(400, gin.H{"msg": "V1 模板写入当前仅实现 Clash，所选规则包含 " + item.Platform})
			return
		}
		result, importErr := rulecenter.ImportClashProvider(text, rulecenter.ImportOptions{
			ProviderName:   item.Name,
			URL:            item.URL,
			Policy:         req.Policy,
			Behavior:       "classical",
			Format:         "yaml",
			Proxy:          req.Proxy,
			ConflictPolicy: req.ConflictPolicy,
			Interval:       3600,
		})
		if importErr != nil {
			c.JSON(400, gin.H{"msg": "导入失败: " + importErr.Error()})
			return
		}
		text = result.Text
		results = append(results, result)
		warnings = append(warnings, result.Warnings...)
	}
	_, validationErrors := validateTemplateText(req.Template, text)
	if len(validationErrors) > 0 {
		c.JSON(400, gin.H{"msg": "导入后模板校验失败", "errors": validationErrors})
		return
	}
	_ = models.SaveTemplateVersion(req.Template, string(body), "before_rule_import")
	if err := atomicWrite(path, text); err != nil {
		c.JSON(500, gin.H{"msg": "模板写入失败: " + err.Error()})
		return
	}
	_ = models.SaveTemplateVersion(req.Template, text, "rule_import")
	c.JSON(200, gin.H{"code": "00000", "data": gin.H{"results": results, "warnings": warnings, "template": req.Template}, "msg": "规则导入完成"})
}
