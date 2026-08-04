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

func (m *module) Manifest() plugin.Manifest {
	return plugin.Manifest{
		MenuPathPrefixes: []string{
			"/projects", "/project-members", "/project-services", "/service-catalog",
			"/service-portrait", "/project-logs", "/project-log-sources", "/log-retention", "/loggie-status",
		},
		APIPrefixes: []string{"/api/v1/projects"},
		Workers:     []string{"log_retention", "kafka_to_es"},
	}
}

func (m *module) Models() []any {
	return []any{
		&model.Project{},
		&model.ProjectMember{},
		&model.Service{},
		&model.ServiceLogSource{},
		&model.LogRetentionPolicy{},
		&model.LoggieAgent{},
		&model.ServiceCatalog{},
		&model.ServiceLink{},
		&model.ChangeEvent{},
	}
}

func (m *module) PostMigrate(db *gorm.DB) error {
	return migrateProjectsDefaultMeta(db)
}

func (m *module) StartWorkers(bgCtx context.Context, rt *plugin.Runtime) error {
	if bgCtx == nil || rt == nil {
		return nil
	}
	if svc, ok := rt.LogRetention.(*service.LogRetentionService); ok && svc != nil && rt.Config != nil {
		go service.RunLogRetentionScheduler(bgCtx, svc, config.ElasticsearchConfig{})
	}
	if kafkaSvc, ok := rt.KafkaToES.(*service.KafkaToESService); ok && kafkaSvc != nil {
		go kafkaSvc.Run(bgCtx)
	}
	return nil
}
