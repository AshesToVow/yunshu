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

	"gorm.io/gorm"
)

var stageKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}$`)

// Service 统一工单引擎：流程定义 + 通用工单 + 排班派单。
type Service struct {
	db            *gorm.DB
	userGroupRepo interfaces.UserGroupRepository
	dutyRepo      interfaces.AlertDutyRepository
	userRepo      interfaces.UserRepository
}

// NewService 创建工单引擎。
func NewService(
	db *gorm.DB,
	userGroupRepo interfaces.UserGroupRepository,
	dutyRepo interfaces.AlertDutyRepository,
	userRepo interfaces.UserRepository,
) *Service {
	return &Service{
		db:            db,
		userGroupRepo: userGroupRepo,
		dutyRepo:      dutyRepo,
		userRepo:      userRepo,
	}
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
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		def, _, err := s.loadDefinitionTx(tx, key)
		if err != nil {
			return err
		}
		if def == nil {
			def = &model.WorkflowDefinition{
				Domain: key.Domain, ProjectID: key.ProjectID, TicketType: key.TicketType,
				Name: key.Domain + " workflow", Enabled: true, ForbidSelfApprove: true,
			}
			if err := tx.Create(def).Error; err != nil {
				return err
			}
		}
		keys := make([]string, 0, len(normalized))
		for _, st := range normalized {
			keys = append(keys, st.Key)
			var existing model.WorkflowStage
			err := tx.Where("definition_id = ? AND stage_key = ?", def.ID, st.Key).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				row := model.WorkflowStage{
					DefinitionID: def.ID, StageKey: st.Key, StageName: st.Name, SortOrder: st.Sort,
					Enabled: st.Enabled, AssigneeRuleType: st.RuleType,
					UserGroupID: st.UserGroupID, DutyMonitorRuleID: st.DutyRuleID,
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
				"stage_name": st.Name, "sort_order": st.Sort, "enabled": st.Enabled,
				"assignee_rule_type": st.RuleType, "user_group_id": st.UserGroupID,
				"duty_monitor_rule_id": st.DutyRuleID,
			}).Error; err != nil {
				return err
			}
		}
		q := tx.Where("definition_id = ?", def.ID)
		if len(keys) > 0 {
			q = q.Where("stage_key NOT IN ?", keys)
		}
		return q.Delete(&model.WorkflowStage{}).Error
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

// resolveFlow 解析流程定义与启用节点；特定类型未配置时回退 default。
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
	def, stages, err := s.resolveFlow(ctx, key)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "CreateTicket", err)
	}
	if def == nil || len(stages) == 0 {
		return nil, constants.ErrBadRequestWithMsg("流程未配置或未启用审批节点")
	}
	payloadJSON := ""
	if len(req.Payload) > 0 {
		b, _ := json.Marshal(req.Payload)
		payloadJSON = string(b)
	}
	var ticket model.WorkflowTicket
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ticket = model.WorkflowTicket{
			DefinitionID: def.ID, Domain: key.Domain, TicketType: key.TicketType,
			ProjectID: req.ProjectID, Title: strings.TrimSpace(req.Title),
			Status: model.WorkflowTicketStatusPending, SubmitterUserID: submitter,
			RefType: strings.TrimSpace(req.RefType), RefID: req.RefID,
			PayloadJSON: payloadJSON, Remark: strings.TrimSpace(req.Remark),
		}
		if err := tx.Create(&ticket).Error; err != nil {
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
			if err := tx.Create(&step).Error; err != nil {
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
	return s.CreateTicket(ctx, CreateTicketRequest{
		Domain: model.WorkflowDomainIncident, TicketType: model.WorkflowTicketTypeIncident,
		Title: title, RefType: "alert_event", RefID: alertEventID,
		Payload: map[string]any{"alert_event_id": alertEventID},
	}, actor)
}

func (s *Service) ListTickets(ctx context.Context, q TicketListQuery) (*pagination.Result[TicketDetail], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	query := s.db.WithContext(ctx).Model(&model.WorkflowTicket{})
	if d := strings.TrimSpace(q.Domain); d != "" {
		query = query.Where("domain = ?", d)
	}
	if tt := strings.TrimSpace(q.TicketType); tt != "" {
		query = query.Where("ticket_type = ?", tt)
	}
	if q.ProjectID != nil {
		query = query.Where("project_id = ?", *q.ProjectID)
	}
	if st := strings.TrimSpace(q.Status); st != "" {
		query = query.Where("status = ?", st)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ListTickets", err)
	}
	var rows []model.WorkflowTicket
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
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
	var row model.WorkflowTicket
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "workflow", "TicketDetail", err)
	}
	return s.ticketDetailFromRow(ctx, row)
}

type ReviewStepRequest struct {
	Approve bool   `json:"approve"`
	Comment string `json:"comment" binding:"omitempty,max=512"`
}

// ReviewStep 审批当前激活步骤。
func (s *Service) ReviewStep(ctx context.Context, ticketID, stepID uint, req ReviewStepRequest, actor *auth.CurrentUser) (*TicketDetail, error) {
	var ticket model.WorkflowTicket
	if err := s.db.WithContext(ctx).First(&ticket, ticketID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "workflow", "ReviewStep", err)
	}
	if ticket.Status != model.WorkflowTicketStatusPending {
		return nil, constants.ErrBadRequestWithMsg("工单不在待审批状态")
	}
	var step model.WorkflowTicketStep
	if err := s.db.WithContext(ctx).Where("id = ? AND ticket_id = ?", stepID, ticketID).First(&step).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "workflow", "ReviewStep", err)
	}
	if step.Status != model.WorkflowStepPending || step.ActivatedAt == nil {
		return nil, constants.ErrBadRequestWithMsg("该审批节点不可操作")
	}
	def, _, _ := s.loadDefinition(ctx, DefinitionKey{Domain: ticket.Domain, ProjectID: ticket.ProjectID, TicketType: ticket.TicketType})
	if def != nil && def.ForbidSelfApprove {
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
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		status := model.WorkflowStepRejected
		ticketStatus := model.WorkflowTicketStatusRejected
		if req.Approve {
			status = model.WorkflowStepApproved
		}
		if err := tx.Model(&step).Updates(map[string]any{
			"status": status, "reviewer_user_id": reviewerID,
			"review_comment": strings.TrimSpace(req.Comment), "reviewed_at": now,
		}).Error; err != nil {
			return err
		}
		if !req.Approve {
			return tx.Model(&ticket).Updates(map[string]any{
				"status": ticketStatus, "closed_at": now,
			}).Error
		}
		var steps []model.WorkflowTicketStep
		if err := tx.Where("ticket_id = ?", ticketID).Order("sort_order ASC, id ASC").Find(&steps).Error; err != nil {
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
			return tx.Model(next).Update("activated_at", now).Error
		}
		return tx.Model(&ticket).Updates(map[string]any{
			"status": model.WorkflowTicketStatusApproved, "closed_at": now,
		}).Error
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ReviewStep", err)
	}
	return s.TicketDetail(ctx, ticketID)
}

func (s *Service) loadDefinition(ctx context.Context, key DefinitionKey) (*model.WorkflowDefinition, []model.WorkflowStage, error) {
	return s.loadDefinitionTx(s.db.WithContext(ctx), key)
}

func (s *Service) loadDefinitionTx(tx *gorm.DB, key DefinitionKey) (*model.WorkflowDefinition, []model.WorkflowStage, error) {
	key = key.normalize()
	var def model.WorkflowDefinition
	err := tx.Where("domain = ? AND project_id = ? AND ticket_type = ?", key.Domain, key.ProjectID, key.TicketType).First(&def).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var stages []model.WorkflowStage
	if err := tx.Where("definition_id = ?", def.ID).Order("sort_order ASC, id ASC").Find(&stages).Error; err != nil {
		return nil, nil, err
	}
	return &def, stages, nil
}

func (s *Service) ticketDetailFromRow(ctx context.Context, row model.WorkflowTicket) (*TicketDetail, error) {
	var steps []model.WorkflowTicketStep
	if err := s.db.WithContext(ctx).Where("ticket_id = ?", row.ID).Order("sort_order ASC, id ASC").Find(&steps).Error; err != nil {
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
	if step.UserGroupID == nil || *step.UserGroupID == 0 {
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
	if len(names) == 0 || s.db == nil {
		return
	}
	ids := make([]uint, 0, len(names))
	for id := range names {
		ids = append(ids, id)
	}
	var groups []model.UserGroup
	if err := s.db.WithContext(ctx).Select("id, name").Where("id IN ?", ids).Find(&groups).Error; err != nil {
		return
	}
	for _, g := range groups {
		names[g.ID] = g.Name
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
