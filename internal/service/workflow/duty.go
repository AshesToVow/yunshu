package workflow

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"yunshu/internal/model"
)

func (s *Service) resolveDutyAssignee(ctx context.Context, st model.WorkflowStage, at time.Time) (*uint, error) {
	if st.AssigneeRuleType != model.WorkflowAssigneeDuty || st.DutyMonitorRuleID == nil || *st.DutyMonitorRuleID == 0 {
		return nil, nil
	}
	if s.dutyRepo == nil {
		return nil, nil
	}
	blocks, err := s.dutyRepo.ListActiveAtRule(ctx, *st.DutyMonitorRuleID, at)
	if err != nil {
		return nil, err
	}
	for _, b := range blocks {
		userIDs, err := parseUintSliceJSON(b.UserIDsJSON)
		if err != nil {
			return nil, err
		}
		for _, id := range userIDs {
			if id > 0 {
				return &id, nil
			}
		}
		deptRoots, err := parseUintSliceJSON(b.DepartmentIDsJSON)
		if err != nil {
			return nil, err
		}
		if len(deptRoots) > 0 && s.userRepo != nil {
			ids, err := s.userRepo.ListActiveIDsByDepartmentSubtree(ctx, deptRoots)
			if err != nil {
				return nil, err
			}
			if len(ids) > 0 {
				uid := ids[0]
				return &uid, nil
			}
		}
	}
	return nil, nil
}

func parseUintSliceJSON(raw string) ([]uint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}
