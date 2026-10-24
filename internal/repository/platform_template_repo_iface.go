package repository

import (
	"context"

	"yunshu/internal/model"
)

// PlatformTemplateListParams filters platform template listing.
type PlatformTemplateListParams struct {
	Category string
	Keyword  string
	Status   *int
	Offset   int
	Limit    int
}

// PlatformTemplateRepo is implemented by *PlatformTemplateRepository.
type PlatformTemplateRepo interface {
	List(ctx context.Context, p PlatformTemplateListParams) ([]model.PlatformTemplate, int64, error)
	GetByID(ctx context.Context, id uint) (*model.PlatformTemplate, error)
	GetByKeyEnabled(ctx context.Context, templateKey string) (*model.PlatformTemplate, error)
	Create(ctx context.Context, row *model.PlatformTemplate) error
	Save(ctx context.Context, row *model.PlatformTemplate) error
	Delete(ctx context.Context, row *model.PlatformTemplate) error
	DeleteVersionsByTemplateID(ctx context.Context, templateID uint) error
	MaxVersion(ctx context.Context, templateID uint) (int, error)
	CreateVersion(ctx context.Context, ver *model.PlatformTemplateVersion) error
	GetVersion(ctx context.Context, templateID uint, version int) (*model.PlatformTemplateVersion, error)
	GetLatestVersion(ctx context.Context, templateID uint) (*model.PlatformTemplateVersion, error)
	UpdateVersionStorageKey(ctx context.Context, versionID uint, storageKey string) error
	UpdatePublishedVersion(ctx context.Context, templateID uint, version int) error
	ListVersions(ctx context.Context, templateID uint) ([]model.PlatformTemplateVersion, error)
	Transaction(ctx context.Context, fn func(PlatformTemplateRepo) error) error
}

var _ PlatformTemplateRepo = (*PlatformTemplateRepository)(nil)
