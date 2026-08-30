package routers

import (
	"github.com/gin-gonic/gin"
	"ppeelink/api"
)

func Status(r *gin.Engine){r.GET("/status",api.PublicStatusPage);r.GET("/api/v1/status/public",api.PublicStatus)}
