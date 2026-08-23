package alert

import (
	"context"
	"encoding/json"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

type AlertRuleChangeService struct {
	db    *gorm.DB
	rules *AlertMonitorRuleService
}

func NewAlertRuleChangeService(db *gorm.DB, rules *AlertMonitorRuleService) *AlertRuleChangeService {
	return &AlertRuleChangeService{db: db, rules: rules}
}

type ProposeRuleChangeRequest struct {
	RuleID      uint                           `json:"rule_id" binding:"required"`
	Payload     AlertMonitorRuleUpsertRequest  `json:"payload" binding:"required"`
	Comment     string                         `json:"comment"`
}

func (s *AlertRuleChangeService) Propose(ctx context.Context, proposerID uint, req ProposeRuleChangeRequest) (*model.AlertMonitorRuleChangeRequest, error) {
	if proposerID == 0 {
		return nil, constants.ErrUnauthorized
	}
	bs, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.rule_change", "Propose", err)
	}
	row := model.AlertMonitorRuleChangeRequest{
		RuleID: req.RuleID, ProposerID: proposerID,
		Status: model.AlertRuleChangePending, PayloadJSON: string(bs), Comment: strings.TrimSpace(req.Comment),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "alert.rule_change", "Propose", err)
	}
	return &row, nil
}

func (s *AlertRuleChangeService) ListPending(ctx context.Context) ([]model.AlertMonitorRuleChangeRequest, error) {
	var list []model.AlertMonitorRuleChangeRequest
	err := s.db.WithContext(ctx).Where("status = ?", model.AlertRuleChangePending).Order("id DESC").Find(&list).Error
	return list, bizerrors.Pass(ctx, "alert.rule_change", "ListPending", err)
}

func (s *AlertRuleChangeService) Approve(ctx context.Context, id, reviewerID uint) (*model.AlertMonitorRule, error) {
	var row model.AlertMonitorRuleChangeRequest
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "alert.rule_change", "Approve", err)
	}
	if row.Status != model.AlertRuleChangePending {
		return nil, constants.ErrBadRequestWithMsg("变更单状态不是待审批")
	}
	var payload AlertMonitorRuleUpsertRequest
	if err := json.Unmarshal([]byte(row.PayloadJSON), &payload); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.rule_change", "Approve", err)
	}
	updated, err := s.rules.Update(ctx, row.RuleID, payload)
	if err != nil {
		return nil, err
	}
	_ = s.db.WithContext(ctx).Model(&row).Updates(map[string]any{
		"status": model.AlertRuleChangeApproved, "reviewer_id": reviewerID,
	}).Error
	return updated, nil
}

func (s *AlertRuleChangeService) Reject(ctx context.Context, id, reviewerID uint, comment string) error {
	res := s.db.WithContext(ctx).Model(&model.AlertMonitorRuleChangeRequest{}).
		Where("id = ? AND status = ?", id, model.AlertRuleChangePending).
		Updates(map[string]any{"status": model.AlertRuleChangeRejected, "reviewer_id": reviewerID, "comment": strings.TrimSpace(comment)})
	if res.Error != nil {
		return bizerrors.Pass(ctx, "alert.rule_change", "Reject", res.Error)
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFound
	}
	return nil
}
