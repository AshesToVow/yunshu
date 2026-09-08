package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type LogPipelineRepository struct {
	db *gorm.DB
}

func NewLogPipelineRepository(db *gorm.DB) LogPipelineRepo {
	return &LogPipelineRepository{db: db}
}

func (r *LogPipelineRepository) List(ctx context.Context, projectID uint, kind string) ([]model.LogPipeline, error) {
	q := r.db.WithContext(ctx).Where("project_id = ?", projectID)
	if k := strings.TrimSpace(kind); k != "" {
		q = q.Where("kind = ?", k)
	}
	var list []model.LogPipeline
	if err := q.Order("updated_at DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *LogPipelineRepository) GetByIDInProject(ctx context.Context, projectID, id uint) (*model.LogPipeline, error) {
	var row model.LogPipeline
	if err := r.db.WithContext(ctx).Where("project_id = ? AND id = ?", projectID, id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *LogPipelineRepository) GetByProjectKindClusterName(ctx context.Context, projectID uint, kind string, clusterID uint, name string) (*model.LogPipeline, error) {
	var row model.LogPipeline
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND kind = ? AND cluster_id = ? AND name = ?", projectID, kind, clusterID, name).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *LogPipelineRepository) GetHostByServer(ctx context.Context, projectID, serverID uint) (*model.LogPipeline, error) {
	var row model.LogPipeline
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND kind = ? AND server_id = ?", projectID, "host", serverID).
		Order("id ASC").
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *LogPipelineRepository) GetUnscopedByProjectAndName(ctx context.Context, projectID uint, name string) (*model.LogPipeline, error) {
	var row model.LogPipeline
	err := r.db.WithContext(ctx).Unscoped().
		Where("project_id = ? AND name = ?", projectID, name).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *LogPipelineRepository) GetUnscopedHostByName(ctx context.Context, projectID uint, name string) (*model.LogPipeline, error) {
	var row model.LogPipeline
	err := r.db.WithContext(ctx).Unscoped().
		Where("project_id = ? AND kind = ? AND name = ?", projectID, "host", name).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *LogPipelineRepository) Create(ctx context.Context, row *model.LogPipeline) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *LogPipelineRepository) Save(ctx context.Context, row *model.LogPipeline) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *LogPipelineRepository) UnscopedSave(ctx context.Context, row *model.LogPipeline) error {
	return r.db.WithContext(ctx).Unscoped().Save(row).Error
}

func (r *LogPipelineRepository) DeleteByIDInProject(ctx context.Context, projectID, id uint) (int64, error) {
	res := r.db.WithContext(ctx).Where("project_id = ? AND id = ?", projectID, id).Delete(&model.LogPipeline{})
	return res.RowsAffected, res.Error
}

func (r *LogPipelineRepository) CreateVersion(ctx context.Context, ver *model.LogPipelineVersion) error {
	return r.db.WithContext(ctx).Create(ver).Error
}

func (r *LogPipelineRepository) ListVersions(ctx context.Context, projectID, pipelineID uint, limit int) ([]model.LogPipelineVersion, error) {
	var list []model.LogPipelineVersion
	q := r.db.WithContext(ctx).Where("project_id = ? AND pipeline_id = ?", projectID, pipelineID).
		Order("version DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *LogPipelineRepository) GetVersion(ctx context.Context, projectID, pipelineID, versionID uint) (*model.LogPipelineVersion, error) {
	var ver model.LogPipelineVersion
	if err := r.db.WithContext(ctx).
		Where("id = ? AND pipeline_id = ? AND project_id = ?", versionID, pipelineID, projectID).
		First(&ver).Error; err != nil {
		return nil, err
	}
	return &ver, nil
}
