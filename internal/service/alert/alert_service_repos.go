package alert

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"yunshu/internal/model"
)

var alertEventProjectBackfillOnce sync.Once

func (s *AlertService) loadEnabledChannels(ctx context.Context) ([]model.AlertChannel, error) {
	ptrs, err := s.channelRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.AlertChannel, len(ptrs))
	for i, ch := range ptrs {
		if ch != nil {
			out[i] = *ch
		}
	}
	return out, nil
}

func (s *AlertService) persistAlertEvent(ctx context.Context, event *model.AlertEvent) error {
	if event == nil {
		return nil
	}
	if event.ProjectID == 0 {
		s.fillAlertEventProjectID(ctx, event)
	}
	if strings.TrimSpace(event.Fingerprint) == "" {
		fillAlertEventFingerprintFromPayload(event, nil)
		if strings.TrimSpace(event.Fingerprint) == "" {
			if fp := alertEventFingerprint(event); fp != "" {
				event.Fingerprint = truncateText(fp, 512)
			}
		}
	}
	if err := s.eventRepo.Create(ctx, event); err != nil {
		return err
	}
	if s.alertStateSvc != nil {
		fp := strings.TrimSpace(event.Fingerprint)
		if fp == "" {
			fp = alertEventFingerprint(event)
		}
		if fp != "" {
			_, _ = s.alertStateSvc.TouchFingerprint(ctx, fp, event.Status)
		}
	}
	return nil
}

func (s *AlertService) fillAlertEventProjectID(ctx context.Context, event *model.AlertEvent) {
	if event == nil || event.ProjectID > 0 {
		return
	}
	if event.DatasourceID > 0 && s.datasourceRepo != nil {
		if ds, err := s.datasourceRepo.GetByID(ctx, event.DatasourceID); err == nil && ds != nil && ds.ProjectID > 0 {
			event.ProjectID = ds.ProjectID
			return
		}
	}
	if pid := projectIDFromMatchedPolicyIDs(ctx, s, event.MatchedPolicyIDs); pid > 0 {
		event.ProjectID = pid
		return
	}
	if pid := projectIDFromRequestPayload(event.RequestPayload); pid > 0 {
		event.ProjectID = pid
	}
}

// ensureAlertEventProjectBackfill 一次性回填历史事件的 project_id（数据源 / 命中订阅节点）。
func (s *AlertService) ensureAlertEventProjectBackfill(ctx context.Context) {
	if s == nil || s.db == nil {
		return
	}
	alertEventProjectBackfillOnce.Do(func() {
		_ = s.db.WithContext(ctx).Exec(`
UPDATE alert_events e
INNER JOIN alert_datasources d ON e.datasource_id = d.id AND d.deleted_at IS NULL
SET e.project_id = d.project_id
WHERE IFNULL(e.project_id, 0) = 0
  AND e.datasource_id > 0
  AND d.project_id > 0
  AND e.deleted_at IS NULL`).Error
		_ = s.db.WithContext(ctx).Exec(`
UPDATE alert_events e
INNER JOIN alert_subscription_nodes n
  ON n.deleted_at IS NULL AND n.project_id > 0
 AND FIND_IN_SET(n.id, e.matched_policy_ids)
SET e.project_id = n.project_id
WHERE IFNULL(e.project_id, 0) = 0
  AND e.matched_policy_ids IS NOT NULL
  AND TRIM(e.matched_policy_ids) <> ''
  AND e.deleted_at IS NULL`).Error
	})
}

func projectIDFromMatchedPolicyIDs(ctx context.Context, s *AlertService, raw string) uint {
	raw = strings.TrimSpace(raw)
	if raw == "" || s == nil || s.db == nil {
		return 0
	}
	parts := strings.Split(raw, ",")
	ids := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil || n == 0 {
			continue
		}
		ids = append(ids, uint(n))
	}
	if len(ids) == 0 {
		return 0
	}
	var pid uint
	_ = s.db.WithContext(ctx).Model(&model.AlertSubscriptionNode{}).
		Select("project_id").Where("id IN ? AND project_id > 0", ids).
		Order("project_id ASC").Limit(1).Scan(&pid)
	return pid
}

func projectIDFromRequestPayload(raw string) uint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return 0
	}
	if id := payloadUintAny(payload["project_id"]); id > 0 {
		return id
	}
	if id := payloadUintAny(payload["projectId"]); id > 0 {
		return id
	}
	if labels, ok := payload["labels"].(map[string]any); ok {
		if id := payloadUintAny(labels["project_id"]); id > 0 {
			return id
		}
	}
	if cloud, ok := payload["cloud"].(map[string]any); ok {
		if id := payloadUintAny(cloud["project_id"]); id > 0 {
			return id
		}
	}
	return 0
}

func alertEventFingerprint(event *model.AlertEvent) string {
	if event == nil {
		return ""
	}
	if fp := strings.TrimSpace(event.Fingerprint); fp != "" {
		return fp
	}
	if fp := strings.TrimSpace(event.GroupKey); fp != "" {
		return fp
	}
	return strings.TrimSpace(event.LabelsDigest)
}
