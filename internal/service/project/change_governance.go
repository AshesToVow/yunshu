package project

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/service/changegate"

	"gorm.io/gorm"
)

type FreezeWindowListQuery struct {
	ProjectID uint `form:"-"`
	Page      int  `form:"page"`
	PageSize  int  `form:"page_size"`
	Enabled   *bool `form:"enabled"`
}

type FreezeWindowUpsertRequest struct {
	ProjectID uint   `json:"-"`
	ID        uint   `json:"id"`
	Name      string `json:"name" binding:"required,max=128"`
	Scope     string `json:"scope" binding:"omitempty,max=32"`
	Env       string `json:"env" binding:"omitempty,max=32"`
	StartsAt  string `json:"starts_at" binding:"required"`
	EndsAt    string `json:"ends_at" binding:"required"`
	Reason    string `json:"reason" binding:"omitempty,max=512"`
	Enabled   *bool  `json:"enabled"`
	CreatedBy uint   `json:"-"`
}

type ConflictCheckQuery struct {
	ProjectID uint   `form:"-"`
	Source    string `form:"source"`
	Env       string `form:"env"`
	ServiceID *uint  `form:"service_id"`
	Namespace string `form:"namespace"`
	Action    string `form:"action"`
}

type ConflictCheckResult struct {
	changegate.CheckResult
	ActiveFreezes []model.ChangeFreezeWindow `json:"active_freezes"`
}

func (s *ChangeEventService) ListFreezeWindows(ctx context.Context, q FreezeWindowListQuery) (*pagination.Result[model.ChangeFreezeWindow], error) {
	if err := s.ensureProject(ctx, q.ProjectID); err != nil {
		return nil, err
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	dbq := s.db.WithContext(ctx).Model(&model.ChangeFreezeWindow{}).Where("project_id = ?", q.ProjectID)
	if q.Enabled != nil {
		dbq = dbq.Where("enabled = ?", *q.Enabled)
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []model.ChangeFreezeWindow
	if err := dbq.Order("starts_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return &pagination.Result[model.ChangeFreezeWindow]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *ChangeEventService) UpsertFreezeWindow(ctx context.Context, req FreezeWindowUpsertRequest) (*model.ChangeFreezeWindow, error) {
	if err := s.ensureProject(ctx, req.ProjectID); err != nil {
		return nil, err
	}
	starts, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartsAt))
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("starts_at 需为 RFC3339")
	}
	ends, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EndsAt))
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("ends_at 需为 RFC3339")
	}
	if !ends.After(starts) {
		return nil, constants.ErrBadRequestWithMsg("ends_at 必须晚于 starts_at")
	}
	scope := strings.ToLower(strings.TrimSpace(req.Scope))
	if scope == "" {
		scope = model.FreezeScopeAll
	}
	switch scope {
	case model.FreezeScopeAll, model.FreezeScopeCicd, model.FreezeScopeK8s, model.FreezeScopeDbmgmt:
	default:
		return nil, constants.ErrBadRequestWithMsg("scope 无效")
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := model.ChangeFreezeWindow{
		ID:        req.ID,
		ProjectID: req.ProjectID,
		Name:      strings.TrimSpace(req.Name),
		Scope:     scope,
		Env:       strings.ToLower(strings.TrimSpace(req.Env)),
		StartsAt:  starts,
		EndsAt:    ends,
		Reason:    strings.TrimSpace(req.Reason),
		Enabled:   enabled,
		CreatedBy: req.CreatedBy,
	}
	if req.ID == 0 {
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
		return &row, nil
	}
	var existing model.ChangeFreezeWindow
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", req.ID, req.ProjectID).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	row.CreatedBy = existing.CreatedBy
	if err := s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
		"name":      row.Name,
		"scope":     row.Scope,
		"env":       row.Env,
		"starts_at": row.StartsAt,
		"ends_at":   row.EndsAt,
		"reason":    row.Reason,
		"enabled":   row.Enabled,
	}).Error; err != nil {
		return nil, err
	}
	_ = s.db.WithContext(ctx).Where("id = ?", req.ID).First(&row).Error
	return &row, nil
}

func (s *ChangeEventService) DeleteFreezeWindow(ctx context.Context, projectID, id uint) error {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", id, projectID).Delete(&model.ChangeFreezeWindow{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFound
	}
	return nil
}

func (s *ChangeEventService) ConflictCheck(ctx context.Context, q ConflictCheckQuery) (*ConflictCheckResult, error) {
	if err := s.ensureProject(ctx, q.ProjectID); err != nil {
		return nil, err
	}
	src := strings.TrimSpace(q.Source)
	if src == "" {
		src = model.ChangeSourceCicd
	}
	peek := changegate.Peek(ctx, changegate.CheckInput{
		ProjectID: q.ProjectID,
		Source:    src,
		Env:       q.Env,
		ServiceID: q.ServiceID,
		Namespace: q.Namespace,
		Action:    q.Action,
	})
	now := time.Now()
	var active []model.ChangeFreezeWindow
	_ = s.db.WithContext(ctx).
		Where("project_id = ? AND enabled = ? AND starts_at <= ? AND ends_at >= ?", q.ProjectID, true, now, now).
		Order("id DESC").Limit(20).Find(&active).Error
	return &ConflictCheckResult{CheckResult: peek, ActiveFreezes: active}, nil
}

func (s *ChangeEventService) ensureProject(ctx context.Context, projectID uint) error {
	if projectID == 0 {
		return constants.ErrBadRequestWithMsg("project_id required")
	}
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFound
		}
		return err
	}
	return nil
}
