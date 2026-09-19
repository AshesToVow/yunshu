package routedeps

import (
	"yunshu/internal/handler"
)

type InspectRouteDeps interface {
	RouteMiddleware
	ProjectAccessDeps
	InspectHandler() *handler.InspectHandler
}
