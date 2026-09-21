package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

var stageKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}$`)

// Service 统一工单引擎：流程定义 + 通用工单 + 排班派单。
type Service struct {
	repo           interfaces.WorkflowRepository
	userGroupRepo  interfaces.UserGroupRepository
	dutyRepo       interfaces.AlertDutyRepository
	userRepo       interfaces.UserRepository
	alertEventRepo interfaces.AlertEventRepository
}

// NewService 创建工单引擎。
func NewService(
	repo interfaces.WorkflowRepository,
	userGroupRepo interfaces.UserGroupRepository,
	dutyRepo interfaces.AlertDutyRepository,
	userRepo interfaces.UserRepository,
) *Service {
	return &Service{
		repo:          repo,
		userGroupRepo: userGroupRepo,
		dutyRepo:      dutyRepo,
		userRepo:      userRepo,
	}
}

// SetAlertEventRepo 注入告警事件查询（告警转故障单取 project_id）。
func (s *Service) SetAlertEventRepo(repo interfaces.AlertEventRepository) {
	if s == nil {
		return
	}
	s.alertEventRepo = repo
}

type DefinitionKey struct {
	Domain     string
	ProjectID  uint
	TicketType string
}

func (k DefinitionKey) normalize() DefinitionKey {
	k.Domain = strings.ToLower(strings.TrimSpace(k.Domain))
	if k.TicketType == "" {
		k.TicketType = model.WorkflowTicketTypeDefault
	}
	return k
}

type StageItem struct {
	StageKey          string `json:"stage_key"`
	StageName         string `json:"stage_name"`
	SortOrder         int    `json:"sort_order"`
	Enabled           bool   `json:"enabled"`
	AssigneeRuleType  string `json:"assignee_rule_type"`
	UserGroupID       *uint  `json:"user_group_id,omitempty"`
	UserGroupName     string `json:"user_group_name,omitempty"`
	DutyMonitorRuleID *uint  `json:"duty_monitor_rule_id,omitempty"`
}

type DefinitionResponse struct {
	Domain     string      `json:"domain"`
	ProjectID  uint        `json:"project_id"`
	TicketType string      `json:"ticket_type"`
	Configured bool        `json:"configured"`
	Stages     []StageItem `json:"stages"`
}

type DefinitionUpsertRequest struct {
	Stages []StageUpsertItem `json:"stages" binding:"required,min=1"`
}

type StageUpsertItem struct {
	StageKey          string `json:"stage_key"`
	StageName         string `json:"stage_name"`
	SortOrder         int    `json:"sort_order"`
	Enabled           bool   `json:"enabled"`
	AssigneeRuleType  string `json:"assignee_rule_type"`
	UserGroupID       *uint  `json:"user_group_id"`
	DutyMonitorRuleID *uint  `json:"duty_monitor_rule_id"`
}

// GetDefinition 读取流程定义；无配置时返回域默认节点骨架。
func (s *Service) GetDefinition(ctx context.Context, key DefinitionKey, defaults []StageItem) (*DefinitionResponse, error) {
	key = key.normalize()
	def, stages, err := s.loadDefinition(ctx, key)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "GetDefinition", err)
	}
	if def == nil {
		out := &DefinitionResponse{
			Domain: key.Domain, ProjectID: key.ProjectID, TicketType: key.TicketType,
			Configured: false, Stages: append([]StageItem(nil), defaults...),
		}
		return out, nil
	}
	groupNames := s.loadUserGroupNameMap(ctx, stages)
	items := make([]StageItem, 0, len(stages))
	for _, st := range stages {
		item := stageToItem(st, groupNames)
		items = append(items, item)
	}
	return &DefinitionResponse{
		Domain: key.Domain, ProjectID: key.ProjectID, TicketType: key.TicketType,
		Configured: true, Stages: items,
	}, nil
}

// UpsertDefinition 保存流程定义（全量替换节点集合）。
func (s *Service) UpsertDefinition(ctx context.Context, key DefinitionKey, req DefinitionUpsertRequest) (*DefinitionResponse, error) {
	key = key.normalize()
	normalized, err := normalizeStages(req.Stages)
	if err != nil {
		return nil, err
	}
	err = s.repo.Transaction(ctx, func(tx interfaces.WorkflowRepository) error {
		def, _, err := tx.LoadDefinition(ctx, key.Domain, key.ProjectID, key.TicketType)
		if err != nil {
			return err
		}
		if def == nil {
			def = &model.WorkflowDefinition{
				Domain: key.Domain, ProjectID: key.ProjectID, TicketType: key.TicketType,
				Name: key.Domain + " workflow", Enabled: true, ForbidSelfApprove: true,
			}
			if err := tx.CreateDefinition(ctx, def); err != nil {
				return err
			}
		}
		keys := make([]string, 0, len(normalized))
		for _, st := range normalized {
			keys = append(keys, st.Key)
			existing, err := tx.GetStageByKey(ctx, def.ID, st.Key)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				row := model.WorkflowStage{
					DefinitionID: def.ID, StageKey: st.Key, StageName: st.Name, SortOrder: st.Sort,
					Enabled: st.Enabled, AssigneeRuleType: st.RuleType,
					UserGroupID: st.UserGroupID, DutyMonitorRuleID: st.DutyRuleID,
				}
				if err := tx.CreateStage(ctx, &row); err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			if err := tx.UpdateStageFields(ctx, existing.ID, map[string]any{
				"stage_name": st.Name, "sort_order": st.Sort, "enabled": st.Enabled,
				"assignee_rule_type": st.RuleType, "user_group_id": st.UserGroupID,
				"duty_monitor_rule_id": st.DutyRuleID,
			}); err != nil {
				return err
			}
		}
		return tx.DeleteStagesNotIn(ctx, def.ID, keys)
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "UpsertDefinition", err)
	}
	return s.GetDefinition(ctx, key, nil)
}

type normalizedStage struct {
	Key, Name, RuleType string
	Sort                int
	Enabled             bool
	UserGroupID         *uint
	DutyRuleID          *uint
}

func normalizeStages(items []StageUpsertItem) ([]normalizedStage, error) {
	out := make([]normalizedStage, 0, len(items))
	seen := map[string]struct{}{}
	for i, st := range items {
		key, err := normalizeStageKey(st.StageKey)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[key]; ok {
			return nil, constants.ErrBadRequestWithMsg("审批节点 Key 重复: " + key)
		}
		seen[key] = struct{}{}
		name := strings.TrimSpace(st.StageName)
		if name == "" {
			name = key
		}
		if utf8.RuneCountInString(name) > 64 {
			return nil, constants.ErrBadRequestWithMsg("审批节点名称过长: " + name)
		}
		ruleType := strings.ToLower(strings.TrimSpace(st.AssigneeRuleType))
		if ruleType == "" {
			ruleType = model.WorkflowAssigneeUserGroup
		}
		if st.Enabled {
			switch ruleType {
			case model.WorkflowAssigneeUserGroup:
				if st.UserGroupID == nil || *st.UserGroupID == 0 {
					return nil, constants.ErrBadRequestWithMsg("启用的审批节点须绑定用户组: " + name)
				}
			case model.WorkflowAssigneeDuty:
				if st.DutyMonitorRuleID == nil || *st.DutyMonitorRuleID == 0 {
					return nil, constants.ErrBadRequestWithMsg("排班派单节点须绑定监控规则: " + name)
				}
			case model.WorkflowAssigneePlatformRole:
				// 平台角色审批（AI 高危操作等），无需绑定用户组
			default:
				return nil, constants.ErrBadRequestWithMsg("不支持的派单规则: " + ruleType)
			}
		}
		sortOrder := st.SortOrder
		if sortOrder <= 0 {
			sortOrder = i + 1
		}
		out = append(out, normalizedStage{
			Key: key, Name: name, Sort: sortOrder, Enabled: st.Enabled, RuleType: ruleType,
			UserGroupID: st.UserGroupID, DutyRuleID: st.DutyMonitorRuleID,
		})
	}
	if len(out) == 0 {
		return nil, constants.ErrBadRequestWithMsg("至少保留一个审批节点")
	}
	return out, nil
}

// EnabledStages 返回项目下已启用的流程节点（供 dbmgmt/cicd 初始化审批步骤）。
// 指定 ticket_type 无启用节点时，回退到同域同项目的 default 流程（与审批流配置页一致）。
func (s *Service) EnabledStages(ctx context.Context, key DefinitionKey) ([]model.WorkflowStage, error) {
	_, stages, err := s.resolveFlow(ctx, key)
	return stages, err
}

// resolveFlow 解析流程定义与启用节点；特定类型未配置时回退 default；
// incident 域再回退到全局 project_id=0（告警转工单开箱）。
func (s *Service) resolveFlow(ctx context.Context, key DefinitionKey) (*model.WorkflowDefinition, []model.WorkflowStage, error) {
	key = key.normalize()
	def, stages, err := s.loadDefinition(ctx, key)
	if err != nil {
		return nil, nil, err
	}
	enabled := filterEnabledStages(stages)
	if len(enabled) > 0 && def != nil {
		return def, enabled, nil
	}
	if key.TicketType != model.WorkflowTicketTypeDefault {
		fallback := DefinitionKey{Domain: key.Domain, ProjectID: key.ProjectID, TicketType: model.WorkflowTicketTypeDefault}
		def2, stages2, err2 := s.loadDefinition(ctx, fallback)
		if err2 != nil {
			return nil, nil, err2
		}
		enabled2 := filterEnabledStages(stages2)
		if len(enabled2) > 0 && def2 != nil {
			return def2, enabled2, nil
		}
	}
	if key.Domain == model.WorkflowDomainIncident && key.ProjectID != 0 {
		for _, tt := range []string{key.TicketType, model.WorkflowTicketTypeDefault} {
			if tt == "" {
				continue
			}
			gKey := DefinitionKey{Domain: key.Domain, ProjectID: 0, TicketType: tt}
			gdef, gstages, gerr := s.loadDefinition(ctx, gKey)
			if gerr != nil {
				return nil, nil, gerr
			}
			genabled := filterEnabledStages(gstages)
			if len(genabled) > 0 && gdef != nil {
				return gdef, genabled, nil
			}
		}
	}
	if def != nil {
		return def, enabled, nil
	}
	return nil, nil, nil
}

func filterEnabledStages(stages []model.WorkflowStage) []model.WorkflowStage {
	out := make([]model.WorkflowStage, 0, len(stages))
	for _, st := range stages {
		if st.Enabled {
			out = append(out, st)
		}
	}
	return out
}

type CreateTicketRequest struct {
	Domain          string         `json:"domain" binding:"required"`
	TicketType      string         `json:"ticket_type"`
	ProjectID       uint           `json:"project_id"`
	Title           string         `json:"title" binding:"required,max=256"`
	Remark          string         `json:"remark" binding:"omitempty,max=512"`
	RefType         string         `json:"ref_type"`
	RefID           uint           `json:"ref_id"`
	Payload         map[string]any `json:"payload"`
	SubmitterUserID uint           `json:"submitter_user_id"`
}

type TicketListQuery struct {
	Domain     string `form:"domain"`
	TicketType string `form:"ticket_type"`
	ProjectID  *uint  `form:"project_id"`
	Status     string `form:"status"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

