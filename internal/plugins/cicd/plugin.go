package cicd

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/plugin"
)

func init() {
	plugin.Register(&module{})
}

type module struct {
	plugin.Base
}

func (m *module) Name() string        { return "cicd" }
func (m *module) Description() string { return "CI/CD 持续集成与交付：Jenkins 打包、MinIO/SSH 发布、执行记录" }

func (m *module) Models() []any {
	return []any{
		&model.CicdService{},
		&model.CicdCiConfig{},
		&model.CicdDeployConfig{},
		&model.CicdBuildRun{},
		&model.CicdReleaseRun{},
		&model.CicdApprovalFlowStage{},
		&model.CicdReleaseApprovalStep{},
	}
}

func (m *module) StartWorkers(bgCtx context.Context, rt *plugin.Runtime) error {
	if bgCtx == nil || rt == nil {
		return nil
	}
	if svc := rt.CicdSvc(); svc != nil {
		go svc.RunSyncWorker(bgCtx)
	}
	return nil
}
