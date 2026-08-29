package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// LogSourceRepo is implemented by *LogSourceRepository.
type LogSourceRepo interface {
	GetByID(ctx context.Context, id uint) (*model.ServiceLogSource, error)
	GetByIDInProject(ctx context.Context, projectID uint, id uint) (*model.ServiceLogSource, error)
	Create(ctx context.Context, it *model.ServiceLogSource) (error)
	Save(ctx context.Context, it *model.ServiceLogSource) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	List(ctx context.Context, p LogSourceListParams) ([]model.ServiceLogSource, int64, error)
	BelongsToProjectServer(ctx context.Context, projectID uint, serverID uint, logSourceID uint) (bool, error)
	ListByProjectAndServer(ctx context.Context, projectID uint, serverID uint) ([]model.ServiceLogSource, error)
}

var _ LogSourceRepo = (*LogSourceRepository)(nil)

// LogRetentionRepo is implemented by *LogRetentionRepository.
type LogRetentionRepo interface {
	GetByScope(ctx context.Context, projectID, serverID uint) (*model.LogRetentionPolicy, error)
	List(ctx context.Context) ([]model.LogRetentionPolicy, error)
	Save(ctx context.Context, it *model.LogRetentionPolicy) error
	DeleteByScope(ctx context.Context, projectID, serverID uint) error
}

var _ LogRetentionRepo = (*LogRetentionRepository)(nil)

// LoggieAgentRepo is implemented by *LoggieAgentRepository.
type LoggieAgentRepo interface {
	GetByToken(ctx context.Context, token string) (*model.LoggieAgent, error)
	GetByProjectAndServer(ctx context.Context, projectID, serverID uint) (*model.LoggieAgent, error)
	ListByProject(ctx context.Context, projectID uint) ([]model.LoggieAgent, error)
	Save(ctx context.Context, it *model.LoggieAgent) error
	DeleteByProjectAndServer(ctx context.Context, projectID, serverID uint) error
	TouchSeen(ctx context.Context, id uint, at time.Time) error
}

var _ LoggieAgentRepo = (*LoggieAgentRepository)(nil)

