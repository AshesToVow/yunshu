package router

import "yunshu/internal/routedeps"

// 窄路由依赖类型别名（实现位于 routedeps，便于 Register* 签名稳定）。
type (
	AIRouteDeps = routedeps.AIRouteDeps
	AlertRouteDeps = routedeps.AlertRouteDeps
	BackupRouteDeps = routedeps.BackupRouteDeps
	CMDBRouteDeps = routedeps.CMDBRouteDeps
	CicdRouteDeps = routedeps.CicdRouteDeps
	CoreRouteDeps = routedeps.CoreRouteDeps
	DbmgmtRouteDeps = routedeps.DbmgmtRouteDeps
	EsmgmtRouteDeps = routedeps.EsmgmtRouteDeps
	InspectRouteDeps = routedeps.InspectRouteDeps
	K8sRouteDeps = routedeps.K8sRouteDeps
	LogPlatformRouteDeps = routedeps.LogPlatformRouteDeps
	PlatformTemplateRouteDeps = routedeps.PlatformTemplateRouteDeps
	ProjectAccessDeps = routedeps.ProjectAccessDeps
	ProjectRouteDeps = routedeps.ProjectRouteDeps
	RouteMiddleware = routedeps.RouteMiddleware
	WorkflowRouteDeps = routedeps.WorkflowRouteDeps
	Bundle         = routedeps.Bundle
)

var _ routedeps.Bundle = (*RouteDeps)(nil)
