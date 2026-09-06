package dbmgmt

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"yunshu/internal/model"
)

func (s *Service) syncApprovalReminders(ctx context.Context) {
	cfg := s.resolvedConfig(ctx)
	if cfg.ApprovalSlaHours <= 0 || s.mailer == nil || !s.mailer.Enabled() {
		return
	}
	intervalHours := cfg.ApprovalReminderIntervalHours
	if intervalHours <= 0 {
		intervalHours = 4
	}
	sla := time.Duration(cfg.ApprovalSlaHours) * time.Hour
	interval := time.Duration(intervalHours) * time.Hour
	now := time.Now()

	accessSteps, _ := s.repo.ListPendingAccessStepsForReminder(ctx, sla)
	for _, step := range accessSteps {
		if s.workflowEngine().HasLinkedTicket(ctx, model.WorkflowRefDbAccessRequest, step.AccessRequestID) {
			continue
		}
		if step.LastRemindedAt != nil && now.Sub(*step.LastRemindedAt) < interval {
			continue
		}
		if err := s.sendAccessReminderEmail(ctx, step); err != nil {
			slog.Default().With("component", "dbmgmt").Warn("access SLA reminder failed", "step_id", step.ID, "error", err)
			continue
		}
		ts := now
		step.LastRemindedAt = &ts
		_ = s.repo.UpdateAccessRequestStep(ctx, &step)
	}

	ticketSteps, _ := s.repo.ListPendingTicketStepsForReminder(ctx, sla)
	for _, step := range ticketSteps {
		if s.workflowEngine().HasLinkedTicket(ctx, model.WorkflowRefDbSqlTicket, step.TicketID) {
			continue
		}
		if step.LastRemindedAt != nil && now.Sub(*step.LastRemindedAt) < interval {
			continue
		}
		if err := s.sendTicketReminderEmail(ctx, step); err != nil {
			slog.Default().With("component", "dbmgmt").Warn("ticket SLA reminder failed", "step_id", step.ID, "error", err)
			continue
		}
		ts := now
		step.LastRemindedAt = &ts
		_ = s.repo.UpdateSqlTicketStep(ctx, &step)
	}
	s.syncWorkflowApprovalReminders(ctx, sla, interval, now)
}

func (s *Service) syncWorkflowApprovalReminders(ctx context.Context, sla, interval time.Duration, now time.Time) {
	if s.db == nil || s.mailer == nil || !s.mailer.Enabled() {
		return
	}
	type row struct {
		StepID           uint
		TicketID         uint
		StageName        string
		ActivatedAt      time.Time
		LastRemindedAt   *time.Time
		UserGroupID      *uint
		AssigneeUserID   *uint
		AssigneeRuleType string
		Domain           string
		RefID            uint
		Title            string
		TicketType       string
	}
	var list []row
	err := s.db.WithContext(ctx).Raw(`
SELECT s.id AS step_id, s.ticket_id, s.stage_name, s.activated_at, s.last_reminded_at,
       s.user_group_id, s.assignee_user_id, s.assignee_rule_type, t.domain, t.ref_id, t.title, t.ticket_type
FROM workflow_ticket_steps s
JOIN workflow_tickets t ON t.id = s.ticket_id AND t.deleted_at IS NULL
WHERE t.domain IN (?, ?) AND t.status = ?
  AND s.status = ? AND s.activated_at IS NOT NULL AND s.deleted_at IS NULL
`, model.WorkflowDomainDbmgmt, model.WorkflowDomainAI, model.WorkflowTicketStatusPending, model.WorkflowStepPending).Scan(&list).Error
	if err != nil {
		slog.Default().With("component", "dbmgmt").Warn("list workflow approval steps failed", "error", err)
		return
	}
	for _, it := range list {
		if now.Sub(it.ActivatedAt) < sla {
			continue
		}
		if it.LastRemindedAt != nil && now.Sub(*it.LastRemindedAt) < interval {
			continue
		}
		userIDs := s.workflowStepNotifyUserIDs(ctx, it.AssigneeUserID, it.UserGroupID, it.AssigneeRuleType)
		if len(userIDs) == 0 {
			continue
		}
		label := "数据库审批"
		switch {
		case it.Domain == model.WorkflowDomainAI:
			label = "AI 高危操作"
		case it.TicketType == model.WorkflowTicketTypeSql:
			label = "SQL 工单"
		case it.TicketType == model.WorkflowTicketTypeAccess:
			label = "权限申请"
		case it.TicketType == model.WorkflowTicketTypeAppUser:
			label = "应用账号申请"
		}
		appName := s.appName
		if appName == "" {
			appName = "Yunshu"
		}
		subject := fmt.Sprintf("[%s] %s待审批超时", appName, label)
		body := fmt.Sprintf("「%s」#%d 在节点「%s」已超时，请尽快处理（统一工单 #%d）。",
			label, it.RefID, it.StageName, it.TicketID)
		if err := s.sendMailToUsers(ctx, userIDs, subject, body); err != nil {
			slog.Default().With("component", "dbmgmt").Warn("workflow SLA reminder failed", "step_id", it.StepID, "error", err)
			continue
		}
		_ = s.db.WithContext(ctx).Model(&model.WorkflowTicketStep{}).
			Where("id = ?", it.StepID).
			Update("last_reminded_at", now).Error
	}
}

