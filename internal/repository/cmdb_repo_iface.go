package repository

import (
	"context"

	"yunshu/internal/model"
)

// ServerRepo is implemented by *ServerRepository.
type ServerRepo interface {
	Create(ctx context.Context, s *model.Server) (error)
	Save(ctx context.Context, s *model.Server) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	GetByID(ctx context.Context, id uint) (*model.Server, error)
	ProjectNameByID(ctx context.Context, projectID uint) (string, error)
	List(ctx context.Context, params ServerListParams) ([]model.Server, int64, error)
	GetByProjectProviderInstance(ctx context.Context, projectID uint, provider string, cloudInstanceID string) (*model.Server, error)
	ListByProjectWithoutGroup(ctx context.Context, projectID uint) ([]model.Server, error)
	ListByProjectGroupProvider(ctx context.Context, projectID uint, groupID uint, provider string) ([]model.Server, error)
	ListByProjectProviderCloud(ctx context.Context, projectID uint, provider string) ([]model.Server, error)
	UpsertCredential(ctx context.Context, cred *model.ServerCredential) (error)
	GetCredentialByServerID(ctx context.Context, serverID uint) (*model.ServerCredential, error)
}

var _ ServerRepo = (*ServerRepository)(nil)

// ServerGroupRepo is implemented by *ServerGroupRepository.
type ServerGroupRepo interface {
	Create(ctx context.Context, item *model.ServerGroup) (error)
	Save(ctx context.Context, item *model.ServerGroup) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	GetByID(ctx context.Context, id uint) (*model.ServerGroup, error)
	ListByProject(ctx context.Context, projectID uint) ([]model.ServerGroup, error)
}

var _ ServerGroupRepo = (*ServerGroupRepository)(nil)

// CloudAccountRepo is implemented by *CloudAccountRepository.
type CloudAccountRepo interface {
	Create(ctx context.Context, item *model.CloudAccount) (error)
	Save(ctx context.Context, item *model.CloudAccount) (error)
	GetByID(ctx context.Context, id uint) (*model.CloudAccount, error)
	ListByProjectAndGroup(ctx context.Context, projectID uint, groupID *uint) ([]model.CloudAccount, error)
	ListEnabledByProject(ctx context.Context, projectID uint, provider string) ([]model.CloudAccount, error)
	DeleteByID(ctx context.Context, id uint) (error)
}

var _ CloudAccountRepo = (*CloudAccountRepository)(nil)

// ServiceRepo is implemented by *ServiceRepository.
type ServiceRepo interface {
	GetByID(ctx context.Context, id uint) (*model.Service, error)
	Create(ctx context.Context, s *model.Service) (error)
	Save(ctx context.Context, s *model.Service) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	List(ctx context.Context, p ServiceListParams) ([]model.Service, int64, error)
}

var _ ServiceRepo = (*ServiceRepository)(nil)

