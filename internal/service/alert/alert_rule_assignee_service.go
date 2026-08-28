package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

type AlertRuleAssigneeUpsertRequest struct {
	UserIDsJSON       string `json:"user_ids_json"`
	DepartmentIDsJSON string `json:"department_ids_json"`
	ExtraEmailsJSON   string `json:"extra_emails_json"`
	RecipientMode     string `json:"recipient_mode"`
	NotifyOnResolved  *bool  `json:"notify_on_resolved"`
	Remark            string `json:"remark" binding:"omitempty,max=512"`
}

type AlertRuleAssigneeService struct {
	repo           interfaces.AlertRuleAssigneeRepository
	ruleRepo       interfaces.AlertMonitorRuleRepository
	dsRepo         interfaces.AlertDatasourceRepository
	userRepo       interfaces.UserRepository
	memberRepo     interfaces.ProjectMemberRepository
	departmentRepo interfaces.DepartmentRepository
}

func NewAlertRuleAssigneeService(
	repo interfaces.AlertRuleAssigneeRepository,
	ruleRepo interfaces.AlertMonitorRuleRepository,
	dsRepo interfaces.AlertDatasourceRepository,
	userRepo interfaces.UserRepository,
	memberRepo interfaces.ProjectMemberRepository,
	departmentRepo interfaces.DepartmentRepository,
) *AlertRuleAssigneeService {
	return &AlertRuleAssigneeService{
		repo: repo, ruleRepo: ruleRepo, dsRepo: dsRepo,
		userRepo: userRepo, memberRepo: memberRepo, departmentRepo: departmentRepo,
	}
}

func assigneeParseStringSliceJSON(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var ss []string
	if err := json.Unmarshal([]byte(raw), &ss); err != nil {
		return nil, bizerrors.Pass(context.Background(), "alert.assignee", "assigneeParseStringSliceJSON", err)
	}
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}

func (s *AlertRuleAssigneeService) ListByRule(ctx context.Context, ruleID uint) ([]model.AlertRuleAssignee, error) {
	list, err := s.repo.ListByRule(ctx, ruleID)
	return list, bizerrors.Pass(ctx, "alert.assignee", "ListByRule", err)
}

func (s *AlertRuleAssigneeService) UpsertPrimary(ctx context.Context, ruleID uint, req AlertRuleAssigneeUpsertRequest) (*model.AlertRuleAssignee, error) {
	if _, err := s.ruleRepo.GetByID(ctx, ruleID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsgdfcd891c9a94)
		}
		return nil, bizerrors.Pass(ctx, "alert.assignee", "UpsertPrimary", err)
	}
	row, err := s.repo.GetPrimaryByRule(ctx, ruleID)
	isNew := err == gorm.ErrRecordNotFound
	if err != nil && !isNew {
		return nil, bizerrors.Pass(ctx, "alert.assignee", "UpsertPrimary", err)
	}
	if isNew {
		row = &model.AlertRuleAssignee{}
	}
	row.MonitorRuleID = ruleID
	row.UserIDsJSON = strings.TrimSpace(req.UserIDsJSON)
	row.DepartmentIDsJSON = strings.TrimSpace(req.DepartmentIDsJSON)
	row.ExtraEmailsJSON = strings.TrimSpace(req.ExtraEmailsJSON)
	row.RecipientMode = normalizeRecipientMode(req.RecipientMode)
	row.Remark = strings.TrimSpace(req.Remark)
	if req.NotifyOnResolved != nil {
		row.NotifyOnResolved = *req.NotifyOnResolved
	}
	if isNew {
		if err := s.repo.Create(ctx, row); err != nil {
			return nil, bizerrors.Pass(ctx, "alert.assignee", "UpsertPrimary", err)
		}
		return row, nil
	}
	if err := s.repo.Save(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.assignee", "UpsertPrimary", err)
	}
	return row, nil
}

