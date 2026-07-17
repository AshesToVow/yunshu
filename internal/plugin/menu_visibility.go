package plugin

import (
	"context"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/model"

	"gorm.io/gorm"
)

// SyncMenuVisibility 将禁用插件对应的菜单 status 置为 0（仅处理 catalog 可识别的 path，不自动启用）。
func SyncMenuVisibility(ctx context.Context, db *gorm.DB, cfg *config.PluginsConfig) error {
	if db == nil || cfg == nil {
		return nil
	}
	var menus []model.Menu
	if err := db.WithContext(ctx).Where("path <> '' AND path IS NOT NULL").Find(&menus).Error; err != nil {
		return err
	}
	for _, m := range menus {
		path := strings.TrimSpace(m.Path)
		if path == "" {
			continue
		}
		if ResolveMenuPathPlugin(path) == "" {
			continue
		}
		if IsMenuPathAllowed(path, cfg) {
			continue
		}
		if m.Status == 0 {
			continue
		}
		if err := db.WithContext(ctx).Model(&model.Menu{}).Where("id = ?", m.ID).Update("status", 0).Error; err != nil {
			return err
		}
	}
	return nil
}
