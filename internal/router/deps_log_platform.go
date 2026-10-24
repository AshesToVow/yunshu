package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

// LogPlatformRouteDeps 插件/模块路由窄依赖。

func (d *RouteDeps) LogPlatformHandler() *handler.LogPlatformHandler {
	if d == nil {
		return nil
	}
	return d.logPlatformHandler
}

func (d *RouteDeps) LoggieHandler() *handler.LoggieHandler {
	if d == nil {
		return nil
	}
	return d.loggieHandler
}

var _ LogPlatformRouteDeps = (*RouteDeps)(nil)

var _ routedeps.LogPlatformRouteDeps = (*RouteDeps)(nil)