func (s *AlertRuleAssigneeService) resolveRuleProjectID(ctx context.Context, ruleID uint) (uint, error) {
	rule, err := s.ruleRepo.GetByID(ctx, ruleID)
	if err != nil {
		return 0, bizerrors.Pass(ctx, "alert.assignee", "resolveRuleProjectID", err)
	}
	ds, err := s.dsRepo.GetByID(ctx, rule.DatasourceID)
	if err != nil {
		return 0, bizerrors.Pass(ctx, "alert.assignee", "resolveRuleProjectID", err)
	}
	return ds.ProjectID, nil
}

func (s *AlertRuleAssigneeService) expandDepartmentProjectMemberUserIDs(ctx context.Context, projectID uint, deptRootIDs []uint) ([]uint, error) {
	if projectID == 0 || len(deptRootIDs) == 0 || s.memberRepo == nil || s.departmentRepo == nil {
		return nil, nil
	}
	seen := map[uint]struct{}{}
	var deptIDs []uint
	for _, root := range deptRootIDs {
		if root == 0 {
			continue
		}
		sub, err := s.departmentRepo.ListDescendantIDsAndSelf(ctx, root)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "alert.assignee", "expandDepartmentProjectMemberUserIDs", err)
		}
		for _, id := range sub {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			deptIDs = append(deptIDs, id)
		}
	}
	if len(deptIDs) == 0 {
		return nil, nil
	}
	uids, err := s.repo.ListProjectMemberUserIDsByDepartments(ctx, projectID, deptIDs)
	return uids, bizerrors.Pass(ctx, "alert.assignee", "expandDepartmentProjectMemberUserIDs", err)
}

func (s *AlertRuleAssigneeService) leaderUserIDsFromDepartmentRoots(ctx context.Context, deptRootIDs []uint) ([]uint, error) {
	if s.departmentRepo == nil || len(deptRootIDs) == 0 {
		return nil, nil
	}
	seen := map[uint]struct{}{}
	var out []uint
	for _, root := range deptRootIDs {
		if root == 0 {
			continue
		}
		d, err := s.departmentRepo.GetByID(ctx, root)
		if err != nil || d == nil || d.LeaderID == nil || *d.LeaderID == 0 {
			continue
		}
		if _, ok := seen[*d.LeaderID]; ok {
			continue
		}
		seen[*d.LeaderID] = struct{}{}
		out = append(out, *d.LeaderID)
	}
	return out, nil
}

// NotifyOnResolvedEnabled 规则是否配置「恢复时通知处理人」。
func (s *AlertRuleAssigneeService) NotifyOnResolvedEnabled(ctx context.Context, ruleID uint) bool {
	list, err := s.ListByRule(ctx, ruleID)
	if err != nil {
		return false
	}
	for _, row := range list {
		if row.NotifyOnResolved {
			return true
		}
	}
	return false
}

// RecipientModeForRule 返回规则收件优先级；无配置时默认 assignee_and_cc。
func (s *AlertRuleAssigneeService) RecipientModeForRule(ctx context.Context, ruleID uint) string {
	list, err := s.ListByRule(ctx, ruleID)
	if err != nil || len(list) == 0 {
		return RecipientModeAssigneeAndCC
	}
	return normalizeRecipientMode(list[0].RecipientMode)
}

// ResolveHandlerSummary 事件台展示用：处理人邮箱摘要（不含值班展开细节）。
func (s *AlertRuleAssigneeService) ResolveHandlerSummary(ctx context.Context, ruleID uint) string {
	emails, err := s.ResolveNotifyEmailsDirectUsers(ctx, ruleID)
	if err != nil || len(emails) == 0 {
		return ""
	}
	if len(emails) <= 2 {
		return strings.Join(emails, ", ")
	}
	return fmt.Sprintf("%s 等 %d 人", strings.Join(emails[:2], ", "), len(emails))
}

