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

	"gorm.io/gorm"
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
	return workflowsvc.NewService(s.db, s.userGroupRepo, nil, s.userRepo)
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
	return workflowsvc.EnabledLegacyCicdStages(ctx, s.db, s.userGroupRepo, projectID)
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
	if err := s.db.WithContext(ctx).Create(&steps).Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(release).Updates(map[string]any{
		"current_stage_key": stages[0].StageKey,
	}).Error
}

func (s *Service) getCurrentPendingStep(ctx context.Context, releaseRunID uint) (*model.CicdReleaseApprovalStep, error) {
	var step model.CicdReleaseApprovalStep
	err := s.db.WithContext(ctx).
		Where("release_run_id = ? AND status = ?", releaseRunID, model.CicdApprovalStepPending).
		Order("sort_order ASC, id ASC").
		First(&step).Error
	if err != nil {
		return nil, err
	}
	return &step, nil
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
		if err := s.db.WithContext(ctx).Model(next).Updates(map[string]any{
			"activated_at":     now,
			"last_reminded_at": nil,
		}).Error; err != nil {
			return err
		}
		return s.db.WithContext(ctx).Model(release).Updates(map[string]any{
			"current_stage_key": next.StageKey,
		}).Error
	}
	return s.db.WithContext(ctx).Model(release).Updates(map[string]any{
		"status":            model.CicdRunStatusPendingExecution,
		"current_stage_key": "",
	}).Error
}

func (s *Service) nextPendingStepAfter(ctx context.Context, releaseRunID uint, afterSort int) (*model.CicdReleaseApprovalStep, error) {
	var step model.CicdReleaseApprovalStep
	err := s.db.WithContext(ctx).
		Where("release_run_id = ? AND status = ? AND sort_order > ?", releaseRunID, model.CicdApprovalStepPending, afterSort).
		Order("sort_order ASC, id ASC").
		First(&step).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &step, nil
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
	var groups []model.UserGroup
	_ = s.db.WithContext(ctx).Select("id, name").Where("id IN ?", ids).Find(&groups).Error
	for _, g := range groups {
		out[g.ID] = g.Name
	}
	return out
}

func (s *Service) filterReleaseRunsForApprover(dbq *gorm.DB, userID uint) *gorm.DB {
	// 无审批步骤的旧工单：与单级审批一致，项目成员均可见（审批时走 legacy 逻辑）
	noSteps := s.db.Table("cicd_release_approval_steps AS s0").
		Select("1").
		Where("s0.release_run_id = cicd_release_runs.id")
	currentStep := s.db.Table("cicd_release_approval_steps AS s").
		Select("1").
		Joins("JOIN user_group_users AS ugu ON ugu.user_group_id = s.user_group_id AND ugu.user_id = ?", userID).
		Where("s.release_run_id = cicd_release_runs.id").
		Where("s.status = ?", model.CicdApprovalStepPending).
		Where("s.user_group_id IS NOT NULL AND s.user_group_id > 0").
		Where(`s.sort_order = (
			SELECT MIN(s2.sort_order) FROM cicd_release_approval_steps s2
			WHERE s2.release_run_id = cicd_release_runs.id AND s2.status = ?
		)`, model.CicdApprovalStepPending)
	return dbq.Where("cicd_release_runs.status = ?", model.CicdRunStatusPendingApproval).
		Where("NOT EXISTS (?) OR EXISTS (?)", noSteps, currentStep)
}

// filterReleaseRunsApprovalDone 当前用户已处理过的审批（通过或驳回）。
func (s *Service) filterReleaseRunsApprovalDone(dbq *gorm.DB, userID uint) *gorm.DB {
	actedStep := s.db.Table("cicd_release_approval_steps AS s").
		Select("1").
		Where("s.release_run_id = cicd_release_runs.id").
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.CicdApprovalStepApproved, model.CicdApprovalStepRejected})
	legacy := s.db.Table("cicd_release_runs AS lr").
		Select("1").
		Where("lr.id = cicd_release_runs.id").
		Where("lr.reviewer_user_id = ?", userID).
		Where("lr.reviewed_at IS NOT NULL")
	return dbq.Where("EXISTS (?) OR EXISTS (?)", actedStep, legacy)
}

// filterReleaseRunsExecutionDone 当前用户作为提交人且已触发执行（非待审/待执行）。
func (s *Service) filterReleaseRunsExecutionDone(dbq *gorm.DB, userID uint) *gorm.DB {
	return dbq.Where("submitter_user_id = ?", userID).
		Where("status NOT IN ?", []string{
			model.CicdRunStatusPendingApproval,
			model.CicdRunStatusPendingExecution,
		})
}

