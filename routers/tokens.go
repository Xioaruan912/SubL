package routers

import (
	"github.com/gin-gonic/gin"
	"ppeelink/api"
)

func Tokens(r *gin.Engine){g:=r.Group("/api/v1/tokens");g.GET("/list",api.APITokenList);g.POST("/create",api.APITokenCreate);g.POST("/revoke",api.APITokenRevoke)}
