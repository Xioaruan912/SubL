package middlewares

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"ppeelink/models"
)

func auditAction(method, path string) string {
	clean := strings.TrimPrefix(path, "/api/v1/")
	clean = strings.Trim(clean, "/")
	if clean == "" {
		clean = "root"
	}
	return strings.ToLower(method) + ":" + clean
}

// AuditTrail records only metadata for state-changing requests. It never
// stores request bodies, query strings, headers, credentials or proxy links.
func AuditTrail(c *gin.Context) {
	method := c.Request.Method
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		c.Next()
		return
	}
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/static/") {
		c.Next()
		return
	}
	c.Next()
	actor, _ := c.Get("username")
	actorText, _ := actor.(string)
	if actorText == "" {
		actorText = "anonymous"
	}
	authType, _ := c.Get("authType")
	authText, _ := authType.(string)
	entry := models.AuditLog{Actor: actorText, AuthType: authText, IP: c.ClientIP(), Method: method, Path: path, Status: c.Writer.Status(), Action: auditAction(method, path), CreatedAt: time.Now()}
	_ = models.DB.Create(&entry).Error
}