// ResolveNotifyEmailsDirectUsers 仅解析规则处理人中的「显式用户」与「额外邮箱」，不展开部门子树（用于 critical 邮件兜底，避免误发项目内非处理人）。
func (s *AlertRuleAssigneeService) ResolveNotifyEmailsDirectUsers(ctx context.Context, ruleID uint) ([]string, error) {
	list, err := s.ListByRule(ctx, ruleID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.assignee", "ResolveNotifyEmailsDirectUsers", err)
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(e string) {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			return
		}
		if _, ok := seen[e]; ok {
			return
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	if len(list) == 0 || s.userRepo == nil {
		return nil, nil
	}
	uidSet := map[uint]struct{}{}
	for _, row := range list {
		for _, id := range parseUintSliceJSON(row.UserIDsJSON) {
			if id > 0 {
				uidSet[id] = struct{}{}
			}
		}
	}
	var allUIDs []uint
	for id := range uidSet {
		allUIDs = append(allUIDs, id)
	}
	if len(allUIDs) > 0 {
		users, err := s.userRepo.ListByIDs(ctx, allUIDs)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "alert.assignee", "ResolveNotifyEmailsDirectUsers", err)
		}
		for i := range users {
			if users[i].Email != nil {
				add(*users[i].Email)
			}
		}
	}
	for _, row := range list {
		extras, _ := assigneeParseStringSliceJSON(row.ExtraEmailsJSON)
		for _, e := range extras {
			add(e)
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// ResolveNotifyEmails 合并规则处理人邮箱：显式用户 + 所选部门在「规则所属项目」内的成员 + 所选根部门的负责人 + 额外邮箱。
func (s *AlertRuleAssigneeService) ResolveNotifyEmails(ctx context.Context, ruleID uint) ([]string, error) {
	list, err := s.ListByRule(ctx, ruleID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.assignee", "ResolveNotifyEmails", err)
	}
	projectID, err := s.resolveRuleProjectID(ctx, ruleID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.assignee", "ResolveNotifyEmails", err)
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(e string) {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			return
		}
		if _, ok := seen[e]; ok {
			return
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	if len(list) == 0 || s.userRepo == nil {
		return nil, nil
	}
	uidSet := map[uint]struct{}{}
	for _, row := range list {
		for _, id := range parseUintSliceJSON(row.UserIDsJSON) {
			if id > 0 {
				uidSet[id] = struct{}{}
			}
		}
		deptRoots := parseUintSliceJSON(row.DepartmentIDsJSON)
		more, e1 := s.expandDepartmentProjectMemberUserIDs(ctx, projectID, deptRoots)
		if e1 != nil {
			return nil, e1
		}
		for _, id := range more {
			uidSet[id] = struct{}{}
		}
		leaders, e2 := s.leaderUserIDsFromDepartmentRoots(ctx, deptRoots)
		if e2 != nil {
			return nil, e2
		}
		for _, id := range leaders {
			uidSet[id] = struct{}{}
		}
	}
	var allUIDs []uint
	for id := range uidSet {
		allUIDs = append(allUIDs, id)
	}
	if len(allUIDs) > 0 {
		users, err := s.userRepo.ListByIDs(ctx, allUIDs)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "alert.assignee", "ResolveNotifyEmails", err)
		}
		for i := range users {
			if users[i].Email != nil {
				add(*users[i].Email)
			}
		}
	}
	for _, row := range list {
		extras, _ := assigneeParseStringSliceJSON(row.ExtraEmailsJSON)
		for _, e := range extras {
			add(e)
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// ResolveNotifyPhones 合并规则处理人手机号：显式用户 + 部门在项目内的成员 + 根部门负责人。
func (s *AlertRuleAssigneeService) ResolveNotifyPhones(ctx context.Context, ruleID uint) ([]string, error) {
	list, err := s.ListByRule(ctx, ruleID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.assignee", "ResolveNotifyPhones", err)
	}
	projectID, err := s.resolveRuleProjectID(ctx, ruleID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.assignee", "ResolveNotifyPhones", err)
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
	if len(list) == 0 || s.userRepo == nil {
		return nil, nil
	}
	uidSet := map[uint]struct{}{}
	for _, row := range list {
		for _, id := range parseUintSliceJSON(row.UserIDsJSON) {
			if id > 0 {
				uidSet[id] = struct{}{}
			}
		}
		deptRoots := parseUintSliceJSON(row.DepartmentIDsJSON)
		more, e1 := s.expandDepartmentProjectMemberUserIDs(ctx, projectID, deptRoots)
		if e1 != nil {
			return nil, e1
		}
		for _, id := range more {
			uidSet[id] = struct{}{}
		}
		leaders, e2 := s.leaderUserIDsFromDepartmentRoots(ctx, deptRoots)
		if e2 != nil {
			return nil, e2
		}
		for _, id := range leaders {
			uidSet[id] = struct{}{}
		}
	}
	var allUIDs []uint
	for id := range uidSet {
		allUIDs = append(allUIDs, id)
	}
	if len(allUIDs) > 0 {
		users, err := s.userRepo.ListByIDs(ctx, allUIDs)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "alert.assignee", "ResolveNotifyPhones", err)
		}
		for i := range users {
			add(users[i].Phone)
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (s *AlertRuleAssigneeService) Delete(ctx context.Context, id uint) error {
	n, err := s.repo.Delete(ctx, id)
	if err != nil {
		return bizerrors.Pass(ctx, "alert.assignee", "Delete", err)
	}
	if n == 0 {
		return constants.ErrNotFoundWithMsg(constants.ErrMsg8faff6dbdd1d)
	}
	return nil
}

func marshalUintSliceJSON(ids []uint) string {
	if len(ids) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ids)
	return string(b)
}

// PruneUserFromAllAssignees 从所有监控规则处理人配置中移除已删除用户 ID。
func (s *AlertRuleAssigneeService) PruneUserFromAllAssignees(ctx context.Context, userID uint) error {
	if userID == 0 {
		return nil
	}
	rows, err := s.repo.ListAll(ctx)
	if err != nil {
		return bizerrors.Pass(ctx, "alert.assignee", "PruneUserFromAllAssignees", err)
	}
	for i := range rows {
		ids := parseUintSliceJSON(rows[i].UserIDsJSON)
		if len(ids) == 0 {
			continue
		}
		filtered := make([]uint, 0, len(ids))
		for _, id := range ids {
			if id != userID {
				filtered = append(filtered, id)
			}
		}
		if len(filtered) == len(ids) {
			continue
		}
		if err := s.repo.UpdateFields(ctx, rows[i].ID, map[string]any{
			"user_ids_json": marshalUintSliceJSON(filtered),
		}); err != nil {
			return bizerrors.Pass(ctx, "alert.assignee", "PruneUserFromAllAssignees", err)
		}
	}
	return nil
}

// PruneDepartmentFromAllAssignees 从所有监控规则处理人配置中移除已删除部门 ID。
func (s *AlertRuleAssigneeService) PruneDepartmentFromAllAssignees(ctx context.Context, departmentID uint) error {
	if departmentID == 0 {
		return nil
	}
	rows, err := s.repo.ListAll(ctx)
	if err != nil {
		return bizerrors.Pass(ctx, "alert.assignee", "PruneDepartmentFromAllAssignees", err)
	}
	for i := range rows {
		ids := parseUintSliceJSON(rows[i].DepartmentIDsJSON)
		if len(ids) == 0 {
			continue
		}
		filtered := make([]uint, 0, len(ids))
		for _, id := range ids {
			if id != departmentID {
				filtered = append(filtered, id)
			}
		}
		if len(filtered) == len(ids) {
			continue
		}
		if err := s.repo.UpdateFields(ctx, rows[i].ID, map[string]any{
			"department_ids_json": marshalUintSliceJSON(filtered),
		}); err != nil {
			return bizerrors.Pass(ctx, "alert.assignee", "PruneDepartmentFromAllAssignees", err)
		}
	}
	return nil
}
