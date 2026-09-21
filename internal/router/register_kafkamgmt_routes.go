package router

import "github.com/gin-gonic/gin"

func RegisterKafkamgmtRoutes(api *gin.RouterGroup, d KafkamgmtRouteDeps) {
	if d == nil || d.KafkamgmtHandler() == nil {
		return
	}
	h := d.KafkamgmtHandler()
	g := api.Group("/kafkamgmt")
	g.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())

	g.GET("/connections", h.ListConnections)
	g.POST("/connections", h.CreateConnection)
	g.POST("/connections/import-from-dict", h.ImportConnectionFromDict)
	g.POST("/connections/test", h.TestConnection)
	g.PUT("/connections/:id", h.UpdateConnection)
	g.DELETE("/connections/:id", h.DeleteConnection)
	g.POST("/connections/:id/ping", h.PingConnection)

	g.GET("/brokers", h.ListBrokers)
	g.GET("/topics", h.ListTopics)
	g.GET("/topics/detail", h.GetTopicDetail)
	g.GET("/topics/config", h.GetTopicConfig)
	g.POST("/topics", h.CreateTopic)
	g.DELETE("/topics", h.DeleteTopic)
	g.POST("/topics/partitions", h.CreatePartitions)
	g.POST("/topics/produce", h.Produce)
	g.POST("/topics/consume", h.Consume)

	g.GET("/groups", h.ListGroups)
	g.GET("/groups/members", h.GetGroupMembers)
	g.GET("/groups/lag", h.GetGroupLag)
	g.DELETE("/groups", h.DeleteGroup)
}
