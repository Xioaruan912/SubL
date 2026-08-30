package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"ppeelink/models"
)

func AuditLogList(c *gin.Context) {
	limit := 100
	if v, err := strconv.Atoi(c.DefaultQuery("limit", "100")); err == nil && v > 0 && v <= 500 {
		limit = v
	}
	var items []models.AuditLog
	if err := models.DB.Order("id desc").Limit(limit).Find(&items).Error; err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "读取审计日志失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": items, "msg": "管理操作审计"})
}
