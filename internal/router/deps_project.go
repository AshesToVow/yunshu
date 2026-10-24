package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

// ProjectRouteDeps 插件/模块路由窄依赖。

func (d *RouteDeps) ProjectHandler() *handler.ProjectHandler {
	if d == nil {
		return nil
	}
	return d.projectHandler
}

func (d *RouteDeps) ProjectCatalogHandler() *handler.ProjectCatalogHandler {
	if d == nil {
		return nil
	}
	return d.projectCatalogHandler
}

func (d *RouteDeps) ClusterLogHandler() *handler.ClusterLogHandler {
	if d == nil {
		return nil
	}
	return d.clusterLogHandler
}

var _ ProjectRouteDeps = (*RouteDeps)(nil)

var _ routedeps.ProjectRouteDeps = (*RouteDeps)(nil)
