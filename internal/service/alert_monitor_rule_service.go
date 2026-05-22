package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type AlertMonitorRuleListQuery struct {
	DatasourceID *uint  `form:"datasource_id"`
	ProjectID    *uint  `form:"project_id"`
	Keyword      string `form:"keyword"`
	Enabled      *bool  `form:"enabled"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

type AlertMonitorRuleUpsertRequest struct {
	DatasourceID        uint   `json:"datasource_id" binding:"required"`
	ProjectID           *uint  `json:"project_id"`
	Name                string `json:"name" binding:"required,max=128"`
	Expr                string `json:"expr" binding:"required"`
	ForSeconds          int    `json:"for_seconds"`
	EvalIntervalSeconds int    `json:"eval_interval_seconds"`
	Severity            string `json:"severity" binding:"omitempty,max=32"`
	ThresholdUnit       string `json:"threshold_unit" binding:"omitempty,max=32"`
	LabelsJSON          string `json:"labels_json"`
	AnnotationsJSON     string `json:"annotations_json"`
	Enabled             *bool  `json:"enabled"`
}

type AlertMonitorRuleService struct {
	ruleRepo interfaces.AlertMonitorRuleRepository
	dsRepo   interfaces.AlertDatasourceRepository
	redis    *redis.Client
}

type AlertMonitorRuleListItem struct {
	model.AlertMonitorRule
	PolicySilenceActive           bool  `json:"policy_silence_active"`
	PolicySilenceRemainingSeconds int64 `json:"policy_silence_remaining_seconds"`
}

func NewAlertMonitorRuleService(
	ruleRepo interfaces.AlertMonitorRuleRepository,
	dsRepo interfaces.AlertDatasourceRepository,
	redisClient *redis.Client,
) *AlertMonitorRuleService {
	return &AlertMonitorRuleService{ruleRepo: ruleRepo, dsRepo: dsRepo, redis: redisClient}
}

func (s *AlertMonitorRuleService) List(ctx context.Context, q AlertMonitorRuleListQuery) ([]AlertMonitorRuleListItem, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.ruleRepo.List(ctx, repository.AlertMonitorRuleListFilter{
		DatasourceID: q.DatasourceID,
		ProjectID:    q.ProjectID,
		Keyword:      q.Keyword,
		Enabled:      q.Enabled,
	}, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.rule", "List", err)
	}
	out := make([]AlertMonitorRuleListItem, 0, len(list))
	for _, row := range list {
		item := AlertMonitorRuleListItem{AlertMonitorRule: row}
		if s.redis != nil {
			key := "alert:policy:silence:rule:" + fmt.Sprintf("%d", row.ID)
			if ttl, err := s.redis.TTL(ctx, key).Result(); err == nil && ttl > 0 {
				item.PolicySilenceActive = true
				item.PolicySilenceRemainingSeconds = int64(ttl / time.Second)
			}
		}
		out = append(out, item)
	}
	return out, total, page, pageSize, nil
}

func (s *AlertMonitorRuleService) ListEnabled(ctx context.Context) ([]model.AlertMonitorRule, error) {
	list, err := s.ruleRepo.ListEnabled(ctx)
	return list, bizerrors.Pass(ctx, "alert.rule", "ListEnabled", err)
}

func (s *AlertMonitorRuleService) Get(ctx context.Context, id uint) (*model.AlertMonitorRule, error) {
	row, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsgdfcd891c9a94)
		}
		return nil, bizerrors.Pass(ctx, "alert.rule", "Get", err)
	}
	return row, nil
}

func (s *AlertMonitorRuleService) Create(ctx context.Context, req AlertMonitorRuleUpsertRequest) (*model.AlertMonitorRule, error) {
	if _, err := s.dsRepo.GetByID(ctx, req.DatasourceID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgaf3782e3e26f)
		}
		return nil, bizerrors.Pass(ctx, "alert.rule", "Create", err)
	}
	ev := req.EvalIntervalSeconds
	if ev <= 0 {
		ev = 30
	}
	if ev < 5 {
		ev = 5
	}
	sev := strings.TrimSpace(req.Severity)
	if sev == "" {
		sev = "warning"
	}
	unit := strings.TrimSpace(req.ThresholdUnit)
	if unit == "" {
		unit = "raw"
	}
	row := model.AlertMonitorRule{
		DatasourceID:        req.DatasourceID,
		Name:                strings.TrimSpace(req.Name),
		Expr:                strings.TrimSpace(req.Expr),
		ForSeconds:          req.ForSeconds,
		EvalIntervalSeconds: ev,
		Severity:            sev,
		ThresholdUnit:       unit,
		LabelsJSON:          strings.TrimSpace(req.LabelsJSON),
		AnnotationsJSON:     strings.TrimSpace(req.AnnotationsJSON),
		Enabled:             req.Enabled == nil || *req.Enabled,
	}
	if err := s.ruleRepo.Create(ctx, &row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.rule", "Create", err)
	}
	return &row, nil
}

func (s *AlertMonitorRuleService) Update(ctx context.Context, id uint, req AlertMonitorRuleUpsertRequest) (*model.AlertMonitorRule, error) {
	row, err := s.Get(ctx, id)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.rule", "Update", err)
	}
	if req.DatasourceID > 0 && req.DatasourceID != row.DatasourceID {
		if _, err := s.dsRepo.GetByID(ctx, req.DatasourceID); err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgaf3782e3e26f)
			}
			return nil, bizerrors.Pass(ctx, "alert.rule", "Update", err)
		}
		row.DatasourceID = req.DatasourceID
	}
	if strings.TrimSpace(req.Name) != "" {
		row.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.Expr) != "" {
		row.Expr = strings.TrimSpace(req.Expr)
	}
	row.ForSeconds = req.ForSeconds
	if req.EvalIntervalSeconds > 0 {
		row.EvalIntervalSeconds = req.EvalIntervalSeconds
		if row.EvalIntervalSeconds < 5 {
			row.EvalIntervalSeconds = 5
		}
	}
	if strings.TrimSpace(req.Severity) != "" {
		row.Severity = strings.TrimSpace(req.Severity)
	}
	if strings.TrimSpace(req.ThresholdUnit) != "" {
		row.ThresholdUnit = strings.TrimSpace(req.ThresholdUnit)
	}
	row.LabelsJSON = strings.TrimSpace(req.LabelsJSON)
	row.AnnotationsJSON = strings.TrimSpace(req.AnnotationsJSON)
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.ruleRepo.Save(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.rule", "Update", err)
	}
	return row, nil
}

func (s *AlertMonitorRuleService) Delete(ctx context.Context, id uint) error {
	err := s.ruleRepo.DeleteCascade(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return constants.ErrNotFoundWithMsg(constants.ErrMsgdfcd891c9a94)
	}
	return bizerrors.Pass(ctx, "alert.rule", "Delete", err)
}
