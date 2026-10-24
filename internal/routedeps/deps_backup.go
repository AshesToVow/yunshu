package routedeps

import (
	"yunshu/internal/handler"
)

type BackupRouteDeps interface {
	RouteMiddleware
	ProjectAccessDeps
	MysqlBackupHandler() *handler.MysqlBackupHandler
}
