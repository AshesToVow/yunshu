package alert

import (
	"context"
	"encoding/json"
	"strings"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
)

type AlertRuleChangeService struct {
	repo  interfaces.AlertRuleChangeRepository
	rules *AlertMonitorRuleService
}

func NewAlertRuleChangeService(repo interfaces.AlertRuleChangeRepository, rules *AlertMonitorRuleService) *AlertRuleChangeService {
	return &AlertRuleChangeService{repo: repo, rules: rules}
}

type ProposeRuleChangeRequest struct {
	RuleID  uint                          `json:"rule_id" binding:"required"`
	Payload AlertMonitorRuleUpsertRequest `json:"payload" binding:"required"`
	Comment string                        `json:"comment"`
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
	if err := s.repo.Create(ctx, &row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.rule_change", "Propose", err)
	}
	return &row, nil
}

func (s *AlertRuleChangeService) ListPending(ctx context.Context) ([]model.AlertMonitorRuleChangeRequest, error) {
	list, err := s.repo.ListPending(ctx)
	return list, bizerrors.Pass(ctx, "alert.rule_change", "ListPending", err)
}

func (s *AlertRuleChangeService) Approve(ctx context.Context, id, reviewerID uint) (*model.AlertMonitorRule, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
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
	_ = s.repo.UpdateFields(ctx, row.ID, map[string]any{
		"status": model.AlertRuleChangeApproved, "reviewer_id": reviewerID,
	})
	return updated, nil
}

func (s *AlertRuleChangeService) Reject(ctx context.Context, id, reviewerID uint, comment string) error {
	rows, err := s.repo.UpdatePending(ctx, id, map[string]any{
		"status": model.AlertRuleChangeRejected, "reviewer_id": reviewerID, "comment": strings.TrimSpace(comment),
	})
	if err != nil {
		return bizerrors.Pass(ctx, "alert.rule_change", "Reject", err)
	}
	if rows == 0 {
		return constants.ErrNotFound
	}
	return nil
}
