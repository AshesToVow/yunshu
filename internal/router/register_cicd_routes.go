package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterCicdRoutes CI/CD：应用服务、CI 配置、发布配置、打包/发布记录（项目作用域）。
func RegisterCicdRoutes(api *gin.RouterGroup, d *RouteDeps) {
	// Jenkins HMAC 回调（无登录；鉴权靠 cicd_jenkins_callback_hmac_secret）
	cicdPublic := api.Group("/cicd")
	cicdPublic.POST("/jenkins/callback", d.cicdHandler.JenkinsCallback)

	// 镜像仓库注册中心 / 浏览 / 清理（平台级，需登录）
	regs := api.Group("/registries")
	regs.Use(d.authMiddleware, d.authorize, d.opAudit)
	regs.GET("", d.cicdHandler.ListRegistries)
	regs.POST("", d.cicdHandler.CreateRegistry)
	regs.GET("/browse/projects", d.cicdHandler.ListHarborProjects)
	regs.GET("/browse/repositories", d.cicdHandler.ListHarborRepositories)
	regs.GET("/browse/artifacts", d.cicdHandler.ListHarborArtifacts)
	regs.POST("/browse/artifacts/delete", d.cicdHandler.DeleteHarborArtifact)
	regs.GET("/cleanup-policies", d.cicdHandler.ListCleanupPolicies)
	regs.POST("/cleanup-policies", d.cicdHandler.CreateCleanupPolicy)
	regs.PUT("/cleanup-policies/:policyId", d.cicdHandler.UpdateCleanupPolicy)
	regs.DELETE("/cleanup-policies/:policyId", d.cicdHandler.DeleteCleanupPolicy)
	regs.POST("/cleanup-policies/:policyId/run", d.cicdHandler.RunCleanupPolicy)
	regs.GET("/:registryId", d.cicdHandler.GetRegistry)
	regs.PUT("/:registryId", d.cicdHandler.UpdateRegistry)
	regs.DELETE("/:registryId", d.cicdHandler.DeleteRegistry)
	regs.POST("/:registryId/ping", d.cicdHandler.PingRegistry)

	tpl := api.Group("/pipeline-templates")
	tpl.Use(d.authMiddleware, d.authorize, d.opAudit)
	tpl.GET("", d.cicdHandler.ListPipelineTemplates)
	tpl.POST("", d.cicdHandler.CreatePipelineTemplate)
	tpl.PUT("/:templateId", d.cicdHandler.UpdatePipelineTemplate)

	projectRoutes := api.Group("/projects")
	projectRoutes.Use(d.authMiddleware, d.authorize, d.opAudit)
	projectScoped := projectRoutes.Group("/:id", middleware.RequireProjectMemberAccess(d.projectMemberRepo, d.projectRepo, d.app.Logger))

	projectScoped.GET("/registry-binding", d.cicdHandler.GetProjectRegistryBinding)
	projectScoped.PUT("/registry-binding", d.cicdHandler.UpsertProjectRegistryBinding)
	projectScoped.DELETE("/registry-binding", d.cicdHandler.DeleteProjectRegistryBinding)

	cicdGroup := projectScoped.Group("/cicd")
	cicdGroup.GET("/services", d.cicdHandler.ListServices)
	cicdGroup.POST("/services", d.cicdHandler.CreateService)
	cicdGroup.GET("/services/:serviceId", d.cicdHandler.GetService)
	cicdGroup.PUT("/services/:serviceId", d.cicdHandler.UpdateService)
	cicdGroup.DELETE("/services/:serviceId", d.cicdHandler.DeleteService)

	cicdGroup.GET("/services/:serviceId/ci-config", d.cicdHandler.GetCiConfig)
	cicdGroup.PUT("/services/:serviceId/ci-config", d.cicdHandler.UpsertCiConfig)

	cicdGroup.GET("/services/:serviceId/deploy-configs", d.cicdHandler.ListDeployConfigs)
	cicdGroup.POST("/services/:serviceId/deploy-configs", d.cicdHandler.CreateDeployConfig)
	cicdGroup.PUT("/services/:serviceId/deploy-configs/:configId", d.cicdHandler.UpdateDeployConfig)
	cicdGroup.DELETE("/services/:serviceId/deploy-configs/:configId", d.cicdHandler.DeleteDeployConfig)

	cicdGroup.GET("/services/:serviceId/artifacts", d.cicdHandler.ListArtifacts)
	cicdGroup.GET("/services/:serviceId/helm-scaffold", d.cicdHandler.DownloadHelmScaffold)
	cicdGroup.GET("/helm-scaffold", d.cicdHandler.DownloadHelmScaffoldPreview)

	cicdGroup.POST("/services/:serviceId/builds", d.cicdHandler.TriggerBuild)
	cicdGroup.POST("/services/:serviceId/releases", d.cicdHandler.TriggerRelease)

	cicdGroup.GET("/build-runs", d.cicdHandler.ListBuildRuns)
	cicdGroup.GET("/build-runs/:runId", d.cicdHandler.GetBuildRun)
	cicdGroup.GET("/build-runs/:runId/log", d.cicdHandler.GetBuildRunLog)
	cicdGroup.GET("/build-runs/:runId/stages", d.cicdHandler.ListBuildRunStages)
	cicdGroup.GET("/build-runs/:runId/artifacts-meta", d.cicdHandler.ListBuildRunArtifactsMeta)
	cicdGroup.DELETE("/build-runs/:runId", d.cicdHandler.DeleteBuildRun)

	cicdGroup.GET("/approval-flow", d.cicdHandler.GetApprovalFlow)
	cicdGroup.PUT("/approval-flow", d.cicdHandler.UpsertApprovalFlow)

	cicdGroup.GET("/release-runs", d.cicdHandler.ListReleaseRuns)
	cicdGroup.GET("/release-runs/:runId", d.cicdHandler.GetReleaseRun)
	cicdGroup.GET("/release-runs/:runId/approval-steps", d.cicdHandler.ListReleaseApprovalSteps)
	cicdGroup.POST("/release-runs/:runId/approve", d.cicdHandler.ApproveReleaseRun)
	cicdGroup.POST("/release-runs/:runId/reject", d.cicdHandler.RejectReleaseRun)
	cicdGroup.POST("/release-runs/:runId/execute", d.cicdHandler.ExecuteReleaseRun)
	cicdGroup.POST("/release-runs/:runId/terminate", d.cicdHandler.TerminateReleaseRun)
	cicdGroup.POST("/release-runs/batch-approve", d.cicdHandler.BatchApproveReleaseRuns)
	cicdGroup.POST("/release-runs/batch-reject", d.cicdHandler.BatchRejectReleaseRuns)
	cicdGroup.POST("/release-runs/batch-execute", d.cicdHandler.BatchExecuteReleaseRuns)
	cicdGroup.POST("/release-runs/batch-terminate", d.cicdHandler.BatchTerminateReleaseRuns)
	cicdGroup.GET("/release-runs/:runId/log", d.cicdHandler.GetReleaseRunLog)
	cicdGroup.POST("/release-runs/:runId/verify", d.cicdHandler.VerifyReleaseRun)
	cicdGroup.POST("/release-runs/:runId/platform-rollback", d.cicdHandler.PlatformRollbackRelease)
	cicdGroup.DELETE("/release-runs/:runId", d.cicdHandler.DeleteReleaseRun)

	projectScoped.GET("/cicd-access-grants", d.cicdHandler.ListCicdGrants)
	projectScoped.POST("/cicd-access-grants", d.cicdHandler.UpsertCicdGrant)
	projectScoped.POST("/cicd-access-grants/bulk", d.cicdHandler.BulkUpsertCicdGrants)
	projectScoped.POST("/cicd-access-grants/bootstrap", d.cicdHandler.BootstrapCicdGrants)
	projectScoped.DELETE("/cicd-access-grants/:grantId", d.cicdHandler.DeleteCicdGrant)
}
