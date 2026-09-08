package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

// CoreRouteDeps 平台内核路由窄依赖（含挂在 core 下的 workflow / platform-template）。

func (d *RouteDeps) SystemHandler() *handler.SystemHandler {
	if d == nil {
		return nil
	}
	return d.systemHandler
}

func (d *RouteDeps) AuthHandler() *handler.AuthHandler {
	if d == nil {
		return nil
	}
	return d.authHandler
}

func (d *RouteDeps) RegistrationHandler() *handler.RegistrationHandler {
	if d == nil {
		return nil
	}
	return d.regHandler
}

func (d *RouteDeps) UserHandler() *handler.UserHandler {
	if d == nil {
		return nil
	}
	return d.userHandler
}

func (d *RouteDeps) DepartmentHandler() *handler.DepartmentHandler {
	if d == nil {
		return nil
	}
	return d.departmentHandler
}

func (d *RouteDeps) RoleHandler() *handler.RoleHandler {
	if d == nil {
		return nil
	}
	return d.roleHandler
}

func (d *RouteDeps) UserGroupHandler() *handler.UserGroupHandler {
	if d == nil {
		return nil
	}
	return d.userGroupHandler
}

func (d *RouteDeps) PermissionHandler() *handler.PermissionHandler {
	if d == nil {
		return nil
	}
	return d.permissionHandler
}

func (d *RouteDeps) PolicyHandler() *handler.PolicyHandler {
	if d == nil {
		return nil
	}
	return d.policyHandler
}

func (d *RouteDeps) AdminHandler() *handler.AdminHandler {
	if d == nil {
		return nil
	}
	return d.adminHandler
}

func (d *RouteDeps) MenuHandler() *handler.MenuHandler {
	if d == nil {
		return nil
	}
	return d.menuHandler
}

func (d *RouteDeps) DictEntryHandler() *handler.DictEntryHandler {
	if d == nil {
		return nil
	}
	return d.dictEntryHandler
}

func (d *RouteDeps) LoginLogHandler() *handler.LoginLogHandler {
	if d == nil {
		return nil
	}
	return d.loginLogHandler
}

func (d *RouteDeps) OperationLogHandler() *handler.OperationLogHandler {
	if d == nil {
		return nil
	}
	return d.opLogHandler
}

func (d *RouteDeps) PluginHandler() *handler.PluginHandler {
	if d == nil {
		return nil
	}
	return d.pluginHandler
}

var _ CoreRouteDeps = (*RouteDeps)(nil)

var _ routedeps.CoreRouteDeps = (*RouteDeps)(nil)