// filterReleaseRunsForMineUser 待办「全部」：待处理 + 已处理。
func (s *Service) filterReleaseRunsApprovalMine(dbq *gorm.DB, userID uint) *gorm.DB {
	pending := s.approvalPendingExistsSubquery(userID)
	done := s.approvalDoneExistsSubquery(userID)
	return dbq.Where("EXISTS (?) OR EXISTS (?)", pending, done)
}

func (s *Service) filterReleaseRunsExecutionMine(dbq *gorm.DB, userID uint) *gorm.DB {
	pending := s.db.Table("cicd_release_runs AS r").
		Select("1").
		Where("r.id = cicd_release_runs.id").
		Where("r.submitter_user_id = ?", userID).
		Where("r.status = ?", model.CicdRunStatusPendingExecution)
	done := s.db.Table("cicd_release_runs AS r").
		Select("1").
		Where("r.id = cicd_release_runs.id").
		Where("r.submitter_user_id = ?", userID).
		Where("r.status NOT IN ?", []string{
			model.CicdRunStatusPendingApproval,
			model.CicdRunStatusPendingExecution,
		})
	return dbq.Where("EXISTS (?) OR EXISTS (?)", pending, done)
}

func (s *Service) approvalPendingExistsSubquery(userID uint) *gorm.DB {
	noSteps := s.db.Table("cicd_release_approval_steps AS s0").
		Select("1").
		Where("s0.release_run_id = r.id")
	currentStep := s.db.Table("cicd_release_approval_steps AS s").
		Select("1").
		Joins("JOIN user_group_users AS ugu ON ugu.user_group_id = s.user_group_id AND ugu.user_id = ?", userID).
		Where("s.release_run_id = r.id").
		Where("s.status = ?", model.CicdApprovalStepPending).
		Where("s.user_group_id IS NOT NULL AND s.user_group_id > 0").
		Where(`s.sort_order = (
			SELECT MIN(s2.sort_order) FROM cicd_release_approval_steps s2
			WHERE s2.release_run_id = r.id AND s2.status = ?
		)`, model.CicdApprovalStepPending)
	return s.db.Table("cicd_release_runs AS r").
		Select("1").
		Where("r.id = cicd_release_runs.id").
		Where("r.status = ?", model.CicdRunStatusPendingApproval).
		Where("NOT EXISTS (?) OR EXISTS (?)", noSteps, currentStep)
}

func (s *Service) approvalDoneExistsSubquery(userID uint) *gorm.DB {
	actedStep := s.db.Table("cicd_release_approval_steps AS s").
		Select("1").
		Where("s.release_run_id = r.id").
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.CicdApprovalStepApproved, model.CicdApprovalStepRejected})
	legacy := s.db.Table("cicd_release_runs AS lr").
		Select("1").
		Where("lr.id = r.id").
		Where("lr.reviewer_user_id = ?", userID).
		Where("lr.reviewed_at IS NOT NULL")
	return s.db.Table("cicd_release_runs AS r").
		Select("1").
		Where("r.id = cicd_release_runs.id").
		Where("EXISTS (?) OR EXISTS (?)", actedStep, legacy)
}

// backfillPendingReleaseSteps 为历史待审工单补建审批步骤（配置审批流之后提交的旧数据）。
func (s *Service) backfillPendingReleaseSteps(ctx context.Context, projectID uint) error {
	var runs []model.CicdReleaseRun
	if err := s.db.WithContext(ctx).
		Where("project_id = ? AND status = ? AND audit_enabled = ?", projectID, model.CicdRunStatusPendingApproval, true).
		Find(&runs).Error; err != nil {
		return err
	}
	for i := range runs {
		has, err := s.hasApprovalSteps(ctx, runs[i].ID)
		if err != nil || has {
			continue
		}
		_ = s.initReleaseApprovalSteps(ctx, &runs[i])
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
	var steps []model.CicdReleaseApprovalStep
	_ = s.db.WithContext(ctx).Where("release_run_id IN ?", runIDs).Order("sort_order ASC, id ASC").Find(&steps).Error
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
			if item.ReviewerUserID != nil && *item.ReviewerUserID == userID && item.ReviewedAt != nil {
				item.MineStatus = releaseMineStatusDone
			} else {
				item.MineStatus = releaseMineStatusPending
			}
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