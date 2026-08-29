package cmd

import "yunshu/internal/model"

func seedPermissionsInspect() []model.Permission {
	return []model.Permission{
		{Name: "巡检计划查询", Resource: "/api/v1/projects/:id/inspect/plan", Action: "GET", Description: "Get project inspect plan"},
		{Name: "巡检计划更新", Resource: "/api/v1/projects/:id/inspect/plan", Action: "PUT", Description: "Update project inspect plan"},
		{Name: "巡检项列表", Resource: "/api/v1/projects/:id/inspect/items", Action: "GET", Description: "List inspect items"},
		{Name: "巡检项创建", Resource: "/api/v1/projects/:id/inspect/items", Action: "POST", Description: "Create inspect item"},
		{Name: "巡检项更新", Resource: "/api/v1/projects/:id/inspect/items/:itemId", Action: "PUT", Description: "Update inspect item"},
		{Name: "巡检项删除", Resource: "/api/v1/projects/:id/inspect/items/:itemId", Action: "DELETE", Description: "Delete inspect item"},
		{Name: "巡检项同步模板", Resource: "/api/v1/projects/:id/inspect/items/sync-template", Action: "POST", Description: "Sync inspect items from global template"},
		{Name: "巡检项重置模板", Resource: "/api/v1/projects/:id/inspect/items/reset-template", Action: "POST", Description: "Reset inspect items from global template"},
		{Name: "巡检记录列表", Resource: "/api/v1/projects/:id/inspect/runs", Action: "GET", Description: "List inspect runs"},
		{Name: "巡检趋势", Resource: "/api/v1/projects/:id/inspect/runs/trends", Action: "GET", Description: "List inspect run trends"},
		{Name: "巡检报告迁移 MinIO", Resource: "/api/v1/projects/:id/inspect/migrate-reports-to-minio", Action: "POST", Description: "Migrate local inspect reports to MinIO"},
		{Name: "巡检立即执行", Resource: "/api/v1/projects/:id/inspect/runs", Action: "POST", Description: "Start inspect run"},
		{Name: "巡检记录详情", Resource: "/api/v1/projects/:id/inspect/runs/:runId", Action: "GET", Description: "Get inspect run"},
		{Name: "巡检报告HTML", Resource: "/api/v1/projects/:id/inspect/runs/:runId/report.html", Action: "GET", Description: "Download inspect HTML report"},
		{Name: "巡检报告PDF", Resource: "/api/v1/projects/:id/inspect/runs/:runId/report.pdf", Action: "GET", Description: "Download inspect PDF report"},
		{Name: "巡检报告PDF上传", Resource: "/api/v1/projects/:id/inspect/runs/:runId/report.pdf", Action: "POST", Description: "Upload inspect PDF from browser"},
		{Name: "巡检报告PDF检查", Resource: "/api/v1/projects/:id/inspect/runs/:runId/report.pdf/check", Action: "GET", Description: "Check inspect PDF exists"},
		{Name: "巡检报告Excel", Resource: "/api/v1/projects/:id/inspect/runs/:runId/report.xlsx", Action: "GET", Description: "Download inspect Excel report"},
		{Name: "巡检报告打印版", Resource: "/api/v1/projects/:id/inspect/runs/:runId/report.print.html", Action: "GET", Description: "Download inspect print HTML"},
		{Name: "巡检邮件重发", Resource: "/api/v1/projects/:id/inspect/runs/:runId/resend-email", Action: "POST", Description: "Resend inspect report email"},
		{Name: "巡检报告模板列表", Resource: "/api/v1/projects/:id/inspect/report-templates", Action: "GET", Description: "List inspect report templates"},
		{Name: "巡检报告模板创建", Resource: "/api/v1/projects/:id/inspect/report-templates", Action: "POST", Description: "Create inspect report template"},
		{Name: "巡检报告模板更新", Resource: "/api/v1/projects/:id/inspect/report-templates/:templateId", Action: "PUT", Description: "Update inspect report template"},
		{Name: "巡检报告模板删除", Resource: "/api/v1/projects/:id/inspect/report-templates/:templateId", Action: "DELETE", Description: "Delete inspect report template"},
		{Name: "巡检报告模板复制", Resource: "/api/v1/projects/:id/inspect/report-templates/copy", Action: "POST", Description: "Copy inspect report template"},
		{Name: "巡检报告模板预览", Resource: "/api/v1/projects/:id/inspect/report-templates/preview", Action: "POST", Description: "Preview inspect report template"},
	}
}

