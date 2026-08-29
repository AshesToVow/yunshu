package repository

import (
	"context"

	"yunshu/internal/model"
)

// ProjectRepo is implemented by *ProjectRepository.
type ProjectRepo interface {
	Create(ctx context.Context, p *model.Project) (error)
	Save(ctx context.Context, p *model.Project) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	GetByID(ctx context.Context, id uint) (*model.Project, error)
	List(ctx context.Context, params ProjectListParams) ([]model.Project, int64, error)
	ListVisibleToUser(ctx context.Context, userID uint, params ProjectListParams) ([]model.Project, int64, error)
}

var _ ProjectRepo = (*ProjectRepository)(nil)

// ProjectMemberRepo is implemented by *ProjectMemberRepository.
type ProjectMemberRepo interface {
	ListByProject(ctx context.Context, projectID uint) ([]model.ProjectMember, error)
	ListUserIDsByProject(ctx context.Context, projectID uint) ([]uint, error)
	DeleteByUserID(ctx context.Context, userID uint) (error)
	ListRolesByUserAndProjectIDs(ctx context.Context, userID uint, projectIDs []uint) (map[uint]string, error)
	ListProjectIDsByUser(ctx context.Context, userID uint) ([]uint, error)
	GetByID(ctx context.Context, id uint) (*model.ProjectMember, error)
	GetByProjectAndUser(ctx context.Context, projectID uint, userID uint) (*model.ProjectMember, error)
	Create(ctx context.Context, row *model.ProjectMember) (error)
	Save(ctx context.Context, row *model.ProjectMember) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	DeleteByProject(ctx context.Context, projectID uint) (error)
	ListDisplayByProject(ctx context.Context, projectID uint) ([]ProjectMemberListRow, error)
}

var _ ProjectMemberRepo = (*ProjectMemberRepository)(nil)

