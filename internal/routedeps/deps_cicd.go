package routedeps

import (
	"yunshu/internal/handler"
)

type CicdRouteDeps interface {
	RouteMiddleware
	ProjectAccessDeps
	CicdHandler() *handler.CicdHandler
}
