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
		for _, id := range parseUintSliceJSON(b.UserIDsJSON) {
			if id > 0 {
				return &id, nil
			}
		}
		deptRoots := parseUintSliceJSON(b.DepartmentIDsJSON)
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

func parseUintSliceJSON(raw string) []uint {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil
	}
	return ids
}