type TicketDetail struct {
	model.WorkflowTicket
	Steps []TicketStepItem `json:"steps"`
}

type TicketStepItem struct {
	model.WorkflowTicketStep
	UserGroupName string `json:"user_group_name,omitempty"`
	ReviewerName  string `json:"reviewer_name,omitempty"`
	AssigneeName  string `json:"assignee_name,omitempty"`
}

// CreateTicket 创建通用工单并初始化审批步骤。
func (s *Service) CreateTicket(ctx context.Context, req CreateTicketRequest, actor *auth.CurrentUser) (*TicketDetail, error) {
	key := DefinitionKey{Domain: req.Domain, ProjectID: req.ProjectID, TicketType: req.TicketType}.normalize()
	submitter := req.SubmitterUserID
	if submitter == 0 && actor != nil {
		submitter = actor.ID
	}
	if key.Domain == model.WorkflowDomainIncident {
		if err := EnsureDefaultIncidentDefinition(ctx, s.repo); err != nil {
			return nil, bizerrors.Pass(ctx, "workflow", "CreateTicket.ensureIncident", err)
		}
	}
	def, stages, err := s.resolveFlow(ctx, key)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "CreateTicket", err)
	}
	if def == nil || len(stages) == 0 {
		return nil, constants.ErrBadRequestWithMsg("流程未配置或未启用审批节点（请在流程中心配置 incident 故障单，或使用全局默认流程）")
	}
	payloadJSON := ""
	if len(req.Payload) > 0 {
		b, _ := json.Marshal(req.Payload)
		payloadJSON = string(b)
	}
	var ticket model.WorkflowTicket
	err = s.repo.Transaction(ctx, func(tx interfaces.WorkflowRepository) error {
		ticket = model.WorkflowTicket{
			DefinitionID: def.ID, Domain: key.Domain, TicketType: key.TicketType,
			ProjectID: req.ProjectID, Title: strings.TrimSpace(req.Title),
			Status: model.WorkflowTicketStatusPending, SubmitterUserID: submitter,
			RefType: strings.TrimSpace(req.RefType), RefID: req.RefID,
			PayloadJSON: payloadJSON, Remark: strings.TrimSpace(req.Remark),
		}
		if err := tx.CreateTicket(ctx, &ticket); err != nil {
			return err
		}
		now := time.Now()
		for i, st := range stages {
			assigneeID, err := s.resolveDutyAssignee(ctx, st, now)
			if err != nil {
				return err
			}
			step := model.WorkflowTicketStep{
				TicketID: ticket.ID, StageKey: st.StageKey, StageName: st.StageName,
				SortOrder: st.SortOrder, Status: model.WorkflowStepPending,
				AssigneeRuleType: st.AssigneeRuleType, UserGroupID: st.UserGroupID,
				DutyMonitorRuleID: st.DutyMonitorRuleID, AssigneeUserID: assigneeID,
			}
			if i == 0 {
				step.ActivatedAt = &now
			}
			if err := tx.CreateStep(ctx, &step); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "CreateTicket", err)
	}
	return s.TicketDetail(ctx, ticket.ID)
}

