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
