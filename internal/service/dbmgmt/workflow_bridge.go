package dbmgmt

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	workflowsvc "yunshu/internal/service/workflow"
)

func (s *Service) createDbmgmtWorkflowTicket(ctx context.Context, ticketType, refType, title string, projectID, refID, submitterUserID uint) error {
	wf := s.workflowEngine()
	if wf.HasLinkedTicket(ctx, refType, refID) {
		return nil
	}
	_, err := wf.CreateLinkedTicket(ctx, workflowsvc.LinkedTicketInput{
		Domain: model.WorkflowDomainDbmgmt, TicketType: ticketType,
		ProjectID: projectID, Title: title, SubmitterUserID: submitterUserID,
		RefType: refType, RefID: refID,
	})
	return err
}

func (s *Service) reviewDbmgmtViaWorkflow(ctx context.Context, refType string, refID uint, approve bool, comment string, actor *auth.CurrentUser, onApproved func(*workflowsvc.TicketDetail) error) error {
	wf := s.workflowEngine()
	if !wf.HasLinkedTicket(ctx, refType, refID) {
		return workflowsvc.ErrNoLinkedTicket
	}
	detail, err := wf.ReviewLinkedStep(ctx, refType, refID, approve, comment, actor)
	if err != nil {
		return err
	}
	if !approve {
		return s.syncDbmgmtRejected(ctx, refType, refID)
	}
	if detail.Status == model.WorkflowTicketStatusApproved {
		if onApproved != nil {
			return onApproved(detail)
		}
	}
	return nil
}

func (s *Service) syncDbmgmtRejected(ctx context.Context, refType string, refID uint) error {
	switch refType {
	case model.WorkflowRefDbSqlTicket:
		var ticket model.DbSqlTicket
		if err := s.db.WithContext(ctx).First(&ticket, refID).Error; err != nil {
			return err
		}
		ticket.Status = model.DbTicketStatusRejected
		return s.repo.UpdateSqlTicket(ctx, &ticket)
	case model.WorkflowRefDbAccessRequest:
		var req model.DbAccessRequest
		if err := s.db.WithContext(ctx).First(&req, refID).Error; err != nil {
			return err
		}
		req.Status = model.DbAccessRequestStatusRejected
		return s.repo.UpdateAccessRequest(ctx, &req)
	case model.WorkflowRefDbAppUserRequest:
		var req model.DbAppUserRequest
		if err := s.db.WithContext(ctx).First(&req, refID).Error; err != nil {
			return err
		}
		req.Status = model.DbAccessRequestStatusRejected
		return s.repo.UpdateAppUserRequest(ctx, &req)
	}
	return nil
}

func sqlTicketWorkflowTitle(ticket *model.DbSqlTicket) string {
	if ticket == nil {
		return "SQL 工单"
	}
	parts := []string{"SQL 工单"}
	if ticket.DatabaseName != "" {
		parts = append(parts, ticket.DatabaseName)
	}
	if ticket.Reason != "" {
		parts = append(parts, strings.TrimSpace(ticket.Reason))
	}
	return strings.Join(parts, " · ")
}

func accessRequestWorkflowTitle(req *model.DbAccessRequest) string {
	if req == nil {
		return "权限申请"
	}
	if req.DatabaseName != "" {
		return fmt.Sprintf("权限申请 · %s", req.DatabaseName)
	}
	return "权限申请"
}

func appUserRequestWorkflowTitle(req *model.DbAppUserRequest) string {
	if req == nil {
		return "应用用户申请"
	}
	if req.MySQLUser != "" {
		return fmt.Sprintf("应用用户申请 · %s", req.MySQLUser)
	}
	return "应用用户申请"
}

func (s *Service) finalizeSqlTicketApproval(ctx context.Context, projectID, ticketID uint) error {
	ticket, err := s.repo.GetSqlTicketInProject(ctx, projectID, ticketID)
	if err != nil {
		return err
	}
	ticket.Status = model.DbTicketStatusPendingExecution
	return s.repo.UpdateSqlTicket(ctx, ticket)
}

func (s *Service) finalizeAccessRequestApproval(ctx context.Context, projectID, requestID uint) error {
	req, err := s.repo.GetAccessRequestInProject(ctx, projectID, requestID)
	if err != nil {
		return err
	}
	req.Status = model.DbAccessRequestStatusApproved
	if err := s.repo.UpdateAccessRequest(ctx, req); err != nil {
		return err
	}
	return s.grantFromAccessRequest(ctx, req)
}

func (s *Service) finalizeAppUserRequestApproval(ctx context.Context, projectID, requestID uint) error {
	req, err := s.repo.GetAppUserRequestInProject(ctx, projectID, requestID)
	if err != nil {
		return err
	}
	req.Status = model.DbAccessRequestStatusApproved
	if err := s.repo.UpdateAppUserRequest(ctx, req); err != nil {
		return err
	}
	return s.executeAppUserRequest(ctx, req)
}

func isWorkflowNoTicket(err error) bool {
	return err != nil && (err == workflowsvc.ErrNoLinkedTicket || err.Error() == workflowsvc.ErrNoLinkedTicket.Error())
}

func workflowOrLegacyErr(err error) error {
	if isWorkflowNoTicket(err) {
		return constants.ErrBadRequestWithMsg("工单审批状态异常，请联系管理员")
	}
	return err
}

func (s *Service) isFinalWorkflowApproval(ctx context.Context, refType string, refID uint) bool {
	wf := s.workflowEngine()
	ticket, err := wf.GetTicketByRef(ctx, refType, refID)
	if err != nil {
		return false
	}
	var pending int64
	_ = s.db.WithContext(ctx).Model(&model.WorkflowTicketStep{}).
		Where("ticket_id = ? AND status = ?", ticket.ID, model.WorkflowStepPending).
		Count(&pending).Error
	return pending <= 1
}
