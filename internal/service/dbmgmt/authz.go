package dbmgmt

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/projectaccess"

	"gorm.io/gorm"
)

func (s *Service) projectMemberRole(ctx context.Context, projectID uint, actor *auth.CurrentUser) (role string, super bool, err error) {
	if actor == nil || actorUserID(actor) == 0 {
		return "", false, constants.ErrForbidden
	}
	if isSuperAdmin(actor) {
		return "owner", true, nil
	}
	if s.memberRepo == nil {
		return "", false, constants.ErrForbidden
	}
	m, err := s.memberRepo.GetByProjectAndUser(ctx, projectID, actorUserID(actor))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", false, constants.ErrForbidden
		}
		return "", false, err
	}
	return m.Role, false, nil
}

func (s *Service) isProjectAdminOrOwner(ctx context.Context, projectID uint, actor *auth.CurrentUser) (bool, error) {
	role, super, err := s.projectMemberRole(ctx, projectID, actor)
	if err != nil {
		return false, err
	}
	if super {
		return true, nil
	}
	return projectaccess.RoleAtLeast(role, "admin"), nil
}

func (s *Service) requireProjectAdminOrOwner(ctx context.Context, projectID uint, actor *auth.CurrentUser) error {
	ok, err := s.isProjectAdminOrOwner(ctx, projectID, actor)
	if err != nil {
		return err
	}
	if !ok {
		return constants.ErrProjectAdminRequired
	}
	return nil
}

func (s *Service) requireInstanceManage(ctx context.Context, projectID, instanceID uint, actor *auth.CurrentUser) error {
	if isSuperAdmin(actor) {
		return nil
	}
	if ok, _ := s.isProjectAdminOrOwner(ctx, projectID, actor); ok {
		return nil
	}
	perm, err := s.GetEffectivePermission(ctx, projectID, instanceID, actor)
	if err != nil {
		return err
	}
	if perm.CanManage {
		return nil
	}
	return constants.ErrForbidden
}

func grantMatchesDatabase(g model.DbAccessGrant, database string) bool {
	db := strings.TrimSpace(g.DatabaseName)
	target := strings.TrimSpace(database)
	// 空库名 = 整实例授权，匹配任意目标库（含未选库的实例级 DDL）。
	if db == "" {
		return true
	}
	// 具名库授权不能覆盖「未指定库」的操作。
	if target == "" {
		return false
	}
	return strings.EqualFold(db, target)
}

func (s *Service) mergeScopedGrant(dst *EffectivePermission, g model.DbAccessGrant) {
	// 空库名历史写授权不再赋予 DML/DDL/导入/管理，避免整实例写 blast radius。
	if strings.TrimSpace(g.DatabaseName) == "" {
		sanitized := g
		sanitized.CanDML = false
		sanitized.CanDDL = false
		sanitized.CanImport = false
		sanitized.CanManage = false
		s.mergeGrant(dst, sanitized)
		return
	}
	s.mergeGrant(dst, g)
}

func (s *Service) effectivePermissionForDatabase(ctx context.Context, projectID, instanceID uint, database string, actor *auth.CurrentUser) (*EffectivePermission, error) {
	base, err := s.GetEffectivePermission(ctx, projectID, instanceID, actor)
	if err != nil {
		return nil, err
	}
	if base.CanManage {
		return base, nil
	}
	target := strings.TrimSpace(database)
	scoped := &EffectivePermission{}
	hasScoped := false
	now := timeNow()
	for _, k := range principalRefs(actor) {
		grants, err := s.repo.ListGrantsForPrincipal(ctx, projectID, instanceID, k.kind, k.ref)
		if err != nil {
			return nil, err
		}
		for _, g := range grants {
			if !grantStillValid(g, now) {
				continue
			}
			if !grantMatchesDatabase(g, target) {
				continue
			}
			hasScoped = true
			s.mergeScopedGrant(scoped, g)
		}
	}
	if !hasScoped {
		// 未命中目标库的授权时，不回退到「其它库上的权限并集」，避免跨库串权。
		return scoped, nil
	}
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	if inst.ReadOnly {
		scoped.CanDML, scoped.CanDDL, scoped.CanImport = false, false, false
	}
	return scoped, nil
}

func (s *Service) canViewTicketSQL(ctx context.Context, projectID uint, ticket *model.DbSqlTicket, actor *auth.CurrentUser) bool {
	if actor == nil || ticket == nil {
		return false
	}
	if isSuperAdmin(actor) {
		return true
	}
	if ticket.SubmitterUserID == actorUserID(actor) {
		return true
	}
	if ok, _ := s.isProjectAdminOrOwner(ctx, projectID, actor); ok {
		return true
	}
	perm, err := s.GetEffectivePermission(ctx, projectID, ticket.InstanceID, actor)
	if err == nil && perm != nil && perm.CanManage {
		return true
	}
	return false
}

// ticketSQLVisibility 批量判定工单 SQL 可见性（口径同 canViewTicketSQL）：
// 项目角色只查一次，实例 Manage 权限按实例 ID 缓存，避免列表逐条查询。
func (s *Service) ticketSQLVisibility(ctx context.Context, projectID uint, actor *auth.CurrentUser, tickets []model.DbSqlTicket) []bool {
	out := make([]bool, len(tickets))
	if actor == nil {
		return out
	}
	super := auth.IsSuperAdminRole(actor.RoleCodes)
	projectAdmin := false
	if !super {
		projectAdmin, _ = s.isProjectAdminOrOwner(ctx, projectID, actor)
	}
	uid := actorUserID(actor)
	manageCache := make(map[uint]bool, 4)
	for i := range tickets {
		t := &tickets[i]
		if super || t.SubmitterUserID == uid || projectAdmin {
			out[i] = true
			continue
		}
		canManage, ok := manageCache[t.InstanceID]
		if !ok {
			perm, err := s.GetEffectivePermission(ctx, projectID, t.InstanceID, actor)
			canManage = err == nil && perm != nil && perm.CanManage
			manageCache[t.InstanceID] = canManage
		}
		out[i] = canManage
	}
	return out
}

func timeNow() time.Time {
	return timeNowFunc()
}

var timeNowFunc = func() time.Time {
	return time.Now()
}