// CreateIncidentFromAlert 告警转故障工单。
func (s *Service) CreateIncidentFromAlert(ctx context.Context, alertEventID uint, title string, actor *auth.CurrentUser) (*TicketDetail, error) {
	if title == "" {
		title = "告警转工单 #" + itoa(alertEventID)
	}
	var projectID uint
	if s.alertEventRepo != nil && alertEventID > 0 {
		ev, err := s.alertEventRepo.GetByID(ctx, alertEventID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrNotFoundWithMsg("告警事件不存在")
			}
			return nil, bizerrors.Pass(ctx, "workflow", "CreateIncidentFromAlert", err)
		}
		projectID = ev.ProjectID
	}
	return s.CreateTicket(ctx, CreateTicketRequest{
		Domain: model.WorkflowDomainIncident, TicketType: model.WorkflowTicketTypeIncident,
		ProjectID: projectID,
		Title:     title, RefType: "alert_event", RefID: alertEventID,
		Payload: map[string]any{"alert_event_id": alertEventID, "project_id": projectID},
	}, actor)
}

// CreateIncidentFromAlertRef 支持投递事件 ID 或指纹（当前告警）转故障工单。
type CreateIncidentFromAlertRefRequest struct {
	Title        string `json:"title"`
	AlertEventID uint   `json:"alert_event_id"`
	Fingerprint  string `json:"fingerprint"`
	ProjectID    uint   `json:"project_id"`
}

