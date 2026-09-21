package router

import (
	"github.com/gin-gonic/gin"
	"yunshu/internal/routedeps"
)

// RouteMiddleware 路由注册共用的鉴权/审计中间件（窄接口，避免插件依赖整个 RouteDeps）。

func (d *RouteDeps) AuthMiddleware() gin.HandlerFunc {
	if d == nil {
		return nil
	}
	return d.authMiddleware
}

func (d *RouteDeps) WSAuthMiddleware() gin.HandlerFunc {
	if d == nil {
		return nil
	}
	return d.wsAuthMiddleware
}

func (d *RouteDeps) Authorize() gin.HandlerFunc {
	if d == nil {
		return nil
	}
	return d.authorize
}

func (d *RouteDeps) K8sScopeAuthorize() gin.HandlerFunc {
	if d == nil {
		return nil
	}
	return d.k8sScopeAuthorize
}

func (d *RouteDeps) OpAudit() gin.HandlerFunc {
	if d == nil {
		return nil
	}
	return d.opAudit
}

var _ routedeps.RouteMiddleware = (*RouteDeps)(nil)
