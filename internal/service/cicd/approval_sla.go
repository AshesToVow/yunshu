package cicd

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"yunshu/internal/model"
)

type approvalReminderCandidate struct {
	Step    model.CicdReleaseApprovalStep
	Release model.CicdReleaseRun
}

func (s *Service) syncApprovalReminders(ctx context.Context) {
	cfg := s.resolvedConfig(ctx)
	if !cfg.Enabled || cfg.ApprovalSlaHours <= 0 {
		return
	}
	if s.mailer == nil || !s.mailer.Enabled() {
		return
	}
	intervalHours := cfg.ApprovalReminderIntervalHours
	if intervalHours <= 0 {
		intervalHours = 4
	}
	s.backfillPendingStepActivatedAt(ctx)

	sla := time.Duration(cfg.ApprovalSlaHours) * time.Hour
	interval := time.Duration(intervalHours) * time.Hour
	now := time.Now()

	var releases []model.CicdReleaseRun
	if err := s.db.WithContext(ctx).
		Where("status = ? AND audit_enabled = ?", model.CicdRunStatusPendingApproval, true).
		Find(&releases).Error; err != nil {
		slog.Default().With("component", "cicd").Warn("list pending approval releases failed", "error", err)
		return
	}
	for _, rel := range releases {
		// 已关联统一工单的由 syncWorkflowApprovalReminders 催办，避免双发
		if rel.WorkflowTicketID != nil && *rel.WorkflowTicketID > 0 {
			continue
		}
		step, err := s.getCurrentPendingStep(ctx, rel.ID)
		if err != nil || step == nil {
			continue
		}
		if step.ActivatedAt == nil {
			continue
		}
		if now.Sub(*step.ActivatedAt) < sla {
			continue
		}
		if step.LastRemindedAt != nil && now.Sub(*step.LastRemindedAt) < interval {
			continue
		}
		candidate := approvalReminderCandidate{Step: *step, Release: rel}
		if err := s.sendApprovalReminderEmail(ctx, candidate); err != nil {
			slog.Default().With("component", "cicd").Warn(
				"approval SLA reminder email failed",
				"release_id", rel.ID,
				"step_id", step.ID,
				"error", err,
			)
			continue
		}
		ts := now
		_ = s.db.WithContext(ctx).Model(&model.CicdReleaseApprovalStep{}).
			Where("id = ?", step.ID).
			Update("last_reminded_at", ts).Error
	}
	s.syncWorkflowApprovalReminders(ctx, sla, interval, now)
}

// syncWorkflowApprovalReminders 对已切到统一引擎的发布工单按 workflow_ticket_steps 催办。
func (s *Service) syncWorkflowApprovalReminders(ctx context.Context, sla, interval time.Duration, now time.Time) {
	type row struct {
		StepID          uint
		TicketID        uint
		StageName       string
		ActivatedAt     time.Time
		LastRemindedAt  *time.Time
		UserGroupID     *uint
		AssigneeUserID  *uint
		RefID           uint
		Title           string
		ProjectID       uint
		SubmitterUserID uint
	}
	var list []row
	err := s.db.WithContext(ctx).Raw(`
SELECT s.id AS step_id, s.ticket_id, s.stage_name, s.activated_at, s.last_reminded_at,
       s.user_group_id, s.assignee_user_id, t.ref_id, t.title, t.project_id, t.submitter_user_id
FROM workflow_ticket_steps s
JOIN workflow_tickets t ON t.id = s.ticket_id AND t.deleted_at IS NULL
WHERE t.domain = ? AND t.ticket_type = ? AND t.status = ?
  AND s.status = ? AND s.activated_at IS NOT NULL AND s.deleted_at IS NULL
`, model.WorkflowDomainCicd, model.WorkflowTicketTypeRelease, model.WorkflowTicketStatusPending,
		model.WorkflowStepPending).Scan(&list).Error
	if err != nil {
		slog.Default().With("component", "cicd").Warn("list workflow approval steps failed", "error", err)
		return
	}
	for _, it := range list {
		if now.Sub(it.ActivatedAt) < sla {
			continue
		}
		if it.LastRemindedAt != nil && now.Sub(*it.LastRemindedAt) < interval {
			continue
		}
		userIDs := s.workflowStepNotifyUserIDs(ctx, it.AssigneeUserID, it.UserGroupID)
		if len(userIDs) == 0 {
			continue
		}
		emails := s.collectUserEmails(ctx, userIDs)
		if len(emails) == 0 {
			continue
		}
		waitHours := int(now.Sub(it.ActivatedAt).Hours())
		if waitHours < 1 {
			waitHours = 1
		}
		appName := strings.TrimSpace(s.appName)
		if appName == "" {
			appName = "Yunshu"
		}
		subject := fmt.Sprintf("[%s CI/CD] 发布审批超时提醒 - %s", appName, strings.TrimSpace(it.Title))
		body := fmt.Sprintf("发布工单 #%d（统一工单 #%d）在节点「%s」已等待超过 %d 小时，请尽快审批。\n项目ID：%d",
			it.RefID, it.TicketID, it.StageName, waitHours, it.ProjectID)
		sent := false
		for _, email := range emails {
			if err := s.mailer.Send(ctx, email, subject, body); err == nil {
				sent = true
			}
		}
		if !sent {
			continue
		}
		_ = s.db.WithContext(ctx).Model(&model.WorkflowTicketStep{}).
			Where("id = ?", it.StepID).
			Update("last_reminded_at", now).Error
	}
}

