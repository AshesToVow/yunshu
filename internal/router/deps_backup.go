package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

// BackupRouteDeps 插件/模块路由窄依赖。

func (d *RouteDeps) MysqlBackupHandler() *handler.MysqlBackupHandler {
	if d == nil {
		return nil
	}
	return d.mysqlBackupHandler
}

var _ BackupRouteDeps = (*RouteDeps)(nil)

var _ routedeps.BackupRouteDeps = (*RouteDeps)(nil)
