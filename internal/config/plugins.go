package config

// PluginsConfig 业务插件启停（GVA 风格 modules.enabled）。
type PluginsConfig struct {
	// Enabled 启用的插件名列表；空则使用默认全集：core, k8s, alert, project, backup。
	Enabled []string `mapstructure:"enabled"`
}
