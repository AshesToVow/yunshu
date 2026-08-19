package plugingate

import (
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/model"
	"yunshu/internal/plugin"
)

// ResolveMenuPathPlugin 解析控制台菜单 path 所属插件；未命中返回空（不受插件开关约束）。
func ResolveMenuPathPlugin(path string) string {
	p := normalizeUIPath(path)
	if p == "" || p == "/" {
		return "k8s"
	}
	for _, m := range plugin.All() {
		mf := plugin.ResolveManifest(m)
		for _, prefix := range mf.MenuPathPrefixes {
			if pathMatchesPrefix(p, prefix) {
				return m.Name()
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
	if pluginName := resolveCicdAPIResource(r); pluginName != "" {
		return pluginName
	}
	if pluginName := resolveDbmgmtAPIResource(r); pluginName != "" {
		return pluginName
	}
	if pluginName := resolveBackupAPIResource(r); pluginName != "" {
		return pluginName
	}
	if pluginName := resolveInspectAPIResource(r); pluginName != "" {
		return pluginName
	}
	if pluginName := resolveCmdbAPIResource(r); pluginName != "" {
		return pluginName
	}
	for _, m := range plugin.All() {
		mf := plugin.ResolveManifest(m)
		for _, prefix := range mf.APIPrefixes {
			prefix = strings.TrimSpace(strings.ToLower(prefix))
			if prefix == "" {
				continue
			}
			if r == prefix || strings.HasPrefix(r, strings.TrimRight(prefix, "/")+"/") {
				return m.Name()
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
	return isPluginAndDepsEnabled(cfg, pluginName)
}

// IsAPIResourceAllowed permission resource 是否对当前启用的插件可见。
func IsAPIResourceAllowed(resource string, cfg *config.PluginsConfig) bool {
	pluginName := ResolveAPIResourcePlugin(resource)
	if pluginName == "" {
		return true
	}
	return isPluginAndDepsEnabled(cfg, pluginName)
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
				if len(child.Children) > 0 {
					out = append(out, child.Children...)
				}
				continue
			}
			if len(it.Children) > 0 && len(child.Children) == 0 {
				continue
			}
			out = append(out, child)
		}
		return out
	}
	return walk(items)
}

func isPluginAndDepsEnabled(cfg *config.PluginsConfig, name string) bool {
	if !isPluginEnabled(cfg, name) {
		return false
	}
	for _, m := range plugin.All() {
		if !strings.EqualFold(m.Name(), name) {
			continue
		}
		mf := plugin.ResolveManifest(m)
		for _, dep := range mf.DependsOn {
			if !isPluginEnabled(cfg, dep) {
				return false
			}
		}
		return true
	}
	return true
}

func resolveCicdAPIResource(resource string) string {
	cicdOverview := []string{
		"/api/v1/overview/project-launches",
		"/api/v1/overview/release-by-person",
		"/api/v1/cicd/jenkins/callback",
		"/api/v1/registries",
		"/api/v1/pipeline-templates",
	}
	for _, p := range cicdOverview {
		if resource == p || strings.HasPrefix(resource, p+"/") {
			return "cicd"
		}
	}
	if strings.Contains(resource, "/projects/") && strings.Contains(resource, "/cicd") {
		return "cicd"
	}
	if strings.Contains(resource, "/projects/") && strings.Contains(resource, "/registry-binding") {
		return "cicd"
	}
	return ""
}

func resolveDbmgmtAPIResource(resource string) string {
	if strings.Contains(resource, "/projects/") && strings.Contains(resource, "/dbmgmt") {
		return "dbmgmt"
	}
	return ""
}

func resolveBackupAPIResource(resource string) string {
	if strings.Contains(resource, "/projects/") && strings.Contains(resource, "/mysql-backup") {
		return "backup"
	}
	return ""
}

func resolveInspectAPIResource(resource string) string {
	if strings.Contains(resource, "/projects/") && strings.Contains(resource, "/inspect") {
		return "inspect"
	}
	return ""
}

func resolveCmdbAPIResource(resource string) string {
	if strings.Contains(resource, "/cloud-accounts") || strings.Contains(resource, "/server-groups") {
		return "cmdb"
	}
	if strings.Contains(resource, "/projects/") &&
		(strings.Contains(resource, "/servers") ||
			strings.Contains(resource, "/server-access") ||
			strings.Contains(resource, "/ssh")) {
		return "cmdb"
	}
	return ""
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
	if prefix == "/" {
		return path == "/"
	}
	// "/alert-" 匹配 "/alert-channels"（连字符前缀，非目录形式）
	if strings.HasSuffix(prefix, "-") {
		return strings.HasPrefix(path, prefix)
	}
	return strings.HasPrefix(path, prefix+"/")
}

func isPluginEnabled(cfg *config.PluginsConfig, name string) bool {
	enabled := plugin.ResolveEnabled(cfg)
	return enabled[strings.ToLower(strings.TrimSpace(name))]
}
