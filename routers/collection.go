package routers

import (
	"ppeelink/api"

	"github.com/gin-gonic/gin"
)

func Collections(r *gin.Engine) {
	r.GET("/cc/", api.GetCollectionClient)
	g := r.Group("/api/v1/collections")
	{
		g.GET("/list", api.CollectionList)
		g.POST("/add", api.CollectionAdd)
		g.POST("/update", api.CollectionUpdate)
		g.DELETE("/delete", api.CollectionDelete)
		g.POST("/reset-token", api.CollectionResetToken)
	}
	// 订阅到期/用量相关
	r.POST("/api/v1/subcription/check-alerts", api.SubscriptionCheckAlerts)
	r.GET("/api/v1/subcription/usage", api.SubscriptionUsage)
}
