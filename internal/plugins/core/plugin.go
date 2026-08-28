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

func (m *module) Manifest() plugin.Manifest {
	return plugin.Manifest{
		MenuPathPrefixes: []string{
			"/users", "/departments", "/roles", "/permissions", "/policies", "/registrations",
			"/menus", "/login-logs", "/operation-logs", "/banned-ips", "/dict-entries",
			"/personal-settings", "/user-groups", "/plugins",
		},
		APIPrefixes: []string{
			"/api/v1/users", "/api/v1/departments", "/api/v1/roles", "/api/v1/permissions",
			"/api/v1/policies", "/api/v1/registrations", "/api/v1/menus", "/api/v1/login-logs",
			"/api/v1/operation-logs", "/api/v1/security", "/api/v1/dict/entries", "/api/v1/dict-entries",
			"/api/v1/user-groups", "/api/v1/plugins", "/api/v1/auth/logout", "/api/v1/auth/me",
			"/api/v1/auth/password", "/api/v1/auth/ws-ticket", "/api/v1/overview", "/api/v1/workflow",
		},
	}
}

func (m *module) Models() []any {
	return []any{
		&model.Department{},
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.UserRole{},
		&model.RegistrationRequest{},
		&model.Menu{},
		&model.MenuPermissionBinding{},
		&model.LoginLog{},
		&model.OperationLog{},
		&model.DictEntry{},
		&model.UserGroup{},
		&model.UserGroupUser{},
		&model.PlatformSavedQuery{},
		&model.WorkflowDefinition{},
		&model.WorkflowStage{},
		&model.WorkflowTicket{},
		&model.WorkflowTicketStep{},
	}
}

func (m *module) PreMigrate(db *gorm.DB) error {
	return bootstrapPreMigrate(db)
}

func (m *module) PostMigrate(db *gorm.DB) error {
	return bootstrapPostMigrateCore(db)
}