func (s *Service) workflowStepNotifyUserIDs(ctx context.Context, assigneeID, groupID *uint) []uint {
	if assigneeID != nil && *assigneeID > 0 {
		return []uint{*assigneeID}
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

func (s *Service) backfillPendingStepActivatedAt(ctx context.Context) {
	type row struct {
		ID uint
	}
	var ids []row
	err := s.db.WithContext(ctx).Raw(`
SELECT s.id FROM cicd_release_approval_steps s
JOIN cicd_release_runs r ON r.id = s.release_run_id
WHERE r.status = ? AND s.status = ? AND s.activated_at IS NULL
AND s.sort_order = (
  SELECT MIN(s2.sort_order) FROM cicd_release_approval_steps s2
  WHERE s2.release_run_id = s.release_run_id AND s2.status = ?
)`, model.CicdRunStatusPendingApproval, model.CicdApprovalStepPending, model.CicdApprovalStepPending).Scan(&ids).Error
	if err != nil || len(ids) == 0 {
		return
	}
	now := time.Now()
	for _, id := range ids {
		_ = s.db.WithContext(ctx).Model(&model.CicdReleaseApprovalStep{}).
			Where("id = ? AND activated_at IS NULL", id.ID).
			Update("activated_at", now).Error
	}
}

func (s *Service) sendApprovalReminderEmail(ctx context.Context, c approvalReminderCandidate) error {
	if c.Step.UserGroupID == nil || *c.Step.UserGroupID == 0 {
		return fmt.Errorf("approval step has no user group")
	}
	memberIDs, err := s.userGroupRepo.ListMemberUserIDs(ctx, *c.Step.UserGroupID)
	if err != nil {
		return err
	}
	emails := s.collectUserEmails(ctx, memberIDs)
	if len(emails) == 0 {
		return fmt.Errorf("no approver emails in user group")
	}

	projectName := s.lookupProjectName(ctx, c.Release.ProjectID)
	serviceName := s.lookupServiceName(ctx, c.Release.ServiceID)
	waitHours := int(time.Since(*c.Step.ActivatedAt).Hours())
	if waitHours < 1 {
		waitHours = 1
	}
	appName := strings.TrimSpace(s.appName)
	if appName == "" {
		appName = "Yunshu"
	}
	subject := fmt.Sprintf("[%s CI/CD] 发布审批超时提醒 - %s", appName, strings.TrimSpace(c.Release.Title))
	textBody := fmt.Sprintf(`您好，

发布工单 #%d 已在审批节点「%s」等待超过 %d 小时，请及时处理。

项目：%s
应用：%s
环境：%s
提交人：%s
工单标题：%s

请登录平台进入 CI/CD 待审核列表完成审批。`,
		c.Release.ID,
		c.Step.StageName,
		waitHours,
		projectName,
		serviceName,
		strings.TrimSpace(c.Release.Tenv),
		strings.TrimSpace(c.Release.SubmitterName),
		strings.TrimSpace(c.Release.Title),
	)

	var sendErr error
	for _, email := range emails {
		if err := s.mailer.Send(ctx, email, subject, textBody); err != nil {
			sendErr = err
		}
	}
	return sendErr
}

func (s *Service) collectUserEmails(ctx context.Context, userIDs []uint) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		email := s.lookupUserEmailByID(ctx, id)
		if email == "" {
			continue
		}
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}
		out = append(out, email)
	}
	return out
}

func (s *Service) lookupProjectName(ctx context.Context, projectID uint) string {
	if projectID == 0 {
		return "-"
	}
	var row model.Project
	if err := s.db.WithContext(ctx).Select("name").Where("id = ?", projectID).First(&row).Error; err != nil {
		return fmt.Sprintf("#%d", projectID)
	}
	if name := strings.TrimSpace(row.Name); name != "" {
		return name
	}
	return fmt.Sprintf("#%d", projectID)
}

func (s *Service) lookupServiceName(ctx context.Context, serviceID uint) string {
	if serviceID == 0 {
		return "-"
	}
	var row model.CicdService
	if err := s.db.WithContext(ctx).Select("name, identifier").Where("id = ?", serviceID).First(&row).Error; err != nil {
		return fmt.Sprintf("#%d", serviceID)
	}
	if name := strings.TrimSpace(row.Name); name != "" {
		return name
	}
	if id := strings.TrimSpace(row.Identifier); id != "" {
		return id
	}
	return fmt.Sprintf("#%d", serviceID)
}
