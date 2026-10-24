package routedeps

import (
	"yunshu/internal/bootstrap"
	"yunshu/internal/handler"
)

type WorkflowRouteDeps interface {
	RouteMiddleware
	App() *bootstrap.App
	WorkflowHandler() *handler.WorkflowHandler
}
