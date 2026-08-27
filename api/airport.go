package api

import (
	"strconv"
	"time"

	"ppeelink/models"

	"github.com/gin-gonic/gin"
)

// 获取机场列表
func AirportList(c *gin.Context) {
	airports, err := models.GetAirports()
	if err != nil {
		c.JSON(500, gin.H{"msg": "获取机场列表失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"data": airports,
		"msg":  "获取成功",
	})
}

// 添加机场
func AirportAdd(c *gin.Context) {
	var a models.Airport
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(400, gin.H{"msg": "参数错误"})
		return
	}
	if a.Name == "" || a.URL == "" {
		c.JSON(400, gin.H{"msg": "名称和URL不能为空"})
		return
	}
	
	now := time.Now()
	a.LastSync = &now

	if err := a.Add(); err != nil {
		c.JSON(500, gin.H{"msg": "添加失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "添加成功"})
}

// 修改机场
func AirportUpdate(c *gin.Context) {
	var a models.Airport
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(400, gin.H{"msg": "参数错误"})
		return
	}
	if a.ID == 0 {
		c.JSON(400, gin.H{"msg": "ID不能为空"})
		return
	}
	
	var existing models.Airport
	existing.ID = a.ID
	if err := existing.Find(); err != nil {
		c.JSON(404, gin.H{"msg": "机场不存在"})
		return
	}
	
	existing.Name = a.Name
	existing.URL = a.URL
	existing.AutoCleanup = a.AutoCleanup
	existing.IsDedicated = a.IsDedicated
	
	if err := existing.Update(); err != nil {
		c.JSON(500, gin.H{"msg": "更新失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "修改成功"})
}

// 删除机场
func AirportDelete(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"msg": "无效的ID"})
		return
	}
	a := models.Airport{ID: id}
	if err := a.Delete(); err != nil {
		c.JSON(500, gin.H{"msg": "删除失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "删除成功"})
}

// 手动同步机场 (调用 Cron 的逻辑)
func AirportSync(c *gin.Context) {
	var body struct {
		ID int `json:"id" form:"id"`
	}
	// 支持 JSON 或 Form
	if err := c.ShouldBind(&body); err != nil || body.ID == 0 {
		// 备用方式
		idStr := c.PostForm("id")
		if idStr != "" {
			body.ID, _ = strconv.Atoi(idStr)
		}
		if body.ID == 0 {
			c.JSON(400, gin.H{"msg": "无效的ID"})
			return
		}
	}
	
	a := models.Airport{ID: body.ID}
	if err := a.Find(); err != nil {
		c.JSON(404, gin.H{"msg": "机场不存在"})
		return
	}

	go SyncAirportNodeTask(a.ID)

	c.JSON(200, gin.H{"code": "00000", "msg": "已在后台启动同步和测活任务，请稍后刷新查看最新数据。"})
}
