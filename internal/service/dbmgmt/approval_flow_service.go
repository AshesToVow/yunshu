package dbmgmt

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

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
	StageKey    string `json:"stage_key"`
	StageName   string `json:"stage_name"`
	SortOrder   int    `json:"sort_order"`
	Enabled     bool   `json:"enabled"`
	UserGroupID *uint  `json:"user_group_id"`
}

var (
	defaultDbApprovalStages = []struct {
		Key, Name string
		Sort      int
	}{
		{model.DbApprovalStageDBALead, "DBA 负责人", 1},
		{model.DbApprovalStageSecurityLead, "安全负责人", 2},
		{model.DbApprovalStageOpsLead, "运维负责人", 3},
	}
	dbStageKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}$`)
)

func dbStageNameByKey(key string) string {
	for _, d := range defaultDbApprovalStages {
		if d.Key == key {
			return d.Name
		}
	}
	return key
}

func generateDbStageKey() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "custom_" + hex.EncodeToString(b[:]), nil
}

func normalizeDbStageKey(raw string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		return generateDbStageKey()
	}
	if !dbStageKeyPattern.MatchString(key) {
		return "", constants.ErrBadRequestWithMsg("审批节点 Key 须为小写字母开头，仅含 a-z/0-9/_，长度 2-32: " + raw)
	}
	return key, nil
}

func (s *Service) GetApprovalFlow(ctx context.Context, projectID uint) (*ApprovalFlowResponse, error) {
	rows, err := s.repo.ListApprovalFlowStages(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		items := make([]ApprovalFlowStageItem, 0, len(defaultDbApprovalStages))
		for _, d := range defaultDbApprovalStages {
			items = append(items, ApprovalFlowStageItem{
				StageKey: d.Key, StageName: d.Name, SortOrder: d.Sort, Enabled: false,
			})
		}
		return &ApprovalFlowResponse{ProjectID: projectID, Stages: items}, nil
	}
	groupNames := s.loadUserGroupNameMap(ctx, rows)
	items := make([]ApprovalFlowStageItem, 0, len(rows))
	for _, row := range rows {
		item := ApprovalFlowStageItem{
			StageKey: row.StageKey, StageName: row.StageName, SortOrder: row.SortOrder,
			Enabled: row.Enabled, UserGroupID: row.UserGroupID,
		}
		if row.UserGroupID != nil {
			item.UserGroupName = groupNames[*row.UserGroupID]
		}
		items = append(items, item)
	}
	return &ApprovalFlowResponse{ProjectID: projectID, Stages: items}, nil
}

func (s *Service) UpsertApprovalFlow(ctx context.Context, projectID uint, req ApprovalFlowUpsertRequest, actor *auth.CurrentUser) (*ApprovalFlowResponse, error) {
	if err := s.requireProjectAdminOrOwner(ctx, projectID, actor); err != nil {
		return nil, err
	}
	type normalizedStage struct {
		Key         string
		Name        string
		Sort        int
		Enabled     bool
		UserGroupID *uint
	}
	normalized := make([]normalizedStage, 0, len(req.Stages))
	seen := map[string]struct{}{}
	for i, st := range req.Stages {
		key, err := normalizeDbStageKey(st.StageKey)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[key]; ok {
			return nil, constants.ErrBadRequestWithMsg("审批节点 Key 重复: " + key)
		}
		seen[key] = struct{}{}
		name := strings.TrimSpace(st.StageName)
		if name == "" {
			name = dbStageNameByKey(key)
		}
		if utf8.RuneCountInString(name) > 64 {
			return nil, constants.ErrBadRequestWithMsg("审批节点名称过长: " + name)
		}
		if st.Enabled && (st.UserGroupID == nil || *st.UserGroupID == 0) {
			return nil, constants.ErrBadRequestWithMsg("启用的审批节点须绑定用户组: " + name)
		}
		if st.UserGroupID != nil && *st.UserGroupID > 0 {
			if _, err := s.userGroupRepo.GetByID(ctx, *st.UserGroupID); err != nil {
				return nil, constants.ErrBadRequestWithMsg("用户组不存在")
			}
		}
		sortOrder := st.SortOrder
		if sortOrder <= 0 {
			sortOrder = i + 1
		}
		var groupID *uint
		if st.UserGroupID != nil && *st.UserGroupID > 0 {
			groupID = st.UserGroupID
		}
		normalized = append(normalized, normalizedStage{
			Key: key, Name: name, Sort: sortOrder, Enabled: st.Enabled, UserGroupID: groupID,
		})
	}
	if len(normalized) == 0 {
		return nil, constants.ErrBadRequestWithMsg("至少保留一个审批节点")
	}
	keys := make([]string, 0, len(normalized))
	for _, st := range normalized {
		keys = append(keys, st.Key)
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, st := range normalized {
			var existing model.DbApprovalFlowStage
			err := tx.Where("project_id = ? AND stage_key = ?", projectID, st.Key).First(&existing).Error
			if err == gorm.ErrRecordNotFound {
				row := model.DbApprovalFlowStage{
					ProjectID: projectID, StageKey: st.Key, StageName: st.Name, SortOrder: st.Sort,
					Enabled: st.Enabled, UserGroupID: st.UserGroupID,
				}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			if err := tx.Model(&existing).Updates(map[string]any{
				"stage_name":    st.Name,
				"sort_order":    st.Sort,
				"enabled":       st.Enabled,
				"user_group_id": st.UserGroupID,
			}).Error; err != nil {
				return err
			}
		}
		q := tx.Where("project_id = ?", projectID)
		if len(keys) > 0 {
			q = q.Where("stage_key NOT IN ?", keys)
		}
		return q.Delete(&model.DbApprovalFlowStage{}).Error
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

// forbidSelfApprove 职责分离：提交人不得审批自己的单据（超级管理员可豁免）。
func (s *Service) forbidSelfApprove(ctx context.Context, actor *auth.CurrentUser, submitterUserID uint) error {
	cfg := s.resolvedConfig(ctx)
	if !cfg.ForbidSelfApprove {
		return nil
	}
	if actor != nil && auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	if submitterUserID == 0 {
		return nil
	}
	if actorUserID(actor) == submitterUserID {
		return constants.ErrForbiddenWithMsg("职责分离：提交人不可审批自己的申请/工单")
	}
	return nil
}

func (s *Service) initAccessRequestSteps(ctx context.Context, req *model.DbAccessRequest) error {
	stages, err := s.loadEnabledFlowStages(ctx, req.ProjectID)
	if err != nil {
		return err
	}
	if len(stages) == 0 {
		return constants.ErrBadRequestWithMsg("请先在「审批流配置」中启用至少一级审批后再提交权限申请")
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
		return constants.ErrBadRequestWithMsg("请先在「审批流配置」中启用至少一级审批后再提交 SQL 工单")
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

func isFinalAccessApprovalStep(steps []model.DbAccessRequestStep, cur *model.DbAccessRequestStep) bool {
	if cur == nil {
		return false
	}
	for _, st := range steps {
		if st.SortOrder > cur.SortOrder {
			return false
		}
	}
	return true
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
