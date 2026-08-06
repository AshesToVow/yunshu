package cicd

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/plugin"
	cicdsvc "yunshu/internal/service/cicd"
)

func init() {
	plugin.Register(&module{})
}

type module struct {
	plugin.Base
}

func (m *module) Name() string        { return "cicd" }
func (m *module) Description() string { return "CI/CD 持续集成与交付：Jenkins 打包、MinIO/SSH 发布、执行记录" }

func (m *module) Manifest() plugin.Manifest {
	return plugin.Manifest{
		MenuPathPrefixes: []string{"/cicd"},
		APIPrefixes: []string{
			"/api/v1/overview/project-launches",
			"/api/v1/overview/release-by-person",
			"/api/v1/cicd/jenkins/callback",
			"/api/v1/registries",
			"/api/v1/pipeline-templates",
		},
		DependsOn: []string{"project"},
		Workers:   []string{"cicd_jenkins_sync", "cicd_image_cleanup"},
	}
}

func (m *module) Models() []any {
	return []any{
		&model.CicdService{},
		&model.CicdCiConfig{},
		&model.CicdDeployConfig{},
		&model.CicdBuildRun{},
		&model.CicdReleaseRun{},
		&model.CicdApprovalFlowStage{},
		&model.CicdReleaseApprovalStep{},
		&model.CicdAccessGrant{},
		&model.CicdRunStage{},
		&model.CicdArtifact{},
		&model.ImageRegistry{},
		&model.ProjectRegistryBinding{},
		&model.ImageCleanupPolicy{},
		&model.CicdPipelineTemplate{},
	}
}

func (m *module) StartWorkers(bgCtx context.Context, rt *plugin.Runtime) error {
	if bgCtx == nil || rt == nil {
		return nil
	}
	if svc, ok := rt.Cicd.(*cicdsvc.Service); ok && svc != nil {
		go svc.RunSyncWorker(bgCtx)
		go svc.RunImageCleanupWorker(bgCtx)
	}
	return nil
}
