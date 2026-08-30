package api

import (
	"fmt"
	"log"
	"ppeelink/middlewares"
	"ppeelink/models"
	"ppeelink/utils"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

type loginAttempt struct {
	Failures     int
	First        time.Time
	BlockedUntil time.Time
}

var loginAttempts = struct {
	sync.Mutex
	Items map[string]loginAttempt
}{Items: map[string]loginAttempt{}}

const loginWindow = 10 * time.Minute
const loginBlock = 15 * time.Minute
const loginMaxFailures = 5

func loginAttemptKey(c *gin.Context, username string) string {
	return c.ClientIP() + "|" + strings.ToLower(strings.TrimSpace(username))
}
func loginBlocked(key string) (bool, time.Duration) {
	loginAttempts.Lock()
	defer loginAttempts.Unlock()
	item, ok := loginAttempts.Items[key]
	if !ok {
		return false, 0
	}
	now := time.Now()
	if !item.BlockedUntil.IsZero() && now.Before(item.BlockedUntil) {
		return true, time.Until(item.BlockedUntil)
	}
	if now.Sub(item.First) > loginWindow {
		delete(loginAttempts.Items, key)
		return false, 0
	}
	return false, 0
}
func recordLoginFailure(key string) {
	loginAttempts.Lock()
	defer loginAttempts.Unlock()
	now := time.Now()
	item := loginAttempts.Items[key]
	if item.First.IsZero() || now.Sub(item.First) > loginWindow {
		item = loginAttempt{First: now}
	}
	item.Failures++
	if item.Failures >= loginMaxFailures {
		item.BlockedUntil = now.Add(loginBlock)
	}
	loginAttempts.Items[key] = item
}
func clearLoginFailures(key string) {
	loginAttempts.Lock()
	delete(loginAttempts.Items, key)
	loginAttempts.Unlock()
}

// 获取token
func GetToken(username string) (string, error) {
	// 过期时间天
	ExpireDays := models.ReadConfig().ExpireDays
	c := &middlewares.JwtClaims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24 * time.Duration(ExpireDays)).Unix(), // 设置过期时间
			IssuedAt:  time.Now().Unix(),                                                 // 签发时间
			Subject:   username,                                                          // 用户
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(middlewares.Secret)
}

// 获取captcha图形验证码
func GetCaptcha(c *gin.Context) {
	id, bs4, _, err := utils.GetCaptcha()
	if err != nil {
		log.Println("获取验证码失败")
		c.JSON(400, gin.H{
			"msg": "获取验证码失败",
		})
		return
	}
	c.JSON(200, gin.H{
		"code": "00000",
		"data": gin.H{
			"captchaKey":    id,
			"captchaBase64": bs4,
		},
		"msg": "获取验证码成功",
	})

}

// 用户登录
func UserLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	captchaCode := c.PostForm("captchaCode")
	captchaKey := c.PostForm("captchaKey")
	attemptKey := loginAttemptKey(c, username)
	if blocked, remaining := loginBlocked(attemptKey); blocked {
		c.JSON(429, gin.H{"code": "42900", "msg": fmt.Sprintf("登录失败次数过多，请约 %d 分钟后重试", int(remaining.Minutes())+1)})
		return
	}
	// 验证验证码
	if !utils.VerifyCaptcha(captchaKey, captchaCode) {
		log.Println("验证码错误")
		c.JSON(400, gin.H{
			"msg": "验证码错误",
		})
		return
	}
	user := &models.User{Username: username, Password: password}
	err := user.Verify()
	if err != nil {
		recordLoginFailure(attemptKey)
		log.Println("账号或者密码错误")
		c.JSON(400, gin.H{
			"msg": "账号或者密码错误",
		})
		return
	}
	clearLoginFailures(attemptKey)
	// 生成token
	token, err := GetToken(username)
	if err != nil {
		log.Println("获取token失败", err)
		c.JSON(400, gin.H{
			"msg": "获取token失败",
		})
		return
	}
	// 登录成功返回token
	c.JSON(200, gin.H{
		"code": "00000",
		"data": gin.H{
			"accessToken":  token,
			"tokenType":    "Bearer",
			"refreshToken": nil,
			"expires":      nil,
		},
		"msg": "登录成功",
	})
}
func UserOut(c *gin.Context) {
	// 拿到jwt中的username
	if _, Is := c.Get("username"); Is {
		c.JSON(200, gin.H{
			"code": "00000",
			"msg":  "退出成功",
		})
	}
}
