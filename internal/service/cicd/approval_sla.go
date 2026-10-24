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

	releases, err := s.repo.ListPendingApprovalReleases(ctx, 0)
	if err != nil {
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
		_ = s.repo.MarkApprovalStepsReminded(ctx, []uint{step.ID}, now)
	}
	s.syncWorkflowApprovalReminders(ctx, sla, interval, now)
}

// syncWorkflowApprovalReminders 对已切到统一引擎的发布工单按 workflow_ticket_steps 催办。
func (s *Service) syncWorkflowApprovalReminders(ctx context.Context, sla, interval time.Duration, now time.Time) {
	list, err := s.repo.ListWorkflowApprovalReminderRows(ctx)
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
		_ = s.repo.UpdateWorkflowTicketStepFields(ctx, it.StepID, map[string]any{"last_reminded_at": now})
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
	ids, err := s.repo.ListLegacyApprovalReminderSeedIDs(ctx)
	if err != nil || len(ids) == 0 {
		return
	}
	now := time.Now()
	for _, id := range ids {
		_ = s.repo.UpdateApprovalStepFields(ctx, id, map[string]any{"activated_at": now})
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
	name, err := s.repo.GetProjectName(ctx, projectID)
	if err != nil {
		return fmt.Sprintf("#%d", projectID)
	}
	if name = strings.TrimSpace(name); name != "" {
		return name
	}
	return fmt.Sprintf("#%d", projectID)
}

func (s *Service) lookupServiceName(ctx context.Context, serviceID uint) string {
	if serviceID == 0 {
		return "-"
	}
	brief, err := s.repo.GetServiceBrief(ctx, serviceID)
	if err != nil {
		return fmt.Sprintf("#%d", serviceID)
	}
	if name := strings.TrimSpace(brief.Name); name != "" {
		return name
	}
	if id := strings.TrimSpace(brief.Identifier); id != "" {
		return id
	}
	return fmt.Sprintf("#%d", serviceID)
}
