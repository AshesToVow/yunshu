package plugin

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"log/slog"

	"yunshu/internal/config"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var registry []Module

// RouteBinder ? router ?????????? plugin ? router ?????
type RouteBinder func(name string, api *gin.RouterGroup, rt *Runtime) error

var routeBinder RouteBinder

// Register ? init ??????GVA ?? compile-time ?????
func Register(m Module) {
	if m == nil {
		return
	}
	registry = append(registry, m)
}

// SetRouteBinder ?? HTTP ???????? router ??????
func SetRouteBinder(fn RouteBinder) {
	routeBinder = fn
}

// All ??????????? Name ????
func All() []Module {
	out := append([]Module(nil), registry...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name() < out[j].Name()
	})
	return out
}

// DefaultEnabled ??? plugins.enabled ?????????
func DefaultEnabled() []string {
	return []string{"core", "k8s", "alert", "project", "cmdb", "backup", "cicd", "dbmgmt", "inspect", "ai"}
}

// ResolveEnabled ???????????????
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

// FilterEnabled ??????????????????????
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

// Migrate ??????? PreMigrate ? AutoMigrate(Models) ? PostMigrate?
func Migrate(db *gorm.DB, cfg *config.PluginsConfig) error {
	if db == nil {
		return nil
	}
	log := slog.Default().With("component", "plugin.migrate")
	for _, m := range FilterEnabled(cfg) {
		log.Info("Migrating plugin", "plugin", m.Name())
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

// RegisterRoutes ????????? HTTP ???? router ??? RouteBinder??
func RegisterRoutes(api *gin.RouterGroup, rt *Runtime, cfg *config.PluginsConfig) error {
	if rt == nil {
		return fmt.Errorf("plugin runtime required")
	}
	if routeBinder == nil {
		return fmt.Errorf("plugin route binder not configured")
	}
	log := slog.Default().With("component", "plugin.routes")
	for _, m := range FilterEnabled(cfg) {
		log.Info("Registering routes", "plugin", m.Name())
		if err := routeBinder(m.Name(), api, rt); err != nil {
			return fmt.Errorf("plugin %s routes: %w", m.Name(), err)
		}
	}
	return nil
}

// StartWorkers ??????????????
func StartWorkers(bgCtx context.Context, rt *Runtime, cfg *config.PluginsConfig) error {
	if rt == nil {
		return fmt.Errorf("plugin runtime required")
	}
	log := slog.Default().With("component", "plugin.workers")
	for _, m := range FilterEnabled(cfg) {
		log.Info("Starting workers", "plugin", m.Name())
		if err := m.StartWorkers(bgCtx, rt); err != nil {
			return fmt.Errorf("plugin %s workers: %w", m.Name(), err)
		}
	}
	return nil
}

// RegisteredNames ?????????????????
func RegisteredNames() []string {
	names := make([]string, 0, len(registry))
	for _, m := range registry {
		names = append(names, m.Name())
	}
	sort.Strings(names)
	return names
}
