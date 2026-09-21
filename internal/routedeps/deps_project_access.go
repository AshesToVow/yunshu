package routedeps

import (
	"yunshu/internal/interfaces"
	logx "yunshu/internal/pkg/logger"
)

type ProjectAccessDeps interface {
	ProjectMemberRepo() interfaces.ProjectMemberRepository
	ProjectRepo() interfaces.ProjectRepository
	AppLogger() *logx.Logger
}
