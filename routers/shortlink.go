package routers

import (
	"ppeelink/api"

	"github.com/gin-gonic/gin"
)

func ShortLinks(r *gin.Engine) {
	r.GET("/s/:code", api.ShortLinkRedirect)
	group := r.Group("/api/v1/shortlinks")
	{
		group.POST("/create", api.ShortLinkCreate)
		group.GET("/list", api.ShortLinkList)
		group.DELETE("/delete", api.ShortLinkDelete)
	}
	r.GET("/api/v1/subcription/import-links", api.SubscriptionImportLinks)
}
