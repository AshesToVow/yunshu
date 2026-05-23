package alert

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

type AlertMaintenanceListQuery struct {
	ProjectID uint   `form:"projectId"`
	Keyword   string `form:"keyword"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type AlertMaintenanceUpsertRequest struct {
	Name         string    `json:"name" binding:"required,max=128"`
	MatchersJSON string    `json:"matchers_json" binding:"required"`
	StartsAt     time.Time `json:"starts_at" binding:"required"`
	EndsAt       time.Time `json:"ends_at" binding:"required"`
	Comment      string    `json:"comment" binding:"omitempty,max=512"`
	ProjectID    uint      `json:"project_id"`
	Enabled      *bool     `json:"enabled"`
}

type AlertMaintenanceService struct {
	repo interfaces.AlertMaintenanceWindowRepository
}

func NewAlertMaintenanceService(repo interfaces.AlertMaintenanceWindowRepository) *AlertMaintenanceService {
	return &AlertMaintenanceService{repo: repo}
}

func (s *AlertMaintenanceService) List(ctx context.Context, q AlertMaintenanceListQuery) ([]model.AlertMaintenanceWindow, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.repo.ListPaged(ctx, q.ProjectID, q.Keyword, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.maintenance", "List", err)
	}
	return list, total, page, pageSize, nil
}

func (s *AlertMaintenanceService) Create(ctx context.Context, userID uint, req AlertMaintenanceUpsertRequest) (*model.AlertMaintenanceWindow, error) {
	if _, err := ParseSilenceMatchersJSON(req.MatchersJSON); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.maintenance", "Create", err)
	}
	if !req.EndsAt.After(req.StartsAt) {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgc1f741f96c03)
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := model.AlertMaintenanceWindow{
		Name:         strings.TrimSpace(req.Name),
		MatchersJSON: strings.TrimSpace(req.MatchersJSON),
		StartsAt:     req.StartsAt,
		EndsAt:       req.EndsAt,
		Comment:      strings.TrimSpace(req.Comment),
		ProjectID:    req.ProjectID,
		Enabled:      enabled,
		CreatedBy:    userID,
	}
	if err := s.repo.Create(ctx, &row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.maintenance", "Create", err)
	}
	return &row, nil
}

func (s *AlertMaintenanceService) Update(ctx context.Context, id uint, req AlertMaintenanceUpsertRequest) (*model.AlertMaintenanceWindow, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsgde63e900b907)
		}
		return nil, bizerrors.Pass(ctx, "alert.maintenance", "Update", err)
	}
	if _, err := ParseSilenceMatchersJSON(req.MatchersJSON); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.maintenance", "Update", err)
	}
	if !req.EndsAt.After(req.StartsAt) {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgc1f741f96c03)
	}
	row.Name = strings.TrimSpace(req.Name)
	row.MatchersJSON = strings.TrimSpace(req.MatchersJSON)
	row.StartsAt = req.StartsAt
	row.EndsAt = req.EndsAt
	row.Comment = strings.TrimSpace(req.Comment)
	if req.ProjectID > 0 {
		row.ProjectID = req.ProjectID
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.repo.Save(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.maintenance", "Update", err)
	}
	return row, nil
}

func (s *AlertMaintenanceService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return bizerrors.Pass(ctx, "alert.maintenance", "Delete", err)
	}
	return nil
}

func (s *AlertMaintenanceService) FirstMatchingID(ctx context.Context, labels map[string]string, t time.Time) (uint, bool, error) {
	list, err := s.repo.ListActiveAt(ctx, t)
	if err != nil {
		return 0, false, bizerrors.Pass(ctx, "alert.maintenance", "FirstMatchingID", err)
	}
	for _, row := range list {
		ms, err := ParseSilenceMatchersJSON(row.MatchersJSON)
		if err != nil {
			continue
		}
		if LabelsMatchSilenceMatchers(ms, labels) {
			return row.ID, true, nil
		}
	}
	return 0, false, nil
}
