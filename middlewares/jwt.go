package middlewares

import (
	"errors"
	"net/http"
	"ppeelink/models"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// 随机密钥

// var Secret = []byte("sublink") // 秘钥
var Secret = []byte(models.ReadConfig().JwtSecret) // 从配置文件读取JWT密钥

// JwtClaims jwt声明
type JwtClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// AuthorToken 验证token中间件
func AuthorToken(c *gin.Context) {
	// 定义白名单
	list := []string{"/static", "/api/v1/auth/login", "/api/v1/auth/captcha", "/c/", "/api/v1/version", "/status", "/api/v1/status/public"}
	// 如果是首页直接跳过
	if c.Request.URL.Path == "/" {
		c.Next()
		return
	}
	// 如果是白名单直接跳过
	for _, v := range list {
		if strings.HasPrefix(c.Request.URL.Path, v) {
			c.Next()
			return
		}
	}
	authorization := strings.TrimSpace(c.Request.Header.Get("Authorization"))
	if authorization == "" {
		c.JSON(400, gin.H{"msg": "请求未携带token"})
		c.Abort()
		return
	}
	credential := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	if credential == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "token为空"})
		c.Abort()
		return
	}
	if strings.Count(credential, ".") == 2 {
		if mc, err := ParseToken(credential); err == nil {
			c.Set("username", mc.Username)
			c.Set("authType", "jwt")
			c.Next()
			return
		}
	}
	var apiToken models.APIToken
	if err := models.DB.Where("token_hash = ? AND enabled = ?", models.HashAPIToken(credential), true).First(&apiToken).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "无效的 API Token"})
		c.Abort()
		return
	}
	if apiToken.ExpiresAt != nil && time.Now().After(*apiToken.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "API Token 已过期"})
		c.Abort()
		return
	}
	required := "read"
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead && c.Request.Method != http.MethodOptions {
		required = "write"
	}
	adminPaths := []string{"/api/v1/tokens", "/api/v1/tasks/safe-publish", "/api/v1/tasks/system-deploy", "/api/v1/ops/backup/import", "/api/v1/audit"}
	for _, prefix := range adminPaths {
		if strings.HasPrefix(c.Request.URL.Path, prefix) {
			required = "admin"
			break
		}
	}
	if !apiToken.HasScope(required) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "API Token 权限不足，需要 " + required})
		c.Abort()
		return
	}
	now := time.Now()
	_ = models.DB.Model(&apiToken).Update("last_used_at", &now).Error
	c.Set("username", "api-token:"+apiToken.Name)
	c.Set("authType", "api-token")
	c.Set("apiTokenId", apiToken.ID)
	c.Set("apiScopes", apiToken.Scopes)
	c.Next()
}

// ParseToken 解析JWT
func ParseToken(tokenString string) (*JwtClaims, error) {
	// 解析token
	token, err := jwt.ParseWithClaims(tokenString, &JwtClaims{}, func(token *jwt.Token) (i interface{}, err error) {
		return Secret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*JwtClaims); ok && token.Valid { // 校验token
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
