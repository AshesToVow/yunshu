package cmdb

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/projectacl"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

// ServerAccessPerm 某台机器的有效权限。
type ServerAccessPerm struct {
	CanView   bool `json:"can_view"`
	CanExec   bool `json:"can_exec"`
	CanManage bool `json:"can_manage"`
}

type ServerGrantItem struct {
	model.ServerAccessGrant
	ServerName string `json:"server_name,omitempty"`
	ServerHost string `json:"server_host,omitempty"`
	Username   string `json:"username,omitempty"`
	Nickname   string `json:"nickname,omitempty"`
}

type ServerGrantUpsertRequest struct {
	ProjectID uint   `json:"-"`
	ServerID  uint   `json:"server_id" binding:"required"`
	UserID    uint   `json:"user_id" binding:"required"`
	CanView   *bool  `json:"can_view"`
	CanExec   bool   `json:"can_exec"`
	CanManage bool   `json:"can_manage"`
	Remark    string `json:"remark"`
	CreatedBy *uint  `json:"-"`
}

type ServerGrantBulkRequest struct {
	ProjectID uint   `json:"-"`
	UserID    uint   `json:"user_id" binding:"required"`
	ServerIDs []uint `json:"server_ids" binding:"required"`
	CanView   bool   `json:"can_view"`
	CanExec   bool   `json:"can_exec"`
	CanManage bool   `json:"can_manage"`
	CreatedBy *uint  `json:"-"`
}

type BootstrapServerGrantsRequest struct {
	ProjectID uint  `json:"-"`
	CreatedBy *uint `json:"-"`
}

func (s *Service) AssertCanCreateServer(ctx context.Context, projectID uint, actor *auth.CurrentUser) error {
	full, err := projectacl.FullAccess(ctx, s.memberRepo, projectID, actor)
	if err != nil {
		return err
	}
	if !full {
		return constants.ErrForbidden
	}
	return nil
}

func (s *Service) assertCanManageServerGrants(ctx context.Context, projectID uint, actor *auth.CurrentUser) error {
	ok, err := projectacl.CanManageGrants(ctx, s.memberRepo, projectID, actor)
	if err != nil {
		return err
	}
	if !ok {
		return constants.ErrProjectAdminRequired
	}
	return nil
}

// EffectiveServerAccess 解析当前用户对指定服务器的有效权限。
func (s *Service) EffectiveServerAccess(ctx context.Context, projectID, serverID uint, actor *auth.CurrentUser) (*ServerAccessPerm, error) {
	full, err := projectacl.FullAccess(ctx, s.memberRepo, projectID, actor)
	if err != nil {
		return nil, err
	}
	if full {
		return &ServerAccessPerm{CanView: true, CanExec: true, CanManage: true}, nil
	}
	if actor == nil || actor.ID == 0 {
		return &ServerAccessPerm{}, nil
	}
	kind, ref := projectacl.UserPrincipalRef(actor.ID)
	g, err := s.accessGrantRepo.Get(ctx, projectID, serverID, kind, ref)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &ServerAccessPerm{}, nil
		}
		return nil, err
	}
	return &ServerAccessPerm{CanView: g.CanView, CanExec: g.CanExec, CanManage: g.CanManage}, nil
}

func (s *Service) AssertServerAccess(ctx context.Context, projectID, serverID uint, actor *auth.CurrentUser, need string) error {
	perm, err := s.EffectiveServerAccess(ctx, projectID, serverID, actor)
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(need)) {
	case "manage":
		if perm.CanManage {
			return nil
		}
		return constants.ErrForbiddenWithMsg("当前账号对该服务器无管理权限")
	case "exec":
		if perm.CanExec || perm.CanManage {
			return nil
		}
		if perm.CanView {
			return constants.ErrForbiddenWithMsg("仅有查看权限，不能 SSH/执行；请在「资源授权」中勾选 SSH/执行")
		}
		return constants.ErrForbiddenWithMsg("当前账号对该服务器无 SSH/执行权限")
	default: // view
		if perm.CanView || perm.CanExec || perm.CanManage {
			return nil
		}
		return constants.ErrForbiddenWithMsg("当前账号对该服务器无查看权限")
	}
}

// visibleServerScope 返回 (unrestricted, serverIDs)。unrestricted 时不过滤。
func (s *Service) visibleServerScope(ctx context.Context, projectID uint, actor *auth.CurrentUser) (bool, []uint, error) {
	full, err := projectacl.FullAccess(ctx, s.memberRepo, projectID, actor)
	if err != nil {
		return false, nil, err
	}
	if full {
		return true, nil, nil
	}
	if actor == nil || actor.ID == 0 {
		return false, []uint{}, nil
	}
	kind, ref := projectacl.UserPrincipalRef(actor.ID)
	ids, err := s.accessGrantRepo.ListVisibleServerIDs(ctx, projectID, kind, ref)
	if err != nil {
		return false, nil, err
	}
	return false, ids, nil
}

