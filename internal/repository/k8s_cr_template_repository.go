package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type K8sCrTemplateRepository struct {
	db *gorm.DB
}

func NewK8sCrTemplateRepository(db *gorm.DB) K8sCrTemplateRepo {
	return &K8sCrTemplateRepository{db: db}
}

func (r *K8sCrTemplateRepository) List(ctx context.Context, f K8sCrTemplateListFilter) ([]model.K8sCrTemplate, error) {
	q := r.db.WithContext(ctx).Model(&model.K8sCrTemplate{})
	if f.ProjectID > 0 {
		q = q.Where("project_id IN (0, ?)", f.ProjectID)
	}
	if k := strings.TrimSpace(f.Kind); k != "" {
		q = q.Where("gvk_kind = ?", k)
	}
	var list []model.K8sCrTemplate
	err := q.Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *K8sCrTemplateRepository) GetByID(ctx context.Context, id uint) (*model.K8sCrTemplate, error) {
	var row model.K8sCrTemplate
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *K8sCrTemplateRepository) Create(ctx context.Context, row *model.K8sCrTemplate) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *K8sCrTemplateRepository) Save(ctx context.Context, row *model.K8sCrTemplate) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *K8sCrTemplateRepository) DeleteByID(ctx context.Context, id uint) (int64, error) {
	res := r.db.WithContext(ctx).Delete(&model.K8sCrTemplate{}, id)
	return res.RowsAffected, res.Error
}
