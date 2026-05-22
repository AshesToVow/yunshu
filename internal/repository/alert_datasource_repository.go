package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertDatasourceRepository struct {
	db *gorm.DB
}

func NewAlertDatasourceRepository(db *gorm.DB) AlertDatasourceRepo {
	return &AlertDatasourceRepository{db: db}
}

func (r *AlertDatasourceRepository) ListWithProject(ctx context.Context, f AlertDatasourceListFilter, offset, limit int) ([]AlertDatasourceListRow, int64, error) {
	tx := r.db.WithContext(ctx).Table("alert_datasources ad").
		Select("ad.*, p.name AS project_name").
		Joins("LEFT JOIN projects p ON p.id = ad.project_id AND p.deleted_at IS NULL")
	if f.ProjectID > 0 {
		tx = tx.Where("ad.project_id = ?", f.ProjectID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("ad.name LIKE ? OR ad.base_url LIKE ? OR ad.remark LIKE ? OR p.name LIKE ?", like, like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []AlertDatasourceListRow
	err := tx.Order("ad.id ASC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AlertDatasourceRepository) GetByID(ctx context.Context, id uint) (*model.AlertDatasource, error) {
	var row model.AlertDatasource
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AlertDatasourceRepository) Create(ctx context.Context, row *model.AlertDatasource) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *AlertDatasourceRepository) Save(ctx context.Context, row *model.AlertDatasource) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *AlertDatasourceRepository) Delete(ctx context.Context, id uint) (int64, error) {
	res := r.db.WithContext(ctx).Delete(&model.AlertDatasource{}, id)
	return res.RowsAffected, res.Error
}

var _ AlertDatasourceRepo = (*AlertDatasourceRepository)(nil)
