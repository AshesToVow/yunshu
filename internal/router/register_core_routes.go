package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterCoreRoutes 平台内核：认证、RBAC、菜单、字典、审计日志等。
func RegisterCoreRoutes(api *gin.RouterGroup, d CoreRouteDeps) {
	api.GET("/health", d.SystemHandler().Health)
	// 兼容旧探活；正式探针见 /livez、/readyz（进程根路径）
	api.GET("/ready", d.SystemHandler().Health)

	authGroup := api.Group("/auth")
	authGroup.POST("/verification-code", d.AuthHandler().SendEmailCode)
	authGroup.POST("/login-code", d.AuthHandler().SendLoginCodeByUsername)
	authGroup.POST("/password-login-code", d.AuthHandler().SendPasswordLoginCode)
	authGroup.POST("/login", d.AuthHandler().Login)
	authGroup.POST("/email-login", d.AuthHandler().EmailLogin)
	authGroup.POST("/register", middleware.RegistrationRateLimit(d.App().Redis), d.RegistrationHandler().Apply)
	authGroup.GET("/password-policy", d.AuthHandler().GetPasswordPolicy)
	authGroup.POST("/refresh", d.AuthHandler().Refresh)
	authGroup.POST("/logout", d.AuthHandler().Logout)

	authAuthed := authGroup.Group("")
	authAuthed.Use(d.AuthMiddleware(), d.OpAudit())
	authAuthed.POST("/ws-ticket", d.AuthHandler().CreateWSTicket)
	authAuthed.GET("/me", d.AuthHandler().Me)
	authAuthed.PUT("/me", d.AuthHandler().UpdateProfile)
	authAuthed.PUT("/password", d.AuthHandler().ChangePassword)

	users := api.Group("/users")
	users.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	users.GET("", d.UserHandler().List)
	users.GET("/export", d.UserHandler().Export)
	users.GET("/import-template", d.UserHandler().ImportTemplate)
	users.POST("/import", d.UserHandler().Import)
	users.POST("", d.UserHandler().Create)
	users.GET("/:id", d.UserHandler().Detail)
	users.PUT("/:id", d.UserHandler().Update)
	users.DELETE("/:id", d.UserHandler().Delete)
	users.PUT("/:id/roles", d.UserHandler().AssignRoles)

	departments := api.Group("/departments")
	departments.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	departments.GET("/tree", d.DepartmentHandler().Tree)
	departments.GET("/:id", d.DepartmentHandler().Detail)
	departments.POST("", d.DepartmentHandler().Create)
	departments.PUT("/:id", d.DepartmentHandler().Update)
	departments.DELETE("/:id", d.DepartmentHandler().Delete)

	roles := api.Group("/roles")
	roles.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	roles.GET("", d.RoleHandler().List)
	roles.POST("", d.RoleHandler().Create)
	roles.GET("/:id", d.RoleHandler().Detail)
	roles.PUT("/:id", d.RoleHandler().Update)
	roles.DELETE("/:id", d.RoleHandler().Delete)

	userGroups := api.Group("/user-groups")
	userGroups.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	userGroups.GET("", d.UserGroupHandler().List)
	userGroups.POST("", d.UserGroupHandler().Create)
	userGroups.GET("/:id", d.UserGroupHandler().Detail)
	userGroups.PUT("/:id", d.UserGroupHandler().Update)
	userGroups.DELETE("/:id", d.UserGroupHandler().Delete)
	userGroups.PUT("/:id/users", d.UserGroupHandler().AssignUsers)

	permissions := api.Group("/permissions")
	permissions.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	permissions.GET("", d.PermissionHandler().List)
	permissions.POST("/k8s-scope/batch", d.PermissionHandler().BatchSetK8sScope)
	permissions.POST("", d.PermissionHandler().Create)
	permissions.GET("/:id", d.PermissionHandler().Detail)
	permissions.PUT("/:id", d.PermissionHandler().Update)
	permissions.DELETE("/:id", d.PermissionHandler().Delete)
	registerPermissionSyncRoute(permissions, d)

	policies := api.Group("/policies")
	policies.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	policies.GET("/menu-links", d.PolicyHandler().MenuLinks)
	policies.GET("/conflicts", d.PolicyHandler().Conflicts)
	policies.POST("/conflicts/fix-menu-entry", d.PolicyHandler().FixMenuEntryAPIs)
	policies.POST("/conflicts/fix-disabled-plugin", d.PolicyHandler().FixDisabledPluginPolicies)
	policies.GET("/permission-tree", d.PolicyHandler().PermissionTree)
	policies.POST("/simulate", d.PolicyHandler().Simulate)
	policies.GET("", d.PolicyHandler().List)
	policies.POST("", d.PolicyHandler().Grant)
	policies.DELETE("", d.PolicyHandler().Revoke)

	registrations := api.Group("/registrations")
	registrations.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	registrations.GET("", d.RegistrationHandler().List)
	registrations.POST("/:id/review", d.RegistrationHandler().Review)

	admin := api.Group("/security")
	admin.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	admin.GET("/banned-ips", d.AdminHandler().ListBannedIPs)
	admin.POST("/banned-ips/unban", d.AdminHandler().UnbanIP)

	menusTree := api.Group("/menus")
	// 侧栏树：登录即可；按用户角色过滤，不要求菜单管理写权限。
	menusTree.Use(d.AuthMiddleware(), d.OpAudit())
	menusTree.GET("/tree", d.MenuHandler().Tree)

	menus := api.Group("/menus")
	menus.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	menus.GET("", d.MenuHandler().List)
	menus.POST("", d.MenuHandler().Create)
	menus.PUT("/status", d.MenuHandler().BatchStatus)
	menus.GET("/:id/bindings", d.MenuHandler().GetBindings)
	menus.PUT("/:id/bindings", d.MenuHandler().ReplaceBindings)
	menus.PUT("/:id", d.MenuHandler().Update)
	menus.DELETE("/:id", d.MenuHandler().Delete)

	dictEntries := api.Group("/dict/entries")
	dictEntries.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	dictEntries.GET("", d.DictEntryHandler().List)
	dictEntries.POST("", d.DictEntryHandler().Create)
	dictEntries.POST("/:id/reveal-value", d.DictEntryHandler().RevealValue)
	dictEntries.PUT("/:id", d.DictEntryHandler().Update)
	dictEntries.DELETE("/:id", d.DictEntryHandler().Delete)

	dictOptions := api.Group("/dict/options")
	dictOptions.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	dictOptions.GET("/:dictType", d.DictEntryHandler().Options)

	loginLogs := api.Group("/login-logs")
	loginLogs.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	loginLogs.GET("/export", d.LoginLogHandler().Export)
	loginLogs.GET("", d.LoginLogHandler().List)
	loginLogs.POST("/delete", d.LoginLogHandler().BatchDelete)
	loginLogs.DELETE("/:id", d.LoginLogHandler().Delete)

	operationLogs := api.Group("/operation-logs")
	operationLogs.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	operationLogs.GET("/export", d.OperationLogHandler().Export)
	operationLogs.GET("", d.OperationLogHandler().List)
	operationLogs.POST("/delete", d.OperationLogHandler().BatchDelete)
	operationLogs.DELETE("/:id", d.OperationLogHandler().Delete)

	plugins := api.Group("/plugins")
	plugins.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	plugins.GET("", d.PluginHandler().List)

	registerWorkflowRoutes(api, d)
	registerPlatformTemplateRoutes(api, d)
}
