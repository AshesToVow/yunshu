package plugin

// Manifest 插件契约：菜单/API 归属与依赖（单一事实源，供前后端与 plugingate 消费）。
type Manifest struct {
	// MenuPathPrefixes 控制台菜单 path 前缀
	MenuPathPrefixes []string `json:"menu_path_prefixes"`
	// APIPrefixes Casbin resource / API 路径前缀
	APIPrefixes []string `json:"api_prefixes"`
	// DependsOn 启用本插件时要求同时启用的其它插件（如 cicd→project）
	DependsOn []string `json:"depends_on"`
	// Workers 后台任务说明（展示用）
	Workers []string `json:"workers,omitempty"`
}

// ManifestProvider 可选：插件声明契约。未实现时 Manifest 为空。
type ManifestProvider interface {
	Manifest() Manifest
}

// ResolveManifest 读取模块契约。
func ResolveManifest(m Module) Manifest {
	if mp, ok := m.(ManifestProvider); ok {
		return mp.Manifest()
	}
	return Manifest{}
}
