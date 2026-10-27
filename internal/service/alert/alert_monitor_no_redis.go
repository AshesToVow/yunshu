package alert

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// monitorNoRedisStateTTL 无 Redis 时进程内 firing 状态的最长保留时间：
// 规则被删除/停用后不会再评估，靠该 TTL 回收，避免 map 随历史规则无界增长。
const monitorNoRedisStateTTL = 7 * 24 * time.Hour

// evaluateMonitorRuleNoRedis 在无 Redis 时做简化状态机（进程内），避免监控规则完全不评估。
func (s *AlertService) evaluateMonitorRuleNoRedis(ctx context.Context, rule *model.AlertMonitorRule, firing bool, labels, annotations map[string]string, fp string, now time.Time) {
	s.monitorEvalMu.Lock()
	defer s.monitorEvalMu.Unlock()
	if s.monitorNoRedisActive == nil {
		s.monitorNoRedisActive = make(map[string]time.Time)
	}
	_, wasActive := s.monitorNoRedisActive[fp]

	if firing {
		s.monitorNoRedisActive[fp] = now
		if !wasActive {
			s.emitMonitorPlatformFiring(ctx, rule, labels, annotations, fp, now)
		}
		return
	}
	if wasActive {
		delete(s.monitorNoRedisActive, fp)
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
}

func (s *AlertService) pruneMonitorNoRedisState(now time.Time) {
	s.monitorEvalMu.Lock()
	defer s.monitorEvalMu.Unlock()
	for fp, last := range s.monitorNoRedisActive {
		if now.Sub(last) > monitorNoRedisStateTTL {
			delete(s.monitorNoRedisActive, fp)
		}
	}
}

func (s *AlertService) shouldEvalRuleNoRedis(ruleID uint, intervalSec int, now time.Time) bool {
	if intervalSec < 5 {
		intervalSec = 5
	}
	s.monitorEvalMu.Lock()
	defer s.monitorEvalMu.Unlock()
	if s.cloudExpiryNoRedisLastEval == nil {
		s.cloudExpiryNoRedisLastEval = make(map[uint]time.Time)
	}
	last, ok := s.cloudExpiryNoRedisLastEval[ruleID]
	if !ok || now.Sub(last) >= time.Duration(intervalSec)*time.Second {
		s.cloudExpiryNoRedisLastEval[ruleID] = now
		return true
	}
	return false
}