func (s *Service) CreateIncidentFromAlertRef(ctx context.Context, req CreateIncidentFromAlertRefRequest, actor *auth.CurrentUser) (*TicketDetail, error) {
	if req.AlertEventID > 0 {
		return s.CreateIncidentFromAlert(ctx, req.AlertEventID, req.Title, actor)
	}
	fp := strings.TrimSpace(req.Fingerprint)
	if fp == "" {
		return nil, constants.ErrBadRequestWithMsg("需要 alert_event_id 或 fingerprint")
	}
	projectID := req.ProjectID
	var linkedEventID uint
	if s.alertEventRepo != nil {
		if ev, err := s.alertEventRepo.GetByFingerprint(ctx, fp); err == nil && ev != nil {
			linkedEventID = ev.ID
			if projectID == 0 {
				projectID = ev.ProjectID
			}
		}
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "告警转工单 " + fp
	}
	if linkedEventID > 0 {
		return s.CreateTicket(ctx, CreateTicketRequest{
			Domain: model.WorkflowDomainIncident, TicketType: model.WorkflowTicketTypeIncident,
			ProjectID: projectID,
			Title:     title, RefType: "alert_event", RefID: linkedEventID,
			Payload: map[string]any{
				"alert_event_id": linkedEventID,
				"fingerprint":    fp,
				"project_id":     projectID,
			},
		}, actor)
	}
	return s.CreateTicket(ctx, CreateTicketRequest{
		Domain: model.WorkflowDomainIncident, TicketType: model.WorkflowTicketTypeIncident,
		ProjectID: projectID,
		Title:     title, RefType: "alert_fingerprint", RefID: 0,
		Payload: map[string]any{"fingerprint": fp, "project_id": projectID},
	}, actor)
}

