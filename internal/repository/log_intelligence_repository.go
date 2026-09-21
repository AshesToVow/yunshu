package repository

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type LogIntelligenceRepository struct {
	db *gorm.DB
}

func NewLogIntelligenceRepository(db *gorm.DB) LogIntelligenceRepo {
	return &LogIntelligenceRepository{db: db}
}

func (r *LogIntelligenceRepository) ListPatterns(ctx context.Context, f LogPatternListFilter) ([]model.LogPattern, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&model.LogPattern{}).Where("project_id = ?", f.ProjectID)
	if lv := strings.TrimSpace(f.Level); lv != "" {
		dbq = dbq.Where("level = ?", strings.ToUpper(lv))
	}
	if sn := strings.TrimSpace(f.ServiceName); sn != "" {
		dbq = dbq.Where("service_name = ?", sn)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		dbq = dbq.Where("signature LIKE ? OR sample LIKE ?", like, like)
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.LogPattern
	if err := dbq.Order("hit_count DESC, last_seen_at DESC").
		Offset(f.Offset).Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *LogIntelligenceRepository) GetPatternBySignature(ctx context.Context, projectID uint, signature string) (*model.LogPattern, error) {
	var row model.LogPattern
	if err := r.db.WithContext(ctx).Where("project_id = ? AND signature = ?", projectID, signature).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *LogIntelligenceRepository) CreatePattern(ctx context.Context, row *model.LogPattern) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *LogIntelligenceRepository) SavePattern(ctx context.Context, row *model.LogPattern) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *LogIntelligenceRepository) ListAnomalies(ctx context.Context, f LogAnomalyListFilter) ([]model.LogAnomaly, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&model.LogAnomaly{}).Where("project_id = ?", f.ProjectID)
	if st := strings.TrimSpace(f.Status); st != "" {
		dbq = dbq.Where("status = ?", st)
	}
	if tp := strings.TrimSpace(f.AnomalyType); tp != "" {
		dbq = dbq.Where("anomaly_type = ?", tp)
	}
	if f.AssigneeID != nil && *f.AssigneeID > 0 {
		dbq = dbq.Where("assignee_id = ?", *f.AssigneeID)
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.LogAnomaly
	if err := dbq.Order("detected_at DESC").
		Offset(f.Offset).Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *LogIntelligenceRepository) GetAnomalyByIDInProject(ctx context.Context, projectID, anomalyID uint) (*model.LogAnomaly, error) {
	var row model.LogAnomaly
	if err := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", anomalyID, projectID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *LogIntelligenceRepository) CreateAnomaly(ctx context.Context, row *model.LogAnomaly) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *LogIntelligenceRepository) SaveAnomaly(ctx context.Context, row *model.LogAnomaly) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *LogIntelligenceRepository) CountAnomaliesByTypeSignature(ctx context.Context, projectID uint, anomalyType, signature string) (int64, error) {
	var cnt int64
	err := r.db.WithContext(ctx).Model(&model.LogAnomaly{}).
		Where("project_id = ? AND anomaly_type = ? AND signature = ?", projectID, anomalyType, signature).
		Count(&cnt).Error
	return cnt, err
}

func (r *LogIntelligenceRepository) GetRecentSpikeAnomaly(ctx context.Context, projectID uint, since time.Time) (*model.LogAnomaly, error) {
	var row model.LogAnomaly
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND anomaly_type = ? AND detected_at >= ?", projectID, model.LogAnomalyTypeErrorSpike, since).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *LogIntelligenceRepository) CountOpenAnomalies(ctx context.Context, projectID uint) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.LogAnomaly{}).
		Where("project_id = ? AND status = ?", projectID, model.LogAnomalyStatusOpen).
		Count(&n).Error
	return n, err
}

func (r *LogIntelligenceRepository) GetAlertEventByID(ctx context.Context, id uint) (*model.AlertEvent, error) {
	var ev model.AlertEvent
	if err := r.db.WithContext(ctx).First(&ev, id).Error; err != nil {
		return nil, err
	}
	return &ev, nil
}

func (r *LogIntelligenceRepository) GetAlertEventByFingerprint(ctx context.Context, fingerprint string) (*model.AlertEvent, error) {
	var ev model.AlertEvent
	if err := r.db.WithContext(ctx).
		Where("fingerprint = ?", strings.TrimSpace(fingerprint)).
		Order("created_at DESC").First(&ev).Error; err != nil {
		return nil, err
	}
	return &ev, nil
}

func (r *LogIntelligenceRepository) ListAlertEventsInRange(ctx context.Context, projectID uint, from, to time.Time, limit int) ([]model.AlertEvent, error) {
	var alerts []model.AlertEvent
	q := r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("project_id = ? AND created_at >= ? AND created_at <= ?", projectID, from, to).
		Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&alerts).Error
	return alerts, err
}

func (r *LogIntelligenceRepository) GetChangeEventByID(ctx context.Context, id uint) (*model.ChangeEvent, error) {
	var ch model.ChangeEvent
	if err := r.db.WithContext(ctx).First(&ch, id).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *LogIntelligenceRepository) ListChangeEventsInRange(ctx context.Context, projectID uint, from, to time.Time, limit int) ([]model.ChangeEvent, error) {
	var changes []model.ChangeEvent
	q := r.db.WithContext(ctx).Model(&model.ChangeEvent{}).
		Where("project_id = ? AND started_at >= ? AND started_at <= ?", projectID, from, to).
		Order("started_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&changes).Error
	return changes, err
}
