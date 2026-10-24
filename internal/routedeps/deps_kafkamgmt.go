package routedeps

import (
	"yunshu/internal/handler"
)

type KafkamgmtRouteDeps interface {
	RouteMiddleware
	KafkamgmtHandler() *handler.KafkamgmtHandler
}
