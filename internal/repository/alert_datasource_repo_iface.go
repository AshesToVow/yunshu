package repository

import (
	"context"

	"yunshu/internal/model"
)

// AlertDatasourceListFilter filters datasource list queries.
type AlertDatasourceListFilter struct {
	ProjectID uint
	Keyword   string
}

// AlertDatasourceListRow includes joined project name for list APIs.
type AlertDatasourceListRow struct {
	model.AlertDatasource
	ProjectName string `gorm:"column:project_name"`
}

// AlertDatasourceRepo is implemented by *AlertDatasourceRepository.
type AlertDatasourceRepo interface {
	ListWithProject(ctx context.Context, f AlertDatasourceListFilter, offset, limit int) ([]AlertDatasourceListRow, int64, error)
	GetByID(ctx context.Context, id uint) (*model.AlertDatasource, error)
	Create(ctx context.Context, row *model.AlertDatasource) error
	Save(ctx context.Context, row *model.AlertDatasource) error
	Delete(ctx context.Context, id uint) (rowsAffected int64, err error)
}
