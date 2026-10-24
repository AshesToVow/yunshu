package routedeps

import (
	"yunshu/internal/handler"
)

type ProjectRouteDeps interface {
	RouteMiddleware
	ProjectAccessDeps
	ProjectHandler() *handler.ProjectHandler
	ProjectCatalogHandler() *handler.ProjectCatalogHandler
	LogPlatformHandler() *handler.LogPlatformHandler
	LoggieHandler() *handler.LoggieHandler
	ClusterLogHandler() *handler.ClusterLogHandler
}
