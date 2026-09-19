package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

// InspectRouteDeps 巡检插件路由依赖。

func (d *RouteDeps) InspectHandler() *handler.InspectHandler {
	if d == nil {
		return nil
	}
	return d.inspectHandler
}

var _ InspectRouteDeps = (*RouteDeps)(nil)

var _ routedeps.InspectRouteDeps = (*RouteDeps)(nil)
