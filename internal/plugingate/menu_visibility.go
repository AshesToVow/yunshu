package plugingate

import (
	"context"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/menu"
	"yunshu/internal/model"

	"gorm.io/gorm"
)

// DesiredMenuStatus 根据插件开关与内置目录计算目标 status。
// 返回 managed=false 表示该 path 不受插件可见性管理（保持原 status）。
func DesiredMenuStatus(path string, current int, cfg *config.PluginsConfig, catalogStatus map[string]int) (want int, managed bool) {
	path = strings.TrimSpace(path)
	if path == "" || ResolveMenuPathPlugin(path) == "" {
		return current, false
	}
	if !IsMenuPathAllowed(path, cfg) {
		return 0, true
	}
	if catalogWant, ok := catalogStatus[path]; ok {
		return catalogWant, true
	}
	return current, true
}

// SyncMenuVisibility 按 plugins.enabled 双向同步菜单 status：
// - 插件不可见：status=0
// - 插件可见且 path 在内置 catalog：恢复为 catalog 期望 status
// - 插件可见但不在 catalog：不改 status（保留管理员手工开关）
func SyncMenuVisibility(ctx context.Context, db *gorm.DB, cfg *config.PluginsConfig) error {
	if db == nil || cfg == nil {
		return nil
	}
	catalogStatus := menu.PathStatusMap()
	var menus []model.Menu
	if err := db.WithContext(ctx).Where("path <> '' AND path IS NOT NULL").Find(&menus).Error; err != nil {
		return err
	}
	for _, m := range menus {
		want, managed := DesiredMenuStatus(m.Path, m.Status, cfg, catalogStatus)
		if !managed || want == m.Status {
			continue
		}
		if err := db.WithContext(ctx).Model(&model.Menu{}).Where("id = ?", m.ID).Update("status", want).Error; err != nil {
			return err
		}
	}
	return nil
}
