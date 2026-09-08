package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterInspectRoutes(api *gin.RouterGroup, d InspectRouteDeps) {
	h := d.InspectHandler()
	api.GET("/inspect/pdf-libs/:name", d.AuthMiddleware(), d.Authorize(), h.ServePDFLib)

	projectRoutes := api.Group("/projects")
	projectRoutes.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	projectScoped := projectRoutes.Group("/:id", middleware.RequireProjectMemberAccess(d.ProjectMemberRepo(), d.ProjectRepo(), d.AppLogger()))

	inspect := projectScoped.Group("/inspect")
	inspect.GET("/plan", h.GetPlan)
	inspect.GET("/storage-info", h.GetStorageInfo)
	inspect.PUT("/plan", h.UpdatePlan)
	inspect.GET("/items", h.ListItems)
	inspect.POST("/items", h.CreateItem)
	inspect.PUT("/items/:itemId", h.UpdateItem)
	inspect.DELETE("/items/:itemId", h.DeleteItem)
	inspect.POST("/items/sync-template", h.SyncItems)
	inspect.POST("/items/reset-template", h.ResetItems)
	inspect.POST("/items/:itemId/promote-alert", h.PromoteItemAlert)
	inspect.GET("/runs", h.ListRuns)
	inspect.GET("/runs/trends", h.ListRunTrends)
	inspect.POST("/runs", h.CreateRun)
	inspect.GET("/runs/:runId", h.GetRun)
	inspect.GET("/runs/:runId/report.pdf/check", h.CheckReportPDF)
	inspect.GET("/runs/:runId/report.html", h.ReportHTML)
	inspect.GET("/runs/:runId/report.pdf", h.ReportPDF)
	inspect.POST("/runs/:runId/report.pdf", h.SaveReportPDF)
	inspect.GET("/runs/:runId/report.xlsx", h.ReportExcel)
	inspect.GET("/runs/:runId/report.print.html", h.ReportPrint)
	inspect.POST("/runs/:runId/resend-email", h.ResendEmail)
	inspect.POST("/migrate-reports-to-minio", h.MigrateReportsToMinIO)

	inspect.GET("/report-templates", h.ListReportTemplates)
	inspect.POST("/report-templates", h.CreateReportTemplate)
	inspect.PUT("/report-templates/:templateId", h.UpdateReportTemplate)
	inspect.DELETE("/report-templates/:templateId", h.DeleteReportTemplate)
	inspect.POST("/report-templates/copy", h.CopyReportTemplate)
	inspect.POST("/report-templates/preview", h.PreviewReportTemplate)
}
