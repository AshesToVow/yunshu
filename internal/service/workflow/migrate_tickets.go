package workflow

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

// MigrateLegacyTickets 将仍在 pending 的 dbmgmt/cicd/ai 业务工单 backfill 到 workflow_tickets。
// 项目未配置审批流时跳过该条（不阻断 migrate/seed）。
func MigrateLegacyTickets(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	svc := NewService(db, nil, nil, nil)
	log := slog.Default().With("component", "workflow.migrate_tickets")
	if err := EnsureDefaultAIToolApprovalDefinition(ctx, db); err != nil {
		log.Warn("ensure AI tool approval definition failed", "error", err)
	}
	if err := migratePendingSqlTickets(ctx, svc, db, log); err != nil {
		return err
	}
	if err := migratePendingAccessRequests(ctx, svc, db, log); err != nil {
		return err
	}
	if err := migratePendingAppUserRequests(ctx, svc, db, log); err != nil {
		return err
	}
	if err := migratePendingReleaseRuns(ctx, svc, db, log); err != nil {
		return err
	}
	return migratePendingAIToolApprovals(ctx, svc, db, log)
}

func migratePendingAIToolApprovals(ctx context.Context, svc *Service, db *gorm.DB, log *slog.Logger) error {
	var rows []model.AiToolApproval
	if err := db.WithContext(ctx).Where("status = ?", "pending").Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if svc.HasLinkedTicket(ctx, model.WorkflowRefAiToolApproval, row.ID) {
			continue
		}
		title := "AI 高危操作 · " + strings.TrimSpace(row.ToolName)
		if res := strings.TrimSpace(row.Resource); res != "" {
			title += " · " + res
		}
		if _, err := svc.CreateLinkedTicket(ctx, LinkedTicketInput{
			Domain: model.WorkflowDomainAI, TicketType: model.WorkflowTicketTypeToolApproval,
			ProjectID: 0, Title: title, SubmitterUserID: row.UserID,
			RefType: model.WorkflowRefAiToolApproval, RefID: row.ID,
			Payload: map[string]any{
				"tool_name": row.ToolName, "cluster_id": row.ClusterID,
				"namespace": row.Namespace, "resource": row.Resource,
			},
		}); err != nil {
			if isFlowNotConfigured(err) {
				log.Warn("skip AI approval migrate: flow not configured", "approval_id", row.ID)
				continue
			}
			return err
		}
	}
	return nil
}

func isFlowNotConfigured(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "流程未配置") || strings.Contains(msg, "流程定义不存在")
}

func migratePendingSqlTickets(ctx context.Context, svc *Service, db *gorm.DB, log *slog.Logger) error {
	var rows []model.DbSqlTicket
	if err := db.WithContext(ctx).Where("status = ?", model.DbTicketStatusPendingApproval).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if svc.HasLinkedTicket(ctx, model.WorkflowRefDbSqlTicket, row.ID) {
			continue
		}
		title := "SQL 工单"
		if row.DatabaseName != "" {
			title += " · " + row.DatabaseName
		}
		if _, err := svc.CreateLinkedTicket(ctx, LinkedTicketInput{
			Domain: model.WorkflowDomainDbmgmt, TicketType: model.WorkflowTicketTypeSql,
			ProjectID: row.ProjectID, Title: title, SubmitterUserID: row.SubmitterUserID,
			RefType: model.WorkflowRefDbSqlTicket, RefID: row.ID,
		}); err != nil {
			if isFlowNotConfigured(err) {
				log.Warn("skip sql ticket migration: approval flow not configured", "ticket_id", row.ID, "project_id", row.ProjectID)
				continue
			}
			return err
		}
	}
	return migratePendingExecutionSqlTickets(ctx, svc, db, log)
}

// migratePendingExecutionSqlTickets 将「待执行」SQL 工单回填为已通过的统一工单，便于提交人在待办中执行。
func migratePendingExecutionSqlTickets(ctx context.Context, svc *Service, db *gorm.DB, log *slog.Logger) error {
	var rows []model.DbSqlTicket
	if err := db.WithContext(ctx).Where("status = ?", model.DbTicketStatusPendingExecution).Find(&rows).Error; err != nil {
		return err
	}
	now := time.Now()
	for _, row := range rows {
		if svc.HasLinkedTicket(ctx, model.WorkflowRefDbSqlTicket, row.ID) {
			// 若已有关联工单但仍为 pending，对齐为已通过
			ticket, err := svc.GetTicketByRef(ctx, model.WorkflowRefDbSqlTicket, row.ID)
			if err == nil && ticket != nil && ticket.Status == model.WorkflowTicketStatusPending {
				_ = markWorkflowTicketApproved(ctx, db, ticket.ID, now)
			}
			continue
		}
		title := "SQL 工单"
		if row.DatabaseName != "" {
			title += " · " + row.DatabaseName
		}
		ticket, err := svc.CreateLinkedTicket(ctx, LinkedTicketInput{
			Domain: model.WorkflowDomainDbmgmt, TicketType: model.WorkflowTicketTypeSql,
			ProjectID: row.ProjectID, Title: title, SubmitterUserID: row.SubmitterUserID,
			RefType: model.WorkflowRefDbSqlTicket, RefID: row.ID,
		})
		if err != nil {
			if isFlowNotConfigured(err) {
				log.Warn("skip sql pending_execution migration: flow not configured", "ticket_id", row.ID, "project_id", row.ProjectID)
				continue
			}
			return err
		}
		if err := markWorkflowTicketApproved(ctx, db, ticket.ID, now); err != nil {
			return err
		}
	}
	return nil
}

