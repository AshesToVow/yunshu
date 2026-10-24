package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterCicdRoutes CI/CD：应用服务、CI 配置、发布配置、打包/发布记录（项目作用域）。
func RegisterCicdRoutes(api *gin.RouterGroup, d CicdRouteDeps) {
	h := d.CicdHandler()
	// Jenkins HMAC 回调（无登录；鉴权靠 cicd_jenkins_callback_hmac_secret）
	cicdPublic := api.Group("/cicd")
	cicdPublic.POST("/jenkins/callback", h.JenkinsCallback)

	// 镜像仓库注册中心 / 浏览 / 清理（平台级，需登录）
	regs := api.Group("/registries")
	regs.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	regs.GET("", h.ListRegistries)
	regs.POST("", h.CreateRegistry)
	regs.GET("/browse/projects", h.ListHarborProjects)
	regs.GET("/browse/repositories", h.ListHarborRepositories)
	regs.GET("/browse/artifacts", h.ListHarborArtifacts)
	regs.POST("/browse/artifacts/delete", h.DeleteHarborArtifact)
	regs.GET("/cleanup-policies", h.ListCleanupPolicies)
	regs.POST("/cleanup-policies", h.CreateCleanupPolicy)
	regs.PUT("/cleanup-policies/:policyId", h.UpdateCleanupPolicy)
	regs.DELETE("/cleanup-policies/:policyId", h.DeleteCleanupPolicy)
	regs.POST("/cleanup-policies/:policyId/run", h.RunCleanupPolicy)
	regs.GET("/:registryId", h.GetRegistry)
	regs.PUT("/:registryId", h.UpdateRegistry)
	regs.DELETE("/:registryId", h.DeleteRegistry)
	regs.POST("/:registryId/ping", h.PingRegistry)

	tpl := api.Group("/pipeline-templates")
	tpl.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	tpl.GET("", h.ListPipelineTemplates)
	tpl.POST("", h.CreatePipelineTemplate)
	tpl.PUT("/:templateId", h.UpdatePipelineTemplate)

	projectRoutes := api.Group("/projects")
	projectRoutes.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	projectScoped := projectRoutes.Group("/:id", middleware.RequireProjectMemberAccess(d.ProjectMemberRepo(), d.ProjectRepo(), d.AppLogger()))

	projectScoped.GET("/registry-binding", h.GetProjectRegistryBinding)
	projectScoped.PUT("/registry-binding", h.UpsertProjectRegistryBinding)
	projectScoped.DELETE("/registry-binding", h.DeleteProjectRegistryBinding)

	cicdGroup := projectScoped.Group("/cicd")
	cicdGroup.GET("/services", h.ListServices)
	cicdGroup.POST("/services", h.CreateService)
	cicdGroup.GET("/services/:serviceId", h.GetService)
	cicdGroup.PUT("/services/:serviceId", h.UpdateService)
	cicdGroup.DELETE("/services/:serviceId", h.DeleteService)

	cicdGroup.GET("/services/:serviceId/ci-config", h.GetCiConfig)
	cicdGroup.PUT("/services/:serviceId/ci-config", h.UpsertCiConfig)

	cicdGroup.GET("/services/:serviceId/deploy-configs", h.ListDeployConfigs)
	cicdGroup.POST("/services/:serviceId/deploy-configs", h.CreateDeployConfig)
	cicdGroup.PUT("/services/:serviceId/deploy-configs/:configId", h.UpdateDeployConfig)
	cicdGroup.DELETE("/services/:serviceId/deploy-configs/:configId", h.DeleteDeployConfig)

	cicdGroup.GET("/services/:serviceId/artifacts", h.ListArtifacts)
	cicdGroup.GET("/services/:serviceId/helm-scaffold", h.DownloadHelmScaffold)
	cicdGroup.GET("/helm-scaffold", h.DownloadHelmScaffoldPreview)

	cicdGroup.POST("/services/:serviceId/builds", h.TriggerBuild)
	cicdGroup.POST("/services/:serviceId/releases", h.TriggerRelease)

	cicdGroup.GET("/build-runs", h.ListBuildRuns)
	cicdGroup.GET("/build-runs/:runId", h.GetBuildRun)
	cicdGroup.GET("/build-runs/:runId/log", h.GetBuildRunLog)
	cicdGroup.GET("/build-runs/:runId/stages", h.ListBuildRunStages)
	cicdGroup.GET("/build-runs/:runId/artifacts-meta", h.ListBuildRunArtifactsMeta)
	cicdGroup.DELETE("/build-runs/:runId", h.DeleteBuildRun)

	cicdGroup.GET("/approval-flow", h.GetApprovalFlow)
	cicdGroup.PUT("/approval-flow", h.UpsertApprovalFlow)

	cicdGroup.GET("/release-runs", h.ListReleaseRuns)
	cicdGroup.GET("/release-runs/:runId", h.GetReleaseRun)
	cicdGroup.GET("/release-runs/:runId/approval-steps", h.ListReleaseApprovalSteps)
	cicdGroup.POST("/release-runs/:runId/approve", h.ApproveReleaseRun)
	cicdGroup.POST("/release-runs/:runId/reject", h.RejectReleaseRun)
	cicdGroup.POST("/release-runs/:runId/execute", h.ExecuteReleaseRun)
	cicdGroup.POST("/release-runs/:runId/terminate", h.TerminateReleaseRun)
	cicdGroup.POST("/release-runs/batch-approve", h.BatchApproveReleaseRuns)
	cicdGroup.POST("/release-runs/batch-reject", h.BatchRejectReleaseRuns)
	cicdGroup.POST("/release-runs/batch-execute", h.BatchExecuteReleaseRuns)
	cicdGroup.POST("/release-runs/batch-terminate", h.BatchTerminateReleaseRuns)
	cicdGroup.GET("/release-runs/:runId/log", h.GetReleaseRunLog)
	cicdGroup.POST("/release-runs/:runId/verify", h.VerifyReleaseRun)
	cicdGroup.POST("/release-runs/:runId/platform-rollback", h.PlatformRollbackRelease)
	cicdGroup.POST("/release-runs/:runId/progressive/promote", h.PromoteProgressiveRelease)
	cicdGroup.POST("/release-runs/:runId/progressive/abort", h.AbortProgressiveRelease)
	cicdGroup.DELETE("/release-runs/:runId", h.DeleteReleaseRun)

	projectScoped.GET("/cicd-access-grants", h.ListCicdGrants)
	projectScoped.POST("/cicd-access-grants", h.UpsertCicdGrant)
	projectScoped.POST("/cicd-access-grants/bulk", h.BulkUpsertCicdGrants)
	projectScoped.POST("/cicd-access-grants/bootstrap", h.BootstrapCicdGrants)
	projectScoped.DELETE("/cicd-access-grants/:grantId", h.DeleteCicdGrant)
}
