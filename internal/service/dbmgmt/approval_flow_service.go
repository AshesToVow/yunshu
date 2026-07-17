package dbmgmt

import (
	"context"
	"slices"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
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

var defaultDbApprovalStages = []struct {
	Key, Name string
	Sort      int
}{
	{model.DbApprovalStageDBALead, "DBA 负责人", 1},
	{model.DbApprovalStageSecurityLead, "安全负责人", 2},
	{model.DbApprovalStageOpsLead, "运维负责人", 3},
}

func dbStageNameByKey(key string) string {
	for _, d := range defaultDbApprovalStages {
		if d.Key == key {
			return d.Name
		}
	}
	return key
}

func (s *Service) GetApprovalFlow(ctx context.Context, projectID uint) (*ApprovalFlowResponse, error) {
	rows, err := s.repo.ListApprovalFlowStages(ctx, projectID)
	if err != nil {
		return nil, err
	}
	byKey := map[string]model.DbApprovalFlowStage{}
	for _, row := range rows {
		byKey[row.StageKey] = row
	}
	groupNames := s.loadUserGroupNameMap(ctx, rows)
	items := make([]ApprovalFlowStageItem, 0, len(defaultDbApprovalStages))
	for _, d := range defaultDbApprovalStages {
		if row, ok := byKey[d.Key]; ok {
			item := ApprovalFlowStageItem{
				StageKey: row.StageKey, StageName: row.StageName, SortOrder: row.SortOrder,
				Enabled: row.Enabled, UserGroupID: row.UserGroupID,
			}
			if row.UserGroupID != nil {
				item.UserGroupName = groupNames[*row.UserGroupID]
			}
			items = append(items, item)
			continue
		}
		items = append(items, ApprovalFlowStageItem{
			StageKey: d.Key, StageName: d.Name, SortOrder: d.Sort, Enabled: false,
		})
	}
	return &ApprovalFlowResponse{ProjectID: projectID, Stages: items}, nil
}

func (s *Service) UpsertApprovalFlow(ctx context.Context, projectID uint, req ApprovalFlowUpsertRequest, actor *auth.CurrentUser) (*ApprovalFlowResponse, error) {
	if err := s.requireProjectAdminOrOwner(ctx, projectID, actor); err != nil {
		return nil, err
	}
	allowed := map[string]struct{}{}
	for _, d := range defaultDbApprovalStages {
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
			return nil, constants.ErrBadRequestWithMsg("启用的审批节点须绑定用户组: " + dbStageNameByKey(key))
		}
		if st.UserGroupID != nil && *st.UserGroupID > 0 {
			if _, err := s.userGroupRepo.GetByID(ctx, *st.UserGroupID); err != nil {
				return nil, constants.ErrBadRequestWithMsg("用户组不存在")
			}
		}
		incoming[key] = st
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, d := range defaultDbApprovalStages {
			st, ok := incoming[d.Key]
			enabled := ok && st.Enabled
			var groupID *uint
			if ok && st.UserGroupID != nil && *st.UserGroupID > 0 {
				groupID = st.UserGroupID
			}
			stage := &model.DbApprovalFlowStage{
				ProjectID: projectID, StageKey: d.Key, StageName: d.Name, SortOrder: d.Sort,
				Enabled: enabled, UserGroupID: groupID,
			}
			var existing model.DbApprovalFlowStage
			err := tx.Where("project_id = ? AND stage_key = ?", projectID, d.Key).First(&existing).Error
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(stage).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			existing.Enabled = enabled
			existing.UserGroupID = groupID
			if err := tx.Save(&existing).Error; err != nil {
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

func (s *Service) loadEnabledFlowStages(ctx context.Context, projectID uint) ([]model.DbApprovalFlowStage, error) {
	var rows []model.DbApprovalFlowStage
	err := s.db.WithContext(ctx).Where("project_id = ? AND enabled = ?", projectID, true).
		Order("sort_order ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (s *Service) loadUserGroupNameMap(ctx context.Context, stages []model.DbApprovalFlowStage) map[uint]string {
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

func (s *Service) userCanApproveStep(ctx context.Context, actor *auth.CurrentUser, groupID *uint) (bool, error) {
	if actor != nil && auth.IsSuperAdminRole(actor.RoleCodes) {
		return true, nil
	}
	userID := actorUserID(actor)
	if userID == 0 || groupID == nil || *groupID == 0 {
		return false, nil
	}
	ids, err := s.userGroupRepo.ListMemberUserIDs(ctx, *groupID)
	if err != nil {
		return false, err
	}
	return slices.Contains(ids, userID), nil
}

func (s *Service) initAccessRequestSteps(ctx context.Context, req *model.DbAccessRequest) error {
	stages, err := s.loadEnabledFlowStages(ctx, req.ProjectID)
	if err != nil {
		return err
	}
	if len(stages) == 0 {
		inst, instErr := s.repo.GetInstance(ctx, req.InstanceID)
		if instErr == nil && inst.Env == model.DbEnvProd {
			return constants.ErrBadRequestWithMsg("生产环境须启用至少一级审批流后方可提交权限申请")
		}
		req.Status = model.DbAccessRequestStatusApproved
		if err := s.repo.UpdateAccessRequest(ctx, req); err != nil {
			return err
		}
		return s.grantFromAccessRequest(ctx, req)
	}
	now := time.Now()
	for i, st := range stages {
		step := &model.DbAccessRequestStep{
			AccessRequestID: req.ID, StageKey: st.StageKey, StageName: st.StageName,
			SortOrder: st.SortOrder, Status: model.DbApprovalStepPending, UserGroupID: st.UserGroupID,
		}
		if i == 0 {
			step.ActivatedAt = &now
		}
		if err := s.repo.CreateAccessRequestStep(ctx, step); err != nil {
			return err
		}
	}
	req.Status = model.DbAccessRequestStatusPending
	return s.repo.UpdateAccessRequest(ctx, req)
}

func (s *Service) initSqlTicketSteps(ctx context.Context, ticket *model.DbSqlTicket) error {
	stages, err := s.loadEnabledFlowStages(ctx, ticket.ProjectID)
	if err != nil {
		return err
	}
	if len(stages) == 0 {
		inst, instErr := s.repo.GetInstance(ctx, ticket.InstanceID)
		if instErr == nil && inst.Env == model.DbEnvProd {
			return constants.ErrBadRequestWithMsg("生产环境须启用至少一级审批流后方可提交 SQL 工单")
		}
		ticket.Status = model.DbTicketStatusPendingExecution
		return s.repo.UpdateSqlTicket(ctx, ticket)
	}
	now := time.Now()
	for i, st := range stages {
		step := &model.DbSqlTicketStep{
			TicketID: ticket.ID, StageKey: st.StageKey, StageName: st.StageName,
			SortOrder: st.SortOrder, Status: model.DbApprovalStepPending, UserGroupID: st.UserGroupID,
		}
		if i == 0 {
			step.ActivatedAt = &now
		}
		if err := s.repo.CreateSqlTicketStep(ctx, step); err != nil {
			return err
		}
	}
	ticket.Status = model.DbTicketStatusPendingApproval
	return s.repo.UpdateSqlTicket(ctx, ticket)
}

func (s *Service) advanceAccessRequestAfterApproval(ctx context.Context, req *model.DbAccessRequest, step *model.DbAccessRequestStep) error {
	steps, err := s.repo.ListAccessRequestSteps(ctx, req.ID)
	if err != nil {
		return err
	}
	var next *model.DbAccessRequestStep
	for i := range steps {
		if steps[i].SortOrder > step.SortOrder && steps[i].Status == model.DbApprovalStepPending {
			next = &steps[i]
			break
		}
	}
	if next != nil {
		now := time.Now()
		next.ActivatedAt = &now
		return s.repo.UpdateAccessRequestStep(ctx, next)
	}
	req.Status = model.DbAccessRequestStatusApproved
	if err := s.repo.UpdateAccessRequest(ctx, req); err != nil {
		return err
	}
	return s.grantFromAccessRequest(ctx, req)
}

func (s *Service) advanceSqlTicketAfterApproval(ctx context.Context, ticket *model.DbSqlTicket, step *model.DbSqlTicketStep) error {
	steps, err := s.repo.ListSqlTicketSteps(ctx, ticket.ID)
	if err != nil {
		return err
	}
	var next *model.DbSqlTicketStep
	for i := range steps {
		if steps[i].SortOrder > step.SortOrder && steps[i].Status == model.DbApprovalStepPending {
			next = &steps[i]
			break
		}
	}
	if next != nil {
		now := time.Now()
		next.ActivatedAt = &now
		return s.repo.UpdateSqlTicketStep(ctx, next)
	}
	ticket.Status = model.DbTicketStatusPendingExecution
	return s.repo.UpdateSqlTicket(ctx, ticket)
}
