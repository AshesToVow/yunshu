package alert

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

type PromqlSavedQueryUpsertRequest struct {
	Name         string `json:"name" binding:"required,max=128"`
	Query        string `json:"query" binding:"required"`
	DatasourceID uint   `json:"datasource_id"`
	Kind         string `json:"kind"`
	ProjectID    uint   `json:"project_id"`
}

type PromqlSavedQueryService struct {
	db *gorm.DB
}

func NewPromqlSavedQueryService(db *gorm.DB) *PromqlSavedQueryService {
	return &PromqlSavedQueryService{db: db}
}

func (s *AlertService) ListPromqlSavedQueries(ctx context.Context, userID uint) ([]model.PlatformSavedQuery, error) {
	return NewPromqlSavedQueryService(s.db).List(ctx, userID)
}

func (s *AlertService) CreatePromqlSavedQuery(ctx context.Context, userID uint, req PromqlSavedQueryUpsertRequest) (*model.PlatformSavedQuery, error) {
	return NewPromqlSavedQueryService(s.db).Create(ctx, userID, req)
}

func (s *AlertService) DeletePromqlSavedQuery(ctx context.Context, userID, id uint) error {
	return NewPromqlSavedQueryService(s.db).Delete(ctx, userID, id)
}

func (s *PromqlSavedQueryService) List(ctx context.Context, userID uint) ([]model.PlatformSavedQuery, error) {
	if s == nil || s.db == nil || userID == 0 {
		return nil, nil
	}
	var list []model.PlatformSavedQuery
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("id DESC").Find(&list).Error
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
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "alert.promql_saved", "Create", err)
	}
	return &row, nil
}

func (s *PromqlSavedQueryService) Delete(ctx context.Context, userID, id uint) error {
	res := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&model.PlatformSavedQuery{})
	if res.Error != nil {
		return bizerrors.Pass(ctx, "alert.promql_saved", "Delete", res.Error)
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFound
	}
	return nil
}
