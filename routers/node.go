package routers

import (
	"ppeelink/api"

	"github.com/gin-gonic/gin"
)

func Nodes(r *gin.Engine) {
	NodesGroup := r.Group("/api/v1/nodes")
	{
		NodesGroup.POST("/add", api.NodeAdd)
		NodesGroup.DELETE("/delete", api.NodeDel)
		NodesGroup.GET("/get", api.NodeGet)
		NodesGroup.GET("/overview", api.NodeOverview)
		NodesGroup.GET("/quality/history", api.NodeQualityHistory)
		NodesGroup.GET("/quality/summary", api.NodeQualitySummary)
		NodesGroup.GET("/quality/matrix", api.NodeQualityMatrix)
		NodesGroup.GET("/health/events", api.NodeHealthEvents)
		NodesGroup.GET("/recommendations", api.NodeRecommendations)
		NodesGroup.GET("/alerts", api.GetAlertSetting)
		NodesGroup.POST("/alerts", api.UpdateAlertSetting)
		NodesGroup.POST("/update", api.NodeUpdadte)
		NodesGroup.GET("/map", api.NodeMap)
		NodesGroup.GET("/ping", api.NodePing)
		NodesGroup.POST("/unlock", api.NodeUnlock)
		NodesGroup.POST("/egress", api.NodeEgress)
		NodesGroup.GET("/egress-targets", api.EgressTargetList)
		NodesGroup.POST("/egress-targets", api.EgressTargetSave)
		NodesGroup.DELETE("/egress-targets", api.EgressTargetDelete)
		NodesGroup.POST("/chinaping", api.NodeChinaPing)
		NodesGroup.POST("/chinaping/stream", api.NodeChinaPingStream)
		NodesGroup.GET("/test/status", api.TestStatus)
		NodesGroup.POST("/test/cancel", api.TestCancel)

	}
	// 分组
	Group := NodesGroup.Group("/group")
	{
		Group.GET("/get", api.GroupNodeGet)        // 添加分组
		Group.GET("/full", api.GroupNodeGetFull)   // 分组完整信息（含 ID/节点数）
		Group.POST("/set", api.GroupNodeSet)       // 绑定创建分组
		Group.POST("/unbind", api.GroupNodeUnbind) // 解除节点与分组绑定
		Group.DELETE("/delete", api.GroupNodeDel)  // 删除分组
		// Group.POST("/update", api.GroupNodeUpdate) // 更新分组
	}
}
