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
		g.GET("/update-impact", api.RuleUpdateImpact)
		g.POST("/apply-update", api.RuleApplyUpdate)
		g.GET("/snapshots", api.RuleSnapshots)
		g.POST("/rollback", api.RuleRollback)
		g.GET("/template-groups", api.RuleTemplateGroups)
		g.POST("/sync", api.RuleSync)
		g.POST("/import", api.RuleImport)
	}
}
