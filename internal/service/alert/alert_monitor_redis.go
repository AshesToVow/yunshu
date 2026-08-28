package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	bizerrors "yunshu/internal/pkg/errors"

	"yunshu/internal/model"
	"yunshu/internal/pkg/alertnotify"

	"github.com/redis/go-redis/v9"
)

func monitorEvalMetaKey(ruleID uint) string {
	return fmt.Sprintf("alert:mon:meta:%d", ruleID)
}

func monitorEvalSeriesStateKey(ruleID uint, fp string) string {
	return fmt.Sprintf("alert:mon:series:%d:%s", ruleID, fp)
}

func monitorEvalTrackedSetKey(ruleID uint) string {
	return fmt.Sprintf("alert:mon:tracked:%d", ruleID)
}

func monitorEvalStateKey(ruleID uint) string {
	// 兼容旧单规则状态键；新逻辑以 meta/series 为准。
	return fmt.Sprintf("alert:mon:state:%d", ruleID)
}

func monitorEvalLockKey(ruleID uint) string {
	return fmt.Sprintf("alert:mon:lock:%d", ruleID)
}

func (s *AlertService) monitorEvalLockAcquire(ctx context.Context, ruleID uint, ttlSec int) bool {
	if s.redis == nil {
		return true
	}
	if ttlSec < 10 {
		ttlSec = 10
	}
	ok, err := s.redis.SetNX(ctx, monitorEvalLockKey(ruleID), "1", time.Duration(ttlSec)*time.Second).Result()
	return err == nil && ok
}

func (s *AlertService) monitorEvalLockRelease(ctx context.Context, ruleID uint) {
	if s.redis == nil {
		return
	}
	_ = s.redis.Del(ctx, monitorEvalLockKey(ruleID)).Err()
}

// redisLastEvalTime 读取上次评估时间（RFC3339Nano），无记录或解析失败时 has=false。
func (s *AlertService) redisLastEvalTime(ctx context.Context, ruleID uint) (t time.Time, has bool) {
	if s.redis == nil {
		return time.Time{}, false
	}
	last, err := s.redis.HGet(ctx, monitorEvalMetaKey(ruleID), "last_eval").Result()
	if err != nil && err != redis.Nil {
		return time.Time{}, false
	}
	if strings.TrimSpace(last) == "" {
		// 兼容旧键
		last, err = s.redis.HGet(ctx, monitorEvalStateKey(ruleID), "last_eval").Result()
		if err != nil && err != redis.Nil {
			return time.Time{}, false
		}
	}
	if strings.TrimSpace(last) == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, last)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, last)
		if err != nil {
			return time.Time{}, false
		}
	}
	return parsed, true
}

func (s *AlertService) shouldEvalRuleRedis(ctx context.Context, ruleID uint, intervalSec int, now time.Time) bool {
	if s.redis == nil {
		return true
	}
	if intervalSec < 5 {
		intervalSec = 5
	}
	last, err := s.redis.HGet(ctx, monitorEvalMetaKey(ruleID), "last_eval").Result()
	if err != nil && err != redis.Nil {
		return true
	}
	if strings.TrimSpace(last) == "" {
		last, err = s.redis.HGet(ctx, monitorEvalStateKey(ruleID), "last_eval").Result()
		if err != nil && err != redis.Nil {
			return true
		}
	}
	if strings.TrimSpace(last) == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339Nano, last)
	if err != nil {
		t, err = time.Parse(time.RFC3339, last)
		if err != nil {
			return true
		}
	}
	return now.Sub(t) >= time.Duration(intervalSec)*time.Second
}

func (s *AlertService) redisTouchLastEval(ctx context.Context, ruleID uint, now time.Time) {
	if s.redis == nil {
		return
	}
	_ = s.redis.HSet(ctx, monitorEvalMetaKey(ruleID), "last_eval", now.UTC().Format(time.RFC3339Nano)).Err()
	_ = s.redis.Expire(ctx, monitorEvalMetaKey(ruleID), 7*24*time.Hour).Err()
}

func parseRFC3339Ptr(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		return nil, bizerrors.Pass(context.Background(), "alert.rule", "parseRFC3339Ptr", err)
	}
	return &t, nil
}

// monitorShouldReingressFiring 持续 firing 时：未成功外发过则重试入站；已外发则仅当 repeat/group 窗口允许时再入站，避免每轮评估写一条「分组节流」历史。
func (s *AlertService) monitorShouldReingressFiring(ctx context.Context, fp string, labels map[string]string, alertname, severity string) bool {
	if !s.alertFiringWasDelivered(ctx, fp) {
		return true
	}
	enriched := s.enrichCanonicalIngressLabels(ctx, labels, "platform-monitor", fp)
	dims := alertnotify.ExtractDims(enriched)
	groupKey := s.computeGroupKey("platform-monitor", "firing", severity, alertname, enriched, dims)
	labelsDigest := s.labelsDigestForGroupTiming("platform-monitor", "firing", severity, alertname, enriched)
	shouldSend, _, _, _, _ := s.peekFiringGroupTiming(ctx, groupKey, labelsDigest)
	return shouldSend
}

func (s *AlertService) emitMonitorPlatformFiring(ctx context.Context, rule *model.AlertMonitorRule, labels, annotations map[string]string, fp string, now time.Time) {
	_ = s.receiveCanonicalSync(ctx, NewCanonicalAlert(
		IngressSourcePlatformMonitor,
		"platform-monitor",
		"firing",
		map[string]string{"alertname": rule.Name},
		labels,
		IngressAlertDetail{
			Status:      "firing",
			Labels:      labels,
			Annotations: annotations,
			StartsAt:    now,
			EndsAt:      now.Add(24 * time.Hour),
			Fingerprint: fp,
		},
	))
}

