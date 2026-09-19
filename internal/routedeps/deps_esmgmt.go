package routedeps

import (
	"yunshu/internal/handler"
)

type EsmgmtRouteDeps interface {
	RouteMiddleware
	EsmgmtHandler() *handler.EsmgmtHandler
}
