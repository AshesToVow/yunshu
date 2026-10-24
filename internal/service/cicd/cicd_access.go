package cicd

import (
	"context"
	"errors"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/projectacl"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type CicdAccessPerm struct {
	CanView    bool `json:"can_view"`
	CanBuild   bool `json:"can_build"`
	CanRelease bool `json:"can_release"`
	CanManage  bool `json:"can_manage"`
}

type CicdGrantItem struct {
	model.CicdAccessGrant
	ServiceName string `json:"service_name,omitempty"`
	Username    string `json:"username,omitempty"`
	Nickname    string `json:"nickname,omitempty"`
}

type CicdGrantUpsertRequest struct {
	ProjectID  uint   `json:"-"`
	ServiceID  uint   `json:"service_id" binding:"required"`
	UserID     uint   `json:"user_id" binding:"required"`
	CanView    *bool  `json:"can_view"`
	CanBuild   bool   `json:"can_build"`
	CanRelease bool   `json:"can_release"`
	CanManage  bool   `json:"can_manage"`
	Remark     string `json:"remark"`
	CreatedBy  *uint  `json:"-"`
}

type CicdGrantBulkRequest struct {
	ProjectID  uint   `json:"-"`
	UserID     uint   `json:"user_id" binding:"required"`
	ServiceIDs []uint `json:"service_ids" binding:"required"`
	CanView    bool   `json:"can_view"`
	CanBuild   bool   `json:"can_build"`
	CanRelease bool   `json:"can_release"`
	CanManage  bool   `json:"can_manage"`
	CreatedBy  *uint  `json:"-"`
}

type BootstrapCicdGrantsRequest struct {
	ProjectID uint  `json:"-"`
	CreatedBy *uint `json:"-"`
}

func (s *Service) assertCanManageCicdGrants(ctx context.Context, projectID uint, actor *auth.CurrentUser) error {
	ok, err := projectacl.CanManageGrants(ctx, s.memberRepo, projectID, actor)
	if err != nil {
		return err
	}
	if !ok {
		return constants.ErrProjectAdminRequired
	}
	return nil
}

func (s *Service) EffectiveCicdAccess(ctx context.Context, projectID, serviceID uint, actor *auth.CurrentUser) (*CicdAccessPerm, error) {
	full, err := projectacl.FullAccess(ctx, s.memberRepo, projectID, actor)
	if err != nil {
		return nil, err
	}
	if full {
		return &CicdAccessPerm{CanView: true, CanBuild: true, CanRelease: true, CanManage: true}, nil
	}
	if actor == nil || actor.ID == 0 {
		return &CicdAccessPerm{}, nil
	}
	grants, err := s.repo.ListAccessGrantsForUser(ctx, projectID, actor.ID)
	if err != nil {
		return nil, err
	}
	for _, g := range grants {
		if g.ServiceID == serviceID {
			return &CicdAccessPerm{CanView: g.CanView, CanBuild: g.CanBuild, CanRelease: g.CanRelease, CanManage: g.CanManage}, nil
		}
	}
	return &CicdAccessPerm{}, nil
}

func (s *Service) AssertCicdAccess(ctx context.Context, projectID, serviceID uint, actor *auth.CurrentUser, need string) error {
	perm, err := s.EffectiveCicdAccess(ctx, projectID, serviceID, actor)
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(need)) {
	case "manage":
		if perm.CanManage {
			return nil
		}
	case "release":
		if perm.CanRelease || perm.CanManage {
			return nil
		}
	case "build":
		if perm.CanBuild || perm.CanManage {
			return nil
		}
	default:
		if perm.CanView || perm.CanBuild || perm.CanRelease || perm.CanManage {
			return nil
		}
	}
	return constants.ErrForbidden
}

func (s *Service) AssertCanCreateCicdService(ctx context.Context, projectID uint, actor *auth.CurrentUser) error {
	full, err := projectacl.FullAccess(ctx, s.memberRepo, projectID, actor)
	if err != nil {
		return err
	}
	if !full {
		return constants.ErrForbidden
	}
	return nil
}

func (s *Service) visibleCicdServiceScope(ctx context.Context, projectID uint, actor *auth.CurrentUser) (bool, []uint, error) {
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
	grants, err := s.repo.ListAccessGrantsForUser(ctx, projectID, actor.ID)
	if err != nil {
		return false, nil, err
	}
	ids := make([]uint, 0, len(grants))
	for _, g := range grants {
		ids = append(ids, g.ServiceID)
	}
	return false, ids, nil
}

