package plugin

import (
	"yunshu/internal/config"

	"gorm.io/gorm"
)

// Runtime 插件运行时共享依赖（由 router.Register 注入，避免依赖 bootstrap 包）。
type Runtime struct {
	DB                      *gorm.DB
	Config                  *config.Config
	YamlK8sEventForwardBase config.K8sEventForwardConfig
	Enabled                 map[string]bool
	// Deps 为 *router.RouteDeps，使用 any 避免 plugin ↔ router 循环依赖。
	Deps any
	// 下列字段由 router 填充，供插件后台任务使用。
	K8sRuntime   any // *service.K8sRuntimeService
	MysqlBackup  any // *service.MysqlBackupService
	Dbmgmt       any // *dbmgmt.Service
	Cicd         any // *cicd.Service
	Alert        any // *service.AlertService
	LogRetention any // *service.LogRetentionService
	KafkaToES    any // *service.KafkaToESService
}

// IsEnabled 判断某插件是否在当前配置下启用。
func (rt *Runtime) IsEnabled(name string) bool {
	if rt == nil || rt.Enabled == nil {
		return true
	}
	return rt.Enabled[name]
}
