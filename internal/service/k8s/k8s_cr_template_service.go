package k8s

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

type K8sCrTemplateService struct {
	db *gorm.DB
}

func NewK8sCrTemplateService(db *gorm.DB) *K8sCrTemplateService {
	return &K8sCrTemplateService{db: db}
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
	db := s.db.WithContext(ctx).Model(&model.K8sCrTemplate{})
	if projectID > 0 {
		db = db.Where("project_id IN (0, ?)", projectID)
	}
	if k := strings.TrimSpace(kind); k != "" {
		db = db.Where("gvk_kind = ?", k)
	}
	var list []model.K8sCrTemplate
	err := db.Order("sort_order ASC, id ASC").Find(&list).Error
	return list, bizerrors.Pass(ctx, "k8s.cr_template", "List", err)
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
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "k8s.cr_template", "Create", err)
	}
	return &row, nil
}

func (s *K8sCrTemplateService) Update(ctx context.Context, id uint, req K8sCrTemplateUpsertRequest) (*model.K8sCrTemplate, error) {
	var row model.K8sCrTemplate
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "k8s.cr_template", "Update", err)
	}
	return &row, nil
}

func (s *K8sCrTemplateService) Delete(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.K8sCrTemplate{}, id)
	if res.Error != nil {
		return bizerrors.Pass(ctx, "k8s.cr_template", "Delete", res.Error)
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFound
	}
	return nil
}
