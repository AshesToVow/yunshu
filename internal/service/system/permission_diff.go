package system

import (
	"fmt"
	"sort"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apiroute"
)

// RoutePolicyDiff 启动期「路由 - 权限点 - Casbin 策略」三方差异快照。
// 只报告不删除：差异可能来自手工维护的权限点或插件未启用，自动清理会误删授权。
type RoutePolicyDiff struct {
	// RouteWithoutPermission 有路由但没有对应权限点（新接口漏同步 → 授权页看不到）。
	RouteWithoutPermission []string `json:"route_without_permission"`
	// PermissionWithoutRoute 有权限点但路由不存在（接口下线/改名后残留，仍可被授权）。
	PermissionWithoutRoute []string `json:"permission_without_route"`
	// PolicyWithoutPermission 有策略但权限点不存在（权限点被删后策略未清理，属孤儿授权）。
	PolicyWithoutPermission []string `json:"policy_without_permission"`
}

// IsEmpty 三方完全一致时返回 true。
func (d RoutePolicyDiff) IsEmpty() bool {
	return len(d.RouteWithoutPermission) == 0 && len(d.PermissionWithoutRoute) == 0 && len(d.PolicyWithoutPermission) == 0
}

// Samples 返回每类最多 n 条样本，便于日志限长输出。
func (d RoutePolicyDiff) Samples(n int) map[string][]string {
	return map[string][]string{
		"route_without_permission":  truncate(d.RouteWithoutPermission, n),
		"permission_without_route":  truncate(d.PermissionWithoutRoute, n),
		"policy_without_permission": truncate(d.PolicyWithoutPermission, n),
	}
}

func truncate(items []string, n int) []string {
	if n <= 0 || len(items) <= n {
		return items
	}
	return items[:n]
}

// BuildRoutePolicyDiff 计算差异。routes 应只包含最终对外生效的路由（调用方先做插件过滤），
// permissions/policies 中非 "/" 开头的资源（如自定义伪权限）不参与比对。
func BuildRoutePolicyDiff(routes []apiroute.Entry, permissions []model.Permission, policies [][]string) RoutePolicyDiff {
	routeKeys := make(map[string]struct{}, len(routes))
	for _, r := range routes {
		p := strings.TrimSpace(r.Path)
		if !strings.HasPrefix(p, "/") {
			continue
		}
		routeKeys[routeKey(r.Method, p)] = struct{}{}
	}

	permissionKeys := make(map[string]struct{}, len(permissions))
	for _, p := range permissions {
		resource := strings.TrimSpace(p.Resource)
		if !strings.HasPrefix(resource, "/") {
			continue
		}
		permissionKeys[routeKey(p.Action, resource)] = struct{}{}
	}

	diff := RoutePolicyDiff{}
	for key := range routeKeys {
		if _, ok := permissionKeys[key]; !ok {
			diff.RouteWithoutPermission = append(diff.RouteWithoutPermission, key)
		}
	}
	for key := range permissionKeys {
		if _, ok := routeKeys[key]; !ok {
			diff.PermissionWithoutRoute = append(diff.PermissionWithoutRoute, key)
		}
	}
	seenPolicy := make(map[string]struct{}, len(policies))
	for _, policy := range policies {
		if len(policy) < 3 {
			continue
		}
		resource := strings.TrimSpace(policy[1])
		if !strings.HasPrefix(resource, "/") {
			continue
		}
		key := routeKey(policy[2], resource)
		if _, ok := permissionKeys[key]; ok {
			continue
		}
		line := fmt.Sprintf("%s -> %s", strings.TrimSpace(policy[0]), key)
		if _, dup := seenPolicy[line]; dup {
			continue
		}
		seenPolicy[line] = struct{}{}
		diff.PolicyWithoutPermission = append(diff.PolicyWithoutPermission, line)
	}

	sort.Strings(diff.RouteWithoutPermission)
	sort.Strings(diff.PermissionWithoutRoute)
	sort.Strings(diff.PolicyWithoutPermission)
	return diff
}

func routeKey(method, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(path)
}
