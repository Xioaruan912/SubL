package routers

import (
	"github.com/gin-gonic/gin"
	"ppeelink/api"
)

func Regression(r *gin.Engine) {
	g := r.Group("/api/v1/regressions")
	{
		g.GET("/list", api.RoutingRegressionList)
		g.POST("/save", api.RoutingRegressionSave)
		g.DELETE("/delete", api.RoutingRegressionDelete)
		g.POST("/evaluate", api.RoutingRegressionEvaluate)
		g.POST("/compare", api.RoutingRegressionCompare)
	}
}
