package system

import (
	"context"
	"errors"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apiroute"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/permcatalog"
	"yunshu/internal/plugin"
	"yunshu/internal/plugingate"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

// CatalogEntry 权限目录项：目录登记信息 + 当前进程里的路由与数据库登记状态。
type CatalogEntry struct {
	Group       string `json:"group"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Auth        bool   `json:"auth"`
	// Registered 权限表中已存在该权限点（可被授权）。
	Registered bool `json:"registered"`
	// Live 当前进程注册了同 method+path 的路由（否则登记了也无法生效）。
	Live bool `json:"live"`
}

// CatalogGroup 目录分组。
type CatalogGroup struct {
	Title   string         `json:"title"`
	Entries []CatalogEntry `json:"entries"`
}

// CatalogResult 目录快照。
type CatalogResult struct {
	Groups []CatalogGroup `json:"groups"`
	// Missing 需要登录鉴权但尚未登记到权限表的条目数。
	Missing int `json:"missing"`
	// UnregisteredRoutes 有路由但目录里没有人工元数据的条目（新增接口需补目录）。
	UnregisteredRoutes []string `json:"unregistered_routes"`
}

const unregisteredGroupTitle = "运行路由（目录未登记）"

// Catalog 汇总单一来源目录（permcatalog）+ 实际路由 + 权限表，供权限页展示与补全。
func (s *PermissionSyncService) Catalog(ctx context.Context, engine *gin.Engine) (*CatalogResult, error) {
	if s == nil || s.permissionRepo == nil {
		return nil, bizerrors.Pass(ctx, "permission.sync", "Catalog", errors.New("permission repo required"))
	}
	perms, err := s.permissionRepo.ListAll(ctx)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "permission.sync", "Catalog", err)
	}
	registered := make(map[string]struct{}, len(perms))
	for _, p := range perms {
		registered[permcatalog.Key(p.Action, p.Resource)] = struct{}{}
	}
	live := make(map[string]struct{})
	for _, e := range apiroute.Collect(engine) {
		if s.plugins != nil && !plugingate.IsAPIResourceAllowed(e.Path, s.plugins) {
			continue
		}
		live[permcatalog.Key(e.Method, e.Path)] = struct{}{}
	}

	out := &CatalogResult{Groups: make([]CatalogGroup, 0, 12)}
	catalogKeys := make(map[string]struct{})
	for _, g := range permcatalog.Groups() {
		group := CatalogGroup{Title: g.Title, Entries: make([]CatalogEntry, 0, len(g.Entries))}
		for _, p := range g.Entries {
			if s.plugins != nil && !plugingate.IsAPIResourceAllowed(p.Resource, s.plugins) {
				continue
			}
			key := permcatalog.Key(p.Action, p.Resource)
			catalogKeys[key] = struct{}{}
			_, isRegistered := registered[key]
			_, isLive := live[key]
			group.Entries = append(group.Entries, CatalogEntry{
				Group:       g.Title,
				Name:        p.Name,
				Description: p.Description,
				Method:      strings.ToUpper(strings.TrimSpace(p.Action)),
				Path:        strings.TrimSpace(p.Resource),
				Auth:        true,
				Registered:  isRegistered,
				Live:        isLive,
			})
			if !isRegistered && isLive {
				out.Missing++
			}
		}
		out.Groups = append(out.Groups, group)
	}

	// 有路由但目录未收编：一并列出，避免新接口只进权限表、失去人工命名。
	unregistered := make([]CatalogEntry, 0)
	for _, e := range apiroute.Collect(engine) {
		if s.plugins != nil && !plugingate.IsAPIResourceAllowed(e.Path, s.plugins) {
			continue
		}
		key := permcatalog.Key(e.Method, e.Path)
		if _, ok := catalogKeys[key]; ok {
			continue
		}
		out.UnregisteredRoutes = append(out.UnregisteredRoutes, key)
		_, isRegistered := registered[key]
		unregistered = append(unregistered, CatalogEntry{
			Group:       unregisteredGroupTitle,
			Name:        apiroute.DefaultName(e.Method, e.Path),
			Description: apiroute.AutoSyncDescription(""),
			Method:      strings.ToUpper(strings.TrimSpace(e.Method)),
			Path:        strings.TrimSpace(e.Path),
			Auth:        true,
			Registered:  isRegistered,
			Live:        true,
		})
		if !isRegistered {
			out.Missing++
		}
	}
	if len(unregistered) > 0 {
		out.Groups = append(out.Groups, CatalogGroup{Title: unregisteredGroupTitle, Entries: unregistered})
	}
	return out, nil
}

// ReclaimResult 启动期孤儿权限回收统计。
type ReclaimResult struct {
	DeletedPermissions int      `json:"deleted_permissions"`
	RemovedPolicies    int      `json:"removed_policies"`
	Skipped            int      `json:"skipped"`
	Aborted            bool     `json:"aborted"`
	Reason             string   `json:"reason,omitempty"`
	Samples            []string `json:"samples,omitempty"`
}

// defaultMaxReclaim 单次回收上限：超过说明路由注册异常（例如插件加载失败），宁可不删。
const defaultMaxReclaim = 100

const reclaimSampleLimit = 10

// ReclaimOrphans 回收「路由已下线」的权限点与孤儿策略。
// 只处理可推导来源（auto-synced 或目录已登记）的行，人工自建权限点保持不动；
// 禁用插件的路径跳过；单次超过 defaultMaxReclaim 直接放弃并留待人工确认。
func (s *PermissionSyncService) ReclaimOrphans(ctx context.Context, engine *gin.Engine, enforcer *casbin.SyncedEnforcer) (*ReclaimResult, error) {
	if s == nil || s.permissionRepo == nil {
		return nil, bizerrors.Pass(ctx, "permission.sync", "ReclaimOrphans", errors.New("permission repo required"))
	}
	out := &ReclaimResult{}
	live := make(map[string]struct{})
	for _, e := range apiroute.Collect(engine) {
		if s.plugins != nil && !plugingate.IsAPIResourceAllowed(e.Path, s.plugins) {
			continue
		}
		live[permcatalog.Key(e.Method, e.Path)] = struct{}{}
	}
	// 路由表为空通常意味着装配失败；此时回收会把整库权限删光。
	if len(live) == 0 {
		out.Aborted = true
		out.Reason = "no live api routes"
		return out, nil
	}

	perms, err := s.permissionRepo.ListAll(ctx)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "permission.sync", "ReclaimOrphans", err)
	}
	manifestPrefixes := s.pluginManifestPrefixes()
	permissionKeys := make(map[string]struct{}, len(perms))
	stale := make([]model.Permission, 0)
	for _, p := range perms {
		resource := strings.TrimSpace(p.Resource)
		action := strings.ToUpper(strings.TrimSpace(p.Action))
		key := permcatalog.Key(action, resource)
		permissionKeys[key] = struct{}{}
		if !strings.HasPrefix(resource, "/api/v1") || permcatalog.IsPseudoResource(resource) {
			continue
		}
		if _, ok := manifestPrefixes[resource]; ok {
			// 插件 Manifest 前缀只有登记项、没有同名路由，删了下次同步还会重建。
			out.Skipped++
			continue
		}
		if _, ok := live[key]; ok {
			continue
		}
		managed := apiroute.IsAutoSyncedDescription(p.Description)
		if !managed {
			_, managed = permcatalog.Lookup(action, resource)
		}
		if !managed {
			out.Skipped++
			continue
		}
		if s.plugins != nil && !plugingate.IsAPIResourceAllowed(resource, s.plugins) {
			out.Skipped++
			continue
		}
		stale = append(stale, p)
	}

	orphanPolicies := make(map[string][2]string)
	if enforcer != nil {
		for _, rule := range enforcer.GetPolicy() {
			if len(rule) < 3 {
				continue
			}
			resource := strings.TrimSpace(rule[1])
			action := strings.ToUpper(strings.TrimSpace(rule[2]))
			if !strings.HasPrefix(resource, "/api/v1") || permcatalog.IsPseudoResource(resource) {
				continue
			}
			if _, ok := manifestPrefixes[resource]; ok {
				continue
			}
			key := permcatalog.Key(action, resource)
			if _, ok := permissionKeys[key]; ok {
				continue
			}
			if _, ok := live[key]; ok {
				continue
			}
			if s.plugins != nil && !plugingate.IsAPIResourceAllowed(resource, s.plugins) {
				out.Skipped++
				continue
			}
			orphanPolicies[key] = [2]string{resource, action}
		}
	}

	if len(stale)+len(orphanPolicies) > defaultMaxReclaim {
		out.Aborted = true
		out.Reason = "exceeds reclaim threshold"
		return out, nil
	}

	for _, p := range stale {
		if err := s.permissionRepo.Delete(ctx, &p); err != nil {
			return nil, bizerrors.Pass(ctx, "permission.sync", "ReclaimOrphans", err)
		}
		out.DeletedPermissions++
		if len(out.Samples) < reclaimSampleLimit {
			out.Samples = append(out.Samples, permcatalog.Key(p.Action, p.Resource))
		}
		if enforcer == nil {
			continue
		}
		removed := len(enforcer.GetFilteredPolicy(1, p.Resource, strings.ToUpper(strings.TrimSpace(p.Action))))
		if err := RemovePermissionPolicies(enforcer, p.Resource, p.Action); err != nil {
			return nil, bizerrors.Pass(ctx, "permission.sync", "ReclaimOrphans", err)
		}
		out.RemovedPolicies += removed
		delete(orphanPolicies, permcatalog.Key(p.Action, p.Resource))
	}

	for _, pair := range orphanPolicies {
		if enforcer == nil {
			break
		}
		removed := len(enforcer.GetFilteredPolicy(1, pair[0], pair[1]))
		if err := RemovePermissionPolicies(enforcer, pair[0], pair[1]); err != nil {
			return nil, bizerrors.Pass(ctx, "permission.sync", "ReclaimOrphans", err)
		}
		out.RemovedPolicies += removed
		if len(out.Samples) < reclaimSampleLimit {
			out.Samples = append(out.Samples, "policy "+permcatalog.Key(pair[1], pair[0]))
		}
	}

	if out.DeletedPermissions > 0 || out.RemovedPolicies > 0 {
		notifyPolicyChange()
	}
	return out, nil
}

// pluginManifestPrefixes 已启用插件声明的 API 前缀。
// 这些前缀只作为权限分组入口登记（无同名路由），回收时必须跳过，否则每轮启动都会删了又建。
func (s *PermissionSyncService) pluginManifestPrefixes() map[string]struct{} {
	out := make(map[string]struct{})
	if s == nil {
		return out
	}
	for _, m := range plugin.FilterEnabled(s.plugins) {
		for _, prefix := range plugin.ResolveManifest(m).APIPrefixes {
			prefix = strings.TrimSpace(prefix)
			if prefix == "" {
				continue
			}
			out[prefix] = struct{}{}
		}
	}
	return out
}
