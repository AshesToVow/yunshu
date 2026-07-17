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