func (s *Service) workflowStepNotifyUserIDs(ctx context.Context, assigneeID, groupID *uint, ruleType string) []uint {
	if assigneeID != nil && *assigneeID > 0 {
		return []uint{*assigneeID}
	}
	if ruleType == model.WorkflowAssigneePlatformRole {
		return s.platformRoleApproverUserIDs(ctx)
	}
	if groupID == nil || *groupID == 0 || s.userGroupRepo == nil {
		return nil
	}
	ids, err := s.userGroupRepo.ListMemberUserIDs(ctx, *groupID)
	if err != nil {
		return nil
	}
	return ids
}

func (s *Service) platformRoleApproverUserIDs(ctx context.Context) []uint {
	if s.userRepo == nil {
		return nil
	}
	seen := map[uint]struct{}{}
	out := make([]uint, 0)
	for _, code := range []string{"admin", "ops-admin", "ai-approver", "super-admin"} {
		ids, err := s.userRepo.ListUserIDsByRoleCode(ctx, code)
		if err != nil {
			continue
		}
		for _, id := range ids {
			if id == 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

func (s *Service) sendAccessReminderEmail(ctx context.Context, step model.DbAccessRequestStep) error {
	if step.UserGroupID == nil {
		return nil
	}
	ids, err := s.userGroupRepo.ListMemberUserIDs(ctx, *step.UserGroupID)
	if err != nil || len(ids) == 0 {
		return err
	}
	subject := fmt.Sprintf("[%s] 数据库权限申请待审批", s.appName)
	body := fmt.Sprintf("审批节点「%s」已超时，请尽快处理权限申请 #%d。", step.StageName, step.AccessRequestID)
	return s.sendMailToUsers(ctx, ids, subject, body)
}

func (s *Service) sendTicketReminderEmail(ctx context.Context, step model.DbSqlTicketStep) error {
	if step.UserGroupID == nil {
		return nil
	}
	ids, err := s.userGroupRepo.ListMemberUserIDs(ctx, *step.UserGroupID)
	if err != nil || len(ids) == 0 {
		return err
	}
	subject := fmt.Sprintf("[%s] 数据库 SQL 工单待审批", s.appName)
	body := fmt.Sprintf("审批节点「%s」已超时，请尽快处理 SQL 工单 #%d。", step.StageName, step.TicketID)
	return s.sendMailToUsers(ctx, ids, subject, body)
}

func (s *Service) sendMailToUsers(ctx context.Context, userIDs []uint, subject, body string) error {
	for _, id := range userIDs {
		u, err := s.userRepo.GetByID(ctx, id)
		if err != nil || u == nil || u.Email == nil || *u.Email == "" {
			continue
		}
		if err := s.mailer.Send(ctx, *u.Email, subject, body); err != nil {
			return err
		}
	}
	return nil
}
