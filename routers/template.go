package routers

import (
	"ppeelink/api"

	"github.com/gin-gonic/gin"
)

func Templates(r *gin.Engine) {
	TempsGroup := r.Group("/api/v1/template")
	{
		TempsGroup.POST("/add", api.AddTemp)
		TempsGroup.POST("/delete", api.DelTemp)
		TempsGroup.GET("/get", api.GetTempS)
		TempsGroup.POST("/update", api.UpdateTemp)
		TempsGroup.POST("/build", api.TemplateBuild)
		TempsGroup.GET("/default", api.TemplateGetDefault)
	}

}
