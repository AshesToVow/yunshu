package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type HarborMergeRepository struct {
	db *gorm.DB
}

func NewHarborMergeRepository(db *gorm.DB) HarborMergeRepo {
	return &HarborMergeRepository{db: db}
}

func (r *HarborMergeRepository) GetDefaultEnabledHarborRegistry(ctx context.Context) (*model.ImageRegistry, error) {
	var reg model.ImageRegistry
	err := r.db.WithContext(ctx).
		Where("is_default = ? AND status = 1 AND type = ?", true, model.ImageRegistryTypeHarbor).
		Order("id ASC").First(&reg).Error
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *HarborMergeRepository) GetProjectRegistryBinding(
	ctx context.Context,
	projectID uint,
) (*model.ProjectRegistryBinding, error) {
	var bind model.ProjectRegistryBinding
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).First(&bind).Error; err != nil {
		return nil, err
	}
	return &bind, nil
}

func (r *HarborMergeRepository) GetImageRegistryByID(ctx context.Context, id uint) (*model.ImageRegistry, error) {
	var reg model.ImageRegistry
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&reg).Error; err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *HarborMergeRepository) GetProjectHarborFields(
	ctx context.Context,
	projectID uint,
) (harborURL, harborProject string, err error) {
	var p model.Project
	if err = r.db.WithContext(ctx).
		Select("harbor_url", "harbor_project").
		Where("id = ?", projectID).
		First(&p).Error; err != nil {
		return "", "", err
	}
	return p.HarborURL, p.HarborProject, nil
}
