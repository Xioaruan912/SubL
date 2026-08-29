package routers

import (
	"ppeelink/api"

	"github.com/gin-gonic/gin"
)

func Rules(r *gin.Engine) {
	g := r.Group("/api/v1/rules")
	{
		g.GET("/sources", api.RuleSources)
		g.GET("/catalog", api.RuleCatalog)
		g.GET("/preview", api.RulePreview)
		g.GET("/template-groups", api.RuleTemplateGroups)
		g.POST("/sync", api.RuleSync)
		g.POST("/import", api.RuleImport)
	}
}
