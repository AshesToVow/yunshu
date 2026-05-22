package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// AlertChannelListFilter filters channel list queries.
type AlertChannelListFilter struct {
	Keyword string
}

// AlertEventListFilter filters alert event history list queries.
type AlertEventListFilter struct {
	Keyword         string
	Cluster         string
	AlertIP         string
	Status          string
	MonitorPipeline string
	DatasourceID    uint
	ProjectID       uint
	GroupKey        string
	Category        string
}

// AlertHistoryStatsRow is aggregated history dashboard stats.
type AlertHistoryStatsRow struct {
	Total                   int64
	Firing                  int64
	Resolved                int64
	Success                 int64
	Failed                  int64
	TodayCreated            int64
	ClusterValues           []string
	MonitorPipelineValues   []string
	DatasourceFilterOptions []AlertDatasourceFilterOption
}

// AlertDatasourceFilterOption is a datasource option for history filters.
type AlertDatasourceFilterOption struct {
	ID   uint   `gorm:"column:id"`
	Name string `gorm:"column:name"`
}

// AlertEventRepo is implemented by *AlertEventRepository.
type AlertEventRepo interface {
	Create(ctx context.Context, event *model.AlertEvent) error
	GetByFingerprint(ctx context.Context, fingerprint string) (*model.AlertEvent, error)
	UpdateStatus(ctx context.Context, fingerprint, status string) error
	List(ctx context.Context, f AlertEventListFilter, offset, limit int) ([]model.AlertEvent, int64, error)
	ListFiringByGroupKeys(ctx context.Context, groupKeys []string) ([]model.AlertEvent, error)
	HistoryStats(ctx context.Context, dayStart, dayEnd time.Time) (*AlertHistoryStatsRow, error)
}

// AlertChannelRepo is implemented by *AlertChannelRepository.
type AlertChannelRepo interface {
	ListEnabled(ctx context.Context) ([]*model.AlertChannel, error)
	List(ctx context.Context, f AlertChannelListFilter) ([]model.AlertChannel, error)
	GetByID(ctx context.Context, id uint) (*model.AlertChannel, error)
	Create(ctx context.Context, ch *model.AlertChannel) error
	Save(ctx context.Context, ch *model.AlertChannel) error
	Delete(ctx context.Context, id uint) error
}
