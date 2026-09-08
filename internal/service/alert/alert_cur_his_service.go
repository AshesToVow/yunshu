package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
)

type AlertCurEventListQuery struct {
	ProjectID    uint   `form:"project_id"`
	DatasourceID uint   `form:"datasource_id"`
	Severity     string `form:"severity"`
	Keyword      string `form:"keyword"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

// AlertCurEventView 当前告警列表 DTO（含处理人摘要、认领状态、最新进展）。
type AlertCurEventView struct {
	model.AlertCurEvent
	HandlerSummary string     `json:"handler_summary"`
	Acked          bool       `json:"acked"`
	AckBy          string     `json:"ack_by,omitempty"`
	AckExpiresAt   *time.Time `json:"ack_expires_at,omitempty"`
	LatestNote     string     `json:"latest_note,omitempty"`
	LatestNoteBy   string     `json:"latest_note_by,omitempty"`
	LatestNoteAt   *time.Time `json:"latest_note_at,omitempty"`
}

type AlertHisEventListQuery struct {
	ProjectID    uint   `form:"project_id"`
	DatasourceID uint   `form:"datasource_id"`
	Severity     string `form:"severity"`
	Keyword      string `form:"keyword"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

// UpsertCurEvent 写入/更新当前告警（屏蔽后不调用）。
func (s *AlertService) UpsertCurEvent(ctx context.Context, row *model.AlertCurEvent) error {
	if s == nil || s.curHisRepo == nil || row == nil {
		return nil
	}
	fp := strings.TrimSpace(row.Fingerprint)
	if fp == "" {
		return nil
	}
	row.Fingerprint = fp
	if strings.TrimSpace(row.Status) == "" {
		row.Status = "firing"
	}
	now := time.Now().UTC()
	if row.StartsAt.IsZero() {
		row.StartsAt = now
	}
	row.UpdatedAt = now
	err := s.curHisRepo.UpsertCurEvent(ctx, row)
	return bizerrors.Pass(ctx, "alert.cur", "UpsertCurEvent", err)
}

// ResolveCurEvent 将当前告警迁入历史并删除当前行。
func (s *AlertService) ResolveCurEvent(ctx context.Context, fingerprint string, resolvedAt time.Time) error {
	if s == nil || s.curHisRepo == nil {
		return nil
	}
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return nil
	}
	if resolvedAt.IsZero() {
		resolvedAt = time.Now().UTC()
	}
	return bizerrors.Pass(ctx, "alert.cur", "ResolveCurEvent", s.curHisRepo.ResolveCurEvent(ctx, fp, resolvedAt))
}

func (s *AlertService) ListCurEvents(ctx context.Context, q AlertCurEventListQuery) ([]AlertCurEventView, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	if s == nil || s.curHisRepo == nil {
		return nil, 0, page, pageSize, nil
	}
	list, total, err := s.curHisRepo.ListCurEvents(ctx, repository.AlertCurEventListFilter{
		ProjectID:    q.ProjectID,
		DatasourceID: q.DatasourceID,
		Severity:     q.Severity,
		Keyword:      q.Keyword,
	}, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.cur", "ListCurEvents", err)
	}
	return s.enrichCurEventViews(ctx, list), total, page, pageSize, nil
}

