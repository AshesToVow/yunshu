package routedeps

import (
	"github.com/gin-gonic/gin"
)

type RouteMiddleware interface {
	AuthMiddleware() gin.HandlerFunc
	WSAuthMiddleware() gin.HandlerFunc
	Authorize() gin.HandlerFunc
	K8sScopeAuthorize() gin.HandlerFunc
	OpAudit() gin.HandlerFunc
}
