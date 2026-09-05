package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

// LinkedTicketInput 创建与业务实体关联的统一工单。
type LinkedTicketInput struct {
	Domain          string
	TicketType      string
	ProjectID       uint
	Title           string
	SubmitterUserID uint
	RefType         string
	RefID           uint
	Payload         map[string]any
}

// GetTicketByRef 按业务引用查找工单（同 ref 多类型时取最新一条）。
func (s *Service) GetTicketByRef(ctx context.Context, refType string, refID uint) (*model.WorkflowTicket, error) {
	return s.GetTicketByRefType(ctx, refType, refID, "")
}

// GetTicketByRefType 按业务引用 + 工单类型查找。
func (s *Service) GetTicketByRefType(ctx context.Context, refType string, refID uint, ticketType string) (*model.WorkflowTicket, error) {
	refType = strings.TrimSpace(refType)
	if refType == "" || refID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	q := s.db.WithContext(ctx).Where("ref_type = ? AND ref_id = ?", refType, refID)
	if tt := strings.TrimSpace(ticketType); tt != "" {
		q = q.Where("ticket_type = ?", tt)
	}
	var row model.WorkflowTicket
	err := q.Order("id DESC").First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// CreateLinkedTicket 基于流程定义创建关联工单（dbmgmt/cicd 等业务写入 workflow_tickets）。
func (s *Service) CreateLinkedTicket(ctx context.Context, in LinkedTicketInput) (*model.WorkflowTicket, error) {
	key := DefinitionKey{Domain: in.Domain, ProjectID: in.ProjectID, TicketType: in.TicketType}.normalize()
	def, stages, err := s.resolveFlow(ctx, key)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "CreateLinkedTicket", err)
	}
	if def == nil || len(stages) == 0 {
		return nil, constants.ErrBadRequestWithMsg("流程未配置或未启用审批节点")
	}
	payloadJSON := ""
	if len(in.Payload) > 0 {
		b, _ := json.Marshal(in.Payload)
		payloadJSON = string(b)
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = in.RefType + " #" + itoa(in.RefID)
	}
	// 工单 ticket_type 保留业务类型；definition 可能来自 default 回退
	ticketType := key.TicketType
	var ticket model.WorkflowTicket
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ticket = model.WorkflowTicket{
			DefinitionID: def.ID, Domain: key.Domain, TicketType: ticketType,
			ProjectID: in.ProjectID, Title: title, Status: model.WorkflowTicketStatusPending,
			SubmitterUserID: in.SubmitterUserID, RefType: strings.TrimSpace(in.RefType), RefID: in.RefID,
			PayloadJSON: payloadJSON,
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
		return nil, bizerrors.Pass(ctx, "workflow", "CreateLinkedTicket", err)
	}
	return &ticket, nil
}

// CreateInfoTicket 创建无需审批的关联工单（如 CICD 变更单登记）。
func (s *Service) CreateInfoTicket(ctx context.Context, in LinkedTicketInput, status string) (*model.WorkflowTicket, error) {
	if status == "" {
		status = model.WorkflowTicketStatusApproved
	}
	payloadJSON := ""
	if len(in.Payload) > 0 {
		b, _ := json.Marshal(in.Payload)
		payloadJSON = string(b)
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = in.RefType + " #" + itoa(in.RefID)
	}
	key := DefinitionKey{Domain: in.Domain, ProjectID: in.ProjectID, TicketType: in.TicketType}.normalize()
	def, _, _ := s.loadDefinition(ctx, key)
	defID := uint(0)
	if def != nil {
		defID = def.ID
	}
	ticket := model.WorkflowTicket{
		DefinitionID: defID, Domain: key.Domain, TicketType: key.TicketType,
		ProjectID: in.ProjectID, Title: title, Status: status,
		SubmitterUserID: in.SubmitterUserID, RefType: strings.TrimSpace(in.RefType), RefID: in.RefID,
		PayloadJSON: payloadJSON,
	}
	if err := s.db.WithContext(ctx).Create(&ticket).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "CreateInfoTicket", err)
	}
	return &ticket, nil
}

// GetActiveStep 返回当前待审批步骤。
func (s *Service) GetActiveStep(ctx context.Context, ticketID uint) (*model.WorkflowTicketStep, error) {
	var step model.WorkflowTicketStep
	err := s.db.WithContext(ctx).
		Where("ticket_id = ? AND status = ? AND activated_at IS NOT NULL", ticketID, model.WorkflowStepPending).
		Order("sort_order ASC, id ASC").
		First(&step).Error
	if err != nil {
		return nil, err
	}
	return &step, nil
}

// ReviewLinkedStep 审批关联工单并返回最新工单状态。
func (s *Service) ReviewLinkedStep(ctx context.Context, refType string, refID uint, approve bool, comment string, actor *auth.CurrentUser) (*TicketDetail, error) {
	return s.ReviewLinkedStepTyped(ctx, refType, refID, "", approve, comment, actor)
}

// ReviewLinkedStepTyped 按工单类型审批关联工单。
func (s *Service) ReviewLinkedStepTyped(ctx context.Context, refType string, refID uint, ticketType string, approve bool, comment string, actor *auth.CurrentUser) (*TicketDetail, error) {
	ticket, err := s.GetTicketByRefType(ctx, refType, refID, ticketType)
	if err != nil {
		return nil, err
	}
	step, err := s.GetActiveStep(ctx, ticket.ID)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("无待审批步骤")
	}
	return s.ReviewStep(ctx, ticket.ID, step.ID, ReviewStepRequest{Approve: approve, Comment: comment}, actor)
}

// HasLinkedTicket 业务实体是否已迁入 workflow_tickets。
func (s *Service) HasLinkedTicket(ctx context.Context, refType string, refID uint) bool {
	_, err := s.GetTicketByRef(ctx, refType, refID)
	return err == nil
}

func (s *Service) HasLinkedTicketType(ctx context.Context, refType string, refID uint, ticketType string) bool {
	_, err := s.GetTicketByRefType(ctx, refType, refID, ticketType)
	return err == nil
}

// CloseLinkedTicket 将关联统一工单标记为 closed（业务执行完成/取消后调用）。
func (s *Service) CloseLinkedTicket(ctx context.Context, refType string, refID uint) error {
	ticket, err := s.GetTicketByRef(ctx, refType, refID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if ticket.Status == model.WorkflowTicketStatusClosed || ticket.Status == model.WorkflowTicketStatusCancelled {
		return nil
	}
	now := time.Now()
	return s.db.WithContext(ctx).Model(ticket).Updates(map[string]any{
		"status": model.WorkflowTicketStatusClosed, "closed_at": now,
	}).Error
}

// CloseLinkedTicketTyped 按工单类型关闭关联统一工单。
func (s *Service) CloseLinkedTicketTyped(ctx context.Context, refType string, refID uint, ticketType string) error {
	ticket, err := s.GetTicketByRefType(ctx, refType, refID, ticketType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if ticket.Status == model.WorkflowTicketStatusClosed || ticket.Status == model.WorkflowTicketStatusCancelled {
		return nil
	}
	now := time.Now()
	return s.db.WithContext(ctx).Model(ticket).Updates(map[string]any{
		"status": model.WorkflowTicketStatusClosed, "closed_at": now,
	}).Error
}

// ErrNoLinkedTicket 业务实体尚未关联统一工单。
var ErrNoLinkedTicket = errors.New("no linked workflow ticket")