func (s *AlertService) enrichCurEventViews(ctx context.Context, list []model.AlertCurEvent) []AlertCurEventView {
	out := make([]AlertCurEventView, 0, len(list))
	if len(list) == 0 {
		return out
	}
	fps := make([]string, 0, len(list))
	ruleIDs := map[uint]struct{}{}
	ruleByFP := map[string]uint{}
	for _, row := range list {
		fps = append(fps, row.Fingerprint)
		rid := monitorRuleIDFromLabelsJSON(row.LabelsJSON)
		if rid > 0 {
			ruleIDs[rid] = struct{}{}
			ruleByFP[row.Fingerprint] = rid
		}
	}
	acks, _ := s.ListActiveAcksByFingerprints(ctx, fps)
	notes, _ := s.ListLatestNotesByFingerprints(ctx, fps)
	handlerByRule := map[uint]string{}
	if s.assigneeSvc != nil {
		for rid := range ruleIDs {
			handlerByRule[rid] = s.assigneeSvc.ResolveHandlerSummary(ctx, rid)
		}
	}
	dutyByRule := map[uint]string{}
	if s.dutySvc != nil {
		for rid := range ruleIDs {
			emails, _ := s.dutySvc.ResolveNotifyEmailsAtRule(ctx, rid, time.Now())
			if len(emails) == 0 {
				continue
			}
			if len(emails) <= 2 {
				dutyByRule[rid] = "值班 " + strings.Join(emails, ", ")
			} else {
				dutyByRule[rid] = fmt.Sprintf("值班 %s 等 %d 人", strings.Join(emails[:2], ", "), len(emails))
			}
		}
	}
	for _, row := range list {
		v := AlertCurEventView{AlertCurEvent: row}
		if rid := ruleByFP[row.Fingerprint]; rid > 0 {
			parts := make([]string, 0, 2)
			if h := strings.TrimSpace(handlerByRule[rid]); h != "" {
				parts = append(parts, h)
			}
			if d := strings.TrimSpace(dutyByRule[rid]); d != "" {
				parts = append(parts, d)
			}
			v.HandlerSummary = strings.Join(parts, " · ")
		}
		if ack, ok := acks[row.Fingerprint]; ok && ack.Acked {
			v.Acked = true
			v.AckBy = ack.UserName
			v.AckExpiresAt = ack.ExpiresAt
		}
		if note, ok := notes[row.Fingerprint]; ok {
			v.LatestNote = note.Content
			v.LatestNoteBy = note.UserName
			at := note.CreatedAt
			v.LatestNoteAt = &at
		}
		out = append(out, v)
	}
	return out
}

func monitorRuleIDFromLabelsJSON(raw string) uint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return 0
	}
	v, ok := m["monitor_rule_id"]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		if t > 0 {
			return uint(t)
		}
	case string:
		if n, err := strconv.ParseUint(strings.TrimSpace(t), 10, 64); err == nil {
			return uint(n)
		}
	}
	return 0
}

func (s *AlertService) ListHisEvents(ctx context.Context, q AlertHisEventListQuery) ([]model.AlertHisEvent, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	if s == nil || s.curHisRepo == nil {
		return nil, 0, page, pageSize, nil
	}
	list, total, err := s.curHisRepo.ListHisEvents(ctx, repository.AlertHisEventListFilter{
		ProjectID:    q.ProjectID,
		DatasourceID: q.DatasourceID,
		Severity:     q.Severity,
		Keyword:      q.Keyword,
	}, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.his", "ListHisEvents", err)
	}
	return list, total, page, pageSize, nil
}

func buildCurEventFromIngress(
	source, receiver, title, severity, status, envLabel, groupKey string,
	dsID uint, labels, annotations map[string]string,
	fp string, startsAt time.Time, value string,
) *model.AlertCurEvent {
	lb, _ := json.Marshal(labels)
	ab, _ := json.Marshal(annotations)
	projectID := uint(0)
	if v := strings.TrimSpace(labels["project_id"]); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			projectID = uint(n)
		}
	}
	summary := strings.TrimSpace(annotations["summary"])
	if summary == "" {
		summary = title
	}
	if startsAt.IsZero() {
		startsAt = time.Now().UTC()
	}
	return &model.AlertCurEvent{
		Fingerprint:     fp,
		Alertname:       title,
		Severity:        severity,
		Status:          status,
		Source:          source,
		Receiver:        receiver,
		Cluster:         envLabel,
		ProjectID:       projectID,
		DatasourceID:    dsID,
		GroupKey:        groupKey,
		LabelsJSON:      string(lb),
		AnnotationsJSON: string(ab),
		Summary:         truncateText(summary, 512),
		Value:           truncateText(value, 128),
		StartsAt:        startsAt,
	}
}