func (s *Service) ListTickets(ctx context.Context, q TicketListQuery) (*pagination.Result[TicketDetail], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	rows, total, err := s.repo.ListTickets(ctx, repository.WorkflowTicketListParams{
		Domain: strings.TrimSpace(q.Domain), TicketType: strings.TrimSpace(q.TicketType),
		ProjectID: q.ProjectID, Status: strings.TrimSpace(q.Status),
		Offset: (page - 1) * pageSize, Limit: pageSize,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ListTickets", err)
	}
	items := make([]TicketDetail, 0, len(rows))
	for _, row := range rows {
		detail, err := s.ticketDetailFromRow(ctx, row)
		if err != nil {
			return nil, err
		}
		items = append(items, *detail)
	}
	return &pagination.Result[TicketDetail]{
		List: items, Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func (s *Service) TicketDetail(ctx context.Context, id uint) (*TicketDetail, error) {
	row, err := s.repo.GetTicket(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "workflow", "TicketDetail", err)
	}
	return s.ticketDetailFromRow(ctx, *row)
}

type ReviewStepRequest struct {
	Approve bool   `json:"approve"`
	Comment string `json:"comment" binding:"omitempty,max=512"`
}

// ReviewStep 审批当前激活步骤（乐观锁：仅 pending 可抢占）。
func (s *Service) ReviewStep(ctx context.Context, ticketID, stepID uint, req ReviewStepRequest, actor *auth.CurrentUser) (*TicketDetail, error) {
	ticketPtr, err := s.repo.GetTicket(ctx, ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "workflow", "ReviewStep", err)
	}
	ticket := *ticketPtr
	if ticket.Status != model.WorkflowTicketStatusPending {
		return nil, constants.ErrBadRequestWithMsg("工单不在待审批状态")
	}
	stepPtr, err := s.repo.GetStep(ctx, ticketID, stepID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "workflow", "ReviewStep", err)
	}
	step := *stepPtr
	if step.Status != model.WorkflowStepPending || step.ActivatedAt == nil {
		return nil, constants.ErrBadRequestWithMsg("该审批节点不可操作")
	}
	// 按工单绑定的 definition_id 读取职责分离开关（避免 ticket_type 回退 default 后查不到配置）
	forbidSelf := true
	if ticket.DefinitionID > 0 {
		if def, err := s.repo.GetDefinitionByID(ctx, ticket.DefinitionID); err == nil && def != nil {
			forbidSelf = def.ForbidSelfApprove
		}
	} else {
		def, _, _ := s.resolveFlow(ctx, DefinitionKey{Domain: ticket.Domain, ProjectID: ticket.ProjectID, TicketType: ticket.TicketType})
		if def != nil {
			forbidSelf = def.ForbidSelfApprove
		}
	}
	if forbidSelf {
		if err := forbidSelfApprove(actor, ticket.SubmitterUserID); err != nil {
			return nil, err
		}
	}
	ok, err := s.userCanReviewStep(ctx, actor, step)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, constants.ErrForbidden
	}
	now := time.Now()
	reviewerID := actorUserID(actor)
	err = s.repo.Transaction(ctx, func(tx interfaces.WorkflowRepository) error {
		status := model.WorkflowStepRejected
		if req.Approve {
			status = model.WorkflowStepApproved
		}
		n, err := tx.ClaimStepReview(ctx, stepID, ticketID, map[string]any{
			"status": status, "reviewer_user_id": reviewerID,
			"review_comment": strings.TrimSpace(req.Comment), "reviewed_at": now,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return errStepConflict
		}
		if !req.Approve {
			return tx.UpdateTicketFields(ctx, ticket.ID, map[string]any{
				"status": model.WorkflowTicketStatusRejected, "closed_at": now,
			})
		}
		steps, err := tx.ListStepsByTicket(ctx, ticketID)
		if err != nil {
			return err
		}
		var next *model.WorkflowTicketStep
		for i := range steps {
			if steps[i].SortOrder > step.SortOrder && steps[i].Status == model.WorkflowStepPending {
				next = &steps[i]
				break
			}
		}
		if next != nil {
			return tx.ActivateStep(ctx, next.ID, now)
		}
		return tx.UpdateTicketFields(ctx, ticket.ID, map[string]any{
			"status": model.WorkflowTicketStatusApproved, "closed_at": now,
		})
	})
	if err != nil {
		if errors.Is(err, errStepConflict) {
			return nil, err
		}
		return nil, bizerrors.Pass(ctx, "workflow", "ReviewStep", err)
	}
	return s.TicketDetail(ctx, ticketID)
}

