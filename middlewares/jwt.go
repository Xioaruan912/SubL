package middlewares

import (
	"errors"
	"log"
	"net/http"
	"ppeelink/models"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// Secret 在配置初始化完成后由 InitSecret 加载，避免包初始化阶段读到空配置。
var Secret []byte

// InitSecret 加载 JWT 密钥；密钥为空时终止启动，避免用空密钥签发/校验 token。
func InitSecret() {
	Secret = []byte(models.ReadConfig().JwtSecret)
	if len(Secret) == 0 {
		log.Fatal("JWT 密钥为空，已拒绝启动以避免空密钥 token 风险")
	}
}

// JwtClaims jwt声明
type JwtClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// AuthorToken 验证token中间件
func AuthorToken(c *gin.Context) {
	// 定义白名单
	list := []string{"/static", "/api/v1/auth/login", "/api/v1/auth/captcha", "/c/", "/api/v1/version", "/status", "/api/v1/status/public", "/favicon.ico"}
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
	required := requiredScope(c.Request.Method, c.Request.URL.Path)
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

// requiredScope 根据请求方法与路径判定 API Token 所需权限。
// 读操作默认 read；高风险写操作（模板、规则、系统部署、备份导入）与
// 令牌/审计、备份导出需要 admin；其余写操作需要 write。
func requiredScope(method, path string) string {
	isRead := method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
	for _, prefix := range []string{"/api/v1/tokens", "/api/v1/audit"} {
		if strings.HasPrefix(path, prefix) {
			return "admin"
		}
	}
	if isRead {
		for _, prefix := range []string{"/api/v1/ops/backup/export", "/api/v1/ops/backup/inspect"} {
			if strings.HasPrefix(path, prefix) {
				return "admin"
			}
		}
		return "read"
	}
	for _, prefix := range []string{
		"/api/v1/template",
		"/api/v1/rules",
		"/api/v1/ops/backup/import",
		"/api/v1/tasks/safe-publish",
		"/api/v1/tasks/system-deploy",
	} {
		if strings.HasPrefix(path, prefix) {
			return "admin"
		}
	}
	return "write"
}

// ParseToken 解析JWT
func ParseToken(tokenString string) (*JwtClaims, error) {
	if len(Secret) == 0 {
		return nil, errors.New("JWT 密钥未初始化")
	}
	// 只接受 HS256，避免算法混淆
	token, err := jwt.ParseWithClaims(tokenString, &JwtClaims{}, func(token *jwt.Token) (i interface{}, err error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
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
