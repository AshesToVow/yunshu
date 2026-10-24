package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

// DbmgmtRouteDeps 数据库管理插件路由依赖。

func (d *RouteDeps) DbmgmtHandler() *handler.DbmgmtHandler {
	if d == nil {
		return nil
	}
	return d.dbmgmtHandler
}

var _ DbmgmtRouteDeps = (*RouteDeps)(nil)

var _ routedeps.DbmgmtRouteDeps = (*RouteDeps)(nil)
