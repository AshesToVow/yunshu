package routedeps

import (
	"yunshu/internal/handler"
)

type PlatformTemplateRouteDeps interface {
	RouteMiddleware
	PlatformTemplateHandler() *handler.PlatformTemplateHandler
}
