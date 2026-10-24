package routedeps

import (
	"yunshu/internal/handler"
)

type LogPlatformRouteDeps interface {
	RouteMiddleware
	LogPlatformHandler() *handler.LogPlatformHandler
	LoggieHandler() *handler.LoggieHandler
}
