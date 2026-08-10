package router

import "github.com/gin-gonic/gin"

func RegisterAIRoutes(api *gin.RouterGroup, d *RouteDeps) {
	if d == nil || d.aiHandler == nil {
		return
	}
	g := api.Group("/ai")
	g.Use(d.authMiddleware, d.authorize, d.opAudit)
	g.GET("/status", d.aiHandler.Status)
	g.POST("/ping", d.aiHandler.Ping)
	g.POST("/chat", d.aiHandler.Chat)
	g.POST("/k8s/pod-diagnose", d.aiHandler.PodDiagnose)
	g.POST("/cicd/build-fail", d.aiHandler.CicdBuildFail)
	g.POST("/alert/explain", d.aiHandler.AlertExplain)
}
