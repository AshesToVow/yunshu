package alert

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"

	"github.com/redis/go-redis/v9"
)

const (
	escalationQueueKey       = "alert:escalation:queue"
	escalationPendingPrefix  = "alert:escalation:pending:"
	escalationLevelPrefix    = "alert:escalation:level:"
	escalationSendLockPrefix = "alert:escalation:sendlock:"
)

// escalationPendingEnvelope 待升级投递信封（到期后按 TargetLevel 通知对应接收组）。
type escalationPendingEnvelope struct {
	Fingerprint  string            `json:"fingerprint"`
	TargetLevel  int               `json:"target_level"`
	GroupKey     string            `json:"group_key"`
	LabelsDigest string            `json:"labels_digest"`
	Source       string            `json:"source"`
	Title        string            `json:"title"`
	Severity     string            `json:"severity"`
	Status       string            `json:"status"`
	EnvLabel     string            `json:"env_label"`
	Labels       map[string]string `json:"labels"`
	Outgoing     map[string]any    `json:"outgoing"`
}

func escalationLevelKey(fingerprint string) string {
	return escalationLevelPrefix + strings.TrimSpace(fingerprint)
}

func escalationPendingKey(fingerprint string) string {
	return escalationPendingPrefix + strings.TrimSpace(fingerprint)
}

func escalationSendLockKey(fingerprint string) string {
	return escalationSendLockPrefix + strings.TrimSpace(fingerprint)
}

func (s *AlertService) currentEscalationLevel(ctx context.Context, fingerprint string) int {
	fp := strings.TrimSpace(fingerprint)
	if s == nil || s.redis == nil || fp == "" {
		return 0
	}
	raw, err := s.redis.Get(ctx, escalationLevelKey(fp)).Result()
	if err != nil || strings.TrimSpace(raw) == "" {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 0 {
		return 0
	}
	return normalizeEscalationLevel(n)
}

func (s *AlertService) setEscalationLevel(ctx context.Context, fingerprint string, level int) {
	fp := strings.TrimSpace(fingerprint)
	if s == nil || s.redis == nil || fp == "" {
		return
	}
	level = normalizeEscalationLevel(level)
	ttl := time.Duration(maxInt(s.cfg.AggregateTTLSeconds, 3600)) * time.Second
	if err := s.redis.Set(ctx, escalationLevelKey(fp), strconv.Itoa(level), ttl).Err(); err != nil {
		alertLog().Warn("set escalation level failed", "error", err, "fingerprint", fp, "level", level)
	}
}

func (s *AlertService) clearEscalationState(ctx context.Context, fingerprint string) {
	fp := strings.TrimSpace(fingerprint)
	if s == nil || s.redis == nil || fp == "" {
		return
	}
	pipe := s.redis.TxPipeline()
	pipe.Del(ctx, escalationLevelKey(fp))
	pipe.Del(ctx, escalationPendingKey(fp))
	pipe.ZRem(ctx, escalationQueueKey, fp)
	if _, err := pipe.Exec(ctx); err != nil {
		alertLog().Warn("clear escalation state failed", "error", err, "fingerprint", fp)
	}
}

// nextEscalationDelayAmongGroups 在已匹配接收组中取 targetLevel 的最短有效等待秒数。
func nextEscalationDelayAmongGroups(groups []*CachedReceiverGroup, targetLevel int) (delaySec int, ok bool) {
	if targetLevel <= 0 {
		return 0, false
	}
	minDelay := 0
	for _, g := range groups {
		if g == nil || !g.IsActiveNow() {
			continue
		}
		if normalizeEscalationLevel(g.EscalationLevel) != targetLevel {
			continue
		}
		d := effectiveEscalationDelaySeconds(targetLevel, g.EscalationDelaySeconds)
		if !ok || d < minDelay {
			minDelay = d
			ok = true
		}
	}
	return minDelay, ok
}

func (s *AlertService) matchedActiveReceiverGroups(ctx context.Context, status string, labels map[string]string) []*CachedReceiverGroup {
	ids := s.matchedReceiverGroupIDs(ctx, status, labels)
	out := make([]*CachedReceiverGroup, 0, len(ids))
	if s.receiverGroupCache == nil {
		return out
	}
	for _, gid := range ids {
		g, err := s.receiverGroupCache.Get(gid)
		if err != nil || g == nil {
			continue
		}
		out = append(out, g)
	}
	return out
}

func (s *AlertService) matchedReceiverGroupIDs(ctx context.Context, status string, labels map[string]string) []uint {
	if s.subscriptionSvc == nil {
		return nil
	}
	projectID := s.resolveProjectIDForAlertRouting(ctx, labels)
	severity := strings.TrimSpace(labels["severity"])
	var ids []uint
	tryMatch := func(pid uint) {
		route, ok := s.subscriptionSvc.MatchRouteDetailed(ctx, pid, labels, severity, status)
		if !ok || len(route.ReceiverGroupIDs) == 0 {
			return
		}
		ids = append(ids, route.ReceiverGroupIDs...)
	}
	if projectID > 0 {
		tryMatch(projectID)
	}
	tryMatch(0)
	return uniqUint(ids)
}

// maybeScheduleEscalation 在当前层级处理后，若存在更高层接收组则排队升级。
func (s *AlertService) maybeScheduleEscalation(ctx context.Context, env escalationPendingEnvelope, currentLevel int) {
	if s == nil || s.redis == nil {
		return
	}
	fp := strings.TrimSpace(env.Fingerprint)
	if fp == "" || !strings.EqualFold(strings.TrimSpace(env.Status), "firing") {
		return
	}
	nextLevel := normalizeEscalationLevel(currentLevel + 1)
	if nextLevel <= currentLevel {
		return
	}
	groups := s.matchedActiveReceiverGroups(ctx, env.Status, env.Labels)
	delaySec, ok := nextEscalationDelayAmongGroups(groups, nextLevel)
	if !ok {
		return
	}
	env.TargetLevel = nextLevel
	if env.Labels == nil {
		env.Labels = map[string]string{}
	}
	if env.Outgoing == nil {
		env.Outgoing = map[string]any{}
	}
	env.Labels = cloneStringMap(env.Labels)
	env.Outgoing = cloneInterfaceMap(env.Outgoing)
	raw, err := json.Marshal(env)
	if err != nil {
		alertLog().Warn("marshal escalation pending failed", "error", err, "fingerprint", fp)
		return
	}
	due := time.Now().UTC().Add(time.Duration(delaySec) * time.Second).Unix()
	ttl := time.Duration(maxInt(s.cfg.AggregateTTLSeconds, delaySec+3600)) * time.Second
	key := escalationPendingKey(fp)
	// 已有更早/同级待升级任务时不覆盖（避免每次重复 firing 重置倒计时）。
	existingDue, zerr := s.redis.ZScore(ctx, escalationQueueKey, fp).Result()
	if zerr == nil && int64(existingDue) > 0 && int64(existingDue) <= due {
		return
	}
	pipe := s.redis.TxPipeline()
	pipe.Set(ctx, key, raw, ttl)
	pipe.ZAdd(ctx, escalationQueueKey, redis.Z{Score: float64(due), Member: fp})
	if _, err := pipe.Exec(ctx); err != nil {
		alertLog().Warn("save escalation pending failed", "error", err, "fingerprint", fp, "level", nextLevel)
		return
	}
	alertLog().Info("escalation scheduled",
		"fingerprint", fp,
		"from_level", currentLevel,
		"target_level", nextLevel,
		"delay_seconds", delaySec,
	)
}

func (s *AlertService) loadEscalationPending(ctx context.Context, fingerprint string) *escalationPendingEnvelope {
	fp := strings.TrimSpace(fingerprint)
	if s == nil || s.redis == nil || fp == "" {
		return nil
	}
	raw, err := s.redis.Get(ctx, escalationPendingKey(fp)).Result()
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}
	var env escalationPendingEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		alertLog().Warn("unmarshal escalation pending failed", "error", err, "fingerprint", fp)
		return nil
	}
	if env.Labels == nil {
		env.Labels = map[string]string{}
	}
	if env.Outgoing == nil {
		env.Outgoing = map[string]any{}
	}
	return &env
}

