package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

// PlatformTemplateRouteDeps 插件/模块路由窄依赖。

func (d *RouteDeps) PlatformTemplateHandler() *handler.PlatformTemplateHandler {
	if d == nil {
		return nil
	}
	return d.platformTplHandler
}

var _ PlatformTemplateRouteDeps = (*RouteDeps)(nil)

var _ routedeps.PlatformTemplateRouteDeps = (*RouteDeps)(nil)
