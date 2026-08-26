package api

import (
	"path/filepath"
	"strings"

	"ppeelink/client"
	"ppeelink/models"

	"github.com/gin-gonic/gin"
)

// ClientList 返回客户端下载中心列表
// GET /api/v1/clients/list
func ClientList(c *gin.Context) {
	items := client.StatusList()
	c.JSON(200, gin.H{
		"code": "00000",
		"data": gin.H{
			"items":    items,
			"lastCheck": client.LastChecked().Unix(),
		},
		"msg": "客户端列表",
	})
}

// ClientCheck 手动触发检查更新
// POST /api/v1/clients/check
func ClientCheck(c *gin.Context) {
	// 后台异步执行，避免请求阻塞（下载可能耗时长）
	go func() {
		_ = client.CheckAll()
	}()
	c.JSON(200, gin.H{"code": "00000", "msg": "检查更新已启动"})
}

// ClientDownload 下载客户端文件（需登录）
// GET /api/v1/clients/download?client=&platform=
func ClientDownload(c *gin.Context) {
	name := c.Query("client")
	platform := c.Query("platform")
	if name == "" || platform == "" {
		c.JSON(400, gin.H{"code": "40000", "msg": "缺少 client 或 platform 参数"})
		return
	}
	// 校验平台合法性
	valid := false
	for _, src := range client.Sources {
		if src.Name != name {
			continue
		}
		for _, plat := range src.Platforms {
			if plat.Key == platform {
				valid = true
				break
			}
		}
	}
	if !valid {
		c.JSON(400, gin.H{"code": "40000", "msg": "无效的客户端或平台"})
		return
	}
	rec, err := (&models.ClientVersion{}).ByClientPlatform(name, platform)
	if err != nil || rec.Status != "ready" || rec.FileName == "" {
		c.JSON(404, gin.H{"code": "40400", "msg": "文件尚未就绪，请先检查更新"})
		return
	}
	path := filepath.Join("downloads", rec.FileName)
	// 附件下载文件名：client_platform_version.ext
	dlName := strings.TrimSuffix(rec.FileName, filepath.Ext(rec.FileName)) + "_" + rec.Version + filepath.Ext(rec.FileName)
	c.FileAttachment(path, dlName)
}