// errStepConflict 并发审批冲突：本节点已被他人处理。
var errStepConflict = constants.ErrBadRequestWithMsg("审批节点状态已变更，请刷新后重试")

func (s *Service) loadDefinition(ctx context.Context, key DefinitionKey) (*model.WorkflowDefinition, []model.WorkflowStage, error) {
	key = key.normalize()
	return s.repo.LoadDefinition(ctx, key.Domain, key.ProjectID, key.TicketType)
}

func (s *Service) ticketDetailFromRow(ctx context.Context, row model.WorkflowTicket) (*TicketDetail, error) {
	steps, err := s.repo.ListStepsByTicket(ctx, row.ID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ticketDetailFromRow", err)
	}
	groupNames := map[uint]string{}
	userNames := map[uint]string{}
	for _, st := range steps {
		if st.UserGroupID != nil && *st.UserGroupID > 0 {
			groupNames[*st.UserGroupID] = ""
		}
		if st.ReviewerUserID != nil && *st.ReviewerUserID > 0 {
			userNames[*st.ReviewerUserID] = ""
		}
		if st.AssigneeUserID != nil && *st.AssigneeUserID > 0 {
			userNames[*st.AssigneeUserID] = ""
		}
	}
	s.fillUserGroupNames(ctx, groupNames)
	s.fillUserNames(ctx, userNames)
	stepItems := make([]TicketStepItem, 0, len(steps))
	for _, st := range steps {
		item := TicketStepItem{WorkflowTicketStep: st}
		if st.UserGroupID != nil {
			item.UserGroupName = groupNames[*st.UserGroupID]
		}
		if st.ReviewerUserID != nil {
			item.ReviewerName = userNames[*st.ReviewerUserID]
		}
		if st.AssigneeUserID != nil {
			item.AssigneeName = userNames[*st.AssigneeUserID]
		}
		stepItems = append(stepItems, item)
	}
	return &TicketDetail{WorkflowTicket: row, Steps: stepItems}, nil
}


