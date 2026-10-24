package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

// CMDBRouteDeps CMDB 插件路由依赖。

func (d *RouteDeps) CMDBHandler() *handler.CMDBHandler {
	if d == nil {
		return nil
	}
	return d.cmdbHandler
}

var _ CMDBRouteDeps = (*RouteDeps)(nil)

var _ routedeps.CMDBRouteDeps = (*RouteDeps)(nil)
