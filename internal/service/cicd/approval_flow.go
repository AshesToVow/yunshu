package cicd

import (
	"context"
	"slices"
	"strings"

	"yunshu/internal/model"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/constants"

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
	ProjectID uint                    `json:"project_id"`
	Stages    []ApprovalFlowStageItem `json:"stages"`
}

type ApprovalFlowUpsertRequest struct {
	Stages []ApprovalFlowStageUpsertItem `json:"stages" binding:"required,min=1"`
}

type ApprovalFlowStageUpsertItem struct {
	StageKey    string `json:"stage_key" binding:"required"`
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

var defaultApprovalFlowStages = []struct {
	Key, Name string
	Sort      int
}{
	{model.CicdApprovalStageTestLead, "测试负责人", 1},
	{model.CicdApprovalStageRDLead, "研发负责人", 2},
	{model.CicdApprovalStageProductLead, "项目/产品负责人", 3},
	{model.CicdApprovalStageOpsLead, "运维负责人", 4},
}

func stageNameByKey(key string) string {
	for _, d := range defaultApprovalFlowStages {
		if d.Key == key {
			return d.Name
		}
	}
	return key
}

func (s *Service) GetApprovalFlow(ctx context.Context, projectID uint) (*ApprovalFlowResponse, error) {
	var rows []model.CicdApprovalFlowStage
	if err := s.db.WithContext(ctx).Where("project_id = ?", projectID).Order("sort_order ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		stages := make([]ApprovalFlowStageItem, 0, len(defaultApprovalFlowStages))
		for _, d := range defaultApprovalFlowStages {
			stages = append(stages, ApprovalFlowStageItem{
				StageKey:  d.Key,
				StageName: d.Name,
				SortOrder: d.Sort,
				Enabled:   true,
			})
		}
		return &ApprovalFlowResponse{ProjectID: projectID, Stages: stages}, nil
	}
	groupNames := s.loadUserGroupNameMap(ctx, rows)
	items := make([]ApprovalFlowStageItem, 0, len(rows))
	for _, row := range rows {
		item := ApprovalFlowStageItem{
			StageKey:    row.StageKey,
			StageName:   row.StageName,
			SortOrder:   row.SortOrder,
			Enabled:     row.Enabled,
			UserGroupID: row.UserGroupID,
		}
		if row.UserGroupID != nil {
			item.UserGroupName = groupNames[*row.UserGroupID]
		}
		items = append(items, item)
	}
	return &ApprovalFlowResponse{ProjectID: projectID, Stages: items}, nil
}

func (s *Service) UpsertApprovalFlow(ctx context.Context, projectID uint, req ApprovalFlowUpsertRequest) (*ApprovalFlowResponse, error) {
	allowed := map[string]struct{}{}
	for _, d := range defaultApprovalFlowStages {
		allowed[d.Key] = struct{}{}
	}
	incoming := map[string]ApprovalFlowStageUpsertItem{}
	for _, st := range req.Stages {
		key := strings.TrimSpace(st.StageKey)
		if key == "" {
			continue
		}
		if _, ok := allowed[key]; !ok {
			return nil, constants.ErrBadRequestWithMsg("无效的审批节点: " + key)
		}
		if st.Enabled && (st.UserGroupID == nil || *st.UserGroupID == 0) {
			return nil, constants.ErrBadRequestWithMsg("启用的审批节点须绑定用户组: " + stageNameByKey(key))
		}
		if st.UserGroupID != nil && *st.UserGroupID > 0 {
			if _, err := s.userGroupRepo.GetByID(ctx, *st.UserGroupID); err != nil {
				return nil, constants.ErrBadRequestWithMsg("用户组不存在")
			}
		}
		incoming[key] = st
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, d := range defaultApprovalFlowStages {
			st, ok := incoming[d.Key]
			enabled := ok && st.Enabled
			var groupID *uint
			if ok && st.UserGroupID != nil && *st.UserGroupID > 0 {
				groupID = st.UserGroupID
			}
			var row model.CicdApprovalFlowStage
			err := tx.Where("project_id = ? AND stage_key = ?", projectID, d.Key).First(&row).Error
			if err != nil {
				row = model.CicdApprovalFlowStage{
					ProjectID:   projectID,
					StageKey:    d.Key,
					StageName:   d.Name,
					SortOrder:   d.Sort,
					Enabled:     enabled,
					UserGroupID: groupID,
				}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Model(&row).Updates(map[string]any{
				"enabled":       enabled,
				"user_group_id": groupID,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetApprovalFlow(ctx, projectID)
}

func (s *Service) loadEnabledFlowStages(ctx context.Context, projectID uint) ([]model.CicdApprovalFlowStage, error) {
	var rows []model.CicdApprovalFlowStage
	if err := s.db.WithContext(ctx).
		Where("project_id = ? AND enabled = ?", projectID, true).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
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
		return s.db.WithContext(ctx).Model(release).Updates(map[string]any{
			"status":            model.CicdRunStatusPendingExecution,
			"current_stage_key": "",
		}).Error
	}
	steps := make([]model.CicdReleaseApprovalStep, 0, len(stages))
	for _, st := range stages {
		steps = append(steps, model.CicdReleaseApprovalStep{
			ReleaseRunID: release.ID,
			StageKey:     st.StageKey,
			StageName:    st.StageName,
			SortOrder:    st.SortOrder,
			Status:       model.CicdApprovalStepPending,
			UserGroupID:  st.UserGroupID,
		})
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

func (s *Service) ListReleaseApprovalSteps(ctx context.Context, projectID, runID uint) ([]ReleaseApprovalStepItem, error) {
	var release model.CicdReleaseRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&release).Error; err != nil {
		return nil, bizerrors.NotFound("release run", runID)
	}
	var steps []model.CicdReleaseApprovalStep
	if err := s.db.WithContext(ctx).Where("release_run_id = ?", runID).Order("sort_order ASC, id ASC").Find(&steps).Error; err != nil {
		return nil, err
	}
	items := make([]ReleaseApprovalStepItem, 0, len(steps))
	for _, st := range steps {
		item := ReleaseApprovalStepItem{
			ID:             st.ID,
			StageKey:       st.StageKey,
			StageName:      st.StageName,
			SortOrder:      st.SortOrder,
			Status:         st.Status,
			UserGroupID:    st.UserGroupID,
			ReviewerUserID: st.ReviewerUserID,
			ReviewerName:   st.ReviewerName,
			ReviewComment:  st.ReviewComment,
		}
		if st.ReviewedAt != nil {
			ts := st.ReviewedAt.Format("2006-01-02 15:04:05")
			item.ReviewedAt = &ts
		}
		items = append(items, item)
	}
	return items, nil
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
	return dbq.Where("NOT EXISTS (?) OR EXISTS (?)", noSteps, currentStep)
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
