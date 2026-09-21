package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

// EsmgmtRouteDeps ES 管理插件路由依赖。

func (d *RouteDeps) EsmgmtHandler() *handler.EsmgmtHandler {
	if d == nil {
		return nil
	}
	return d.esmgmtHandler
}

var _ EsmgmtRouteDeps = (*RouteDeps)(nil)

var _ routedeps.EsmgmtRouteDeps = (*RouteDeps)(nil)
