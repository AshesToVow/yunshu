package cicd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"slices"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	workflowsvc "yunshu/internal/service/workflow"
)

type ApprovalFlowStageItem struct {
	StageKey      string `json:"stage_key"`
	StageName     string `json:"stage_name"`
	SortOrder     int    `json:"sort_order"`
	Enabled       bool   `json:"enabled"`
	UserGroupID   *uint  `json:"user_group_id,omitempty"`
	UserGroupName string `json:"user_group_name,omitempty"`
}

type ApprovalFlowResponse struct {
	ProjectID  uint                    `json:"project_id"`
	Stages     []ApprovalFlowStageItem `json:"stages"`
	Configured bool                    `json:"configured"`
}

type ApprovalFlowUpsertRequest struct {
	Stages []ApprovalFlowStageUpsertItem `json:"stages" binding:"required,min=1"`
}

type ApprovalFlowStageUpsertItem struct {
	StageKey    string `json:"stage_key"`
	StageName   string `json:"stage_name"`
	SortOrder   int    `json:"sort_order"`
	Enabled     bool   `json:"enabled"`
	UserGroupID *uint  `json:"user_group_id"`
}

type ReleaseApprovalStepItem struct {
	ID             uint    `json:"id"`
	StageKey       string  `json:"stage_key"`
	StageName      string  `json:"stage_name"`
	SortOrder      int     `json:"sort_order"`
	Status         string  `json:"status"`
	UserGroupID    *uint   `json:"user_group_id,omitempty"`
	UserGroupName  string  `json:"user_group_name,omitempty"`
	ReviewerUserID *uint   `json:"reviewer_user_id,omitempty"`
	ReviewerName   string  `json:"reviewer_name"`
	ReviewComment  string  `json:"review_comment"`
	ReviewedAt     *string `json:"reviewed_at,omitempty"`
}

