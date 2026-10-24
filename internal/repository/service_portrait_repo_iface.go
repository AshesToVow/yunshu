package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// ServicePortraitRepo aggregates read-only queries for service catalog portrait/health.
type ServicePortraitRepo interface {
	GetCicdService(ctx context.Context, projectID, id uint) (*model.CicdService, error)
	LatestReleaseRun(ctx context.Context, projectID, cicdServiceID uint) (*model.CicdReleaseRun, error)
	CountReleaseRunsSince(ctx context.Context, projectID, cicdServiceID uint, since time.Time) (total, failedOrCancelled int64, err error)
	CountFiringAlertsSince(ctx context.Context, severity string, since time.Time) (int64, error)
	CountFailedChangesSince(ctx context.Context, projectID, catalogServiceID uint, source string, since time.Time) (int64, error)
	CountOpenAnomalies(ctx context.Context, projectID uint, severity string) (int64, error)
	CountLogPatterns(ctx context.Context, projectID uint) (int64, error)
	ListRecentOpenAnomalies(ctx context.Context, projectID uint, limit int) ([]model.LogAnomaly, error)
}

var _ ServicePortraitRepo = (*ServicePortraitRepository)(nil)