func (s *Service) userCanReviewStep(ctx context.Context, actor *auth.CurrentUser, step model.WorkflowTicketStep) (bool, error) {
	if actor != nil && auth.IsSuperAdminRole(actor.RoleCodes) {
		return true, nil
	}
	userID := actorUserID(actor)
	if userID == 0 {
		return false, nil
	}
	if step.AssigneeUserID != nil && *step.AssigneeUserID > 0 {
		return *step.AssigneeUserID == userID, nil
	}
	if step.AssigneeRuleType == model.WorkflowAssigneePlatformRole {
		return CanPlatformRoleReview(actor), nil
	}
	// 值班节点未解析到人：允许平台审批角色接手，避免工单永久卡死
	if step.AssigneeRuleType == model.WorkflowAssigneeDuty {
		return CanPlatformRoleReview(actor), nil
	}
	if step.UserGroupID == nil || *step.UserGroupID == 0 {
		return false, nil
	}
	if s.userGroupRepo == nil {
		return false, nil
	}
	ids, err := s.userGroupRepo.ListMemberUserIDs(ctx, *step.UserGroupID)
	if err != nil {
		return false, err
	}
	return slices.Contains(ids, userID), nil
}

func forbidSelfApprove(actor *auth.CurrentUser, submitterUserID uint) error {
	if actor != nil && auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	if submitterUserID == 0 {
		return nil
	}
	if actorUserID(actor) == submitterUserID {
		return constants.ErrForbiddenWithMsg("职责分离：提交人不可审批自己的工单")
	}
	return nil
}

func actorUserID(actor *auth.CurrentUser) uint {
	if actor == nil {
		return 0
	}
	return actor.ID
}

func normalizeStageKey(raw string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		var b [4]byte
		if _, err := rand.Read(b[:]); err != nil {
			return "", err
		}
		return "custom_" + hex.EncodeToString(b[:]), nil
	}
	if !stageKeyPattern.MatchString(key) {
		return "", constants.ErrBadRequestWithMsg("审批节点 Key 须为小写字母开头，仅含 a-z/0-9/_，长度 2-32: " + raw)
	}
	return key, nil
}

func stageToItem(st model.WorkflowStage, groupNames map[uint]string) StageItem {
	item := StageItem{
		StageKey: st.StageKey, StageName: st.StageName, SortOrder: st.SortOrder,
		Enabled: st.Enabled, AssigneeRuleType: st.AssigneeRuleType,
		UserGroupID: st.UserGroupID, DutyMonitorRuleID: st.DutyMonitorRuleID,
	}
	if st.UserGroupID != nil {
		item.UserGroupName = groupNames[*st.UserGroupID]
	}
	return item
}

func (s *Service) loadUserGroupNameMap(ctx context.Context, stages []model.WorkflowStage) map[uint]string {
	names := map[uint]string{}
	for _, st := range stages {
		if st.UserGroupID != nil && *st.UserGroupID > 0 {
			names[*st.UserGroupID] = ""
		}
	}
	s.fillUserGroupNames(ctx, names)
	return names
}

func (s *Service) fillUserGroupNames(ctx context.Context, names map[uint]string) {
	if len(names) == 0 || s.userGroupRepo == nil {
		return
	}
	ids := make([]uint, 0, len(names))
	for id := range names {
		ids = append(ids, id)
	}
	m, err := s.userGroupRepo.ListNamesByIDs(ctx, ids)
	if err != nil {
		return
	}
	for id, name := range m {
		names[id] = name
	}
}

func (s *Service) fillUserNames(ctx context.Context, names map[uint]string) {
	if s.userRepo == nil || len(names) == 0 {
		return
	}
	ids := make([]uint, 0, len(names))
	for id := range names {
		ids = append(ids, id)
	}
	users, err := s.userRepo.ListByIDs(ctx, ids)
	if err != nil {
		return
	}
	for _, u := range users {
		names[u.ID] = u.Nickname
		if names[u.ID] == "" {
			names[u.ID] = u.Username
		}
	}
}

func itoa(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}
