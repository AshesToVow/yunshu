package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

// AIRouteDeps AI 插件路由所需的最小依赖（中间件 + AIHandler）。

func (d *RouteDeps) AIHandler() *handler.AIHandler {
	if d == nil {
		return nil
	}
	return d.aiHandler
}

var _ AIRouteDeps = (*RouteDeps)(nil)

var _ routedeps.AIRouteDeps = (*RouteDeps)(nil)
