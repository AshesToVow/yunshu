package service

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type AlertDutyBlockListQuery struct {
	MonitorRuleID *uint `form:"monitor_rule_id"`
	ProjectID     *uint `form:"project_id"`
	Page          int   `form:"page"`
	PageSize      int   `form:"page_size"`
}

type AlertDutyBlockUpsertRequest struct {
	MonitorRuleID     uint      `json:"monitor_rule_id" binding:"required"`
	StartsAt          time.Time `json:"starts_at" binding:"required"`
	EndsAt            time.Time `json:"ends_at" binding:"required"`
	Title             string    `json:"title" binding:"omitempty,max=128"`
	UserIDsJSON       string    `json:"user_ids_json"`
	DepartmentIDsJSON string    `json:"department_ids_json"`
	ExtraEmailsJSON   string    `json:"extra_emails_json"`
	Remark            string    `json:"remark" binding:"omitempty,max=512"`
}

type AlertDutyService struct {
	repo         interfaces.AlertDutyRepository
	ruleRepo     interfaces.AlertMonitorRuleRepository
	userRepo     interfaces.UserRepository
}

func NewAlertDutyService(
	repo interfaces.AlertDutyRepository,
	ruleRepo interfaces.AlertMonitorRuleRepository,
	userRepo interfaces.UserRepository,
) *AlertDutyService {
	return &AlertDutyService{repo: repo, ruleRepo: ruleRepo, userRepo: userRepo}
}

func (s *AlertDutyService) ListBlocks(ctx context.Context, q AlertDutyBlockListQuery) ([]model.AlertDutyBlock, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.repo.List(ctx, repository.AlertDutyListFilter{
		MonitorRuleID: q.MonitorRuleID,
		ProjectID:     q.ProjectID,
	}, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.duty", "ListBlocks", err)
	}
	return list, total, page, pageSize, nil
}

func (s *AlertDutyService) CreateBlock(ctx context.Context, req AlertDutyBlockUpsertRequest) (*model.AlertDutyBlock, error) {
	if !req.EndsAt.After(req.StartsAt) {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgc1f741f96c03)
	}
	if _, err := s.ruleRepo.GetByID(ctx, req.MonitorRuleID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgdfcd891c9a94)
		}
		return nil, bizerrors.Pass(ctx, "alert.duty", "CreateBlock", err)
	}
	row := model.AlertDutyBlock{
		MonitorRuleID:     req.MonitorRuleID,
		StartsAt:          req.StartsAt,
		EndsAt:            req.EndsAt,
		Title:             strings.TrimSpace(req.Title),
		UserIDsJSON:       strings.TrimSpace(req.UserIDsJSON),
		DepartmentIDsJSON: strings.TrimSpace(req.DepartmentIDsJSON),
		ExtraEmailsJSON:   strings.TrimSpace(req.ExtraEmailsJSON),
		Remark:            strings.TrimSpace(req.Remark),
	}
	if err := s.repo.Create(ctx, &row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.duty", "CreateBlock", err)
	}
	return &row, nil
}

func (s *AlertDutyService) UpdateBlock(ctx context.Context, id uint, req AlertDutyBlockUpsertRequest) (*model.AlertDutyBlock, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsgde63e900b907)
		}
		return nil, bizerrors.Pass(ctx, "alert.duty", "UpdateBlock", err)
	}
	if req.MonitorRuleID > 0 && req.MonitorRuleID != row.MonitorRuleID {
		if _, err := s.ruleRepo.GetByID(ctx, req.MonitorRuleID); err != nil {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgdfcd891c9a94)
		}
		row.MonitorRuleID = req.MonitorRuleID
	}
	if !req.StartsAt.IsZero() {
		row.StartsAt = req.StartsAt
	}
	if !req.EndsAt.IsZero() {
		row.EndsAt = req.EndsAt
	}
	if !row.EndsAt.After(row.StartsAt) {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgc1f741f96c03)
	}
	row.Title = strings.TrimSpace(req.Title)
	row.UserIDsJSON = strings.TrimSpace(req.UserIDsJSON)
	row.DepartmentIDsJSON = strings.TrimSpace(req.DepartmentIDsJSON)
	row.ExtraEmailsJSON = strings.TrimSpace(req.ExtraEmailsJSON)
	row.Remark = strings.TrimSpace(req.Remark)
	if err := s.repo.Save(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.duty", "UpdateBlock", err)
	}
	return row, nil
}

