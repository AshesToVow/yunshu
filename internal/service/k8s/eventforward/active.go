package eventforward

// active 由 k8s 插件在 StartWorkers 中安装，供进程退出时 Stop。
var active *Manager

// SetActive 注册当前进程的事件转发管理器。
func SetActive(m *Manager) {
	active = m
}

// Active 返回已安装的管理器（未启用 k8s 插件时为 nil）。
func Active() *Manager {
	return active
}
