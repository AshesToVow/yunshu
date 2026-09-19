package repository

import (
	"context"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type ServicePortraitRepository struct {
	db *gorm.DB
}

func NewServicePortraitRepository(db *gorm.DB) ServicePortraitRepo {
	return &ServicePortraitRepository{db: db}
}

func (r *ServicePortraitRepository) GetCicdService(ctx context.Context, projectID, id uint) (*model.CicdService, error) {
	var svc model.CicdService
	if err := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", id, projectID).First(&svc).Error; err != nil {
		return nil, err
	}
	return &svc, nil
}

func (r *ServicePortraitRepository) LatestReleaseRun(ctx context.Context, projectID, cicdServiceID uint) (*model.CicdReleaseRun, error) {
	var run model.CicdReleaseRun
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND service_id = ?", projectID, cicdServiceID).
		Order("id DESC").
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *ServicePortraitRepository) CountReleaseRunsSince(
	ctx context.Context,
	projectID, cicdServiceID uint,
	since time.Time,
) (total, failedOrCancelled int64, err error) {
	err = r.db.WithContext(ctx).Model(&model.CicdReleaseRun{}).
		Where("project_id = ? AND service_id = ? AND created_at >= ?", projectID, cicdServiceID, since).
		Count(&total).Error
	if err != nil {
		return 0, 0, err
	}
	err = r.db.WithContext(ctx).Model(&model.CicdReleaseRun{}).
		Where("project_id = ? AND service_id = ? AND created_at >= ? AND status IN ?",
			projectID, cicdServiceID, since, []string{"failed", "cancelled"}).
		Count(&failedOrCancelled).Error
	return total, failedOrCancelled, err
}

func (r *ServicePortraitRepository) CountFiringAlertsSince(ctx context.Context, severity string, since time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("status = ? AND severity = ? AND created_at >= ?", "firing", severity, since).
		Count(&n).Error
	return n, err
}

func (r *ServicePortraitRepository) CountFailedChangesSince(
	ctx context.Context,
	projectID, catalogServiceID uint,
	source string,
	since time.Time,
) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.ChangeEvent{}).
		Where("project_id = ? AND service_id = ? AND source = ? AND status = ? AND started_at >= ?",
			projectID, catalogServiceID, source, model.ChangeStatusFailed, since).
		Count(&n).Error
	return n, err
}

func (r *ServicePortraitRepository) CountOpenAnomalies(ctx context.Context, projectID uint, severity string) (int64, error) {
	q := r.db.WithContext(ctx).Model(&model.LogAnomaly{}).
		Where("project_id = ? AND status = ?", projectID, model.LogAnomalyStatusOpen)
	if severity != "" {
		q = q.Where("severity = ?", severity)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *ServicePortraitRepository) CountLogPatterns(ctx context.Context, projectID uint) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.LogPattern{}).
		Where("project_id = ?", projectID).
		Count(&n).Error
	return n, err
}

func (r *ServicePortraitRepository) ListRecentOpenAnomalies(ctx context.Context, projectID uint, limit int) ([]model.LogAnomaly, error) {
	if limit <= 0 {
		limit = 5
	}
	var rows []model.LogAnomaly
	err := r.db.WithContext(ctx).Model(&model.LogAnomaly{}).
		Where("project_id = ? AND status = ?", projectID, model.LogAnomalyStatusOpen).
		Order("detected_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
