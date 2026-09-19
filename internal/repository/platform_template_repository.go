package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type PlatformTemplateRepository struct {
	db *gorm.DB
}

func NewPlatformTemplateRepository(db *gorm.DB) PlatformTemplateRepo {
	return &PlatformTemplateRepository{db: db}
}

func (r *PlatformTemplateRepository) List(ctx context.Context, p PlatformTemplateListParams) ([]model.PlatformTemplate, int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.PlatformTemplate{})
	if c := p.Category; c != "" {
		tx = tx.Where("category = ?", c)
	}
	if kw := p.Keyword; kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("template_key LIKE ? OR name LIKE ? OR description LIKE ?", like, like, like)
	}
	if p.Status != nil {
		tx = tx.Where("status = ?", *p.Status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.PlatformTemplate
	err := tx.Order("category ASC, template_key ASC").
		Offset(p.Offset).Limit(p.Limit).Find(&rows).Error
	return rows, total, err
}

func (r *PlatformTemplateRepository) GetByID(ctx context.Context, id uint) (*model.PlatformTemplate, error) {
	var row model.PlatformTemplate
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *PlatformTemplateRepository) GetByKeyEnabled(ctx context.Context, templateKey string) (*model.PlatformTemplate, error) {
	var row model.PlatformTemplate
	err := r.db.WithContext(ctx).
		Where("template_key = ? AND status = ?", templateKey, model.PlatformTemplateStatusEnabled).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *PlatformTemplateRepository) Create(ctx context.Context, row *model.PlatformTemplate) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *PlatformTemplateRepository) Save(ctx context.Context, row *model.PlatformTemplate) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *PlatformTemplateRepository) Delete(ctx context.Context, row *model.PlatformTemplate) error {
	return r.db.WithContext(ctx).Delete(row).Error
}

func (r *PlatformTemplateRepository) DeleteVersionsByTemplateID(ctx context.Context, templateID uint) error {
	return r.db.WithContext(ctx).Where("template_id = ?", templateID).Delete(&model.PlatformTemplateVersion{}).Error
}

func (r *PlatformTemplateRepository) MaxVersion(ctx context.Context, templateID uint) (int, error) {
	var maxVer int
	err := r.db.WithContext(ctx).Model(&model.PlatformTemplateVersion{}).
		Where("template_id = ?", templateID).Select("COALESCE(MAX(version),0)").Scan(&maxVer).Error
	return maxVer, err
}

func (r *PlatformTemplateRepository) CreateVersion(ctx context.Context, ver *model.PlatformTemplateVersion) error {
	return r.db.WithContext(ctx).Create(ver).Error
}

func (r *PlatformTemplateRepository) GetVersion(ctx context.Context, templateID uint, version int) (*model.PlatformTemplateVersion, error) {
	var ver model.PlatformTemplateVersion
	err := r.db.WithContext(ctx).
		Where("template_id = ? AND version = ?", templateID, version).First(&ver).Error
	if err != nil {
		return nil, err
	}
	return &ver, nil
}

func (r *PlatformTemplateRepository) GetLatestVersion(ctx context.Context, templateID uint) (*model.PlatformTemplateVersion, error) {
	var ver model.PlatformTemplateVersion
	err := r.db.WithContext(ctx).Where("template_id = ?", templateID).
		Order("version DESC").First(&ver).Error
	if err != nil {
		return nil, err
	}
	return &ver, nil
}

func (r *PlatformTemplateRepository) UpdateVersionStorageKey(ctx context.Context, versionID uint, storageKey string) error {
	return r.db.WithContext(ctx).Model(&model.PlatformTemplateVersion{}).
		Where("id = ?", versionID).Update("storage_key", storageKey).Error
}

func (r *PlatformTemplateRepository) UpdatePublishedVersion(ctx context.Context, templateID uint, version int) error {
	return r.db.WithContext(ctx).Model(&model.PlatformTemplate{}).
		Where("id = ?", templateID).Update("published_version", version).Error
}

func (r *PlatformTemplateRepository) ListVersions(ctx context.Context, templateID uint) ([]model.PlatformTemplateVersion, error) {
	var rows []model.PlatformTemplateVersion
	err := r.db.WithContext(ctx).Where("template_id = ?", templateID).
		Order("version DESC").Find(&rows).Error
	return rows, err
}

func (r *PlatformTemplateRepository) Transaction(ctx context.Context, fn func(PlatformTemplateRepo) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&PlatformTemplateRepository{db: tx})
	})
}
