package plugin

import (
	"yunshu/internal/config"
	"yunshu/internal/model"
	"yunshu/internal/plugingate"
)

func ResolveMenuPathPlugin(path string) string {
	return plugingate.ResolveMenuPathPlugin(path)
}

func ResolveAPIResourcePlugin(resource string) string {
	return plugingate.ResolveAPIResourcePlugin(resource)
}

func IsMenuPathAllowed(path string, cfg *config.PluginsConfig) bool {
	return plugingate.IsMenuPathAllowed(path, cfg)
}

func IsAPIResourceAllowed(resource string, cfg *config.PluginsConfig) bool {
	return plugingate.IsAPIResourceAllowed(resource, cfg)
}

func FilterMenusByPlugins(items []model.Menu, cfg *config.PluginsConfig) []model.Menu {
	return plugingate.FilterMenusByPlugins(items, cfg)
}
