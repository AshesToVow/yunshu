package bootstrap

import (
	"context"

	"yunshu/internal/config"
	"yunshu/internal/menu"
	"yunshu/internal/plugin"

	_ "yunshu/internal/plugins/all" // 注册内置业务插件

	"gorm.io/gorm"
)

// AutoMigrateModels 按启用的业务插件执行库表迁移，并同步内置菜单（与 `go run . migrate` / 服务启动一致）。
func AutoMigrateModels(db *gorm.DB, pluginsCfg *config.PluginsConfig) error {
	if db == nil {
		return nil
	}
	if err := plugin.Migrate(db, pluginsCfg); err != nil {
		return err
	}
	ctx := context.Background()
	if err := menu.Sync(ctx, db); err != nil {
		return err
	}
	return plugin.SyncMenuVisibility(ctx, db, pluginsCfg)
}
