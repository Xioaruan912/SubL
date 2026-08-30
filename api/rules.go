package api

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"ppeelink/models"
	"ppeelink/rulecenter"
)

type affectedRuleTemplate struct {
	Template string `json:"template"`
	Policy   string `json:"policy"`
}
type ruleRegressionImpact struct {
	Template    string `json:"template"`
	Target      string `json:"target"`
	Domain      string `json:"domain"`
	Policy      string `json:"policy"`
	BeforeInSet bool   `json:"beforeInSet"`
	AfterInSet  bool   `json:"afterInSet"`
	Changed     bool   `json:"changed"`
}

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

func findAffectedRuleTemplates(providerName string) []affectedRuleTemplate {
	entries, err := os.ReadDir("template")
	if err != nil {
		return nil
	}
	out := []affectedRuleTemplate{}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".yaml") && !strings.HasSuffix(strings.ToLower(name), ".yml") && !strings.HasSuffix(strings.ToLower(name), ".conf") {
			continue
		}
		body, err := os.ReadFile(filepath.Join("template", name))
		if err != nil {
			continue
		}
		rules := parseSplitRules(string(body))
		matched := false
		for _, r := range rules {
			if r.Kind == "RULE-SET" && strings.EqualFold(strings.TrimSpace(r.Domain), strings.TrimSpace(providerName)) {
				key := name + "\x00" + r.Policy
				if !seen[key] {
					seen[key] = true
					out = append(out, affectedRuleTemplate{Template: name, Policy: r.Policy})
				}
				matched = true
			}
		}
		if !matched && strings.Contains(strings.ToLower(string(body)), strings.ToLower(providerName)) {
			key := name + "\x00"
			if !seen[key] {
				seen[key] = true
				out = append(out, affectedRuleTemplate{Template: name})
			}
		}
	}
	return out
}

func RuleUpdateImpact(c *gin.Context) {
	id := strings.TrimSpace(c.Query("id"))
	if id == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "规则 id 不能为空"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	preview, err := rulecenter.PreviewCatalogUpdate(ctx, id)
	if err != nil {
		c.JSON(502, gin.H{"code": "50200", "msg": "更新预览失败: " + err.Error()})
		return
	}
	affected := findAffectedRuleTemplates(preview.Name)
	targets, _ := models.EnabledEgressTargets()
	domains := make([]string, 0, len(targets))
	nameByDomain := map[string]string{}
	for _, t := range targets {
		domains = append(domains, t.Domain)
		nameByDomain[t.Domain] = t.Name
	}
	matches, matchErr := rulecenter.PreviewCatalogDomainMatches(ctx, id, domains)
	regression := []ruleRegressionImpact{}
	if matchErr == nil {
		for _, tpl := range affected {
			for _, m := range matches {
				if m.Changed {
					regression = append(regression, ruleRegressionImpact{Template: tpl.Template, Target: nameByDomain[m.Domain], Domain: m.Domain, Policy: tpl.Policy, BeforeInSet: m.Before, AfterInSet: m.After, Changed: true})
				}
			}
		}
	}
	warnings := append([]string{}, preview.Warnings...)
	if matchErr != nil {
		warnings = append(warnings, "分流回归比较失败: "+matchErr.Error())
	}
	c.JSON(200, gin.H{"code": "00000", "data": gin.H{"preview": preview, "affectedTemplates": affected, "regression": regression, "warnings": warnings}, "msg": "规则更新影响预览"})
}

func RuleApplyUpdate(c *gin.Context) {
	id := strings.TrimSpace(c.Query("id"))
	if id == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "规则 id 不能为空"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	preview, err := rulecenter.ApplyCatalogUpdate(ctx, id)
	if err != nil {
		c.JSON(502, gin.H{"code": "50200", "msg": "更新失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": preview, "msg": "规则已更新并保留回滚快照"})
}

func RuleSnapshots(c *gin.Context) {
	id := strings.TrimSpace(c.Query("id"))
	items, err := rulecenter.RuleSnapshots(id)
	if err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "读取快照失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": items, "msg": "规则快照"})
}

func RuleRollback(c *gin.Context) {
	id := strings.TrimSpace(c.Query("id"))
	snapshotID64, _ := strconv.ParseUint(c.Query("snapshotId"), 10, 64)
	if id == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "规则 id 不能为空"})
		return
	}
	if err := rulecenter.RollbackCatalog(id, uint(snapshotID64)); err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "回滚失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "规则已回滚"})
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
	name := "规则中心同步"
	if source != "" {
		name += " · " + source
	}
	task, taskCtx, taskErr := createTaskRun(c.Request.Context(), "rule-sync", name, ruleSyncTaskRequest{Source: source}, nil)
	if taskErr != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "创建任务记录失败: " + taskErr.Error()})
		return
	}
	updateTaskProgress(task.ID, 15, "正在同步规则源")
	ctx, cancel := context.WithTimeout(taskCtx, 120*time.Second)
	defer cancel()
	var err error
	if source == "" {
		err = rulecenter.SyncAll(ctx)
	} else {
		err = rulecenter.SyncSource(ctx, source)
	}
	if err != nil {
		if taskCtx.Err() == context.Canceled {
			markTaskCancelled(task.ID)
		} else {
			finishTaskRun(task.ID, err, nil)
		}
		c.JSON(502, gin.H{"msg": "同步失败: " + err.Error()})
		return
	}
	sources, _ := rulecenter.Sources()
	finishTaskRun(task.ID, nil, sources)
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
