package plugin

import (
	"yunshu/internal/config"

	"gorm.io/gorm"
)

// Runtime 插件运行时共享依赖（由 router.Register 注入，避免依赖 bootstrap 包）。
// HTTP 路由依赖不经本结构传递：router 在 SetRouteBinder 闭包中持有 routedeps.Bundle，
// 以避免 plugin → routedeps → handler → … → plugin 循环依赖。
type Runtime struct {
	DB                      *gorm.DB
	Config                  *config.Config
	YamlK8sEventForwardBase config.K8sEventForwardConfig
	Enabled                 map[string]bool
	// Worker 槽位（由 router 填充）；用 plugin.As[*Concrete](rt.Xxx) 取出。
	K8sRuntime      any // *service.K8sRuntimeService
	MysqlBackup     any // *service.MysqlBackupService
	Esmgmt          any // *esmgmt.Service
	Dbmgmt          any // *dbmgmt.Service
	Cicd            any // *cicd.Service
	Alert           any // *service.AlertService
	Inspect         any // *inspect.Service
	LogRetention    any // *service.LogRetentionService
	KafkaToES       any // *service.KafkaToESService
	LogIntelligence any // *service.LogIntelligenceService
}

// IsEnabled 判断某插件是否在当前配置下启用。
func (rt *Runtime) IsEnabled(name string) bool {
	if rt == nil || rt.Enabled == nil {
		return true
	}
	return rt.Enabled[name]
}
