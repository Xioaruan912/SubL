package api

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"ppeelink/models"
)

type routingRegressionResult struct {
	Case            models.RoutingRegressionCase `json:"case"`
	Passed          bool                         `json:"passed"`
	Policy          string                       `json:"policy"`
	ExpectedCountry string                       `json:"expectedCountry,omitempty"`
	ActualCountry   string                       `json:"actualCountry,omitempty"`
	MatchedRule     string                       `json:"matchedRule,omitempty"`
	Reason          string                       `json:"reason"`
}

type routingDiffItem struct {
	Domain       string `json:"domain"`
	BeforePolicy string `json:"beforePolicy"`
	AfterPolicy  string `json:"afterPolicy"`
	BeforeRule   string `json:"beforeRule"`
	AfterRule    string `json:"afterRule"`
	Changed      bool   `json:"changed"`
}

func activeRegressionCases() ([]models.RoutingRegressionCase, error) {
	var cases []models.RoutingRegressionCase
	err := models.DB.Where("enabled = ?", true).Order("id asc").Find(&cases).Error
	return cases, err
}

func regressionDomains(cases []models.RoutingRegressionCase) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(cases))
	for _, item := range cases {
		domain := strings.ToLower(strings.TrimSpace(item.Domain))
		if domain != "" && !seen[domain] {
			seen[domain] = true
			out = append(out, domain)
		}
	}
	return out
}

func evaluateRoutingRegression(ctx context.Context, filename, content string, cases []models.RoutingRegressionCase) ([]routingRegressionResult, templatePreflightReport) {
	report := buildTemplatePreflight(ctx, filename, content, regressionDomains(cases))
	routeByDomain := map[string]templatePreflightRoute{}
	for _, route := range report.Routes {
		routeByDomain[strings.ToLower(route.Domain)] = route
	}
	results := make([]routingRegressionResult, 0, len(cases))
	for _, item := range cases {
		route := routeByDomain[strings.ToLower(strings.TrimSpace(item.Domain))]
		actualCountry := countryFromText(route.Policy)
		if actualCountry == "" {
			actualCountry = policyFilterCountry(content, route.Policy)
		}
		result := routingRegressionResult{Case: item, Passed: true, Policy: route.Policy, ExpectedCountry: item.ExpectedCountry, ActualCountry: actualCountry, MatchedRule: route.MatchedRule}
		reasons := []string{}
		if route.Status != "matched" {
			result.Passed = false
			reasons = append(reasons, "目标没有得到确定规则命中")
		}
		if item.ExpectedPolicy != "" && !strings.EqualFold(strings.TrimSpace(item.ExpectedPolicy), strings.TrimSpace(route.Policy)) {
			result.Passed = false
			reasons = append(reasons, "策略不符合预期")
		}
		if item.ForbiddenPolicy != "" && strings.EqualFold(strings.TrimSpace(item.ForbiddenPolicy), strings.TrimSpace(route.Policy)) {
			result.Passed = false
			reasons = append(reasons, "命中了禁止策略")
		}
		if item.ExpectedCountry != "" && !strings.EqualFold(strings.TrimSpace(item.ExpectedCountry), actualCountry) {
			result.Passed = false
			reasons = append(reasons, "地区不符合预期")
		}
		if len(reasons) == 0 {
			reasons = append(reasons, "符合回归预期")
		}
		result.Reason = strings.Join(reasons, "；")
		results = append(results, result)
	}
	return results, report
}

func compareRoutingTemplates(ctx context.Context, filename, before, after string, cases []models.RoutingRegressionCase) ([]routingDiffItem, []routingRegressionResult, []routingRegressionResult) {
	beforeResults, beforeReport := evaluateRoutingRegression(ctx, filename, before, cases)
	afterResults, afterReport := evaluateRoutingRegression(ctx, filename, after, cases)
	byBefore := map[string]templatePreflightRoute{}
	for _, r := range beforeReport.Routes {
		byBefore[strings.ToLower(r.Domain)] = r
	}
	byAfter := map[string]templatePreflightRoute{}
	for _, r := range afterReport.Routes {
		byAfter[strings.ToLower(r.Domain)] = r
	}
	diffs := []routingDiffItem{}
	for _, domain := range regressionDomains(cases) {
		a, b := byBefore[domain], byAfter[domain]
		diffs = append(diffs, routingDiffItem{Domain: domain, BeforePolicy: a.Policy, AfterPolicy: b.Policy, BeforeRule: a.MatchedRule, AfterRule: b.MatchedRule, Changed: a.Policy != b.Policy || a.MatchedRule != b.MatchedRule})
	}
	return diffs, beforeResults, afterResults
}