func (s *Service) ListCicdGrants(ctx context.Context, projectID uint, actor *auth.CurrentUser, userID, serviceID uint) ([]CicdGrantItem, error) {
	if err := s.assertCanManageCicdGrants(ctx, projectID, actor); err != nil {
		return nil, err
	}
	var rows []model.CicdAccessGrant
	var err error
	if userID > 0 {
		rows, err = s.repo.ListAccessGrantsForUser(ctx, projectID, userID)
		if err != nil {
			return nil, err
		}
		if serviceID > 0 {
			filtered := make([]model.CicdAccessGrant, 0, len(rows))
			for _, r := range rows {
				if r.ServiceID == serviceID {
					filtered = append(filtered, r)
				}
			}
			rows = filtered
		}
	} else {
		rows, _, err = s.repo.ListAccessGrants(ctx, repository.CicdAccessGrantListParams{
			ProjectID: projectID,
			ServiceID: serviceID,
		})
		if err != nil {
			return nil, err
		}
	}
	out := make([]CicdGrantItem, 0, len(rows))
	for _, r := range rows {
		item := CicdGrantItem{CicdAccessGrant: r}
		if brief, err := s.repo.GetServiceBrief(ctx, r.ServiceID); err == nil && brief != nil {
			item.ServiceName = brief.Name
		}
		if uid, ok := projectacl.ParseUserRef(r.PrincipalRef); ok {
			users, err := s.repo.ListUsersByIDs(ctx, []uint{uid})
			if err == nil && len(users) > 0 {
				item.Username, item.Nickname = users[0].Username, users[0].Nickname
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) UpsertCicdGrant(ctx context.Context, req CicdGrantUpsertRequest, actor *auth.CurrentUser) (*model.CicdAccessGrant, error) {
	if err := s.assertCanManageCicdGrants(ctx, req.ProjectID, actor); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetService(ctx, req.ProjectID, req.ServiceID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	canView := true
	if req.CanView != nil {
		canView = *req.CanView
	}
	if req.CanManage {
		canView, req.CanBuild, req.CanRelease = true, true, true
	} else if req.CanRelease || req.CanBuild {
		canView = true
	}
	kind, ref := projectacl.UserPrincipalRef(req.UserID)
	row := model.CicdAccessGrant{
		ProjectID: req.ProjectID, ServiceID: req.ServiceID, PrincipalKind: kind, PrincipalRef: ref,
		CanView: canView, CanBuild: req.CanBuild, CanRelease: req.CanRelease, CanManage: req.CanManage,
		Remark: strings.TrimSpace(req.Remark), CreatedBy: req.CreatedBy,
	}
	if err := s.repo.UpsertAccessGrant(ctx, &row); err != nil {
		return nil, err
	}
	return s.findAccessGrant(ctx, req.ProjectID, req.ServiceID, kind, ref)
}

func (s *Service) findAccessGrant(ctx context.Context, projectID, serviceID uint, kind, ref string) (*model.CicdAccessGrant, error) {
	rows, _, err := s.repo.ListAccessGrants(ctx, repository.CicdAccessGrantListParams{
		ProjectID: projectID,
		ServiceID: serviceID,
	})
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].PrincipalKind == kind && rows[i].PrincipalRef == ref {
			return &rows[i], nil
		}
	}
	return &model.CicdAccessGrant{}, nil
}

func (s *Service) BulkUpsertCicdGrants(ctx context.Context, req CicdGrantBulkRequest, actor *auth.CurrentUser) (int, error) {
	if err := s.assertCanManageCicdGrants(ctx, req.ProjectID, actor); err != nil {
		return 0, err
	}
	if len(req.ServiceIDs) == 0 {
		return 0, constants.ErrBadRequestWithMsg("service_ids required")
	}
	canView, canBuild, canRelease, canManage := req.CanView, req.CanBuild, req.CanRelease, req.CanManage
	if canManage {
		canView, canBuild, canRelease = true, true, true
	} else if canRelease || canBuild {
		canView = true
	} else if !canView {
		canView = true
	}
	kind, ref := projectacl.UserPrincipalRef(req.UserID)
	n := 0
	for _, sid := range req.ServiceIDs {
		if _, err := s.repo.GetService(ctx, req.ProjectID, sid); err != nil {
			continue
		}
		row := model.CicdAccessGrant{
			ProjectID: req.ProjectID, ServiceID: sid, PrincipalKind: kind, PrincipalRef: ref,
			CanView: canView, CanBuild: canBuild, CanRelease: canRelease, CanManage: canManage, CreatedBy: req.CreatedBy,
		}
		if err := s.repo.UpsertAccessGrant(ctx, &row); err == nil {
			n++
		}
	}
	return n, nil
}

func (s *Service) DeleteCicdGrant(ctx context.Context, projectID, grantID uint, actor *auth.CurrentUser) error {
	if err := s.assertCanManageCicdGrants(ctx, projectID, actor); err != nil {
		return err
	}
	n, err := s.repo.DeleteAccessGrant(ctx, projectID, grantID)
	if err != nil {
		return err
	}
	if n == 0 {
		return constants.ErrNotFound
	}
	return nil
}

func (s *Service) BootstrapCicdGrantsForMembers(ctx context.Context, req BootstrapCicdGrantsRequest, actor *auth.CurrentUser) (map[string]int, error) {
	if err := s.assertCanManageCicdGrants(ctx, req.ProjectID, actor); err != nil {
		return nil, err
	}
	members, err := s.repo.ListMemberUserIDs(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	services, err := s.repo.ListServicesByProject(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	granted, skipped := 0, 0
	for _, m := range members {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if role == "owner" || role == "admin" {
			skipped++
			continue
		}
		kind, ref := projectacl.UserPrincipalRef(m.UserID)
		for _, svc := range services {
			row := model.CicdAccessGrant{
				ProjectID: req.ProjectID, ServiceID: svc.ID, PrincipalKind: kind, PrincipalRef: ref,
				CanView: true, CanBuild: false, CanRelease: false, CanManage: false, CreatedBy: req.CreatedBy, Remark: "bootstrap",
			}
			if err := s.repo.UpsertAccessGrant(ctx, &row); err == nil {
				granted++
			}
		}
	}
	return map[string]int{"grants_upserted": granted, "admin_members_skipped": skipped, "services": len(services)}, nil
}
