package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

// CicdRouteDeps CI/CD 插件路由依赖。

func (d *RouteDeps) CicdHandler() *handler.CicdHandler {
	if d == nil {
		return nil
	}
	return d.cicdHandler
}

var _ CicdRouteDeps = (*RouteDeps)(nil)

var _ routedeps.CicdRouteDeps = (*RouteDeps)(nil)
