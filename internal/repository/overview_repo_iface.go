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
	LogAgentsOnlineCount  int64
	LogAgentsOfflineCount int64
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

type OverviewRepo interface {
	DialectName() string
	LoadMetrics(ctx context.Context, regPendingStatus int) (*OverviewMetricsRow, error)
	ListEnabledClusters(ctx context.Context, projectIDs []uint, unrestricted bool) ([]model.K8sCluster, error)
	FillAlertAndAgentStats(ctx context.Context, dayStart, dayEnd, agentCutoff time.Time) (*OverviewStats, error)
	CountReleaseLaunchesByProjectDay(ctx context.Context, start, end time.Time, projectIDs []uint, unrestricted bool) ([]OverviewProjectDayCount, error)
	CountReleaseRunsByPerson(ctx context.Context, start, end time.Time, projectIDs []uint, unrestricted bool) ([]OverviewPersonCount, error)
}
