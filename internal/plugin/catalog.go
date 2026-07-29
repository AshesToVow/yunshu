package plugin

import (
	"strings"

	"yunshu/internal/config"
)

// Info 插件元数据（供管理端展示）。
type Info struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
	Manifest    Manifest `json:"manifest"`
}

// Catalog 返回全部已注册插件及启停状态与契约。
func Catalog(cfg *config.PluginsConfig) []Info {
	enabled := ResolveEnabled(cfg)
	out := make([]Info, 0, len(registry))
	for _, m := range All() {
		name := m.Name()
		out = append(out, Info{
			Name:        name,
			Description: m.Description(),
			Enabled:     enabled[strings.ToLower(name)],
			Manifest:    ResolveManifest(m),
		})
	}
	return out
}

// EnabledNames 返回当前配置下已启用的插件名（保持注册顺序）。
func EnabledNames(cfg *config.PluginsConfig) []string {
	enabled := ResolveEnabled(cfg)
	var names []string
	for _, m := range registry {
		n := m.Name()
		if enabled[strings.ToLower(n)] {
			names = append(names, n)
		}
	}
	return names
}