var (
	defaultApprovalFlowStages = []struct {
		Key, Name string
		Sort      int
	}{
		{model.CicdApprovalStageTestLead, "测试负责人", 1},
		{model.CicdApprovalStageRDLead, "研发负责人", 2},
		{model.CicdApprovalStageProductLead, "项目/产品负责人", 3},
		{model.CicdApprovalStageOpsLead, "运维负责人", 4},
	}
	stageKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}$`)
)

func stageNameByKey(key string) string {
	for _, d := range defaultApprovalFlowStages {
		if d.Key == key {
			return d.Name
		}
	}
	return key
}

func generateStageKey() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "custom_" + hex.EncodeToString(b[:]), nil
}

func normalizeStageKey(raw string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		return generateStageKey()
	}
	if !stageKeyPattern.MatchString(key) {
		return "", constants.ErrBadRequestWithMsg("审批节点 Key 须为小写字母开头，仅含 a-z/0-9/_，长度 2-32: " + raw)
	}
	return key, nil
}

func (s *Service) workflowEngine() *workflowsvc.Service {
	return workflowsvc.NewService(s.workflowRepo, s.userGroupRepo, s.dutyRepo, s.userRepo)
}

func (s *Service) GetApprovalFlow(ctx context.Context, projectID uint) (*ApprovalFlowResponse, error) {
	def, err := s.workflowEngine().GetDefinition(ctx, workflowsvc.DefinitionKey{
		Domain: model.WorkflowDomainCicd, ProjectID: projectID,
	}, workflowsvc.DefaultCicdStages())
	if err != nil {
		return nil, err
	}
	items := make([]ApprovalFlowStageItem, 0, len(def.Stages))
	for _, st := range def.Stages {
		items = append(items, ApprovalFlowStageItem{
			StageKey: st.StageKey, StageName: st.StageName, SortOrder: st.SortOrder,
			Enabled: st.Enabled, UserGroupID: st.UserGroupID, UserGroupName: st.UserGroupName,
		})
	}
	return &ApprovalFlowResponse{ProjectID: projectID, Stages: items, Configured: def.Configured}, nil
}

func (s *Service) UpsertApprovalFlow(ctx context.Context, projectID uint, req ApprovalFlowUpsertRequest, actor *auth.CurrentUser) (*ApprovalFlowResponse, error) {
	if err := s.requireProjectAdmin(ctx, projectID, actor); err != nil {
		return nil, err
	}
	stages := make([]workflowsvc.StageUpsertItem, 0, len(req.Stages))
	for _, st := range req.Stages {
		stages = append(stages, workflowsvc.StageUpsertItem{
			StageKey: st.StageKey, StageName: st.StageName, SortOrder: st.SortOrder,
			Enabled: st.Enabled, AssigneeRuleType: model.WorkflowAssigneeUserGroup,
			UserGroupID: st.UserGroupID,
		})
	}
	_, err := s.workflowEngine().UpsertDefinition(ctx, workflowsvc.DefinitionKey{
		Domain: model.WorkflowDomainCicd, ProjectID: projectID,
	}, workflowsvc.DefinitionUpsertRequest{Stages: stages})
	if err != nil {
		return nil, err
	}
	return s.GetApprovalFlow(ctx, projectID)
}

func (s *Service) loadEnabledFlowStages(ctx context.Context, projectID uint) ([]model.CicdApprovalFlowStage, error) {
	return workflowsvc.EnabledLegacyCicdStagesFrom(ctx, s.workflowEngine(), projectID)
}

func (s *Service) initReleaseApprovalSteps(ctx context.Context, release *model.CicdReleaseRun) error {
	if release == nil {
		return constants.ErrBadRequestWithMsg("工单不存在")
	}
	stages, err := s.loadEnabledFlowStages(ctx, release.ProjectID)
	if err != nil {
		return err
	}
	if len(stages) == 0 {
		return constants.ErrBadRequestWithMsg("请先在「审批流配置」中启用至少一级审批后再提交发布")
	}
	steps := make([]model.CicdReleaseApprovalStep, 0, len(stages))
	now := time.Now()
	for i, st := range stages {
		step := model.CicdReleaseApprovalStep{
			ReleaseRunID: release.ID,
			StageKey:     st.StageKey,
			StageName:    st.StageName,
			SortOrder:    st.SortOrder,
			Status:       model.CicdApprovalStepPending,
			UserGroupID:  st.UserGroupID,
		}
		if i == 0 {
			step.ActivatedAt = &now
		}
		steps = append(steps, step)
	}
	if err := s.repo.CreateApprovalSteps(ctx, steps); err != nil {
		return err
	}
	return s.repo.UpdateReleaseRunFields(ctx, release.ID, map[string]any{
		"current_stage_key": stages[0].StageKey,
	})
}

func (s *Service) getCurrentPendingStep(ctx context.Context, releaseRunID uint) (*model.CicdReleaseApprovalStep, error) {
	return s.repo.GetPendingApprovalStep(ctx, releaseRunID)
}

func (s *Service) userCanApproveStep(ctx context.Context, userID uint, step *model.CicdReleaseApprovalStep) (bool, error) {
	if userID == 0 || step == nil {
		return false, nil
	}
	if step.UserGroupID == nil || *step.UserGroupID == 0 {
		return false, constants.ErrBadRequestWithMsg("当前审批节点未配置用户组")
	}
	ids, err := s.userGroupRepo.ListMemberUserIDs(ctx, *step.UserGroupID)
	if err != nil {
		return false, err
	}
	return slices.Contains(ids, userID), nil
}

func (s *Service) advanceReleaseAfterApproval(ctx context.Context, release *model.CicdReleaseRun, step *model.CicdReleaseApprovalStep) error {
	next, err := s.nextPendingStepAfter(ctx, release.ID, step.SortOrder)
	if err != nil {
		return err
	}
	if next != nil {
		now := time.Now()
		if err := s.repo.UpdateApprovalStepFields(ctx, next.ID, map[string]any{
			"activated_at":     now,
			"last_reminded_at": nil,
		}); err != nil {
			return err
		}
		return s.repo.UpdateReleaseRunFields(ctx, release.ID, map[string]any{
			"current_stage_key": next.StageKey,
		})
	}
	return s.repo.UpdateReleaseRunFields(ctx, release.ID, map[string]any{
		"status":            model.CicdRunStatusPendingExecution,
		"current_stage_key": "",
	})
}

func (s *Service) nextPendingStepAfter(ctx context.Context, releaseRunID uint, afterSort int) (*model.CicdReleaseApprovalStep, error) {
	steps, err := s.repo.ListApprovalStepsByRun(ctx, releaseRunID)
	if err != nil {
		return nil, err
	}
	for _, step := range steps {
		if step.Status == model.CicdApprovalStepPending && step.SortOrder > afterSort {
			st := step
			return &st, nil
		}
	}
	return nil, nil
}

func (s *Service) ListReleaseApprovalSteps(ctx context.Context, projectID, runID uint, actor *auth.CurrentUser) ([]ReleaseApprovalStepItem, error) {
	if _, err := s.assertReleaseRunAccess(ctx, projectID, runID, actor, "view"); err != nil {
		return nil, err
	}
	return s.buildReleaseApprovalStepItems(ctx, runID)
}

func (s *Service) loadUserGroupNameMap(ctx context.Context, stages []model.CicdApprovalFlowStage) map[uint]string {
	ids := make([]uint, 0)
	seen := map[uint]struct{}{}
	for _, st := range stages {
		if st.UserGroupID != nil && *st.UserGroupID > 0 {
			if _, ok := seen[*st.UserGroupID]; !ok {
				seen[*st.UserGroupID] = struct{}{}
				ids = append(ids, *st.UserGroupID)
			}
		}
	}
	out := map[uint]string{}
	if len(ids) == 0 {
		return out
	}
	groups, _ := s.repo.ListUserGroupsByIDs(ctx, ids)
	for _, g := range groups {
		out[g.ID] = g.Name
	}
	return out
}

// backfillPendingReleaseSteps 为历史待审工单补建统一工单（不再写入遗留步骤表）。
func (s *Service) backfillPendingReleaseSteps(ctx context.Context, projectID uint) error {
	runs, err := s.repo.ListPendingApprovalReleases(ctx, projectID)
	if err != nil {
		return err
	}
	wf := s.workflowEngine()
	for i := range runs {
		if wf.HasLinkedTicketType(ctx, model.WorkflowRefCicdReleaseRun, runs[i].ID, model.WorkflowTicketTypeRelease) {
			continue
		}
		_ = s.createReleaseWorkflowTickets(ctx, &runs[i])
	}
	return nil
}

const (
	releaseMineStatusPending = "mine_pending"
	releaseMineStatusDone    = "mine_done"
)

// enrichReleaseRunMineStatus 待办列表按当前用户填充 mine_status（多级审批：我已审完显示已审批）。
func (s *Service) enrichReleaseRunMineStatus(ctx context.Context, items []ReleaseRunItem, userID uint, mineTab string) {
	if userID == 0 || len(items) == 0 {
		return
	}
	switch mineTab {
	case "execution":
		for i := range items {
			submitterID := uint(0)
			if items[i].SubmitterUserID != nil {
				submitterID = *items[i].SubmitterUserID
			}
			if submitterID != userID {
				continue
			}
			switch items[i].Status {
			case model.CicdRunStatusPendingExecution:
				items[i].MineStatus = releaseMineStatusPending
			case model.CicdRunStatusPendingApproval:
				// 仍待他人审批，提交人视角不算待执行
			default:
				items[i].MineStatus = releaseMineStatusDone
			}
		}
		return
	default:
		s.enrichReleaseRunApprovalMineStatus(ctx, items, userID)
	}
}

func (s *Service) enrichReleaseRunApprovalMineStatus(ctx context.Context, items []ReleaseRunItem, userID uint) {
	runIDs := make([]uint, 0, len(items))
	for _, it := range items {
		runIDs = append(runIDs, it.ID)
	}
	steps, _ := s.repo.ListApprovalStepsByRuns(ctx, runIDs)
	byRun := make(map[uint][]model.CicdReleaseApprovalStep, len(items))
	for _, st := range steps {
		byRun[st.ReleaseRunID] = append(byRun[st.ReleaseRunID], st)
	}
	for i := range items {
		item := &items[i]
		if item.Status != model.CicdRunStatusPendingApproval {
			if item.Status == model.CicdRunStatusRejected {
				item.MineStatus = releaseMineStatusDone
			}
			continue
		}
		sts := byRun[item.ID]
		if len(sts) == 0 {
			s.enrichReleaseMineFromWorkflow(ctx, item, userID)
			continue
		}
		for _, st := range sts {
			if st.ReviewerUserID != nil && *st.ReviewerUserID == userID &&
				(st.Status == model.CicdApprovalStepApproved || st.Status == model.CicdApprovalStepRejected) {
				item.MineStatus = releaseMineStatusDone
				break
			}
		}
		if item.MineStatus == releaseMineStatusDone {
			continue
		}
		var current *model.CicdReleaseApprovalStep
		for j := range sts {
			if sts[j].Status == model.CicdApprovalStepPending {
				current = &sts[j]
				break
			}
		}
		if current != nil {
			if ok, _ := s.userCanApproveStep(ctx, userID, current); ok {
				item.MineStatus = releaseMineStatusPending
			}
		}
	}
}

func (s *Service) enrichReleaseMineFromWorkflow(ctx context.Context, item *ReleaseRunItem, userID uint) {
	wf := s.workflowEngine()
	ticket, err := wf.GetTicketByRefType(ctx, model.WorkflowRefCicdReleaseRun, item.ID, model.WorkflowTicketTypeRelease)
	if err != nil || ticket == nil {
		if item.ReviewerUserID != nil && *item.ReviewerUserID == userID && item.ReviewedAt != nil {
			item.MineStatus = releaseMineStatusDone
		} else {
			item.MineStatus = releaseMineStatusPending
		}
		return
	}
	steps, err := s.workflowRepo.ListStepsByTicket(ctx, ticket.ID)
	if err != nil || len(steps) == 0 {
		if item.ReviewerUserID != nil && *item.ReviewerUserID == userID && item.ReviewedAt != nil {
			item.MineStatus = releaseMineStatusDone
		} else {
			item.MineStatus = releaseMineStatusPending
		}
		return
	}
	for _, st := range steps {
		if st.ReviewerUserID != nil && *st.ReviewerUserID == userID &&
			(st.Status == model.WorkflowStepApproved || st.Status == model.WorkflowStepRejected) {
			item.MineStatus = releaseMineStatusDone
			return
		}
	}
	for i := range steps {
		st := &steps[i]
		if st.Status != model.WorkflowStepPending || st.ActivatedAt == nil {
			continue
		}
		if st.AssigneeUserID != nil && *st.AssigneeUserID == userID {
			item.MineStatus = releaseMineStatusPending
			return
		}
		if st.UserGroupID != nil && *st.UserGroupID > 0 {
			if ok, _ := s.userCanApproveStep(ctx, userID, &model.CicdReleaseApprovalStep{UserGroupID: st.UserGroupID}); ok {
				item.MineStatus = releaseMineStatusPending
			}
		}
		return
	}
}
