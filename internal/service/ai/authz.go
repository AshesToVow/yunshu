package ai

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/service/k8s"

	"gorm.io/gorm"
)

// assertProjectMember 校验当前用户对 projectID 的成员资格（超级管理员放行）。
func (s *Service) assertProjectMember(ctx context.Context, actor *auth.CurrentUser, projectID uint) error {
	if projectID == 0 {
		return constants.ErrBadRequestWithMsg("project_id 必填")
	}
	if actor == nil {
		if u, ok := auth.RequestUserFromContext(ctx); ok {
			actor = u
		}
	}
	if actor == nil || actor.ID == 0 {
		return constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	if s.memberRepo == nil {
		return constants.ErrInternal
	}
	_, err := s.memberRepo.GetByProjectAndUser(ctx, projectID, actor.ID)
	if err == gorm.ErrRecordNotFound {
		return constants.ErrForbiddenWithMsg("无权访问该项目")
	}
	if err != nil {
		return err
	}
	return nil
}

// resolveActor 优先用传入 actor，其次从 RequestContext 取完整用户；禁止仅用 ID 的残缺主体。
func resolveActor(ctx context.Context, actor *auth.CurrentUser) *auth.CurrentUser {
	if actor != nil && actor.ID > 0 {
		return actor
	}
	if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil && u.ID > 0 {
		return u
	}
	return nil
}

// canReviewApprovals 超级管理员或具备审批查看全局的运维角色。
func canReviewApprovals(actor *auth.CurrentUser) bool {
	if actor == nil {
		return false
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return true
	}
	for _, c := range actor.RoleCodes {
		switch c {
		case "admin", "ops-admin", "ai-approver":
			return true
		}
	}
	return false
}

// withActorContext 将主体写入 ctx，供 K8s 服务层做 NS deny/allow 过滤。
func withActorContext(ctx context.Context, actor *auth.CurrentUser) context.Context {
	if actor == nil || actor.ID == 0 {
		return ctx
	}
	return auth.WithRequestUser(ctx, actor)
}

// assertK8sClusterAccess 与 HTTP K8sScopeAuthorize 对齐：集群档位 + 命名空间策略。
// minRank 使用 k8s.K8sAccessRank*；超级管理员跳过档位，但仍受 NS 策略约束（与中间件一致时由服务层过滤）。
func (s *Service) assertK8sClusterAccess(
	ctx context.Context,
	actor *auth.CurrentUser,
	clusterID uint,
	namespace string,
	minRank int,
) error {
	if actor == nil || actor.ID == 0 {
		return constants.ErrUnauthorized
	}
	if clusterID == 0 {
		return constants.ErrBadRequestWithMsg("cluster_id 必填")
	}
	if minRank <= 0 {
		minRank = k8s.K8sAccessRankReadonly
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return s.assertNamespacePolicy(ctx, actor, clusterID, namespace)
	}
	if s.accessRepo == nil {
		return constants.ErrInternal
	}
	pack := k8sauth.PackFromCurrentUser(actor)
	rank := s.accessRepo.EffectiveTier(ctx, pack, clusterID)
	if rank < minRank {
		return constants.ErrForbiddenWithMsg("当前主体对该集群权限不足（需集群授权档位）")
	}
	return s.assertNamespacePolicy(ctx, actor, clusterID, namespace)
}

func (s *Service) assertNamespacePolicy(ctx context.Context, actor *auth.CurrentUser, clusterID uint, namespace string) error {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return nil
	}
	pack := k8sauth.PackFromCurrentUser(actor)
	ok, err := k8s.NamespaceAllowedByPolicy(ctx, s.nsDenyRepo, s.nsAllowRepo, pack, clusterID, ns)
	if err != nil {
		return err
	}
	if !ok {
		return constants.ErrForbiddenWithMsg("当前主体无权访问该命名空间")
	}
	return nil
}

func scriptToolRequiresApproval(riskLevel, permission string) bool {
	r := strings.ToUpper(strings.TrimSpace(riskLevel))
	p := strings.ToUpper(strings.TrimSpace(permission))
	if p == "WRITE" || p == "ADMIN" {
		return true
	}
	switch r {
	case "HIGH", "CRITICAL", "WRITE":
		return true
	default:
		return false
	}
}

func errScriptNeedsApproval(name string) error {
	return fmt.Errorf("脚本工具 %s 为高危/写操作，禁止直接执行，请走审批或降低风险等级", name)
}
