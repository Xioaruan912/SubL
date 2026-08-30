package api

import (
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"ppeelink/models"
	"ppeelink/node"
)

var egressTargetKeyRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,79}$`)

func enabledNodeEgressTargets() ([]node.EgressTarget, error) {
	if models.DB == nil {
		return nil, nil
	}
	items, err := models.EnabledEgressTargets()
	if err != nil {
		return nil, err
	}
	result := make([]node.EgressTarget, 0, len(items))
	for _, item := range items {
		result = append(result, node.EgressTarget{
			Key: item.Key, Name: item.Name, Domain: item.Domain, Group: item.Group,
			Path: item.Path, Method: item.Method, ExpectedStatus: item.ExpectedStatus,
			ResponseContains: item.ResponseContains, IPOptional: !item.RequireEgressIP,
			Timeout: time.Duration(item.TimeoutSeconds) * time.Second, Retries: item.Retries,
		})
	}
	return result, nil
}

func validateEgressTarget(item *models.EgressTarget) string {
	models.NormalizeEgressTarget(item)
	if !egressTargetKeyRE.MatchString(item.Key) {
		return "key 仅允许小写字母、数字、下划线和短横线"
	}
	if item.Name == "" || item.Domain == "" || item.Group == "" {
		return "名称、域名和分类不能为空"
	}
	if strings.Contains(item.Domain, "://") || strings.ContainsAny(item.Domain, "/?# ") {
		return "域名只能填写主机名，不要包含协议、路径或参数"
	}
	if net.ParseIP(item.Domain) == nil && !strings.Contains(item.Domain, ".") {
		return "域名格式无效"
	}
	if item.Method != "GET" && item.Method != "HEAD" {
		return "当前仅支持 GET 或 HEAD"
	}
	return ""
}

func EgressTargetList(c *gin.Context) {
	var items []models.EgressTarget
	if err := models.DB.Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "读取检测目标失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": items, "msg": "检测目标"})
}

func EgressTargetSave(c *gin.Context) {
	var input models.EgressTarget
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"code": "40000", "msg": "参数格式错误"})
		return
	}
	if msg := validateEgressTarget(&input); msg != "" {
		c.JSON(400, gin.H{"code": "40000", "msg": msg})
		return
	}
	if input.ID > 0 {
		var existing models.EgressTarget
		if err := models.DB.First(&existing, input.ID).Error; err != nil {
			c.JSON(404, gin.H{"code": "40400", "msg": "检测目标不存在"})
			return
		}
		updates := map[string]any{
			"key": input.Key, "name": input.Name, "domain": input.Domain, "group": input.Group,
			"icon": input.Icon, "path": input.Path, "method": input.Method,
			"expected_status": input.ExpectedStatus, "response_contains": input.ResponseContains,
			"require_egress_ip": input.RequireEgressIP, "timeout_seconds": input.TimeoutSeconds,
			"retries": input.Retries, "enabled": input.Enabled, "sort_order": input.SortOrder,
		}
		if err := models.DB.Model(&existing).Updates(updates).Error; err != nil {
			c.JSON(409, gin.H{"code": "40900", "msg": "保存失败，key 可能已存在"})
			return
		}
		c.JSON(200, gin.H{"code": "00000", "data": existing.ID, "msg": "已更新"})
		return
	}
	if err := models.DB.Create(&input).Error; err != nil {
		c.JSON(409, gin.H{"code": "40900", "msg": "保存失败，key 可能已存在"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": input.ID, "msg": "已新增"})
}

func EgressTargetDelete(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "缺少 id"})
		return
	}
	result := models.DB.Delete(&models.EgressTarget{}, id)
	if result.Error != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "删除失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(404, gin.H{"code": "40400", "msg": "检测目标不存在"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "已删除"})
}
