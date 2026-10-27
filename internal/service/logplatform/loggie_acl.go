package logplatform

import (
	"context"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/projectaccess"
)

// CanViewAgentCredentials 判断当前用户能否下载含凭据的 Agent 文件（env Token、sink 密码）。
// 只读成员仅可查看/下载不含凭据的配置；无用户上下文（后台任务）放行。
func (s *LoggieAgentService) CanViewAgentCredentials(ctx context.Context, projectID uint) bool {
	actor, _ := auth.RequestUserFromContext(ctx)
	if actor == nil || actor.ID == 0 {
		return true
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return true
	}
	if projectID == 0 || s == nil || s.memberRepo == nil {
		return false
	}
	m, err := s.memberRepo.GetByProjectAndUser(ctx, projectID, actor.ID)
	if err != nil || m == nil {
		return false
	}
	return !projectaccess.IsReadonly(m.Role)
}
