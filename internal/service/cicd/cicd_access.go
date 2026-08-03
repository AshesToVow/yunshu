package cicd

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/projectacl"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	kind, ref := projectacl.UserPrincipalRef(actor.ID)
	var g model.CicdAccessGrant
	err = s.db.WithContext(ctx).
		Where("project_id = ? AND service_id = ? AND principal_kind = ? AND principal_ref = ?", projectID, serviceID, kind, ref).
		First(&g).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &CicdAccessPerm{}, nil
		}
		return nil, err
	}
	return &CicdAccessPerm{CanView: g.CanView, CanBuild: g.CanBuild, CanRelease: g.CanRelease, CanManage: g.CanManage}, nil
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
	kind, ref := projectacl.UserPrincipalRef(actor.ID)
	var ids []uint
	err = s.db.WithContext(ctx).Model(&model.CicdAccessGrant{}).
		Where("project_id = ? AND principal_kind = ? AND principal_ref = ? AND (can_view = ? OR can_build = ? OR can_release = ? OR can_manage = ?)",
			projectID, kind, ref, true, true, true, true).
		Pluck("service_id", &ids).Error
	if err != nil {
		return false, nil, err
	}
	if ids == nil {
		ids = []uint{}
	}
	return false, ids, nil
}

func (s *Service) ListCicdGrants(ctx context.Context, projectID uint, actor *auth.CurrentUser, userID, serviceID uint) ([]CicdGrantItem, error) {
	if err := s.assertCanManageCicdGrants(ctx, projectID, actor); err != nil {
		return nil, err
	}
	q := s.db.WithContext(ctx).Model(&model.CicdAccessGrant{}).Where("project_id = ?", projectID)
	if userID > 0 {
		_, ref := projectacl.UserPrincipalRef(userID)
		q = q.Where("principal_kind = ? AND principal_ref = ?", model.ResourcePrincipalUser, ref)
	}
	if serviceID > 0 {
		q = q.Where("service_id = ?", serviceID)
	}
	var rows []model.CicdAccessGrant
	if err := q.Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]CicdGrantItem, 0, len(rows))
	for _, r := range rows {
		item := CicdGrantItem{CicdAccessGrant: r}
		var svc model.CicdService
		if err := s.db.WithContext(ctx).Select("id", "name").Where("id = ?", r.ServiceID).First(&svc).Error; err == nil {
			item.ServiceName = svc.Name
		}
		if uid, ok := projectacl.ParseUserRef(r.PrincipalRef); ok {
			var u model.User
			if err := s.db.WithContext(ctx).Select("id", "username", "nickname").Where("id = ?", uid).First(&u).Error; err == nil {
				item.Username, item.Nickname = u.Username, u.Nickname
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
	var svc model.CicdService
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", req.ServiceID, req.ProjectID).First(&svc).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
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
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "service_id"}, {Name: "principal_kind"}, {Name: "principal_ref"}},
		DoUpdates: clause.AssignmentColumns([]string{"can_view", "can_build", "can_release", "can_manage", "remark", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return nil, err
	}
	_ = s.db.WithContext(ctx).
		Where("project_id = ? AND service_id = ? AND principal_kind = ? AND principal_ref = ?", req.ProjectID, req.ServiceID, kind, ref).
		First(&row).Error
	return &row, nil
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
		var svc model.CicdService
		if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", sid, req.ProjectID).First(&svc).Error; err != nil {
			continue
		}
		row := model.CicdAccessGrant{
			ProjectID: req.ProjectID, ServiceID: sid, PrincipalKind: kind, PrincipalRef: ref,
			CanView: canView, CanBuild: canBuild, CanRelease: canRelease, CanManage: canManage, CreatedBy: req.CreatedBy,
		}
		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "project_id"}, {Name: "service_id"}, {Name: "principal_kind"}, {Name: "principal_ref"}},
			DoUpdates: clause.AssignmentColumns([]string{"can_view", "can_build", "can_release", "can_manage", "updated_at"}),
		}).Create(&row).Error; err == nil {
			n++
		}
	}
	return n, nil
}

func (s *Service) DeleteCicdGrant(ctx context.Context, projectID, grantID uint, actor *auth.CurrentUser) error {
	if err := s.assertCanManageCicdGrants(ctx, projectID, actor); err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", grantID, projectID).Delete(&model.CicdAccessGrant{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFound
	}
	return nil
}

func (s *Service) BootstrapCicdGrantsForMembers(ctx context.Context, req BootstrapCicdGrantsRequest, actor *auth.CurrentUser) (map[string]int, error) {
	if err := s.assertCanManageCicdGrants(ctx, req.ProjectID, actor); err != nil {
		return nil, err
	}
	var members []model.ProjectMember
	if err := s.db.WithContext(ctx).Where("project_id = ?", req.ProjectID).Find(&members).Error; err != nil {
		return nil, err
	}
	var services []model.CicdService
	if err := s.db.WithContext(ctx).Where("project_id = ?", req.ProjectID).Find(&services).Error; err != nil {
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
				CanView: true, CanBuild: true, CanRelease: true, CanManage: false, CreatedBy: req.CreatedBy, Remark: "bootstrap",
			}
			if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "project_id"}, {Name: "service_id"}, {Name: "principal_kind"}, {Name: "principal_ref"}},
				DoUpdates: clause.AssignmentColumns([]string{"can_view", "can_build", "can_release", "updated_at"}),
			}).Create(&row).Error; err == nil {
				granted++
			}
		}
	}
	return map[string]int{"grants_upserted": granted, "admin_members_skipped": skipped, "services": len(services)}, nil
}
