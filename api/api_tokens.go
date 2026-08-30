package api

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"ppeelink/models"
)

type createAPITokenRequest struct { Name string `json:"name"`; Scopes []string `json:"scopes"`; ExpiresInDays int `json:"expiresInDays"` }

func APITokenList(c *gin.Context){var items []models.APIToken;if err:=models.DB.Order("id desc").Find(&items).Error;err!=nil{c.JSON(500,gin.H{"code":"50000","msg":"读取 API Token 失败"});return};c.JSON(200,gin.H{"code":"00000","data":items,"msg":"API Token"})}

func APITokenCreate(c *gin.Context){var req createAPITokenRequest;if err:=c.ShouldBindJSON(&req);err!=nil{c.JSON(400,gin.H{"code":"40000","msg":"参数格式错误"});return};req.Name=strings.TrimSpace(req.Name);if req.Name==""{c.JSON(400,gin.H{"code":"40000","msg":"名称不能为空"});return};allowed:=map[string]bool{"read":true,"write":true,"admin":true};seen:=map[string]bool{};scopes:=[]string{};for _,scope:=range req.Scopes{s:=strings.ToLower(strings.TrimSpace(scope));if allowed[s]&&!seen[s]{seen[s]=true;scopes=append(scopes,s)}};if len(scopes)==0{scopes=[]string{"read"}};plain,prefix,hash,err:=models.GenerateAPIToken();if err!=nil{c.JSON(500,gin.H{"code":"50000","msg":"生成 Token 失败"});return};var expires *time.Time;if req.ExpiresInDays>0{if req.ExpiresInDays>3650{req.ExpiresInDays=3650};value:=time.Now().Add(time.Duration(req.ExpiresInDays)*24*time.Hour);expires=&value};item:=models.APIToken{Name:req.Name,TokenPrefix:prefix,TokenHash:hash,Scopes:strings.Join(scopes,","),ExpiresAt:expires,Enabled:true};if err:=models.DB.Create(&item).Error;err!=nil{c.JSON(500,gin.H{"code":"50000","msg":"保存 Token 失败"});return};c.JSON(200,gin.H{"code":"00000","data":gin.H{"id":item.ID,"token":plain,"prefix":prefix,"scopes":scopes,"expiresAt":expires},"msg":"API Token 已创建；明文仅显示这一次"})}

func APITokenRevoke(c *gin.Context){id:=c.Query("id");if id==""{c.JSON(400,gin.H{"code":"40000","msg":"缺少 id"});return};result:=models.DB.Model(&models.APIToken{}).Where("id = ?",id).Update("enabled",false);if result.Error!=nil{c.JSON(500,gin.H{"code":"50000","msg":"撤销失败"});return};if result.RowsAffected==0{c.JSON(404,gin.H{"code":"40400","msg":"Token 不存在"});return};c.JSON(200,gin.H{"code":"00000","msg":"API Token 已撤销"})}
