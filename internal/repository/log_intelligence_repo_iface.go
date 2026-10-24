package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// LogPatternListFilter filters log pattern list queries.
type LogPatternListFilter struct {
	ProjectID   uint
	Level       string
	ServiceName string
	Keyword     string
	Offset      int
	Limit       int
}

// LogAnomalyListFilter filters log anomaly list queries.
type LogAnomalyListFilter struct {
	ProjectID   uint
	Status      string
	AnomalyType string
	AssigneeID  *uint
	Offset      int
	Limit       int
}

// LogIntelligenceRepo is implemented by *LogIntelligenceRepository.
type LogIntelligenceRepo interface {
	ListPatterns(ctx context.Context, f LogPatternListFilter) ([]model.LogPattern, int64, error)
	GetPatternBySignature(ctx context.Context, projectID uint, signature string) (*model.LogPattern, error)
	CreatePattern(ctx context.Context, row *model.LogPattern) error
	SavePattern(ctx context.Context, row *model.LogPattern) error

	ListAnomalies(ctx context.Context, f LogAnomalyListFilter) ([]model.LogAnomaly, int64, error)
	GetAnomalyByIDInProject(ctx context.Context, projectID, anomalyID uint) (*model.LogAnomaly, error)
	CreateAnomaly(ctx context.Context, row *model.LogAnomaly) error
	SaveAnomaly(ctx context.Context, row *model.LogAnomaly) error
	CountAnomaliesByTypeSignature(ctx context.Context, projectID uint, anomalyType, signature string) (int64, error)
	GetRecentSpikeAnomaly(ctx context.Context, projectID uint, since time.Time) (*model.LogAnomaly, error)
	CountOpenAnomalies(ctx context.Context, projectID uint) (int64, error)

	GetAlertEventByID(ctx context.Context, id uint) (*model.AlertEvent, error)
	GetAlertEventByFingerprint(ctx context.Context, fingerprint string) (*model.AlertEvent, error)
	ListAlertEventsInRange(ctx context.Context, projectID uint, from, to time.Time, limit int) ([]model.AlertEvent, error)
	GetChangeEventByID(ctx context.Context, id uint) (*model.ChangeEvent, error)
	ListChangeEventsInRange(ctx context.Context, projectID uint, from, to time.Time, limit int) ([]model.ChangeEvent, error)
}

var _ LogIntelligenceRepo = (*LogIntelligenceRepository)(nil)
