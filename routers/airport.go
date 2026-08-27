package routers

import (
	"ppeelink/api"

	"github.com/gin-gonic/gin"
)

func AirportRoutes(r *gin.Engine) {
	airport := r.Group("/api/v1/airport")
	{
		airport.GET("/list", api.AirportList)
		airport.GET("/detail", api.AirportDetail)
		airport.POST("/add", api.AirportAdd)
		airport.POST("/update", api.AirportUpdate)
		airport.DELETE("/delete", api.AirportDelete)
		airport.POST("/sync", api.AirportSync)
	}
}