func (s *Service) ListServerGrants(ctx context.Context, projectID uint, actor *auth.CurrentUser, userID, serverID uint) ([]ServerGrantItem, error) {
	if err := s.assertCanManageServerGrants(ctx, projectID, actor); err != nil {
		return nil, err
	}
	params := repository.ServerAccessGrantListParams{ProjectID: projectID, ServerID: serverID}
	if userID > 0 {
		kind, ref := projectacl.UserPrincipalRef(userID)
		params.PrincipalKind, params.PrincipalRef = kind, ref
	}
	rows, err := s.accessGrantRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]ServerGrantItem, 0, len(rows))
	for _, r := range rows {
		item := ServerGrantItem{ServerAccessGrant: r}
		if sv, err := s.serverRepo.GetByID(ctx, r.ServerID); err == nil && sv != nil {
			item.ServerName, item.ServerHost = sv.Name, sv.Host
		}
		if uid, ok := projectacl.ParseUserRef(r.PrincipalRef); ok {
			if u, err := s.userRepo.GetByID(ctx, uid); err == nil && u != nil {
				item.Username, item.Nickname = u.Username, u.Nickname
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) UpsertServerGrant(ctx context.Context, req ServerGrantUpsertRequest, actor *auth.CurrentUser) (*model.ServerAccessGrant, error) {
	if err := s.assertCanManageServerGrants(ctx, req.ProjectID, actor); err != nil {
		return nil, err
	}
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrServerNotFound
		}
		return nil, err
	}
	if sv.ProjectID != req.ProjectID {
		return nil, constants.ErrBadRequestWithMsg("服务器不属于当前项目")
	}
	canView := true
	if req.CanView != nil {
		canView = *req.CanView
	}
	canExec, canManage := req.CanExec, req.CanManage
	if canManage {
		canView, canExec = true, true
	} else if canExec {
		canView = true
	}
	kind, ref := projectacl.UserPrincipalRef(req.UserID)
	row := model.ServerAccessGrant{
		ProjectID:     req.ProjectID,
		ServerID:      req.ServerID,
		PrincipalKind: kind,
		PrincipalRef:  ref,
		CanView:       canView,
		CanExec:       canExec,
		CanManage:     canManage,
		Remark:        strings.TrimSpace(req.Remark),
		CreatedBy:     req.CreatedBy,
	}
	if err := s.accessGrantRepo.Upsert(ctx, &row); err != nil {
		return nil, err
	}
	if g, err := s.accessGrantRepo.Get(ctx, req.ProjectID, req.ServerID, kind, ref); err == nil && g != nil {
		return g, nil
	}
	return &row, nil
}

func (s *Service) BulkUpsertServerGrants(ctx context.Context, req ServerGrantBulkRequest, actor *auth.CurrentUser) (int, error) {
	if err := s.assertCanManageServerGrants(ctx, req.ProjectID, actor); err != nil {
		return 0, err
	}
	if len(req.ServerIDs) == 0 {
		return 0, constants.ErrBadRequestWithMsg("server_ids required")
	}
	canView, canExec, canManage := req.CanView, req.CanExec, req.CanManage
	if canManage {
		canView, canExec = true, true
	} else if canExec {
		canView = true
	} else if !canView {
		canView = true
	}
	kind, ref := projectacl.UserPrincipalRef(req.UserID)
	n := 0
	for _, sid := range req.ServerIDs {
		sv, err := s.serverRepo.GetByID(ctx, sid)
		if err != nil || sv.ProjectID != req.ProjectID {
			continue
		}
		row := model.ServerAccessGrant{
			ProjectID: req.ProjectID, ServerID: sid, PrincipalKind: kind, PrincipalRef: ref,
			CanView: canView, CanExec: canExec, CanManage: canManage, CreatedBy: req.CreatedBy,
		}
		if err := s.accessGrantRepo.Upsert(ctx, &row); err == nil {
			n++
		}
	}
	return n, nil
}

func (s *Service) DeleteServerGrant(ctx context.Context, projectID, grantID uint, actor *auth.CurrentUser) error {
	if err := s.assertCanManageServerGrants(ctx, projectID, actor); err != nil {
		return err
	}
	n, err := s.accessGrantRepo.Delete(ctx, projectID, grantID)
	if err != nil {
		return err
	}
	if n == 0 {
		return constants.ErrNotFound
	}
	return nil
}

// BootstrapServerGrantsForMembers 给项目内全部非 admin 成员授予当前全部服务器 view+exec（迁移用）。
func (s *Service) BootstrapServerGrantsForMembers(ctx context.Context, req BootstrapServerGrantsRequest, actor *auth.CurrentUser) (map[string]int, error) {
	if err := s.assertCanManageServerGrants(ctx, req.ProjectID, actor); err != nil {
		return nil, err
	}
	members, err := s.memberRepo.ListByProject(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	servers, err := s.serverRepo.ListByProject(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	granted, skipped := 0, 0
	for _, m := range members {
		if projectaclRoleIsFull(m.Role) {
			skipped++
			continue
		}
		kind, ref := projectacl.UserPrincipalRef(m.UserID)
		for _, sv := range servers {
			row := model.ServerAccessGrant{
				ProjectID: req.ProjectID, ServerID: sv.ID, PrincipalKind: kind, PrincipalRef: ref,
				CanView: true, CanExec: true, CanManage: false, CreatedBy: req.CreatedBy,
				Remark: "bootstrap",
			}
			if err := s.accessGrantRepo.Upsert(ctx, &row); err == nil {
				granted++
			}
		}
	}
	return map[string]int{"grants_upserted": granted, "admin_members_skipped": skipped, "servers": len(servers)}, nil
}

func projectaclRoleIsFull(role string) bool {
	r := strings.ToLower(strings.TrimSpace(role))
	return r == "owner" || r == "admin"
}
