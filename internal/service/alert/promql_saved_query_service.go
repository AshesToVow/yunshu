package alert

import (
	"context"
	"strings"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
)

type PromqlSavedQueryUpsertRequest struct {
	Name         string `json:"name" binding:"required,max=128"`
	Query        string `json:"query" binding:"required"`
	DatasourceID uint   `json:"datasource_id"`
	Kind         string `json:"kind"`
	ProjectID    uint   `json:"project_id"`
}

type PromqlSavedQueryService struct {
	repo interfaces.PromqlSavedQueryRepository
}

func NewPromqlSavedQueryService(repo interfaces.PromqlSavedQueryRepository) *PromqlSavedQueryService {
	return &PromqlSavedQueryService{repo: repo}
}

func (s *AlertService) ListPromqlSavedQueries(ctx context.Context, userID uint) ([]model.PlatformSavedQuery, error) {
	return NewPromqlSavedQueryService(s.promqlSavedQueryRepo).List(ctx, userID)
}

func (s *AlertService) CreatePromqlSavedQuery(ctx context.Context, userID uint, req PromqlSavedQueryUpsertRequest) (*model.PlatformSavedQuery, error) {
	return NewPromqlSavedQueryService(s.promqlSavedQueryRepo).Create(ctx, userID, req)
}

func (s *AlertService) DeletePromqlSavedQuery(ctx context.Context, userID, id uint) error {
	return NewPromqlSavedQueryService(s.promqlSavedQueryRepo).Delete(ctx, userID, id)
}

func (s *PromqlSavedQueryService) List(ctx context.Context, userID uint) ([]model.PlatformSavedQuery, error) {
	if s == nil || s.repo == nil || userID == 0 {
		return nil, nil
	}
	list, err := s.repo.ListByUser(ctx, userID)
	return list, bizerrors.Pass(ctx, "alert.promql_saved", "List", err)
}

func (s *PromqlSavedQueryService) Create(ctx context.Context, userID uint, req PromqlSavedQueryUpsertRequest) (*model.PlatformSavedQuery, error) {
	if userID == 0 {
		return nil, constants.ErrUnauthorized
	}
	kind := strings.TrimSpace(req.Kind)
	if kind == "" {
		kind = "instant"
	}
	row := model.PlatformSavedQuery{
		UserID: userID, Name: strings.TrimSpace(req.Name), Query: strings.TrimSpace(req.Query),
		DatasourceID: req.DatasourceID, Kind: kind, ProjectID: req.ProjectID,
	}
	if err := s.repo.Create(ctx, &row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.promql_saved", "Create", err)
	}
	return &row, nil
}

func (s *PromqlSavedQueryService) Delete(ctx context.Context, userID, id uint) error {
	rows, err := s.repo.DeleteByUser(ctx, userID, id)
	if err != nil {
		return bizerrors.Pass(ctx, "alert.promql_saved", "Delete", err)
	}
	if rows == 0 {
		return constants.ErrNotFound
	}
	return nil
}