func (s *AlertService) trackMonitorSeries(ctx context.Context, ruleID uint, fp string) {
	if s.redis == nil || strings.TrimSpace(fp) == "" {
		return
	}
	key := monitorEvalTrackedSetKey(ruleID)
	_ = s.redis.SAdd(ctx, key, fp).Err()
	_ = s.redis.Expire(ctx, key, 7*24*time.Hour).Err()
}

func (s *AlertService) untrackMonitorSeries(ctx context.Context, ruleID uint, fp string) {
	if s.redis == nil || strings.TrimSpace(fp) == "" {
		return
	}
	_ = s.redis.SRem(ctx, monitorEvalTrackedSetKey(ruleID), fp).Err()
}

func (s *AlertService) listTrackedMonitorSeries(ctx context.Context, ruleID uint) []string {
	if s.redis == nil {
		return nil
	}
	out, err := s.redis.SMembers(ctx, monitorEvalTrackedSetKey(ruleID)).Result()
	if err != nil {
		return nil
	}
	return out
}

func (s *AlertService) saveMonitorSeriesPayload(ctx context.Context, ruleID uint, fp string, labels, annotations map[string]string) {
	if s.redis == nil {
		return
	}
	lb, _ := json.Marshal(labels)
	ab, _ := json.Marshal(annotations)
	_ = s.redis.HSet(ctx, monitorEvalSeriesStateKey(ruleID, fp), map[string]any{
		"labels_json":      string(lb),
		"annotations_json": string(ab),
	}).Err()
}

func (s *AlertService) loadMonitorSeriesPayload(ctx context.Context, ruleID uint, fp string) (labels, annotations map[string]string) {
	labels = map[string]string{}
	annotations = map[string]string{}
	if s.redis == nil {
		return labels, annotations
	}
	h, err := s.redis.HGetAll(ctx, monitorEvalSeriesStateKey(ruleID, fp)).Result()
	if err != nil {
		return labels, annotations
	}
	if raw := strings.TrimSpace(h["labels_json"]); raw != "" {
		_ = json.Unmarshal([]byte(raw), &labels)
	}
	if raw := strings.TrimSpace(h["annotations_json"]); raw != "" {
		_ = json.Unmarshal([]byte(raw), &annotations)
	}
	return labels, annotations
}

func (s *AlertService) evaluateMonitorRuleWithRedis(ctx context.Context, rule *model.AlertMonitorRule, firing bool, labels map[string]string, annotations map[string]string, fp string, now time.Time) {
	if s.redis == nil {
		s.evaluateMonitorRuleNoRedis(ctx, rule, firing, labels, annotations, fp, now)
		return
	}
	key := monitorEvalSeriesStateKey(rule.ID, fp)
	h, err := s.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return
	}
	active := strings.TrimSpace(h["active_firing"]) == "1"
	pendingStr := strings.TrimSpace(h["pending_since"])
	pendingSince, _ := parseRFC3339Ptr(pendingStr)

	if firing {
		s.trackMonitorSeries(ctx, rule.ID, fp)
		s.saveMonitorSeriesPayload(ctx, rule.ID, fp, labels, annotations)
		if active {
			sev := strings.TrimSpace(labels["severity"])
			if sev == "" {
				sev = "warning"
			}
			if s.monitorShouldReingressFiring(ctx, fp, labels, rule.Name, sev) {
				s.emitMonitorPlatformFiring(ctx, rule, labels, annotations, fp, now)
			}
			_ = s.redis.Expire(ctx, key, 7*24*time.Hour).Err()
			return
		}
		if pendingSince == nil {
			if rule.ForSeconds <= 0 {
				_ = s.redis.HSet(ctx, key, map[string]any{
					"active_firing": "1",
					"pending_since": "",
				}).Err()
				_ = s.redis.Expire(ctx, key, 7*24*time.Hour).Err()
				s.emitMonitorPlatformFiring(ctx, rule, labels, annotations, fp, now)
				return
			}
			t := now.UTC()
			_ = s.redis.HSet(ctx, key, "pending_since", t.Format(time.RFC3339Nano)).Err()
			_ = s.redis.Expire(ctx, key, 7*24*time.Hour).Err()
			return
		}
		forDur := max(time.Duration(rule.ForSeconds)*time.Second, 0)
		if now.Sub(*pendingSince) >= forDur {
			_ = s.redis.HSet(ctx, key, map[string]any{
				"active_firing": "1",
				"pending_since": "",
			}).Err()
			_ = s.redis.Expire(ctx, key, 7*24*time.Hour).Err()
			s.emitMonitorPlatformFiring(ctx, rule, labels, annotations, fp, now)
			return
		}
		return
	}

	if active {
		_ = s.redis.HSet(ctx, key, map[string]any{
			"active_firing": "0",
			"pending_since": "",
		}).Err()
		_ = s.receiveCanonicalSync(ctx, NewCanonicalAlert(
			IngressSourcePlatformMonitor,
			"platform-monitor",
			"resolved",
			map[string]string{"alertname": rule.Name},
			labels,
			IngressAlertDetail{
				Status:      "resolved",
				Labels:      labels,
				Annotations: annotations,
				StartsAt:    now.Add(-time.Minute),
				EndsAt:      now,
				Fingerprint: fp,
			},
		))
	}
	_ = s.redis.HSet(ctx, key, "pending_since", "").Err()
	s.untrackMonitorSeries(ctx, rule.ID, fp)
	_ = s.redis.Del(ctx, key).Err()
}
