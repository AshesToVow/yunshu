// Package routedeps 定义插件 HTTP 路由所需的窄依赖接口。
//
// *router.RouteDeps 实现 Bundle。HTTP 依赖经 router.SetRouteBinder 闭包注入，
// 不挂在 plugin.Runtime 上，以避免 plugin → routedeps → handler → … → plugin 循环。
package routedeps

// Bundle 插件 HTTP 路由所需的完整依赖面（由 *router.RouteDeps 实现）。
type Bundle interface {
	CoreRouteDeps
	K8sRouteDeps
	AlertRouteDeps
	ProjectRouteDeps
	LogPlatformRouteDeps
	CMDBRouteDeps
	BackupRouteDeps
	CicdRouteDeps
	DbmgmtRouteDeps
	InspectRouteDeps
	AIRouteDeps
	EsmgmtRouteDeps
	KafkamgmtRouteDeps
}
