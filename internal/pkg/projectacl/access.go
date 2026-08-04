package projectacl

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/projectaccess"

	"gorm.io/gorm"
)

// FullAccess 项目 owner/admin 或全局超管：对项目内资源隐式全量权限。
func FullAccess(ctx context.Context, memberRepo interfaces.ProjectMemberRepository, projectID uint, actor *auth.CurrentUser) (bool, error) {
	if actor == nil {
		return false, nil
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return true, nil
	}
	if memberRepo == nil || projectID == 0 {
		return false, nil
	}
	m, err := memberRepo.GetByProjectAndUser(ctx, projectID, actor.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return projectaccess.RoleAtLeast(m.Role, "admin"), nil
}

// CanManageGrants 仅 owner/admin/超管可改资源授权。
func CanManageGrants(ctx context.Context, memberRepo interfaces.ProjectMemberRepository, projectID uint, actor *auth.CurrentUser) (bool, error) {
	return FullAccess(ctx, memberRepo, projectID, actor)
}

// UserPrincipalRef 一期仅支持 user 主体。
func UserPrincipalRef(userID uint) (kind, ref string) {
	return "user", strconv.FormatUint(uint64(userID), 10)
}

func ParseUserRef(ref string) (uint, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(ref, 10, 32)
	if err != nil || v == 0 {
		return 0, false
	}
	return uint(v), true
}
