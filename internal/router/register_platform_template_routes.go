package router

import (
	"github.com/gin-gonic/gin"
)

func registerPlatformTemplateRoutes(api *gin.RouterGroup, d PlatformTemplateRouteDeps) {
	if d == nil || d.PlatformTemplateHandler() == nil {
		return
	}
	tpl := api.Group("/platform-templates")
	tpl.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	tpl.GET("", d.PlatformTemplateHandler().List)
	tpl.POST("", d.PlatformTemplateHandler().Create)
	tpl.GET("/resolve/:template_key", d.PlatformTemplateHandler().Resolve)
	tpl.GET("/:id", d.PlatformTemplateHandler().Detail)
	tpl.PUT("/:id", d.PlatformTemplateHandler().Update)
	tpl.DELETE("/:id", d.PlatformTemplateHandler().Delete)
	tpl.GET("/:id/versions", d.PlatformTemplateHandler().ListVersions)
	tpl.GET("/:id/versions/:version", d.PlatformTemplateHandler().GetVersion)
	tpl.POST("/:id/drafts", d.PlatformTemplateHandler().SaveDraft)
	tpl.POST("/:id/publish", d.PlatformTemplateHandler().Publish)
}
