package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

type K8sWorkloadSnapshotRepository struct {
	db *gorm.DB
}

func NewK8sWorkloadSnapshotRepository(db *gorm.DB) K8sWorkloadSnapshotRepo {
	return &K8sWorkloadSnapshotRepository{db: db}
}

func (r *K8sWorkloadSnapshotRepository) Create(ctx context.Context, row *model.K8sWorkloadSnapshot) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *K8sWorkloadSnapshotRepository) GetByID(ctx context.Context, id uint) (*model.K8sWorkloadSnapshot, error) {
	var row model.K8sWorkloadSnapshot
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *K8sWorkloadSnapshotRepository) List(
	ctx context.Context,
	p K8sWorkloadSnapshotListParams,
) (*pagination.Result[model.K8sWorkloadSnapshot], error) {
	page, pageSize := pagination.Normalize(p.Page, p.PageSize)
	q := r.db.WithContext(ctx).Model(&model.K8sWorkloadSnapshot{})
	if p.ProjectID > 0 {
		q = q.Where("project_id = ?", p.ProjectID)
	}
	if p.ClusterID > 0 {
		q = q.Where("cluster_id = ?", p.ClusterID)
	}
	if ns := strings.TrimSpace(p.Namespace); ns != "" {
		q = q.Where("namespace = ?", ns)
	}
	if kind := strings.TrimSpace(p.Kind); kind != "" {
		q = q.Where("kind = ?", kind)
	}
	if name := strings.TrimSpace(p.Name); name != "" {
		q = q.Where("name = ?", name)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []model.K8sWorkloadSnapshot
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return &pagination.Result[model.K8sWorkloadSnapshot]{
		List: list, Total: total, Page: page, PageSize: pageSize,
	}, nil
}
