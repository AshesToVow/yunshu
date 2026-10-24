package routedeps

import (
	"yunshu/internal/handler"
)

type AIRouteDeps interface {
	RouteMiddleware
	AIHandler() *handler.AIHandler
}
