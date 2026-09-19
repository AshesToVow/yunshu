package repository

import (
	"context"

	"yunshu/internal/model"
)

// LogPipelineRepo is implemented by *LogPipelineRepository.
type LogPipelineRepo interface {
	List(ctx context.Context, projectID uint, kind string) ([]model.LogPipeline, error)
	GetByIDInProject(ctx context.Context, projectID, id uint) (*model.LogPipeline, error)
	GetByProjectKindClusterName(ctx context.Context, projectID uint, kind string, clusterID uint, name string) (*model.LogPipeline, error)
	GetHostByServer(ctx context.Context, projectID, serverID uint) (*model.LogPipeline, error)
	GetUnscopedByProjectAndName(ctx context.Context, projectID uint, name string) (*model.LogPipeline, error)
	GetUnscopedHostByName(ctx context.Context, projectID uint, name string) (*model.LogPipeline, error)
	Create(ctx context.Context, row *model.LogPipeline) error
	Save(ctx context.Context, row *model.LogPipeline) error
	UnscopedSave(ctx context.Context, row *model.LogPipeline) error
	DeleteByIDInProject(ctx context.Context, projectID, id uint) (rowsAffected int64, err error)

	CreateVersion(ctx context.Context, ver *model.LogPipelineVersion) error
	ListVersions(ctx context.Context, projectID, pipelineID uint, limit int) ([]model.LogPipelineVersion, error)
	GetVersion(ctx context.Context, projectID, pipelineID, versionID uint) (*model.LogPipelineVersion, error)
}

var _ LogPipelineRepo = (*LogPipelineRepository)(nil)
