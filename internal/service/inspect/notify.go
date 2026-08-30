package inspect

// 巡检邮件通知：报告邮件、异常告警邮件、手动补发，以及收件人解析。

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/mailer"
)

func (s *Service) sendRunEmail(ctx context.Context, plan *model.InspectPlan, run *model.InspectRun, data ReportData, htmlBytes, pdfBytes []byte) error {
	if s.mailer == nil || !s.mailer.Enabled() || plan == nil {
		return nil
	}
	recipients := parseRecipients(plan.RecipientsJSON)
	if len(recipients) == 0 {
		return nil
	}
	subject := fmt.Sprintf("[%s] 巡检报告 %s 分数%.0f", s.appNameOrDefault(), data.Project, data.Score)
	text := data.Summary
	htmlBody := fmt.Sprintf("<p>%s</p><p>严重 %d / 警告 %d / 正常 %d</p>",
		html.EscapeString(data.Summary), run.CriticalCount, run.WarningCount, run.NormalCount)
	atts := []mailer.Attachment{
		{Filename: fmt.Sprintf("inspect-run-%d.html", run.ID), ContentType: "text/html; charset=utf-8", Content: htmlBytes},
	}
	if len(pdfBytes) > 0 {
		atts = append(atts, mailer.Attachment{Filename: fmt.Sprintf("inspect-run-%d.pdf", run.ID), ContentType: "application/pdf", Content: pdfBytes})
	}
	var lastErr error
	for _, to := range recipients {
		if err := s.mailer.SendWithAttachments(ctx, to, subject, text, htmlBody, atts); err != nil {
			lastErr = err
		}
	}
	if lastErr == nil {
		now := time.Now()
		_ = s.db.WithContext(ctx).Model(run).Update("email_sent_at", now).Error
	}
	return lastErr
}

func (s *Service) sendInspectAnomalyEmail(ctx context.Context, plan *model.InspectPlan, run *model.InspectRun, data ReportData) error {
	if s.mailer == nil || !s.mailer.Enabled() || plan == nil || run == nil {
		return nil
	}
	recipients := parseRecipients(plan.RecipientsJSON)
	if len(recipients) == 0 {
		return nil
	}
	subject := fmt.Sprintf("[%s] 巡检异常告警 %s", s.appNameOrDefault(), data.Project)
	if run.Status == "failed" {
		subject += "（执行失败）"
	} else if run.CriticalCount > 0 {
		subject += fmt.Sprintf("（严重 %d）", run.CriticalCount)
	}
	text := strings.TrimSpace(data.Summary)
	if text == "" {
		text = run.ErrorMessage
	}
	htmlBody := fmt.Sprintf("<p><strong>巡检异常</strong></p><p>%s</p><p>严重 %d / 警告 %d / 分数 %.0f</p>",
		html.EscapeString(text), run.CriticalCount, run.WarningCount, data.Score)
	var lastErr error
	for _, to := range recipients {
		if err := s.mailer.SendMultipart(ctx, to, subject, text, htmlBody); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// ResendEmail 依据已归档报告重新发送邮件（不重新采集）。
func (s *Service) ResendEmail(ctx context.Context, projectID, runID uint) error {
	run, err := s.GetRun(ctx, projectID, runID)
	if err != nil {
		return err
	}
	plan, err := s.GetOrCreatePlan(ctx, projectID)
	if err != nil {
		return err
	}
	htmlBytes, _, err := s.ReadReport(ctx, projectID, runID, "html")
	if err != nil {
		return err
	}
	pdfBytes, _, _ := s.ReadReport(ctx, projectID, runID, "pdf")
	if len(pdfBytes) >= 4 && string(pdfBytes[:4]) != "%PDF" {
		pdfBytes = nil
	}
	data := ReportData{
		Project:    fmt.Sprintf("project-%d", projectID),
		Datasource: run.DatasourceName,
		Score:      run.Score,
		Grade:      run.Grade,
		Summary:    run.Summary,
		Timestamp:  time.Now(),
	}
	return s.sendRunEmail(ctx, plan, run, data, htmlBytes, pdfBytes)
}

func parseRecipients(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		list = strings.Split(raw, ",")
	}
	return uniqEmails(list)
}

func uniqEmails(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, e := range in {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" || seen[e] {
			continue
		}
		seen[e] = true
		out = append(out, e)
	}
	return out
}
