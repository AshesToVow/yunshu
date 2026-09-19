# -*- coding: utf-8 -*-
"""Generate cicd_repository*.go implementations."""
from pathlib import Path

ROOT = Path(r"d:/gocode/yunshu/internal/repository")

base = r'''package repository

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CicdRepository struct {
	db *gorm.DB
}

func NewCicdRepository(db *gorm.DB) CicdRepo {
	if db == nil {
		return &CicdRepository{}
	}
	return &CicdRepository{db: db}
}

func (r *CicdRepository) Transaction(ctx context.Context, fn func(CicdRepo) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&CicdRepository{db: tx})
	})
}
'''

services = r'''
func (r *CicdRepository) ListServices(ctx context.Context, p CicdServiceListParams) ([]model.CicdService, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.CicdService{}).Where("project_id = ?", p.ProjectID)
	if p.RestrictIDs {
		if len(p.IDs) == 0 {
			return nil, 0, nil
		}
		q = q.Where("id IN ?", p.IDs)
	}
	if kw := strings.TrimSpace(p.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR identifier LIKE ?", like, like)
	}
	if st := strings.TrimSpace(p.ServiceType); st != "" {
		q = q.Where("service_type = ?", st)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.CicdService
	err := q.Order("id DESC").Offset(p.Offset).Limit(p.Limit).Find(&rows).Error
	return rows, total, err
}

func (r *CicdRepository) GetService(ctx context.Context, projectID, serviceID uint) (*model.CicdService, error) {
	var row model.CicdService
	err := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", serviceID, projectID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) GetServiceByID(ctx context.Context, serviceID uint) (*model.CicdService, error) {
	var row model.CicdService
	if err := r.db.WithContext(ctx).Where("id = ?", serviceID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) GetServiceByProjectIdentifier(ctx context.Context, projectID uint, identifier string) (*model.CicdService, error) {
	var row model.CicdService
	err := r.db.WithContext(ctx).Where("project_id = ? AND identifier = ?", projectID, identifier).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) CreateService(ctx context.Context, row *model.CicdService) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *CicdRepository) SaveService(ctx context.Context, row *model.CicdService) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *CicdRepository) DeleteServiceCascade(ctx context.Context, projectID, serviceID uint) error {
	return r.Transaction(ctx, func(tx CicdRepo) error {
		tr := tx.(*CicdRepository)
		if err := tr.db.Where("service_id = ?", serviceID).Delete(&model.CicdCiConfig{}).Error; err != nil {
			return err
		}
		if err := tr.db.Where("service_id = ?", serviceID).Delete(&model.CicdDeployConfig{}).Error; err != nil {
			return err
		}
		if err := tr.db.Where("service_id = ?", serviceID).Delete(&model.CicdBuildRun{}).Error; err != nil {
			return err
		}
		if err := tr.db.Where("service_id = ?", serviceID).Delete(&model.CicdReleaseRun{}).Error; err != nil {
			return err
		}
		return tr.db.Where("id = ? AND project_id = ?", serviceID, projectID).Delete(&model.CicdService{}).Error
	})
}

func (r *CicdRepository) UpdateServiceJenkinsJob(ctx context.Context, serviceID uint, jobName string) error {
	return r.db.WithContext(ctx).Model(&model.CicdService{}).Where("id = ?", serviceID).Update("jenkins_job", jobName).Error
}

func (r *CicdRepository) CountDuplicateServiceIdentifier(ctx context.Context, projectID, excludeID uint, identifier string) (int64, error) {
	var n int64
	q := r.db.WithContext(ctx).Model(&model.CicdService{}).Where("project_id = ? AND identifier = ?", projectID, identifier)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	err := q.Count(&n).Error
	return n, err
}

func (r *CicdRepository) ListServicesByProject(ctx context.Context, projectID uint) ([]model.CicdService, error) {
	var rows []model.CicdService
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) ListServicesByIDs(ctx context.Context, ids []uint) ([]model.CicdService, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []model.CicdService
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) PluckServiceIDsWithCI(ctx context.Context, serviceIDs []uint) ([]uint, error) {
	if len(serviceIDs) == 0 {
		return nil, nil
	}
	var ids []uint
	err := r.db.WithContext(ctx).Model(&model.CicdCiConfig{}).
		Where("service_id IN ?", serviceIDs).Distinct("service_id").Pluck("service_id", &ids).Error
	return ids, err
}

func (r *CicdRepository) CountDeployConfigsByServiceIDs(ctx context.Context, serviceIDs []uint) ([]CicdServiceDeployCount, error) {
	if len(serviceIDs) == 0 {
		return nil, nil
	}
	var rows []CicdServiceDeployCount
	err := r.db.WithContext(ctx).Model(&model.CicdDeployConfig{}).
		Select("service_id, COUNT(*) AS cnt").
		Where("service_id IN ? AND status = 1", serviceIDs).
		Group("service_id").Scan(&rows).Error
	return rows, err
}

func (r *CicdRepository) ListLatestBuildsByServiceIDs(ctx context.Context, serviceIDs []uint) ([]model.CicdBuildRun, error) {
	if len(serviceIDs) == 0 {
		return nil, nil
	}
	var builds []model.CicdBuildRun
	err := r.db.WithContext(ctx).Where("service_id IN ?", serviceIDs).Order("id DESC").Find(&builds).Error
	return builds, err
}

func (r *CicdRepository) CountCIByService(ctx context.Context, serviceID uint) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.CicdCiConfig{}).Where("service_id = ?", serviceID).Count(&n).Error
	return n, err
}

func (r *CicdRepository) GetCIConfig(ctx context.Context, serviceID uint) (*model.CicdCiConfig, error) {
	var row model.CicdCiConfig
	err := r.db.WithContext(ctx).Where("service_id = ?", serviceID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) CreateCIConfig(ctx context.Context, row *model.CicdCiConfig) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *CicdRepository) SaveCIConfig(ctx context.Context, row *model.CicdCiConfig) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *CicdRepository) ListDeployConfigs(ctx context.Context, serviceID uint) ([]model.CicdDeployConfig, error) {
	var rows []model.CicdDeployConfig
	err := r.db.WithContext(ctx).Where("service_id = ?", serviceID).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *CicdRepository) GetDeployConfig(ctx context.Context, serviceID, configID uint) (*model.CicdDeployConfig, error) {
	var row model.CicdDeployConfig
	err := r.db.WithContext(ctx).Where("id = ? AND service_id = ?", configID, serviceID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) GetDeployConfigByID(ctx context.Context, configID uint) (*model.CicdDeployConfig, error) {
	var row model.CicdDeployConfig
	if err := r.db.WithContext(ctx).Where("id = ?", configID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) CreateDeployConfig(ctx context.Context, row *model.CicdDeployConfig) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *CicdRepository) SaveDeployConfig(ctx context.Context, row *model.CicdDeployConfig) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *CicdRepository) DeleteDeployConfig(ctx context.Context, serviceID, configID uint) error {
	return r.db.WithContext(ctx).Where("id = ? AND service_id = ?", configID, serviceID).Delete(&model.CicdDeployConfig{}).Error
}

func (r *CicdRepository) CountDeployConfigDupName(ctx context.Context, serviceID, excludeID uint, name string) (int64, error) {
	var n int64
	q := r.db.WithContext(ctx).Model(&model.CicdDeployConfig{}).Where("service_id = ? AND name = ?", serviceID, name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	err := q.Count(&n).Error
	return n, err
}

func (r *CicdRepository) ClearDefaultContainerDeploy(ctx context.Context, serviceID uint) error {
	return r.db.WithContext(ctx).Model(&model.CicdDeployConfig{}).
		Where("service_id = ? AND deploy_kind = ?", serviceID, model.CicdDeployKindContainer).
		Update("is_default", false).Error
}

func (r *CicdRepository) GetDefaultContainerDeploy(ctx context.Context, serviceID uint) (*model.CicdDeployConfig, error) {
	var row model.CicdDeployConfig
	err := r.db.WithContext(ctx).
		Where("service_id = ? AND deploy_kind = ? AND is_default = ?", serviceID, model.CicdDeployKindContainer, true).
		Order("id ASC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CicdRepository) GetFirstContainerDeploy(ctx context.Context, serviceID uint) (*model.CicdDeployConfig, error) {
	var row model.CicdDeployConfig
	err := r.db.WithContext(ctx).
		Where("service_id = ? AND deploy_kind = ?", serviceID, model.CicdDeployKindContainer).
		Order("id ASC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}
'''

# Write base+services first; continue with more files
(ROOT / "cicd_repository.go").write_text(base + services, encoding="utf-8")
print("wrote cicd_repository.go")
