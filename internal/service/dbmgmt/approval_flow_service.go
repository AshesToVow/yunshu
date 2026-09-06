package dbmgmt

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

func (s *Service) workflowEngine() *workflowsvc.Service {
	return workflowsvc.NewService(s.db, s.userGroupRepo, s.dutyRepo, s.userRepo)
}

func (s *Service) GetApprovalFlow(ctx context.Context, projectID uint) (*ApprovalFlowResponse, error) {
	def, err := s.workflowEngine().GetDefinition(ctx, workflowsvc.DefinitionKey{
		Domain: model.WorkflowDomainDbmgmt, ProjectID: projectID,
	}, workflowsvc.DefaultDbmgmtStages())
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
	return &ApprovalFlowResponse{ProjectID: projectID, Stages: items}, nil
}

func (s *Service) UpsertApprovalFlow(ctx context.Context, projectID uint, req ApprovalFlowUpsertRequest, actor *auth.CurrentUser) (*ApprovalFlowResponse, error) {
	if err := s.requireProjectAdminOrOwner(ctx, projectID, actor); err != nil {
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
		Domain: model.WorkflowDomainDbmgmt, ProjectID: projectID,
	}, workflowsvc.DefinitionUpsertRequest{Stages: stages})
	if err != nil {
		return nil, err
	}
	return s.GetApprovalFlow(ctx, projectID)
}

func (s *Service) loadEnabledFlowStages(ctx context.Context, projectID uint) ([]model.DbApprovalFlowStage, error) {
	return workflowsvc.EnabledLegacyDbmgmtStages(ctx, s.db, s.userGroupRepo, projectID)
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
	if err := s.createDbmgmtWorkflowTicket(ctx, model.WorkflowTicketTypeAccess, model.WorkflowRefDbAccessRequest,
		accessRequestWorkflowTitle(req), req.ProjectID, req.ID, req.RequesterUserID); err != nil {
		return err
	}
	req.Status = model.DbAccessRequestStatusPending
	return s.repo.UpdateAccessRequest(ctx, req)
}

func (s *Service) initSqlTicketSteps(ctx context.Context, ticket *model.DbSqlTicket) error {
	if err := s.createDbmgmtWorkflowTicket(ctx, model.WorkflowTicketTypeSql, model.WorkflowRefDbSqlTicket,
		sqlTicketWorkflowTitle(ticket), ticket.ProjectID, ticket.ID, ticket.SubmitterUserID); err != nil {
		return err
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
