package api

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"ppeelink/models"
	"ppeelink/utils"

	"github.com/gin-gonic/gin"
)

func baseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = strings.TrimSpace(strings.Split(proto, ",")[0])
	}
	return scheme + "://" + c.Request.Host
}

type shortLinkCreateRequest struct {
	Target string `json:"target"`
	Remark string `json:"remark"`
}

// ShortLinkCreate 创建内部短链。
func ShortLinkCreate(c *gin.Context) {
	var req shortLinkCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Target) == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "target 不能为空"})
		return
	}
	target := strings.TrimSpace(req.Target)
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		c.JSON(400, gin.H{"code": "40000", "msg": "仅支持 http/https 目标"})
		return
	}
	for i := 0; i < 5; i++ {
		link := models.ShortLink{Code: utils.RandString(7), Target: target, Remark: strings.TrimSpace(req.Remark)}
		if err := models.DB.Create(&link).Error; err == nil {
			c.JSON(200, gin.H{"code": "00000", "data": gin.H{"code": link.Code, "url": baseURL(c) + "/s/" + link.Code}, "msg": "短链已创建"})
			return
		}
	}
	c.JSON(500, gin.H{"code": "50000", "msg": "生成短链失败"})
}

// ShortLinkList 列出短链。
func ShortLinkList(c *gin.Context) {
	var items []models.ShortLink
	if err := models.DB.Order("id desc").Limit(200).Find(&items).Error; err != nil {
		c.JSON(500, gin.H{"code": "50000", "msg": "读取短链失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": items, "msg": "短链列表"})
}

// ShortLinkDelete 删除短链。
func ShortLinkDelete(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"code": "40000", "msg": "无效的ID"})
		return
	}
	models.DB.Delete(&models.ShortLink{}, id)
	c.JSON(200, gin.H{"code": "00000", "msg": "已删除"})
}

// ShortLinkRedirect 短链跳转（公开）。
func ShortLinkRedirect(c *gin.Context) {
	code := c.Param("code")
	var link models.ShortLink
	if err := models.DB.Where("code = ?", code).First(&link).Error; err != nil {
		c.String(404, "短链不存在")
		return
	}
	c.Redirect(http.StatusFound, link.Target)
}

// SubscriptionImportLinks 返回订阅的直链与一键导入链接。
func SubscriptionImportLinks(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"code": "40000", "msg": "订阅 id 无效"})
		return
	}
	var sub models.Subcription
	if err := models.DB.First(&sub, id).Error; err != nil {
		c.JSON(404, gin.H{"code": "40400", "msg": "订阅不存在"})
		return
	}
	if sub.Token == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "订阅尚未生成 token，请先重置订阅令牌"})
		return
	}
	base := baseURL(c)
	links := map[string]string{}
	for _, cl := range []string{"clash", "surge", "loon", "v2ray", "singbox", "qx", "shadowrocket"} {
		links[cl] = base + "/c/?token=" + url.QueryEscape(sub.Token) + "&client=" + cl
	}
	imports := map[string]string{
		"clash":        "clash://install-config?url=" + url.QueryEscape(links["clash"]),
		"surge":        "surge://install-config?url=" + url.QueryEscape(links["surge"]),
		"loon":         "loon://install-config?url=" + url.QueryEscape(links["loon"]),
		"shadowrocket": "shadowrocket://add/sub://" + base64.RawURLEncoding.EncodeToString([]byte(links["shadowrocket"])),
	}
	c.JSON(200, gin.H{"code": "00000", "data": gin.H{"links": links, "imports": imports}, "msg": "导入链接"})
}
