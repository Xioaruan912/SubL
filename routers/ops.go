package routers

import (
	"github.com/gin-gonic/gin"
	"ppeelink/api"
)

func Ops(r *gin.Engine){g:=r.Group("/api/v1/ops");g.GET("/backup/export",api.ConfigBackupExport);g.GET("/backup/inspect",api.ConfigBackupInspect);g.POST("/backup/import",api.ConfigBackupImport)}
