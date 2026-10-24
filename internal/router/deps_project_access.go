package router

import (
	"yunshu/internal/interfaces"
	"yunshu/internal/routedeps"
	logx "yunshu/internal/pkg/logger"
)

// ProjectAccessDeps 项目作用域路由所需的成员校验依赖。

func (d *RouteDeps) ProjectMemberRepo() interfaces.ProjectMemberRepository {
	if d == nil {
		return nil
	}
	return d.projectMemberRepo
}

func (d *RouteDeps) ProjectRepo() interfaces.ProjectRepository {
	if d == nil {
		return nil
	}
	return d.projectRepo
}

func (d *RouteDeps) AppLogger() *logx.Logger {
	if d == nil || d.app == nil {
		return nil
	}
	return d.app.Logger
}

var _ routedeps.ProjectAccessDeps = (*RouteDeps)(nil)
