package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// EsmgmtRepo is implemented by *EsmgmtRepository.
type EsmgmtRepo interface {
	ListConnections(ctx context.Context) ([]model.EsmgmtConnection, error)
	GetConnection(ctx context.Context, id uint) (*model.EsmgmtConnection, error)
	GetConnectionOwner(ctx context.Context, id uint) (*model.EsmgmtConnection, error)
	FindDictImportConnection(ctx context.Context, remarkMarker, name string) (*model.EsmgmtConnection, error)
	CountDefaultConnections(ctx context.Context) (int64, error)
	GetDefaultConnection(ctx context.Context) (*model.EsmgmtConnection, error)
	CreateConnectionClearDefaults(ctx context.Context, row *model.EsmgmtConnection) error
	UpdateConnectionClearDefaults(ctx context.Context, id uint, row *model.EsmgmtConnection) error
	DeleteConnection(ctx context.Context, id uint) (int64, error)

	CreateBackupJob(ctx context.Context, job *model.EsmgmtBackupJob) error
	ListBackupJobs(ctx context.Context, connectionID uint, limit int) ([]model.EsmgmtBackupJob, error)
	GetBackupJob(ctx context.Context, id uint) (*model.EsmgmtBackupJob, error)
	UpdateBackupJobFields(ctx context.Context, id uint, fields map[string]any) error
	CountRunningBackupJobs(ctx context.Context, connectionID uint, indexName string) (int64, error)

	CreateRestoreJob(ctx context.Context, job *model.EsmgmtRestoreJob) error
	ListRestoreJobs(ctx context.Context, connectionID uint, limit int) ([]model.EsmgmtRestoreJob, error)
	GetRestoreJob(ctx context.Context, id uint) (*model.EsmgmtRestoreJob, error)
	UpdateRestoreJobFields(ctx context.Context, id uint, fields map[string]any) error

	CreateSchedule(ctx context.Context, row *model.EsmgmtBackupSchedule) error
	GetSchedule(ctx context.Context, id uint) (*model.EsmgmtBackupSchedule, error)
	SaveSchedule(ctx context.Context, row *model.EsmgmtBackupSchedule) error
	DeleteSchedule(ctx context.Context, id uint) (int64, error)
	ListSchedules(ctx context.Context, connectionID uint) ([]model.EsmgmtBackupSchedule, error)
	ListEnabledSchedules(ctx context.Context) ([]model.EsmgmtBackupSchedule, error)
	UpdateScheduleLastScheduledAt(ctx context.Context, id uint, at time.Time) error
}

var _ EsmgmtRepo = (*EsmgmtRepository)(nil)
