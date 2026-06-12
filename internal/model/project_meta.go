package model

// 项目类型（project_type）
const (
	ProjectTypeBusiness  = "business"  // 业务项目
	ProjectTypePlatform  = "platform"  // 平台项目
	ProjectTypeInfra     = "infra"     // 基础设施
	ProjectTypeResearch  = "research"  // 研发/试验
)

// 项目生命周期状态（lifecycle_status），与启用/停用 status 字段独立。
const (
	ProjectLifecyclePlanning  = "planning"  // 规划中
	ProjectLifecycleActive    = "active"    // 进行中
	ProjectLifecycleSuspended = "suspended" // 已暂停
	ProjectLifecycleArchived  = "archived"  // 已归档
)

// ValidProjectTypes 合法项目类型。
func ValidProjectTypes() []string {
	return []string{ProjectTypeBusiness, ProjectTypePlatform, ProjectTypeInfra, ProjectTypeResearch}
}

// ValidProjectLifecycles 合法项目生命周期状态。
func ValidProjectLifecycles() []string {
	return []string{ProjectLifecyclePlanning, ProjectLifecycleActive, ProjectLifecycleSuspended, ProjectLifecycleArchived}
}
