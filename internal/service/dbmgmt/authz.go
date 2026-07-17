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
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return "owner", true, nil
	}
	var m model.ProjectMember
	err = s.db.WithContext(ctx).Where("project_id = ? AND user_id = ?", projectID, actorUserID(actor)).First(&m).Error
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
	if auth.IsSuperAdminRole(actor.RoleCodes) {
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
	if db == "" {
		return true
	}
	if target == "" {
		return true
	}
	return strings.EqualFold(db, target)
}

func (s *Service) mergeScopedGrant(dst *EffectivePermission, g model.DbAccessGrant) {
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
	if target == "" {
		return base, nil
	}
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
		return base, nil
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
	if auth.IsSuperAdminRole(actor.RoleCodes) {
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

func timeNow() time.Time {
	return timeNowFunc()
}

var timeNowFunc = func() time.Time {
	return time.Now()
}
