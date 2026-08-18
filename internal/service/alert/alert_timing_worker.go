package alert

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"yunshu/internal/model"
)

func (s *AlertService) runAlertTimingWorker(ctx context.Context) {
	if s == nil {
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	s.flushDueGroupWaits(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.flushDueGroupWaits(ctx)
		}
	}
}

func (s *AlertService) acquireTimingLeader(ctx context.Context) bool {
	if s == nil || s.redis == nil {
		return true
	}
	token := strings.TrimSpace(s.timingLeaderToken)
	if token == "" {
		token = fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
		s.timingLeaderToken = token
	}
	ok, err := s.redis.SetNX(ctx, groupTimingLeaderKey, token, 8*time.Second).Result()
	if err != nil {
		alertLog().Warn("alert timing leader lock failed", "error", err)
		return false
	}
	if ok {
		return true
	}
	cur, getErr := s.redis.Get(ctx, groupTimingLeaderKey).Result()
	if getErr != nil {
		return false
	}
	if strings.TrimSpace(cur) != token {
		return false
	}
	if expErr := s.redis.Expire(ctx, groupTimingLeaderKey, 8*time.Second).Err(); expErr != nil {
		alertLog().Warn("alert timing leader refresh failed", "error", expErr)
		return false
	}
	return true
}

func (s *AlertService) flushDueGroupWaits(ctx context.Context) {
	if s == nil || s.redis == nil {
		return
	}
	if s.cfg.GroupWaitSeconds <= 0 {
		return
	}
	if !s.acquireTimingLeader(ctx) {
		return
	}
	host := s.ingressHost()
	channels, err := host.LoadEnabledChannels(ctx)
	if err != nil {
		alertLog().Warn("alert timing load channels failed", "error", err)
		return
	}
	for _, gk := range s.listDueGroupWaitKeys(ctx, time.Now().UTC()) {
		env := s.loadGroupWaitPending(ctx, gk)
		if env == nil {
			s.clearGroupWaitPending(ctx, gk)
			continue
		}
		s.flushOneGroupWait(ctx, host, channels, *env)
	}
}

func (s *AlertService) flushOneGroupWait(
	ctx context.Context,
	host IngressHost,
	channels []model.AlertChannel,
	env groupWaitPendingEnvelope,
) {
	gk := strings.TrimSpace(env.GroupKey)
	fp := strings.TrimSpace(env.Fingerprint)
	if gk == "" {
		return
	}
	if !s.tryLockGroupSend(ctx, gk) {
		return
	}
	defer s.unlockGroupSend(ctx, gk)

	if s.groupTimingAlreadySent(ctx, gk) {
		s.clearGroupWaitPending(ctx, gk)
		return
	}
	if fp != "" && host.IsAckActive(ctx, fp) {
		return
	}
	if fp != "" && !s.curEventStillFiring(ctx, fp) {
		s.clearGroupWaitPending(ctx, gk)
		return
	}
	if sid, muted, err := host.FirstMatchingSilenceID(ctx, env.Labels, time.Now()); err == nil && muted {
		host.LogSilenceSuppressed(ctx, env.Title, env.Severity, env.Status, env.EnvLabel, gk, env.LabelsDigest, sid, env.Outgoing)
		s.clearGroupWaitPending(ctx, gk)
		return
	}

	outgoing := env.Outgoing
	if outgoing == nil {
		outgoing = map[string]interface{}{}
	}
	outgoing["group_wait_flush"] = true
	route := host.ChannelRouteForAlert(ctx, env.Status, env.Labels)
	host.ExpandChannelSetForAssigneeNotification(ctx, route.ChannelIDs, route.ReceiverGroupIDs, outgoing)
	outgoing["matchedPolicyIds"] = route.MatchedPolicyIDs
	outgoing["matchedPolicyNames"] = route.MatchedPolicyNames
	if len(route.ChannelIDs) == 0 {
		host.LogNoMatchedChannel(ctx, env.Title, env.Severity, env.Status, env.EnvLabel, gk, env.LabelsDigest, outgoing, "no_policy_matched")
		s.clearGroupWaitPending(ctx, gk)
		return
	}
	if host.ShouldSuppressByRouteSilence(ctx, env.Status, gk, route.MatchedPolicyIDs, route.SilenceSeconds, env.Labels) {
		host.LogSuppressedRouteSilence(ctx, env.Title, env.Severity, env.Status, env.EnvLabel, gk, env.LabelsDigest, route.SilenceSeconds, outgoing)
		s.clearGroupWaitPending(ctx, gk)
		return
	}

	_, okDeliveries := deliverToChannels(
		ctx, host, channels, route.ChannelIDs,
		env.Source, env.Title, env.Severity, env.Status, env.Labels, outgoing,
	)
	if okDeliveries > 0 {
		host.CommitFiringGroupTimingSend(ctx, gk, env.LabelsDigest)
		if fp != "" {
			host.MarkFiringDelivered(ctx, fp)
		}
	}
	s.clearGroupWaitPending(ctx, gk)
}

func (s *AlertService) groupTimingAlreadySent(ctx context.Context, groupKey string) bool {
	if s == nil || s.redis == nil {
		return false
	}
	gk := strings.TrimSpace(groupKey)
	if gk == "" {
		return false
	}
	v, err := s.redis.HGet(ctx, firingGroupTimingRedisKey(gk), "last_sent").Result()
	if err != nil {
		return false
	}
	return strings.TrimSpace(v) != ""
}

func (s *AlertService) curEventStillFiring(ctx context.Context, fingerprint string) bool {
	fp := strings.TrimSpace(fingerprint)
	if s == nil || s.db == nil || fp == "" {
		return true
	}
	var n int64
	if err := s.db.WithContext(ctx).Model(&model.AlertCurEvent{}).Where("fingerprint = ?", fp).Count(&n).Error; err != nil {
		alertLog().Warn("cur event existence check failed", "error", err, "fingerprint", fp)
		return true
	}
	return n > 0
}
