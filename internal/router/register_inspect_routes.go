package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterInspectRoutes(api *gin.RouterGroup, d *RouteDeps) {
	projectRoutes := api.Group("/projects")
	projectRoutes.Use(d.authMiddleware, d.authorize, d.opAudit)
	projectScoped := projectRoutes.Group("/:id", middleware.RequireProjectMemberAccess(d.projectMemberRepo, d.app.Logger))

	inspect := projectScoped.Group("/inspect")
	inspect.GET("/plan", d.inspectHandler.GetPlan)
	inspect.PUT("/plan", d.inspectHandler.UpdatePlan)
	inspect.GET("/items", d.inspectHandler.ListItems)
	inspect.POST("/items", d.inspectHandler.CreateItem)
	inspect.PUT("/items/:itemId", d.inspectHandler.UpdateItem)
	inspect.DELETE("/items/:itemId", d.inspectHandler.DeleteItem)
	inspect.POST("/items/sync-template", d.inspectHandler.SyncItems)
	inspect.POST("/items/reset-template", d.inspectHandler.ResetItems)
	inspect.GET("/runs", d.inspectHandler.ListRuns)
	inspect.POST("/runs", d.inspectHandler.CreateRun)
	inspect.GET("/runs/:runId", d.inspectHandler.GetRun)
	inspect.GET("/runs/:runId/report.html", d.inspectHandler.ReportHTML)
	inspect.GET("/runs/:runId/report.pdf", d.inspectHandler.ReportPDF)
	inspect.GET("/runs/:runId/report.xlsx", d.inspectHandler.ReportExcel)
	inspect.GET("/runs/:runId/report.print.html", d.inspectHandler.ReportPrint)
	inspect.POST("/runs/:runId/resend-email", d.inspectHandler.ResendEmail)

	inspect.GET("/report-templates", d.inspectHandler.ListReportTemplates)
	inspect.POST("/report-templates", d.inspectHandler.CreateReportTemplate)
	inspect.PUT("/report-templates/:templateId", d.inspectHandler.UpdateReportTemplate)
	inspect.DELETE("/report-templates/:templateId", d.inspectHandler.DeleteReportTemplate)
	inspect.POST("/report-templates/copy", d.inspectHandler.CopyReportTemplate)
	inspect.POST("/report-templates/preview", d.inspectHandler.PreviewReportTemplate)
}
