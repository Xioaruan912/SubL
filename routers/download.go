package routers

import (
	"ppeelink/api"

	"github.com/gin-gonic/gin"
)

func Downloads(r *gin.Engine) {
	g := r.Group("/api/v1/clients")
	{
		g.GET("/list", api.ClientList)
		g.POST("/check", api.ClientCheck)
		g.GET("/download", api.ClientDownload)
	}
}