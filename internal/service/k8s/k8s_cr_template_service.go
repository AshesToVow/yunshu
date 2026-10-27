package k8s

import (
	"context"
	"errors"
	"strings"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/projectaccess"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type K8sCrTemplateService struct {
	repo    repository.K8sCrTemplateRepo
	members interfaces.ProjectMemberRepository
}

func NewK8sCrTemplateService(repo repository.K8sCrTemplateRepo, members interfaces.ProjectMemberRepository) *K8sCrTemplateService {
	return &K8sCrTemplateService{repo: repo, members: members}
}

type K8sCrTemplateUpsertRequest struct {
	ProjectID  uint   `json:"project_id"`
	Name       string `json:"name" binding:"required,max=128"`
	GVKGroup   string `json:"gvk_group"`
	GVKVersion string `json:"gvk_version"`
	GVKKind    string `json:"gvk_kind" binding:"required,max=64"`
	Body       string `json:"body" binding:"required"`
	SortOrder  int    `json:"sort_order"`
}

func (s *K8sCrTemplateService) List(ctx context.Context, projectID uint, kind string) ([]model.K8sCrTemplate, error) {
	filter := repository.K8sCrTemplateListFilter{ProjectID: projectID, Kind: kind}
	if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil && !auth.IsSuperAdminRole(u.RoleCodes) {
		if projectID > 0 {
			if err := s.requireProjectRole(ctx, u, projectID, "readonly"); err != nil {
				return nil, err
			}
		} else {
			if s.members == nil {
				return nil, constants.ErrInternal
			}
			ids, err := s.members.ListProjectIDsByUser(ctx, u.ID)
			if err != nil {
				return nil, bizerrors.Pass(ctx, "k8s.cr_template", "ListProjectIDsByUser", err)
			}
			filter.VisibleOnly = true
			filter.ProjectIDs = ids
		}
	}
	list, err := s.repo.List(ctx, filter)
	return list, bizerrors.Pass(ctx, "k8s.cr_template", "List", err)
}

func (s *K8sCrTemplateService) Get(ctx context.Context, id uint) (*model.K8sCrTemplate, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "k8s.cr_template", "Get", err)
	}
	if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil && !auth.IsSuperAdminRole(u.RoleCodes) {
		if err := s.requireProjectRole(ctx, u, row.ProjectID, "readonly"); err != nil {
			return nil, err
		}
	}
	return row, nil
}

func (s *K8sCrTemplateService) Create(ctx context.Context, req K8sCrTemplateUpsertRequest) (*model.K8sCrTemplate, error) {
	if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil {
		if err := s.requireProjectRole(ctx, u, req.ProjectID, "admin"); err != nil {
			return nil, err
		}
	}
	ver := strings.TrimSpace(req.GVKVersion)
	if ver == "" {
		ver = "v1"
	}
	row := model.K8sCrTemplate{
		ProjectID: req.ProjectID, Name: strings.TrimSpace(req.Name),
		GVKGroup: strings.TrimSpace(req.GVKGroup), GVKVersion: ver,
		GVKKind: strings.TrimSpace(req.GVKKind), Body: req.Body, SortOrder: req.SortOrder,
	}
	if err := s.repo.Create(ctx, &row); err != nil {
		return nil, bizerrors.Pass(ctx, "k8s.cr_template", "Create", err)
	}
	return &row, nil
}

func (s *K8sCrTemplateService) Update(ctx context.Context, id uint, req K8sCrTemplateUpsertRequest) (*model.K8sCrTemplate, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "k8s.cr_template", "Update", err)
	}
	if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil {
		if err := s.requireProjectRole(ctx, u, row.ProjectID, "admin"); err != nil {
			return nil, err
		}
	}
	// 归属项目不可通过更新迁移，避免把项目模板提升为全局模板。
	row.Name = strings.TrimSpace(req.Name)
	row.GVKGroup = strings.TrimSpace(req.GVKGroup)
	if v := strings.TrimSpace(req.GVKVersion); v != "" {
		row.GVKVersion = v
	}
	row.GVKKind = strings.TrimSpace(req.GVKKind)
	row.Body = req.Body
	row.SortOrder = req.SortOrder
	if err := s.repo.Save(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "k8s.cr_template", "Update", err)
	}
	return row, nil
}

func (s *K8sCrTemplateService) Delete(ctx context.Context, id uint) error {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrNotFound
		}
		return bizerrors.Pass(ctx, "k8s.cr_template", "Delete", err)
	}
	if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil {
		if err := s.requireProjectRole(ctx, u, row.ProjectID, "admin"); err != nil {
			return err
		}
	}
	if _, err := s.repo.DeleteByID(ctx, id); err != nil {
		return bizerrors.Pass(ctx, "k8s.cr_template", "Delete", err)
	}
	return nil
}

// requireProjectRole 校验当前用户在模板归属项目中的角色；project_id=0 的全局模板仅超管可维护。
func (s *K8sCrTemplateService) requireProjectRole(ctx context.Context, u *auth.CurrentUser, projectID uint, need string) error {
	if projectID == 0 {
		if auth.IsSuperAdminRole(u.RoleCodes) {
			return nil
		}
		return constants.ErrForbiddenWithMsg("全局模板仅超级管理员可维护")
	}
	if auth.IsSuperAdminRole(u.RoleCodes) {
		return nil
	}
	if s.members == nil {
		return constants.ErrInternal
	}
	m, err := s.members.GetByProjectAndUser(ctx, projectID, u.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrForbiddenWithMsg("无权访问该项目模板")
		}
		return bizerrors.Pass(ctx, "k8s.cr_template", "GetByProjectAndUser", err)
	}
	if !projectaccess.RoleAtLeast(m.Role, need) {
		if need == "readonly" {
			return constants.ErrForbiddenWithMsg("无权访问该项目模板")
		}
		return constants.ErrForbiddenWithMsg("需项目管理员权限才能维护模板")
	}
	return nil
}