func (s *AlertService) clearEscalationPendingOnly(ctx context.Context, fingerprint string) {
	fp := strings.TrimSpace(fingerprint)
	if s == nil || s.redis == nil || fp == "" {
		return
	}
	pipe := s.redis.TxPipeline()
	pipe.Del(ctx, escalationPendingKey(fp))
	pipe.ZRem(ctx, escalationQueueKey, fp)
	if _, err := pipe.Exec(ctx); err != nil {
		alertLog().Warn("clear escalation pending failed", "error", err, "fingerprint", fp)
	}
}

func (s *AlertService) listDueEscalationFingerprints(ctx context.Context, now time.Time) []string {
	out := []string{}
	if s == nil || s.redis == nil {
		return out
	}
	members, err := s.redis.ZRangeByScore(ctx, escalationQueueKey, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    strconv.FormatInt(now.Unix(), 10),
		Offset: 0,
		Count:  50,
	}).Result()
	if err != nil {
		alertLog().Warn("list due escalation keys failed", "error", err)
		return out
	}
	seen := map[string]struct{}{}
	for _, m := range members {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	return out
}

func (s *AlertService) tryLockEscalationSend(ctx context.Context, fingerprint string) bool {
	fp := strings.TrimSpace(fingerprint)
	if s == nil || s.redis == nil || fp == "" {
		return true
	}
	ok, err := s.redis.SetNX(ctx, escalationSendLockKey(fp), "1", 30*time.Second).Result()
	return err == nil && ok
}

func (s *AlertService) unlockEscalationSend(ctx context.Context, fingerprint string) {
	fp := strings.TrimSpace(fingerprint)
	if s == nil || s.redis == nil || fp == "" {
		return
	}
	if err := s.redis.Del(ctx, escalationSendLockKey(fp)).Err(); err != nil {
		alertLog().Warn("unlock escalation send failed", "error", err, "fingerprint", fp)
	}
}

