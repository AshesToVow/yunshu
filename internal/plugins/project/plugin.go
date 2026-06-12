package project

import (
	"yunshu/internal/model"
	"yunshu/internal/plugin"

	"gorm.io/gorm"
)

func init() {
	plugin.Register(&module{})
}

type module struct {
	plugin.Base
}

func (m *module) Name() string        { return "project" }
func (m *module) Description() string { return "多租户项目、成员、服务配置、日志 Agent 与日志流" }

func (m *module) Models() []any {
	return []any{
		&model.Project{},
		&model.ProjectMember{},
		&model.Service{},
		&model.ServiceLogSource{},
		&model.LogAgent{},
		&model.AgentDiscovery{},
	}
}

func (m *module) PostMigrate(db *gorm.DB) error {
	if err := migrateLogAgentsClearPlaceholderListenPort(db); err != nil {
		return err
	}
	if err := migrateAgentDiscoveryUniqueIndex(db); err != nil {
		return err
	}
	return migrateProjectsDefaultMeta(db)
}
