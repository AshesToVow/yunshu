package k8s

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/repository"
)

type K8sCrTemplateService struct {
	repo repository.K8sCrTemplateRepo
}

func NewK8sCrTemplateService(repo repository.K8sCrTemplateRepo) *K8sCrTemplateService {
	return &K8sCrTemplateService{repo: repo}
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
	list, err := s.repo.List(ctx, repository.K8sCrTemplateListFilter{
		ProjectID: projectID,
		Kind:      kind,
	})
	return list, bizerrors.Pass(ctx, "k8s.cr_template", "List", err)
}

func (s *K8sCrTemplateService) Get(ctx context.Context, id uint) (*model.K8sCrTemplate, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "k8s.cr_template", "Get", err)
	}
	return row, nil
}

func (s *K8sCrTemplateService) Create(ctx context.Context, req K8sCrTemplateUpsertRequest) (*model.K8sCrTemplate, error) {
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
		return nil, bizerrors.Pass(ctx, "k8s.cr_template", "Update", err)
	}
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
	n, err := s.repo.DeleteByID(ctx, id)
	if err != nil {
		return bizerrors.Pass(ctx, "k8s.cr_template", "Delete", err)
	}
	if n == 0 {
		return constants.ErrNotFound
	}
	return nil
}
