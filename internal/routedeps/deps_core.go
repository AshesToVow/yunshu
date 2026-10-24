package routedeps

import (
	"yunshu/internal/handler"
)

type CoreRouteDeps interface {
	WorkflowRouteDeps
	PlatformTemplateRouteDeps
	SystemHandler() *handler.SystemHandler
	AuthHandler() *handler.AuthHandler
	RegistrationHandler() *handler.RegistrationHandler
	UserHandler() *handler.UserHandler
	DepartmentHandler() *handler.DepartmentHandler
	RoleHandler() *handler.RoleHandler
	UserGroupHandler() *handler.UserGroupHandler
	PermissionHandler() *handler.PermissionHandler
	PolicyHandler() *handler.PolicyHandler
	AdminHandler() *handler.AdminHandler
	MenuHandler() *handler.MenuHandler
	DictEntryHandler() *handler.DictEntryHandler
	LoginLogHandler() *handler.LoginLogHandler
	OperationLogHandler() *handler.OperationLogHandler
	PluginHandler() *handler.PluginHandler
}
