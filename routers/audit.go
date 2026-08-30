package routers

import (
	"github.com/gin-gonic/gin"
	"ppeelink/api"
)

func Audit(r *gin.Engine) { r.GET("/api/v1/audit/list", api.AuditLogList) }
