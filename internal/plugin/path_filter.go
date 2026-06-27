package plugin

import (
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/model"
)

// ResolveMenuPathPlugin 解析控制台菜单 path 所属插件；未命中返回空（不受插件开关约束）。
func ResolveMenuPathPlugin(path string) string {
	p := normalizeUIPath(path)
	if p == "" || p == "/" {
		return "k8s"
	}
	for _, rule := range uiPathRules {
		for _, prefix := range rule.prefixes {
			if pathMatchesPrefix(p, prefix) {
				return rule.plugin
			}
		}
	}
	return ""
}

// ResolveAPIResourcePlugin 解析 Casbin permission resource 所属插件。
func ResolveAPIResourcePlugin(resource string) string {
	r := strings.TrimSpace(strings.ToLower(resource))
	if r == "" {
		return ""
	}
	for _, rule := range apiResourceRules {
		for _, prefix := range rule.prefixes {
			if strings.HasPrefix(r, prefix) {
				return rule.plugin
			}
		}
	}
	return ""
}

// IsMenuPathAllowed 菜单 path 是否对当前启用的插件可见。
func IsMenuPathAllowed(path string, cfg *config.PluginsConfig) bool {
	pluginName := ResolveMenuPathPlugin(path)
	if pluginName == "" {
		return true
	}
	if pluginName == "cmdb" {
		return isPluginEnabled(cfg, "cmdb") && isPluginEnabled(cfg, "project")
	}
	return isPluginEnabled(cfg, pluginName)
}

// IsAPIResourceAllowed permission resource 是否对当前启用的插件可见。
func IsAPIResourceAllowed(resource string, cfg *config.PluginsConfig) bool {
	pluginName := ResolveAPIResourcePlugin(resource)
	if pluginName == "" {
		return true
	}
	if pluginName == "cmdb" {
		return isPluginEnabled(cfg, "cmdb") && isPluginEnabled(cfg, "project")
	}
	return isPluginEnabled(cfg, pluginName)
}

// FilterMenusByPlugins 递归过滤菜单树（禁用插件的菜单及其空目录父节点会被移除）。
func FilterMenusByPlugins(items []model.Menu, cfg *config.PluginsConfig) []model.Menu {
	if cfg == nil {
		return items
	}
	var walk func([]model.Menu) []model.Menu
	walk = func(nodes []model.Menu) []model.Menu {
		out := make([]model.Menu, 0, len(nodes))
		for _, it := range nodes {
			child := it
			if len(it.Children) > 0 {
				child.Children = walk(it.Children)
			}
			path := strings.TrimSpace(it.Path)
			if path != "" && !IsMenuPathAllowed(path, cfg) {
				continue
			}
			if len(child.Children) == 0 && path == "" && len(it.Children) > 0 {
				continue
			}
			if len(it.Children) > 0 && len(child.Children) == 0 && strings.TrimSpace(it.Path) == "" {
				continue
			}
			out = append(out, child)
		}
		return out
	}
	return walk(items)
}

type pathRule struct {
	plugin   string
	prefixes []string
}

var uiPathRules = []pathRule{
	{
		plugin: "core",
		prefixes: []string{
			"/users", "/departments", "/roles", "/permissions", "/policies", "/registrations",
			"/menus", "/login-logs", "/operation-logs", "/banned-ips", "/dict-entries",
			"/personal-settings", "/user-groups", "/plugins",
		},
	},
	{
		plugin: "k8s",
		prefixes: []string{
			"/clusters", "/cluster", "/pods", "/namespaces", "/nodes", "/component-status",
			"/cluster-api-resources", "/horizontal-pod-autoscalers", "/k8s-resource-topology",
			"/deployments", "/statefulsets", "/daemonsets", "/cronjobs", "/jobs",
			"/configmaps", "/secrets", "/ingresses", "/ingress-classes", "/events",
			"/k8s-services", "/persistentvolumes", "/persistentvolumeclaims", "/storageclasses",
			"/crds", "/crs", "/rbac", "/serviceaccounts", "/k8s-scoped-policies",
			"/network-policies", "/k8s/",
		},
	},
	{plugin: "alert", prefixes: []string{"/alert-"}},
	{
		plugin: "project",
		prefixes: []string{
			"/projects", "/application-topology", "/project-members", "/project-services",
			"/project-logs", "/project-log-sources", "/agent-list",
		},
	},
	{plugin: "cmdb", prefixes: []string{"/project-servers", "/server-console"}},
	{plugin: "backup", prefixes: []string{"/mysql-backup"}},
}

var apiResourceRules = []pathRule{
	{
		plugin: "core",
		prefixes: []string{
			"/api/v1/users", "/api/v1/departments", "/api/v1/roles", "/api/v1/permissions",
			"/api/v1/policies", "/api/v1/registrations", "/api/v1/menus", "/api/v1/login-logs",
			"/api/v1/operation-logs", "/api/v1/security", "/api/v1/dict-entries",
			"/api/v1/user-groups", "/api/v1/plugins", "/api/v1/auth/logout", "/api/v1/auth/me",
			"/api/v1/auth/password", "/api/v1/auth/ws-ticket", "/api/v1/overview",
		},
	},
	{
		plugin: "k8s",
		prefixes: []string{
			"/api/v1/clusters", "/api/v1/pods", "/api/v1/namespaces", "/api/v1/nodes",
			"/api/v1/deployments", "/api/v1/statefulsets", "/api/v1/daemonsets", "/api/v1/cronjobs",
			"/api/v1/jobs", "/api/v1/configmaps", "/api/v1/secrets", "/api/v1/services",
			"/api/v1/ingresses", "/api/v1/ingress-classes", "/api/v1/events", "/api/v1/persistentvolumes",
			"/api/v1/persistentvolumeclaims", "/api/v1/storageclasses", "/api/v1/crds", "/api/v1/crs",
			"/api/v1/rbac", "/api/v1/serviceaccounts", "/api/v1/k8s-policies",
			"/api/v1/k8s-namespace-deny-rules", "/api/v1/k8s-namespace-allow-rules",
			"/api/v1/network-policies", "/api/v1/k8s/", "/api/v1/horizontal-pod-autoscalers",
			"/api/v1/component-status", "/api/v1/k8s-event-forward",
		},
	},
	{plugin: "alert", prefixes: []string{"/api/v1/alerts"}},
	{plugin: "project", prefixes: []string{"/api/v1/projects"}},
	{plugin: "cmdb", prefixes: []string{"/api/v1/servers", "/api/v1/cloud-accounts", "/api/v1/server-groups"}},
	{plugin: "backup", prefixes: []string{"/api/v1/mysql-backup"}},
}

func normalizeUIPath(path string) string {
	p := strings.TrimSpace(strings.ToLower(path))
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = strings.TrimRight(p, "/")
	if p == "" {
		return "/"
	}
	return p
}

func pathMatchesPrefix(path, prefix string) bool {
	prefix = normalizeUIPath(prefix)
	if path == prefix {
		return true
	}
	return strings.HasPrefix(path, prefix+"/") || strings.HasPrefix(path, prefix)
}

func isPluginEnabled(cfg *config.PluginsConfig, name string) bool {
	enabled := ResolveEnabled(cfg)
	return enabled[strings.ToLower(strings.TrimSpace(name))]
}
