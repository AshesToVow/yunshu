package plugin

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/pkg/logutil"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var registry []Module

// RouteBinder 由 router 包在启动时注入，避免 plugin ↔ router 循环依赖。
type RouteBinder func(name string, api *gin.RouterGroup, rt *Runtime) error

var routeBinder RouteBinder

// Register 在 init 中注册插件（GVA 风格 compile-time 插件表）。
func Register(m Module) {
	if m == nil {
		return
	}
	registry = append(registry, m)
}

// SetRouteBinder 注册 HTTP 路由绑定函数（仅 router 包应调用）。
func SetRouteBinder(fn RouteBinder) {
	routeBinder = fn
}

// All 返回已注册插件副本（按 Name 排序）。
func All() []Module {
	out := append([]Module(nil), registry...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name() < out[j].Name()
	})
	return out
}

// DefaultEnabled 未配置 plugins.enabled 时的默认启用列表。
func DefaultEnabled() []string {
	return []string{"core", "k8s", "alert", "project", "cmdb", "backup", "cicd"}
}

// ResolveEnabled 根据配置解析启用的插件名集合。
func ResolveEnabled(cfg *config.PluginsConfig) map[string]bool {
	names := DefaultEnabled()
	if cfg != nil && len(cfg.Enabled) > 0 {
		names = cfg.Enabled
	}
	m := make(map[string]bool, len(names))
	for _, n := range names {
		n = strings.TrimSpace(strings.ToLower(n))
		if n == "" {
			continue
		}
		m[n] = true
	}
	return m
}

// FilterEnabled 返回按配置过滤后的插件列表（保持注册顺序）。
func FilterEnabled(cfg *config.PluginsConfig) []Module {
	enabled := ResolveEnabled(cfg)
	var out []Module
	for _, m := range registry {
		if enabled[strings.ToLower(m.Name())] {
			out = append(out, m)
		}
	}
	return out
}

// Migrate 对启用插件执行 PreMigrate → AutoMigrate(Models) → PostMigrate。
func Migrate(db *gorm.DB, cfg *config.PluginsConfig) error {
	if db == nil {
		return nil
	}
	log := logutil.Worker("plugin.migrate")
	for _, m := range FilterEnabled(cfg) {
		log.Infow("Migrating plugin", "plugin", m.Name())
		if err := m.PreMigrate(db); err != nil {
			return fmt.Errorf("plugin %s pre-migrate: %w", m.Name(), err)
		}
		models := m.Models()
		if len(models) > 0 {
			if err := db.AutoMigrate(models...); err != nil {
				return fmt.Errorf("plugin %s auto-migrate: %w", m.Name(), err)
			}
		}
		if err := m.PostMigrate(db); err != nil {
			return fmt.Errorf("plugin %s post-migrate: %w", m.Name(), err)
		}
	}
	return nil
}

// RegisterRoutes 为所有启用插件注册 HTTP 路由（经 router 注入的 RouteBinder）。
func RegisterRoutes(api *gin.RouterGroup, rt *Runtime, cfg *config.PluginsConfig) error {
	if rt == nil {
		return fmt.Errorf("plugin runtime required")
	}
	if routeBinder == nil {
		return fmt.Errorf("plugin route binder not configured")
	}
	log := logutil.Worker("plugin.routes")
	for _, m := range FilterEnabled(cfg) {
		log.Infow("Registering routes", "plugin", m.Name())
		if err := routeBinder(m.Name(), api, rt); err != nil {
			return fmt.Errorf("plugin %s routes: %w", m.Name(), err)
		}
	}
	return nil
}

// StartWorkers 启动所有启用插件的后台任务。
func StartWorkers(bgCtx context.Context, rt *Runtime, cfg *config.PluginsConfig) error {
	if rt == nil {
		return fmt.Errorf("plugin runtime required")
	}
	log := logutil.Worker("plugin.workers")
	for _, m := range FilterEnabled(cfg) {
		log.Infow("Starting workers", "plugin", m.Name())
		if err := m.StartWorkers(bgCtx, rt); err != nil {
			return fmt.Errorf("plugin %s workers: %w", m.Name(), err)
		}
	}
	return nil
}

// RegisteredNames 返回全部已注册插件名（含未启用）。
func RegisteredNames() []string {
	names := make([]string, 0, len(registry))
	for _, m := range registry {
		names = append(names, m.Name())
	}
	sort.Strings(names)
	return names
}
