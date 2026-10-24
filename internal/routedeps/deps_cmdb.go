package routedeps

import (
	"yunshu/internal/handler"
)

type CMDBRouteDeps interface {
	RouteMiddleware
	ProjectAccessDeps
	CMDBHandler() *handler.CMDBHandler
}
