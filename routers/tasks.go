package routers

import (
	"github.com/gin-gonic/gin"
	"ppeelink/api"
)

func Tasks(r *gin.Engine) {
	g := r.Group("/api/v1/tasks")
	g.GET("/list", api.TaskList)
	g.POST("/cancel", api.TaskCancel)
	g.POST("/retry", api.TaskRetry)
}
