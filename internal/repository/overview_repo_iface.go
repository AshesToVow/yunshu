package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

type OverviewMetricsRow struct {
	UsersCount                int64
	ClustersCount             int64
	PendingRegistrationsCount int64
	ServersCount              int64
}

type OverviewStats struct {
	AlertFiringCount      int64
	AlertEventsTodayCount int64
	LoggieAgentsOnlineCount  int64
	LoggieAgentsOfflineCount int64
}

type OverviewPersonCount struct {
	Person string
	Cnt    int64
}

type OverviewProjectDayCount struct {
	ProjectID   uint
	ProjectName string
	Day         string
	Cnt         int64
}

type OverviewLabelCount struct {
	Label string
	Cnt   int64
}

type OverviewAlertBrief struct {
	ID          uint
	Fingerprint string
	Alertname   string
	Severity    string
	Cluster     string
	ProjectID   uint
	Summary     string
	StartsAt    time.Time
}

type OverviewReleaseBrief struct {
	ID            uint
	ProjectID     uint
	ProjectName   string
	Title         string
	Status        string
	Tenv          string
	SubmitterName string
	FinishedAt    *time.Time
	CreatedAt     time.Time
}

type OverviewChangeBrief struct {
	ID        uint
	ProjectID uint
	Source    string
	Action    string
	RiskLevel string
	Status    string
	Summary   string
	StartedAt time.Time
}

type OverviewLoggieBrief struct {
	ID           uint
	ProjectID    uint
	ServerID     uint
	HealthStatus string
	LastError    string
	LastSeenAt   *time.Time
}

type OverviewRepo interface {
	DialectName() string
	LoadMetrics(ctx context.Context, regPendingStatus int) (*OverviewMetricsRow, error)
	ListEnabledClusters(ctx context.Context, projectIDs []uint, unrestricted bool) ([]model.K8sCluster, error)
	FillAlertAndAgentStats(ctx context.Context, dayStart, dayEnd, agentCutoff time.Time) (*OverviewStats, error)
	CountReleaseLaunchesByProjectDay(ctx context.Context, start, end time.Time, projectIDs []uint, unrestricted bool) ([]OverviewProjectDayCount, error)
	CountReleaseRunsByPerson(ctx context.Context, start, end time.Time, projectIDs []uint, unrestricted bool) ([]OverviewPersonCount, error)

	// Screen panels
	CountServersByStatus(ctx context.Context) ([]OverviewLabelCount, error)
	CountAlertsBySeverity(ctx context.Context, projectIDs []uint, unrestricted bool) ([]OverviewLabelCount, error)
	CountFiringAlerts(ctx context.Context, projectIDs []uint, unrestricted bool) (int64, error)
	ListFiringAlertsTop(ctx context.Context, projectIDs []uint, unrestricted bool, limit int) ([]OverviewAlertBrief, error)
	ListRecentReleases(ctx context.Context, projectIDs []uint, unrestricted bool, limit int) ([]OverviewReleaseBrief, error)
	ListRecentChanges(ctx context.Context, projectIDs []uint, unrestricted bool, limit int) ([]OverviewChangeBrief, error)
	CountLoggieByHealth(ctx context.Context) ([]OverviewLabelCount, error)
	ListLoggieOfflineSample(ctx context.Context, agentCutoff time.Time, limit int) ([]OverviewLoggieBrief, error)
	CountAIInvestigations(ctx context.Context, dayStart, dayEnd time.Time) (today int64, open int64, err error)
}