func (s *AlertService) flushDueEscalations(ctx context.Context) {
	if s == nil || s.redis == nil {
		return
	}
	if !s.acquireTimingLeader(ctx) {
		return
	}
	host := s.ingressHost()
	channels, err := host.LoadEnabledChannels(ctx)
	if err != nil {
		alertLog().Warn("escalation load channels failed", "error", err)
		return
	}
	for _, fp := range s.listDueEscalationFingerprints(ctx, time.Now().UTC()) {
		env := s.loadEscalationPending(ctx, fp)
		if env == nil {
			s.clearEscalationPendingOnly(ctx, fp)
			continue
		}
		s.flushOneEscalation(ctx, host, channels, *env)
	}
}

func (s *AlertService) flushOneEscalation(
	ctx context.Context,
	host IngressHost,
	channels []model.AlertChannel,
	env escalationPendingEnvelope,
) {
	fp := strings.TrimSpace(env.Fingerprint)
	if fp == "" {
		return
	}
	if !s.tryLockEscalationSend(ctx, fp) {
		return
	}
	defer s.unlockEscalationSend(ctx, fp)

	if host.IsAckActive(ctx, fp) {
		s.clearEscalationState(ctx, fp)
		return
	}
	if !s.curEventStillFiring(ctx, fp) {
		s.clearEscalationState(ctx, fp)
		return
	}
	if sid, muted, err := host.FirstMatchingSilenceID(ctx, env.Labels, time.Now()); err == nil && muted {
		host.LogSilenceSuppressed(ctx, env.Title, env.Severity, env.Status, env.EnvLabel, env.GroupKey, env.LabelsDigest, sid, env.Outgoing)
		s.clearEscalationState(ctx, fp)
		return
	}

	target := normalizeEscalationLevel(env.TargetLevel)
	if target <= 0 {
		s.clearEscalationPendingOnly(ctx, fp)
		return
	}
	s.setEscalationLevel(ctx, fp, target)
	s.clearEscalationPendingOnly(ctx, fp)

	outgoing := env.Outgoing
	if outgoing == nil {
		outgoing = map[string]any{}
	}
	outgoing["escalation_flush"] = true
	outgoing["escalation_level"] = target

	route := host.ChannelRouteForAlert(ctx, env.Status, env.Labels, fp)
	host.ExpandChannelSetForAssigneeNotification(ctx, route.ChannelIDs, route.ReceiverGroupIDs, outgoing)
	outgoing["matchedPolicyIds"] = route.MatchedPolicyIDs
	outgoing["matchedPolicyNames"] = route.MatchedPolicyNames
	if len(route.ReceiverGroupEmails) > 0 {
		outgoing["receiver_group_emails"] = route.ReceiverGroupEmails
	}
	if len(route.ChannelIDs) == 0 {
		host.LogNoMatchedChannel(ctx, env.Title, env.Severity, env.Status, env.EnvLabel, env.GroupKey, env.LabelsDigest, outgoing, "no_policy_matched_escalation")
		// 本层无通道时仍尝试更高层
		s.maybeScheduleEscalation(ctx, env, target)
		return
	}
	gk := strings.TrimSpace(env.GroupKey)
	if gk != "" && !host.TryLockGroupSend(ctx, gk) {
		// 稍后重试：重新入队
		env.TargetLevel = target
		s.requeueEscalationSoon(ctx, env, 5)
		return
	}
	_, okDeliveries := deliverToChannels(
		ctx, host, channels, route.ChannelIDs,
		env.Source, env.Title, env.Severity, env.Status, env.Labels, outgoing,
	)
	if gk != "" {
		host.UnlockGroupSend(ctx, gk)
	}
	if okDeliveries > 0 {
		host.MarkFiringDelivered(ctx, fp)
		if gk != "" {
			host.CommitFiringGroupTimingSend(ctx, gk, env.LabelsDigest)
		}
	}
	s.maybeScheduleEscalation(ctx, env, target)
}

func (s *AlertService) requeueEscalationSoon(ctx context.Context, env escalationPendingEnvelope, afterSec int) {
	fp := strings.TrimSpace(env.Fingerprint)
	if s == nil || s.redis == nil || fp == "" {
		return
	}
	if afterSec <= 0 {
		afterSec = 5
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return
	}
	due := time.Now().UTC().Add(time.Duration(afterSec) * time.Second).Unix()
	ttl := time.Duration(maxInt(s.cfg.AggregateTTLSeconds, 3600)) * time.Second
	pipe := s.redis.TxPipeline()
	pipe.Set(ctx, escalationPendingKey(fp), raw, ttl)
	pipe.ZAdd(ctx, escalationQueueKey, redis.Z{Score: float64(due), Member: fp})
	if _, err := pipe.Exec(ctx); err != nil {
		alertLog().Warn("requeue escalation failed", "error", err, "fingerprint", fp)
	}
}
