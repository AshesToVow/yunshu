package core

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

func (m *module) Name() string        { return "core" }
func (m *module) Description() string { return "平台内核：认证、RBAC、菜单、字典、审计" }

func (m *module) Models() []any {
	return []any{
		&model.Department{},
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.UserRole{},
		&model.RegistrationRequest{},
		&model.Menu{},
		&model.LoginLog{},
		&model.OperationLog{},
		&model.DictEntry{},
		&model.UserGroup{},
		&model.UserGroupUser{},
	}
}

func (m *module) PreMigrate(db *gorm.DB) error {
	return bootstrapPreMigrate(db)
}

func (m *module) PostMigrate(db *gorm.DB) error {
	return bootstrapPostMigrateCore(db)
}