func (s *AlertDutyService) DeleteBlock(ctx context.Context, id uint) error {
	n, err := s.repo.Delete(ctx, id)
	if err != nil {
		return bizerrors.Pass(ctx, "alert.duty", "DeleteBlock", err)
	}
	if n == 0 {
		return constants.ErrNotFoundWithMsg(constants.ErrMsgde63e900b907)
	}
	return nil
}

func dedupeEmailsLower(emails []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, e := range emails {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}

func (s *AlertDutyService) HasActiveBlockAtRule(ctx context.Context, monitorRuleID uint, t time.Time) (bool, error) {
	return s.repo.HasActiveAtRule(ctx, monitorRuleID, t)
}

func (s *AlertDutyService) ResolveNotifyEmailsAtRule(ctx context.Context, monitorRuleID uint, t time.Time) ([]string, error) {
	blocks, err := s.repo.ListActiveAtRule(ctx, monitorRuleID, t)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.duty", "ResolveNotifyEmailsAtRule", err)
	}
	var emails []string
	for _, b := range blocks {
		uidSet := map[uint]struct{}{}
		for _, id := range parseUintSliceJSON(b.UserIDsJSON) {
			uidSet[id] = struct{}{}
		}
		deptRoots := parseUintSliceJSON(b.DepartmentIDsJSON)
		more, err := s.userRepo.ListActiveIDsByDepartmentSubtree(ctx, deptRoots)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "alert.duty", "ResolveNotifyEmailsAtRule", err)
		}
		for _, id := range more {
			uidSet[id] = struct{}{}
		}
		var all []uint
		for id := range uidSet {
			all = append(all, id)
		}
		if len(all) > 0 {
			users, err := s.userRepo.ListByIDs(ctx, all)
			if err != nil {
				return nil, bizerrors.Pass(ctx, "alert.duty", "ResolveNotifyEmailsAtRule", err)
			}
			for i := range users {
				if users[i].Email != nil {
					emails = append(emails, *users[i].Email)
				}
			}
		}
		extras, _ := assigneeParseStringSliceJSON(b.ExtraEmailsJSON)
		emails = append(emails, extras...)
	}
	return dedupeEmailsLower(emails), nil
}

func (s *AlertDutyService) ResolveNotifyPhonesAtRule(ctx context.Context, monitorRuleID uint, t time.Time) ([]string, error) {
	blocks, err := s.repo.ListActiveAtRule(ctx, monitorRuleID, t)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.duty", "ResolveNotifyPhonesAtRule", err)
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, b := range blocks {
		uidSet := map[uint]struct{}{}
		for _, id := range parseUintSliceJSON(b.UserIDsJSON) {
			uidSet[id] = struct{}{}
		}
		deptRoots := parseUintSliceJSON(b.DepartmentIDsJSON)
		more, err := s.userRepo.ListActiveIDsByDepartmentSubtree(ctx, deptRoots)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "alert.duty", "ResolveNotifyPhonesAtRule", err)
		}
		for _, id := range more {
			uidSet[id] = struct{}{}
		}
		var all []uint
		for id := range uidSet {
			all = append(all, id)
		}
		if len(all) > 0 {
			users, err := s.userRepo.ListByIDs(ctx, all)
			if err != nil {
				return nil, bizerrors.Pass(ctx, "alert.duty", "ResolveNotifyPhonesAtRule", err)
			}
			for i := range users {
				add(users[i].Phone)
			}
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
