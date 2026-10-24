package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertConsulRepository struct {
	db *gorm.DB
}

func NewAlertConsulRepository(db *gorm.DB) AlertConsulRepo {
	return &AlertConsulRepository{db: db}
}

func (r *AlertConsulRepository) ListEndpoints(ctx context.Context, f AlertConsulEndpointListFilter, offset, limit int) ([]model.AlertConsulEndpoint, int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.AlertConsulEndpoint{})
	if f.ProjectID > 0 {
		tx = tx.Where("project_id = ?", f.ProjectID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR address LIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AlertConsulEndpoint
	err := tx.Order("id desc").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AlertConsulRepository) CreateEndpoint(ctx context.Context, row *model.AlertConsulEndpoint) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *AlertConsulRepository) GetEndpointByID(ctx context.Context, id uint) (*model.AlertConsulEndpoint, error) {
	var row model.AlertConsulEndpoint
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AlertConsulRepository) SaveEndpoint(ctx context.Context, row *model.AlertConsulEndpoint) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *AlertConsulRepository) DeleteEndpointWithObjects(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("endpoint_id = ?", id).Delete(&model.AlertMonitorObject{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.AlertConsulEndpoint{}, id).Error
	})
}

func (r *AlertConsulRepository) ListObjects(ctx context.Context, f AlertMonitorObjectListFilter, offset, limit int) ([]model.AlertMonitorObject, int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.AlertMonitorObject{})
	if f.ProjectID > 0 {
		tx = tx.Where("project_id = ?", f.ProjectID)
	}
	if f.EndpointID > 0 {
		tx = tx.Where("endpoint_id = ?", f.EndpointID)
	}
	if role := strings.TrimSpace(f.ExporterRole); role != "" {
		tx = tx.Where("exporter_role = ?", role)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where(
			"service_name LIKE ? OR service_id LIKE ? OR address LIKE ? OR yunshu_project LIKE ?",
			like, like, like, like,
		)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AlertMonitorObject
	err := tx.Order("id desc").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AlertConsulRepository) UpdateEndpointMeta(ctx context.Context, id uint, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&model.AlertConsulEndpoint{}).
		Where("id = ?", id).
		Updates(fields).Error
}

func (r *AlertConsulRepository) UpsertMonitorObject(ctx context.Context, row *model.AlertMonitorObject) error {
	if row == nil {
		return nil
	}
	var existing model.AlertMonitorObject
	qerr := r.db.WithContext(ctx).Unscoped().
		Where("endpoint_id = ? AND service_name = ? AND service_id = ?",
			row.EndpointID, row.ServiceName, row.ServiceID).
		First(&existing).Error
	if qerr == gorm.ErrRecordNotFound {
		return r.db.WithContext(ctx).Create(row).Error
	}
	if qerr != nil {
		return qerr
	}
	row.ID = existing.ID
	row.CreatedAt = existing.CreatedAt
	row.DeletedAt = gorm.DeletedAt{}
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *AlertConsulRepository) ListObjectsByEndpoint(ctx context.Context, endpointID uint) ([]model.AlertMonitorObject, error) {
	var list []model.AlertMonitorObject
	err := r.db.WithContext(ctx).Where("endpoint_id = ?", endpointID).Find(&list).Error
	return list, err
}

func (r *AlertConsulRepository) DeleteObject(ctx context.Context, row *model.AlertMonitorObject) error {
	if row == nil {
		return nil
	}
	return r.db.WithContext(ctx).Delete(row).Error
}

func (r *AlertConsulRepository) Transaction(ctx context.Context, fn func(AlertConsulRepo) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&AlertConsulRepository{db: tx})
	})
}

var _ AlertConsulRepo = (*AlertConsulRepository)(nil)
