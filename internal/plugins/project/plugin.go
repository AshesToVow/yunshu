package project

import (
	"context"

	"yunshu/internal/config"
	"yunshu/internal/model"
	"yunshu/internal/plugin"
	"yunshu/internal/service"

	"gorm.io/gorm"
)

func init() {
	plugin.Register(&module{})
}

type module struct {
	plugin.Base
}

func (m *module) Name() string        { return "project" }
func (m *module) Description() string { return "多租户项目、成员、服务配置与 ES 日志检索" }

func (m *module) Models() []any {
	return []any{
		&model.Project{},
		&model.ProjectMember{},
		&model.Service{},
		&model.ServiceLogSource{},
		&model.LogRetentionPolicy{},
		&model.LoggieAgent{},
	}
}

func (m *module) PostMigrate(db *gorm.DB) error {
	return migrateProjectsDefaultMeta(db)
}

func (m *module) StartWorkers(bgCtx context.Context, rt *plugin.Runtime) error {
	if bgCtx == nil || rt == nil {
		return nil
	}
	if svc := rt.LogRetentionSvc(); svc != nil && rt.Config != nil {
		go service.RunLogRetentionScheduler(bgCtx, svc, config.ElasticsearchConfig{})
	}
	return nil
}