func markWorkflowTicketApproved(ctx context.Context, db *gorm.DB, ticketID uint, now time.Time) error {
	if err := db.WithContext(ctx).Model(&model.WorkflowTicket{}).Where("id = ?", ticketID).Updates(map[string]any{
		"status": model.WorkflowTicketStatusApproved, "closed_at": now,
	}).Error; err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.WorkflowTicketStep{}).Where("ticket_id = ?", ticketID).Updates(map[string]any{
		"status": model.WorkflowStepApproved, "reviewed_at": now,
	}).Error
}

func migratePendingAccessRequests(ctx context.Context, svc *Service, db *gorm.DB, log *slog.Logger) error {
	var rows []model.DbAccessRequest
	if err := db.WithContext(ctx).Where("status = ?", model.DbAccessRequestStatusPending).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if svc.HasLinkedTicket(ctx, model.WorkflowRefDbAccessRequest, row.ID) {
			continue
		}
		title := "权限申请"
		if row.DatabaseName != "" {
			title += " · " + row.DatabaseName
		}
		if _, err := svc.CreateLinkedTicket(ctx, LinkedTicketInput{
			Domain: model.WorkflowDomainDbmgmt, TicketType: model.WorkflowTicketTypeAccess,
			ProjectID: row.ProjectID, Title: title, SubmitterUserID: row.RequesterUserID,
			RefType: model.WorkflowRefDbAccessRequest, RefID: row.ID,
		}); err != nil {
			if isFlowNotConfigured(err) {
				log.Warn("skip access request migration: approval flow not configured", "request_id", row.ID, "project_id", row.ProjectID)
				continue
			}
			return err
		}
	}
	return nil
}

func migratePendingAppUserRequests(ctx context.Context, svc *Service, db *gorm.DB, log *slog.Logger) error {
	var rows []model.DbAppUserRequest
	if err := db.WithContext(ctx).Where("status = ?", model.DbAccessRequestStatusPending).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if svc.HasLinkedTicket(ctx, model.WorkflowRefDbAppUserRequest, row.ID) {
			continue
		}
		title := "应用用户申请"
		if row.MySQLUser != "" {
			title += " · " + row.MySQLUser
		}
		if _, err := svc.CreateLinkedTicket(ctx, LinkedTicketInput{
			Domain: model.WorkflowDomainDbmgmt, TicketType: model.WorkflowTicketTypeAppUser,
			ProjectID: row.ProjectID, Title: title, SubmitterUserID: row.RequesterUserID,
			RefType: model.WorkflowRefDbAppUserRequest, RefID: row.ID,
		}); err != nil {
			if isFlowNotConfigured(err) {
				log.Warn("skip app user request migration: approval flow not configured", "request_id", row.ID, "project_id", row.ProjectID)
				continue
			}
			return err
		}
	}
	return nil
}

func migratePendingReleaseRuns(ctx context.Context, svc *Service, db *gorm.DB, log *slog.Logger) error {
	var rows []model.CicdReleaseRun
	if err := db.WithContext(ctx).Where("status = ? AND audit_enabled = ?", model.CicdRunStatusPendingApproval, true).Find(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		row := rows[i]
		if svc.HasLinkedTicketType(ctx, model.WorkflowRefCicdReleaseRun, row.ID, model.WorkflowTicketTypeRelease) {
			continue
		}
		submitter := uint(0)
		if row.SubmitterUserID != nil {
			submitter = *row.SubmitterUserID
		}
		releaseTicket, err := svc.CreateLinkedTicket(ctx, LinkedTicketInput{
			Domain: model.WorkflowDomainCicd, TicketType: model.WorkflowTicketTypeRelease,
			ProjectID: row.ProjectID, Title: row.Title, SubmitterUserID: submitter,
			RefType: model.WorkflowRefCicdReleaseRun, RefID: row.ID,
		})
		if err != nil {
			if isFlowNotConfigured(err) {
				log.Warn("skip release run migration: approval flow not configured", "run_id", row.ID, "project_id", row.ProjectID)
				continue
			}
			return err
		}
		changeID := row.ChangeWorkflowTicketID
		if changeID == nil || *changeID == 0 {
			changeTicket, err := svc.CreateInfoTicket(ctx, LinkedTicketInput{
				Domain: model.WorkflowDomainOps, TicketType: model.WorkflowTicketTypeChange,
				ProjectID: row.ProjectID, Title: "变更单 · " + row.Title, SubmitterUserID: submitter,
				RefType: model.WorkflowRefCicdReleaseChange, RefID: row.ID,
				Payload: map[string]any{"release_run_id": row.ID, "release_ticket_id": releaseTicket.ID},
			}, model.WorkflowTicketStatusApproved)
			if err != nil {
				return err
			}
			changeID = &changeTicket.ID
		}
		if err := db.WithContext(ctx).Model(&row).Updates(map[string]any{
			"workflow_ticket_id": releaseTicket.ID, "change_workflow_ticket_id": changeID,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}
