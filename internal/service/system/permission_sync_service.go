package system

import (
	"context"
	"errors"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/apiroute"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/plugingate"
	"yunshu/internal/plugin"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PermissionSyncService 启动时将 Gin 路由与插件 Manifest 同步到 permissions 表。
type PermissionSyncService struct {
	permissionRepo interfaces.PermissionRepository
	plugins        *config.PluginsConfig
}

// PermissionSyncResult 同步统计。
type PermissionSyncResult struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
	Total   int `json:"total"`
}

// NewPermissionSyncService 创建权限同步服务。
func NewPermissionSyncService(permissionRepo interfaces.PermissionRepository, plugins *config.PluginsConfig) *PermissionSyncService {
	return &PermissionSyncService{permissionRepo: permissionRepo, plugins: plugins}
}

// SyncFromEngine 扫描 Gin 路由并 upsert 到 permissions（不覆盖已有手工维护的名称）。
func (s *PermissionSyncService) SyncFromEngine(ctx context.Context, engine *gin.Engine) (*PermissionSyncResult, error) {
	if s == nil || s.permissionRepo == nil {
		return nil, bizerrors.Pass(ctx, "permission.sync", "SyncFromEngine", errors.New("permission repo required"))
	}
	entries := apiroute.Collect(engine)
	out := &PermissionSyncResult{Total: len(entries)}
	for _, e := range entries {
		if s.plugins != nil && !plugingate.IsAPIResourceAllowed(e.Path, s.plugins) {
			out.Skipped++
			continue
		}
		pluginName := plugingate.ResolveAPIResourcePlugin(e.Path)
		created, err := s.upsertRoutePermission(ctx, e.Method, e.Path, pluginName)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "permission.sync", "SyncFromEngine", err)
		}
		if created {
			out.Created++
		} else {
			out.Skipped++
		}
	}
	if err := s.syncPluginManifestPrefixes(ctx); err != nil {
		return nil, bizerrors.Pass(ctx, "permission.sync", "SyncFromEngine", err)
	}
	return out, nil
}

func (s *PermissionSyncService) upsertRoutePermission(ctx context.Context, method, path, pluginName string) (created bool, err error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.TrimSpace(path)
	existing, err := s.permissionRepo.GetByResourceAction(ctx, path, method)
	if err == nil && existing != nil {
		return false, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	row := model.Permission{
		Name:        apiroute.DefaultName(method, path),
		Resource:    path,
		Action:      method,
		Description: apiroute.AutoSyncDescription(pluginName),
	}
	if err := s.permissionRepo.Create(ctx, &row); err != nil {
		if existing, getErr := s.permissionRepo.GetByResourceAction(ctx, path, method); getErr == nil && existing != nil {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// syncPluginManifestPrefixes 为插件 Manifest APIPrefixes 登记 GET 入口权限（便于策略树分组）。
func (s *PermissionSyncService) syncPluginManifestPrefixes(ctx context.Context) error {
	enabled := plugin.FilterEnabled(s.plugins)
	for _, m := range enabled {
		mf := plugin.ResolveManifest(m)
		for _, prefix := range mf.APIPrefixes {
			prefix = strings.TrimSpace(prefix)
			if prefix == "" {
				continue
			}
			_, err := s.upsertRoutePermission(ctx, "GET", prefix, m.Name())
			if err != nil {
				return err
			}
		}
	}
	return nil
}
