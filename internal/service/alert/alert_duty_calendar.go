package alert

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type AlertDutyCalendarQuery struct {
	MonitorRuleID *uint     `form:"monitor_rule_id"`
	ProjectID     *uint     `form:"project_id"`
	From          time.Time `form:"from" binding:"required"`
	To            time.Time `form:"to" binding:"required"`
}

type AlertDutyCalendarItem struct {
	model.AlertDutyBlock
	Overlap bool `json:"overlap"`
}

type AlertDutyValidateRequest struct {
	MonitorRuleID uint                        `json:"monitor_rule_id" binding:"required"`
	Blocks        []AlertDutyBlockUpsertRequest `json:"blocks" binding:"required,dive"`
}

type AlertDutyValidateResult struct {
	OK        bool     `json:"ok"`
	Conflicts []string `json:"conflicts,omitempty"`
}

type AlertDutyHandoffRequest struct {
	UserIDsJSON       string `json:"user_ids_json" binding:"required"`
	DepartmentIDsJSON string `json:"department_ids_json"`
	ExtraEmailsJSON   string `json:"extra_emails_json"`
	Remark            string `json:"remark" binding:"omitempty,max=512"`
}

func (s *AlertDutyService) ListCalendar(ctx context.Context, q AlertDutyCalendarQuery) ([]AlertDutyCalendarItem, error) {
	if !q.To.After(q.From) {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgc1f741f96c03)
	}
	list, err := s.repo.ListBetween(ctx, repository.AlertDutyListFilter{
		MonitorRuleID: q.MonitorRuleID,
		ProjectID:     q.ProjectID,
	}, q.From, q.To)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.duty", "ListCalendar", err)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].StartsAt.Before(list[j].StartsAt) })
	out := make([]AlertDutyCalendarItem, 0, len(list))
	for i, b := range list {
		item := AlertDutyCalendarItem{AlertDutyBlock: b}
		for j := i + 1; j < len(list); j++ {
			other := list[j]
			if other.MonitorRuleID != b.MonitorRuleID {
				continue
			}
			if blocksOverlap(b.StartsAt, b.EndsAt, other.StartsAt, other.EndsAt) {
				item.Overlap = true
				break
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *AlertDutyService) ValidateBlocks(ctx context.Context, req AlertDutyValidateRequest) (*AlertDutyValidateResult, error) {
	var conflicts []string
	blocks := req.Blocks
	sort.Slice(blocks, func(i, j int) bool { return blocks[i].StartsAt.Before(blocks[j].StartsAt) })
	for i := range blocks {
		if !blocks[i].EndsAt.After(blocks[i].StartsAt) {
			conflicts = append(conflicts, fmt.Sprintf("块 %d: 结束时间须晚于开始时间", i+1))
		}
		for j := i + 1; j < len(blocks); j++ {
			if blocksOverlap(blocks[i].StartsAt, blocks[i].EndsAt, blocks[j].StartsAt, blocks[j].EndsAt) {
				conflicts = append(conflicts, fmt.Sprintf("块 %d 与块 %d 时间重叠", i+1, j+1))
			}
		}
	}
	existing, err := s.repo.ListBetween(ctx, repository.AlertDutyListFilter{
		MonitorRuleID: &req.MonitorRuleID,
	}, time.Time{}, time.Now().AddDate(5, 0, 0))
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.duty", "ValidateBlocks", err)
	}
	for i, nb := range blocks {
		for _, ex := range existing {
			if blocksOverlap(nb.StartsAt, nb.EndsAt, ex.StartsAt, ex.EndsAt) {
				conflicts = append(conflicts, fmt.Sprintf("块 %d 与已有值班 #%d 重叠", i+1, ex.ID))
			}
		}
	}
	return &AlertDutyValidateResult{OK: len(conflicts) == 0, Conflicts: conflicts}, nil
}

func (s *AlertDutyService) HandoffBlock(ctx context.Context, id uint, req AlertDutyHandoffRequest) (*model.AlertDutyBlock, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsgde63e900b907)
		}
		return nil, bizerrors.Pass(ctx, "alert.duty", "HandoffBlock", err)
	}
	row.UserIDsJSON = strings.TrimSpace(req.UserIDsJSON)
	if v := strings.TrimSpace(req.DepartmentIDsJSON); v != "" {
		row.DepartmentIDsJSON = v
	}
	if v := strings.TrimSpace(req.ExtraEmailsJSON); v != "" {
		row.ExtraEmailsJSON = v
	}
	if v := strings.TrimSpace(req.Remark); v != "" {
		row.Remark = v
	}
	if err := s.repo.Save(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.duty", "HandoffBlock", err)
	}
	return row, nil
}

func blocksOverlap(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}