func RoutingRegressionList(c *gin.Context) {
	var items []models.RoutingRegressionCase
	if err := models.DB.Order("id asc").Find(&items).Error; err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "读取回归用例失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": items, "msg": "分流回归用例"})
}

func RoutingRegressionSave(c *gin.Context) {
	var req models.RoutingRegressionCase
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": "40000", "msg": "参数错误"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Domain = strings.ToLower(strings.TrimSpace(req.Domain))
	req.ExpectedCountry = strings.ToUpper(strings.TrimSpace(req.ExpectedCountry))
	if req.Name == "" || req.Domain == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "名称和域名不能为空"})
		return
	}
	if req.Protocol == "" {
		req.Protocol = "tcp"
	}
	if req.Port == 0 {
		req.Port = 443
	}
	if req.ID == 0 {
		if err := models.DB.Create(&req).Error; err != nil {
			c.JSON(500, gin.H{"code": "50000", "msg": "保存失败"})
			return
		}
	} else {
		if err := models.DB.Model(&models.RoutingRegressionCase{}).Where("id = ?", req.ID).Updates(map[string]any{"name": req.Name, "domain": req.Domain, "expected_policy": req.ExpectedPolicy, "expected_country": req.ExpectedCountry, "forbidden_policy": req.ForbiddenPolicy, "protocol": req.Protocol, "port": req.Port, "enabled": req.Enabled, "updated_at": time.Now()}).Error; err != nil {
			c.JSON(500, gin.H{"code": "50000", "msg": "保存失败"})
			return
		}
	}
	c.JSON(200, gin.H{"code": "00000", "data": req, "msg": "已保存"})
}

func RoutingRegressionDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Query("id"), 10, 64)
	if id == 0 {
		c.JSON(400, gin.H{"code": "40000", "msg": "id 无效"})
		return
	}
	if err := models.DB.Delete(&models.RoutingRegressionCase{}, uint(id)).Error; err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "删除失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "已删除"})
}

func RoutingRegressionEvaluate(c *gin.Context) {
	filename := strings.TrimSpace(c.PostForm("filename"))
	content := c.PostForm("text")
	if content == "" && filename != "" {
		if path, err := safeFilePath(filename); err == nil {
			if body, readErr := os.ReadFile(path); readErr == nil {
				content = string(body)
			}
		}
	}
	if filename == "" || content == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "模板文件名和内容不能为空"})
		return
	}
	cases, err := activeRegressionCases()
	if err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "读取回归用例失败"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	results, report := evaluateRoutingRegression(ctx, filename, content, cases)
	passed := 0
	for _, item := range results {
		if item.Passed {
			passed++
		}
	}
	c.JSON(200, gin.H{"code": "00000", "data": gin.H{"results": results, "passed": passed, "failed": len(results) - passed, "preflightValid": report.Valid}, "msg": "回归检查完成"})
}

func RoutingRegressionCompare(c *gin.Context) {
	filename := strings.TrimSpace(c.PostForm("filename"))
	before := c.PostForm("before")
	after := c.PostForm("after")
	if filename == "" || before == "" || after == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "filename/before/after 不能为空"})
		return
	}
	cases, err := activeRegressionCases()
	if err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "读取回归用例失败"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	diff, beforeResults, afterResults := compareRoutingTemplates(ctx, filename, before, after, cases)
	c.JSON(200, gin.H{"code": "00000", "data": gin.H{"diff": diff, "before": beforeResults, "after": afterResults}, "msg": "模板分流差异"})
}
