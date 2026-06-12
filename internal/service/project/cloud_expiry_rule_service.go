package project

import (
	"context"
	"strings"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

type CloudExpiryRuleListQuery struct {
	ProjectID *uint  `form:"project_id"`
	Provider  string `form:"provider"`
	Keyword   string `form:"keyword"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type CloudExpiryRuleUpsertRequest struct {
	ProjectID       uint   `json:"project_id" binding:"required"`
	Name            string `json:"name" binding:"required,max=128"`
	Provider        string `json:"provider"`
	RegionScope     string `json:"region_scope"`
	AdvanceDays     int    `json:"advance_days"`
	Severity        string `json:"severity" binding:"omitempty,max=32"`
	LabelsJSON      string `json:"labels_json"`
	EvalCronSpec    string `json:"eval_cron_spec"`
	ScheduleEnabled *bool  `json:"schedule_enabled"`
	Enabled         *bool  `json:"enabled"`
}

type CloudExpiryRuleService struct {
	repo interfaces.CloudExpiryRuleRepository
}

func NewCloudExpiryRuleService(repo interfaces.CloudExpiryRuleRepository) *CloudExpiryRuleService {
	return &CloudExpiryRuleService{repo: repo}
}

func (s *CloudExpiryRuleService) List(ctx context.Context, q CloudExpiryRuleListQuery) ([]model.CloudExpiryRule, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	kw := strings.TrimSpace(q.Keyword)
	if q.ProjectID != nil && *q.ProjectID > 0 {
		kw = kw // project filter kept in service layer via future repo extension
	}
	list, total, err := s.repo.List(ctx, kw, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.cloud-expiry", "List", err)
	}
	if q.ProjectID != nil && *q.ProjectID > 0 {
		filtered := make([]model.CloudExpiryRule, 0, len(list))
		for _, row := range list {
			if row.ProjectID == *q.ProjectID {
				filtered = append(filtered, row)
			}
		}
		list = filtered
		total = int64(len(filtered))
	}
	if p := strings.TrimSpace(q.Provider); p != "" {
		filtered := make([]model.CloudExpiryRule, 0, len(list))
		for _, row := range list {
			if strings.TrimSpace(row.Provider) == p {
				filtered = append(filtered, row)
			}
		}
		list = filtered
		total = int64(len(filtered))
	}
	return list, total, page, pageSize, nil
}

func (s *CloudExpiryRuleService) Create(ctx context.Context, req CloudExpiryRuleUpsertRequest) (*model.CloudExpiryRule, error) {
	if err := ValidateCloudExpiryCronSpec(req.EvalCronSpec); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.cloud-expiry", "Create", err)
	}
	days := req.AdvanceDays
	if days <= 0 {
		days = 7
	}
	sev := strings.TrimSpace(req.Severity)
	if sev == "" {
		sev = "warning"
	}
	sched := true
	if req.ScheduleEnabled != nil {
		sched = *req.ScheduleEnabled
	}
	if sched && strings.TrimSpace(req.EvalCronSpec) == "" {
		return nil, constants.ErrBadRequestWithMsg("已启用定时评估时必须填写 eval_cron_spec（Cron 表达式）")
	}
	row := model.CloudExpiryRule{
		ProjectID:           req.ProjectID,
		Name:                strings.TrimSpace(req.Name),
		Provider:            strings.TrimSpace(req.Provider),
		RegionScope:         strings.TrimSpace(req.RegionScope),
		AdvanceDays:         days,
		Severity:            sev,
		LabelsJSON:          strings.TrimSpace(req.LabelsJSON),
		EvalIntervalSeconds: 0,
		EvalCronSpec:        strings.TrimSpace(req.EvalCronSpec),
		ScheduleEnabled:     sched,
		Enabled:             req.Enabled == nil || *req.Enabled,
	}
	if err := s.repo.Create(ctx, &row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.cloud-expiry", "Create", err)
	}
	return &row, nil
}

func (s *CloudExpiryRuleService) Update(ctx context.Context, id uint, req CloudExpiryRuleUpsertRequest) (*model.CloudExpiryRule, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsg34cc3b1e5427)
		}
		return nil, bizerrors.Pass(ctx, "alert.cloud-expiry", "Update", err)
	}
	if err := ValidateCloudExpiryCronSpec(req.EvalCronSpec); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.cloud-expiry", "Update", err)
	}
	if req.ProjectID > 0 {
		row.ProjectID = req.ProjectID
	}
	if v := strings.TrimSpace(req.Name); v != "" {
		row.Name = v
	}
	row.Provider = strings.TrimSpace(req.Provider)
	row.RegionScope = strings.TrimSpace(req.RegionScope)
	if req.AdvanceDays > 0 {
		row.AdvanceDays = req.AdvanceDays
	}
	if v := strings.TrimSpace(req.Severity); v != "" {
		row.Severity = v
	}
	row.LabelsJSON = strings.TrimSpace(req.LabelsJSON)
	row.EvalCronSpec = strings.TrimSpace(req.EvalCronSpec)
	row.EvalIntervalSeconds = 0
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if req.ScheduleEnabled != nil {
		row.ScheduleEnabled = *req.ScheduleEnabled
	}
	if row.ScheduleEnabled && strings.TrimSpace(row.EvalCronSpec) == "" {
		return nil, constants.ErrBadRequestWithMsg("已启用定时评估时必须填写 eval_cron_spec（Cron 表达式）")
	}
	if err := s.repo.Save(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.cloud-expiry", "Update", err)
	}
	return row, nil
}

func (s *CloudExpiryRuleService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFoundWithMsg(constants.ErrMsg34cc3b1e5427)
		}
		return bizerrors.Pass(ctx, "alert.cloud-expiry", "Delete", err)
	}
	return nil
}
