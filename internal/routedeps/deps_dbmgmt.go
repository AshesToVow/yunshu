package routedeps

import (
	"yunshu/internal/handler"
)

type DbmgmtRouteDeps interface {
	RouteMiddleware
	ProjectAccessDeps
	DbmgmtHandler() *handler.DbmgmtHandler
}
