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
	Severity        string // 单值或逗号分隔：critical,warning（P1/P2）
	MonitorPipeline string
	DatasourceID    uint
	ProjectID       uint
	GroupKey        string
	Fingerprint     string
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

// AlertEventGroupRow 按 group_key 聚合的查询行。
type AlertEventGroupRow struct {
	GroupKey   string    `gorm:"column:group_key"`
	Title      string    `gorm:"column:title"`
	Count      int64     `gorm:"column:cnt"`
	LastAt     time.Time `gorm:"column:last_at"`
	Status     string    `gorm:"column:status"`
	Severity   string    `gorm:"column:severity"`
	Cluster    string    `gorm:"column:cluster"`
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
	ListGroupedByGroupKey(ctx context.Context, f AlertEventListFilter, offset, limit int) ([]AlertEventGroupRow, int64, error)
	ListFiringByGroupKeys(ctx context.Context, groupKeys []string) ([]model.AlertEvent, error)
	HistoryStats(ctx context.Context, projectID uint, dayStart, dayEnd time.Time) (*AlertHistoryStatsRow, error)
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
