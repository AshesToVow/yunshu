package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type ClusterLogRepository struct {
	db *gorm.DB
}

func NewClusterLogRepository(db *gorm.DB) ClusterLogRepo {
	return &ClusterLogRepository{db: db}
}

func (r *ClusterLogRepository) ListRules(ctx context.Context, projectID, clusterID uint) ([]model.ClusterLogRule, error) {
	q := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("id desc")
	if clusterID > 0 {
		q = q.Where("cluster_id = ?", clusterID)
	}
	var list []model.ClusterLogRule
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *ClusterLogRepository) GetRuleByIDInProject(ctx context.Context, projectID, ruleID uint) (*model.ClusterLogRule, error) {
	var row model.ClusterLogRule
	if err := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", ruleID, projectID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ClusterLogRepository) CreateRule(ctx context.Context, row *model.ClusterLogRule) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *ClusterLogRepository) SaveRule(ctx context.Context, row *model.ClusterLogRule) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *ClusterLogRepository) DeleteRuleByIDInProject(ctx context.Context, projectID, ruleID uint) (int64, error) {
	res := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", ruleID, projectID).Delete(&model.ClusterLogRule{})
	return res.RowsAffected, res.Error
}

func (r *ClusterLogRepository) ListAgents(ctx context.Context, projectID uint) ([]model.ClusterLogAgent, error) {
	var list []model.ClusterLogAgent
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *ClusterLogRepository) GetAgent(ctx context.Context, projectID, clusterID uint) (*model.ClusterLogAgent, error) {
	var row model.ClusterLogAgent
	if err := r.db.WithContext(ctx).Where("project_id = ? AND cluster_id = ?", projectID, clusterID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ClusterLogRepository) CreateAgent(ctx context.Context, row *model.ClusterLogAgent) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *ClusterLogRepository) SaveAgent(ctx context.Context, row *model.ClusterLogAgent) error {
	return r.db.WithContext(ctx).Save(row).Error
}
