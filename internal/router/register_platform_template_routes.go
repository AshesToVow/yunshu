package router

import (
	"github.com/gin-gonic/gin"
)

func registerPlatformTemplateRoutes(api *gin.RouterGroup, d *RouteDeps) {
	if d == nil || d.platformTplHandler == nil {
		return
	}
	tpl := api.Group("/platform-templates")
	tpl.Use(d.authMiddleware, d.authorize, d.opAudit)
	tpl.GET("", d.platformTplHandler.List)
	tpl.POST("", d.platformTplHandler.Create)
	tpl.GET("/resolve/:template_key", d.platformTplHandler.Resolve)
	tpl.GET("/:id", d.platformTplHandler.Detail)
	tpl.PUT("/:id", d.platformTplHandler.Update)
	tpl.DELETE("/:id", d.platformTplHandler.Delete)
	tpl.GET("/:id/versions", d.platformTplHandler.ListVersions)
	tpl.GET("/:id/versions/:version", d.platformTplHandler.GetVersion)
	tpl.POST("/:id/drafts", d.platformTplHandler.SaveDraft)
	tpl.POST("/:id/publish", d.platformTplHandler.Publish)
}
