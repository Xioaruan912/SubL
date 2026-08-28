package api

import (
	"strconv"
	"strings"
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
	deleteNodes := c.Query("delete_nodes") == "true"
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"msg": "无效的ID"})
		return
	}
	a := models.Airport{ID: id}
	if err := a.Find(); err != nil {
		c.JSON(500, gin.H{"msg": "删除失败: " + err.Error()})
		return
	}
	// 删除机场前，自动解除所有订阅对该机场同名分组的引用
	var gn models.GroupNode
	if err := models.DB.Where("name = ?", a.Name).Preload("Nodes").First(&gn).Error; err == nil && gn.ID != 0 {
		if deleteNodes {
			for _, n := range gn.Nodes {
				_ = n.Del()
			}
		}

		var subs []models.Subcription
		models.DB.Preload("GroupRefs").Find(&subs)
		for i := range subs {
			var refs []models.GroupNode
			for _, r := range subs[i].GroupRefs {
				if r.ID != gn.ID {
					refs = append(refs, r)
				}
			}
			if len(refs) != len(subs[i].GroupRefs) {
				_ = models.DB.Model(&subs[i]).Association("GroupRefs").Replace(refs)
			}
		}
	}
	if err := a.Delete(); err != nil {
		c.JSON(500, gin.H{"msg": "删除失败: " + err.Error()})
		return
	}
	// 删除同名分组（若已无任何引用）
	if gn.ID != 0 {
		var refCnt int64
		models.DB.Table("subcription_groups").Where("group_node_id = ?", gn.ID).Count(&refCnt)
		if refCnt == 0 {
			_ = models.DB.Model(&gn).Association("Nodes").Clear()
			_ = models.DB.Delete(&gn).Error
		}
	}
	
	InvalidateOverview() // 刷新缓存

	c.JSON(200, gin.H{"code": "00000", "msg": "删除成功"})
}

// 机场详情（含同名分组节点列表，供抽屉展示）
func AirportDetail(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"msg": "无效的ID"})
		return
	}
	a := models.Airport{ID: id}
	if err := a.Find(); err != nil {
		c.JSON(404, gin.H{"msg": "机场不存在"})
		return
	}
	var gn models.GroupNode
	nodes := []models.Node{}
	gnErr := models.DB.Where("name = ?", a.Name).Preload("Nodes").First(&gn).Error
	if gnErr == nil {
		nodes = gn.Nodes
	}
	var selected []string
	if a.SelectedNodes != "" {
		for _, s := range strings.Split(a.SelectedNodes, ",") {
			if s = strings.TrimSpace(s); s != "" {
				selected = append(selected, s)
			}
		}
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"data": gin.H{
			"id":             a.ID,
			"name":           a.Name,
			"url":            a.URL,
			"auto_cleanup":   a.AutoCleanup,
			"is_dedicated":   a.IsDedicated,
			"last_sync":      a.LastSync,
			"node_count":     a.NodeCount,
			"group_id":       gn.ID,
			"nodes":          nodes,
			"selected_nodes": selected,
		},
		"msg": "获取成功",
	})
}

// 保存机场勾选节点（按节点名，逗号分隔）
func AirportSelectNodes(c *gin.Context) {
	idStr := c.PostForm("id")
	if idStr == "" {
		c.JSON(400, gin.H{"msg": "无效的ID"})
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"msg": "无效的ID"})
		return
	}
	a := models.Airport{ID: id}
	if err := a.Find(); err != nil {
		c.JSON(404, gin.H{"msg": "机场不存在"})
		return
	}
	// 规范化存储：去重、去空
	seen := map[string]bool{}
	var names []string
	for _, s := range strings.Split(c.PostForm("nodes"), ",") {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			names = append(names, s)
		}
	}
	if err := models.DB.Model(&a).Update("selected_nodes", strings.Join(names, ",")).Error; err != nil {
		c.JSON(500, gin.H{"msg": "保存失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "保存成功"})
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
