package repository

import (
	"context"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type EsmgmtRepository struct {
	db *gorm.DB
}

func NewEsmgmtRepository(db *gorm.DB) EsmgmtRepo {
	if db == nil {
		return &EsmgmtRepository{}
	}
	return &EsmgmtRepository{db: db}
}

func (r *EsmgmtRepository) ListConnections(ctx context.Context) ([]model.EsmgmtConnection, error) {
	var list []model.EsmgmtConnection
	if err := r.db.WithContext(ctx).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *EsmgmtRepository) GetConnection(ctx context.Context, id uint) (*model.EsmgmtConnection, error) {
	var row model.EsmgmtConnection
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *EsmgmtRepository) GetConnectionOwner(ctx context.Context, id uint) (*model.EsmgmtConnection, error) {
	var row model.EsmgmtConnection
	if err := r.db.WithContext(ctx).Select("id", "owner_user_id").First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *EsmgmtRepository) FindDictImportConnection(ctx context.Context, remarkMarker, name string) (*model.EsmgmtConnection, error) {
	var existing model.EsmgmtConnection
	err := r.db.WithContext(ctx).
		Where("remark LIKE ? OR name = ?", "%"+remarkMarker+"%", name).
		Order("id ASC").
		First(&existing).Error
	if err != nil {
		return nil, err
	}
	return &existing, nil
}

func (r *EsmgmtRepository) CountDefaultConnections(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.EsmgmtConnection{}).Where("is_default = ?", true).Count(&n).Error
	return n, err
}

func (r *EsmgmtRepository) GetDefaultConnection(ctx context.Context) (*model.EsmgmtConnection, error) {
	var def model.EsmgmtConnection
	if err := r.db.WithContext(ctx).Where("is_default = ?", true).First(&def).Error; err != nil {
		return nil, err
	}
	return &def, nil
}

func (r *EsmgmtRepository) CreateConnectionClearDefaults(ctx context.Context, row *model.EsmgmtConnection) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.IsDefault {
			if err := tx.Model(&model.EsmgmtConnection{}).
				Where("is_default = ?", true).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(row).Error
	})
}

func (r *EsmgmtRepository) UpdateConnectionClearDefaults(ctx context.Context, id uint, row *model.EsmgmtConnection) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.IsDefault {
			if err := tx.Model(&model.EsmgmtConnection{}).
				Where("is_default = ? AND id <> ?", true, id).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(row).Error
	})
}

func (r *EsmgmtRepository) DeleteConnection(ctx context.Context, id uint) (int64, error) {
	res := r.db.WithContext(ctx).Delete(&model.EsmgmtConnection{}, id)
	return res.RowsAffected, res.Error
}

func (r *EsmgmtRepository) CreateBackupJob(ctx context.Context, job *model.EsmgmtBackupJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *EsmgmtRepository) ListBackupJobs(ctx context.Context, connectionID uint, limit int) ([]model.EsmgmtBackupJob, error) {
	q := r.db.WithContext(ctx).Order("id desc").Limit(limit)
	if connectionID > 0 {
		q = q.Where("connection_id = ?", connectionID)
	}
	var list []model.EsmgmtBackupJob
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *EsmgmtRepository) GetBackupJob(ctx context.Context, id uint) (*model.EsmgmtBackupJob, error) {
	var job model.EsmgmtBackupJob
	if err := r.db.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *EsmgmtRepository) UpdateBackupJobFields(ctx context.Context, id uint, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.EsmgmtBackupJob{}).Where("id = ?", id).Updates(fields).Error
}

func (r *EsmgmtRepository) CountRunningBackupJobs(ctx context.Context, connectionID uint, indexName string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.EsmgmtBackupJob{}).
		Where("connection_id = ? AND index_name = ? AND status IN ?", connectionID, indexName, []string{"pending", "running"}).
		Count(&n).Error
	return n, err
}

func (r *EsmgmtRepository) CreateRestoreJob(ctx context.Context, job *model.EsmgmtRestoreJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *EsmgmtRepository) ListRestoreJobs(ctx context.Context, connectionID uint, limit int) ([]model.EsmgmtRestoreJob, error) {
	q := r.db.WithContext(ctx).Order("id desc").Limit(limit)
	if connectionID > 0 {
		q = q.Where("connection_id = ?", connectionID)
	}
	var list []model.EsmgmtRestoreJob
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *EsmgmtRepository) GetRestoreJob(ctx context.Context, id uint) (*model.EsmgmtRestoreJob, error) {
	var job model.EsmgmtRestoreJob
	if err := r.db.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *EsmgmtRepository) UpdateRestoreJobFields(ctx context.Context, id uint, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.EsmgmtRestoreJob{}).Where("id = ?", id).Updates(fields).Error
}

func (r *EsmgmtRepository) CreateSchedule(ctx context.Context, row *model.EsmgmtBackupSchedule) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *EsmgmtRepository) GetSchedule(ctx context.Context, id uint) (*model.EsmgmtBackupSchedule, error) {
	var row model.EsmgmtBackupSchedule
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *EsmgmtRepository) SaveSchedule(ctx context.Context, row *model.EsmgmtBackupSchedule) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *EsmgmtRepository) DeleteSchedule(ctx context.Context, id uint) (int64, error) {
	res := r.db.WithContext(ctx).Delete(&model.EsmgmtBackupSchedule{}, id)
	return res.RowsAffected, res.Error
}

func (r *EsmgmtRepository) ListSchedules(ctx context.Context, connectionID uint) ([]model.EsmgmtBackupSchedule, error) {
	q := r.db.WithContext(ctx).Order("id desc")
	if connectionID > 0 {
		q = q.Where("connection_id = ?", connectionID)
	}
	var list []model.EsmgmtBackupSchedule
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *EsmgmtRepository) ListEnabledSchedules(ctx context.Context) ([]model.EsmgmtBackupSchedule, error) {
	var list []model.EsmgmtBackupSchedule
	if err := r.db.WithContext(ctx).
		Where("enabled = ? AND cron_spec <> ?", true, "").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *EsmgmtRepository) UpdateScheduleLastScheduledAt(ctx context.Context, id uint, at time.Time) error {
	return r.db.WithContext(ctx).Model(&model.EsmgmtBackupSchedule{}).
		Where("id = ?", id).
		Update("last_scheduled_at", at).Error
}

func (r *EsmgmtRepository) CreateReindexJob(ctx context.Context, job *model.EsmgmtReindexJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *EsmgmtRepository) ListReindexJobs(ctx context.Context, connectionID uint, limit int) ([]model.EsmgmtReindexJob, error) {
	q := r.db.WithContext(ctx).Order("id desc").Limit(limit)
	if connectionID > 0 {
		q = q.Where("connection_id = ?", connectionID)
	}
	var list []model.EsmgmtReindexJob
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *EsmgmtRepository) GetReindexJob(ctx context.Context, id uint) (*model.EsmgmtReindexJob, error) {
	var job model.EsmgmtReindexJob
	if err := r.db.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *EsmgmtRepository) UpdateReindexJobFields(ctx context.Context, id uint, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.EsmgmtReindexJob{}).Where("id = ?", id).Updates(fields).Error
}

func (r *EsmgmtRepository) CountRunningReindexJobs(ctx context.Context, connectionID uint, source, dest string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.EsmgmtReindexJob{}).
		Where("connection_id = ? AND source_index = ? AND dest_index = ? AND status IN ?", connectionID, source, dest, []string{"pending", "running"}).
		Count(&n).Error
	return n, err
}
