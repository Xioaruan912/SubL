package api

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"ppeelink/models"
	"ppeelink/node"

	"github.com/gin-gonic/gin"
)

// parseMemberIDs 解析合集成员订阅 ID（支持 JSON 数组或逗号分隔）。
func parseMemberIDs(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "[") {
		var ids []int
		if err := json.Unmarshal([]byte(raw), &ids); err == nil {
			return ids
		}
	}
	return parseIDList(raw)
}

// collectionNodes 合并合集内所有订阅的节点，可选去重并套用合集处理链。
func collectionNodes(col models.Collection) []models.Node {
	ids := parseMemberIDs(col.MemberIDs)
	merged := []models.Node{}
	seen := map[string]bool{}
	for _, id := range ids {
		var sub models.Subcription
		if err := models.DB.First(&sub, id).Error; err != nil {
			continue
		}
		sub.Pipeline = "" // 合集层统一处理，避免与成员链叠加
		if err := mergeGroupNodes(&sub); err != nil {
			continue
		}
		for _, n := range sub.Nodes {
			if col.Dedupe {
				key := strings.ToLower(strings.TrimSpace(n.Link))
				if key == "" || seen[key] {
					continue
				}
				seen[key] = true
			}
			merged = append(merged, n)
		}
	}
	if strings.TrimSpace(col.Pipeline) != "" {
		if preview, err := ApplySubscriptionPipeline(merged, col.Pipeline); err == nil {
			merged = preview.Nodes
		}
	}
	return merged
}

// renderCollectionNodes 按客户端渲染合并后的节点。
func renderCollectionNodes(client string, nodes []models.Node, configs node.SqlConfig) ([]byte, string, string) {
	urls, flags := collectNodeInputs(nodes)
	switch client {
	case "clash":
		b, err := node.EncodeClashWithFlags(urls, flags, configs)
		if err != nil {
			return []byte(err.Error()), "yaml", "text/plain; charset=utf-8"
		}
		return b, "yaml", "text/plain; charset=utf-8"
	case "surge":
		s, err := node.EncodeSurgeWithFlags(urls, flags, configs)
		if err != nil {
			return []byte(err.Error()), "conf", "text/plain; charset=utf-8"
		}
		return []byte(s), "conf", "text/plain; charset=utf-8"
	case "loon":
		s, err := node.EncodeLoonWithFlags(urls, flags, configs)
		if err != nil {
			return []byte(err.Error()), "conf", "text/plain; charset=utf-8"
		}
		return []byte(s), "conf", "text/plain; charset=utf-8"
	case "singbox":
		s, err := node.EncodeSingbox(urls, configs)
		if err != nil {
			return []byte(err.Error()), "json", "text/plain; charset=utf-8"
		}
		return []byte(s), "json", "application/json; charset=utf-8"
	case "qx":
		s, err := node.EncodeQX(urls, configs)
		if err != nil {
			return []byte(err.Error()), "conf", "text/plain; charset=utf-8"
		}
		return []byte(s), "conf", "text/plain; charset=utf-8"
	default: // v2ray / shadowrocket
		var b strings.Builder
		for _, l := range urls {
			b.WriteString(l)
			b.WriteString("\n")
		}
		return []byte(node.Base64Encode(b.String())), "txt", "text/html; charset=utf-8"
	}
}

// GetCollectionClient 合集统一下发（公开，按 token 匹配）。
func GetCollectionClient(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.Writer.WriteString("token为空")
		return
	}
	var col models.Collection
	if err := models.DB.Where("token = ?", token).First(&col).Error; err != nil {
		c.Writer.WriteString("无效的合集令牌")
		return
	}
	if col.ExpiresAt != nil && time.Now().After(*col.ExpiresAt) {
		c.Writer.WriteString("合集已过期")
		return
	}
	client := normalizeClient(c.Query("client"))
	if client == "" {
		client = clientFromUserAgent(c.Request.UserAgent())
	}
	nodes := collectionNodes(col)
	if len(nodes) == 0 {
		c.Writer.WriteString("合集没有可用节点")
		return
	}
	var configs node.SqlConfig
	if strings.TrimSpace(col.Config) != "" {
		_ = json.Unmarshal([]byte(col.Config), &configs)
	}
	body, ext, contentType := renderCollectionNodes(client, nodes, configs)
	if col.ExpiresAt != nil {
		c.Header("Subscription-Userinfo", "upload=0; download=0; total=0; expire="+strconv.FormatInt(col.ExpiresAt.Unix(), 10))
	}
	c.Header("Content-Disposition", "inline; filename*=utf-8''"+url.QueryEscape(col.Name+"."+ext))
	c.Data(200, contentType, body)
}

// CollectionList 合集列表。
func CollectionList(c *gin.Context) {
	var cols []models.Collection
	if err := models.DB.Order("id desc").Find(&cols).Error; err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "读取合集失败"})
		return
	}
	items := make([]gin.H, 0, len(cols))
	for _, col := range cols {
		_ = col.EnsureToken()
		items = append(items, gin.H{
			"id": col.ID, "name": col.Name, "token": col.Token, "memberIds": col.MemberIDs,
			"dedupe": col.Dedupe, "pipeline": col.Pipeline, "config": col.Config,
			"note": col.Note, "expiresAt": col.ExpiresAt, "createdAt": col.CreatedAt,
		})
	}
	c.JSON(200, gin.H{"code": "00000", "data": items, "msg": "合集列表"})
}

// CollectionAdd 新建合集。
func CollectionAdd(c *gin.Context) {
	var col models.Collection
	if err := c.ShouldBindJSON(&col); err != nil {
		c.JSON(400, gin.H{"code": "40000", "msg": "参数格式错误"})
		return
	}
	col.Name = strings.TrimSpace(col.Name)
	if col.Name == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "名称不能为空"})
		return
	}
	col.ID = 0
	col.Token = models.GenerateToken()
	if err := models.DB.Create(&col).Error; err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "创建失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": col, "msg": "合集已创建"})
}

// CollectionUpdate 修改合集。
func CollectionUpdate(c *gin.Context) {
	var req models.Collection
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		c.JSON(400, gin.H{"code": "40000", "msg": "参数格式错误"})
		return
	}
	var col models.Collection
	if err := models.DB.First(&col, req.ID).Error; err != nil {
		c.JSON(404, gin.H{"code": "40400", "msg": "合集不存在"})
		return
	}
	col.Name = strings.TrimSpace(req.Name)
	col.MemberIDs = req.MemberIDs
	col.Dedupe = req.Dedupe
	col.Pipeline = req.Pipeline
	col.Config = req.Config
	col.Note = req.Note
	col.ExpiresAt = req.ExpiresAt
	if err := models.DB.Save(&col).Error; err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "保存失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": col, "msg": "已保存"})
}

// CollectionDelete 删除合集。
func CollectionDelete(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"code": "40000", "msg": "无效的ID"})
		return
	}
	models.DB.Delete(&models.Collection{}, id)
	c.JSON(200, gin.H{"code": "00000", "msg": "已删除"})
}

// CollectionResetToken 重置合集令牌。
func CollectionResetToken(c *gin.Context) {
	id, err := strconv.Atoi(c.PostForm("id"))
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"code": "40000", "msg": "无效的ID"})
		return
	}
	var col models.Collection
	if err := models.DB.First(&col, id).Error; err != nil {
		c.JSON(404, gin.H{"code": "40400", "msg": "合集不存在"})
		return
	}
	token := models.GenerateToken()
	if err := models.DB.Model(&col).Update("token", token).Error; err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "重置失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": gin.H{"token": token}, "msg": "令牌已重置"})
}
