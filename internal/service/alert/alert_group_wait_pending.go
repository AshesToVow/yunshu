package alert

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	groupWaitQueueKey     = "alert:group:waitqueue"
	groupWaitPendingField = "pending"
	groupSendLockPrefix   = "alert:group:sendlock:"
	groupTimingLeaderKey  = "alert:timing:leader"
)

// groupWaitPendingEnvelope 首次同组等待期间缓存的待发信封，到期由 alert_timing 补发。
type groupWaitPendingEnvelope struct {
	GroupKey     string                 `json:"group_key"`
	Fingerprint  string                 `json:"fingerprint"`
	LabelsDigest string                 `json:"labels_digest"`
	Source       string                 `json:"source"`
	Title        string                 `json:"title"`
	Severity     string                 `json:"severity"`
	Status       string                 `json:"status"`
	EnvLabel     string                 `json:"env_label"`
	FirstSeen    string                 `json:"first_seen"`
	Labels       map[string]string      `json:"labels"`
	Outgoing     map[string]interface{} `json:"outgoing"`
}

func groupSendLockKey(groupKey string) string {
	return groupSendLockPrefix + strings.TrimSpace(groupKey)
}

func groupWaitDueUnix(firstSeen string, wait time.Duration, fallback time.Time) int64 {
	fst, err := time.Parse(time.RFC3339, strings.TrimSpace(firstSeen))
	if err != nil {
		fst = fallback
	}
	return fst.Add(wait).Unix()
}

func (s *AlertService) saveGroupWaitPending(ctx context.Context, env groupWaitPendingEnvelope, firstSeen string) {
	if s == nil || s.redis == nil {
		return
	}
	gk := strings.TrimSpace(env.GroupKey)
	if gk == "" {
		return
	}
	if env.Labels == nil {
		env.Labels = map[string]string{}
	}
	if env.Outgoing == nil {
		env.Outgoing = map[string]interface{}{}
	}
	env.FirstSeen = strings.TrimSpace(firstSeen)
	env.Labels = cloneStringMap(env.Labels)
	env.Outgoing = cloneInterfaceMap(env.Outgoing)
	raw, err := json.Marshal(env)
	if err != nil {
		alertLog().Warn("marshal group wait pending failed", "error", err, "group_key", gk)
		return
	}
	key := firingGroupTimingRedisKey(gk)
	wait := time.Duration(maxInt(0, s.cfg.GroupWaitSeconds)) * time.Second
	due := groupWaitDueUnix(firstSeen, wait, time.Now().UTC())
	pipe := s.redis.TxPipeline()
	pipe.HSet(ctx, key, groupWaitPendingField, raw)
	pipe.Expire(ctx, key, time.Duration(s.cfg.AggregateTTLSeconds)*time.Second)
	pipe.ZAdd(ctx, groupWaitQueueKey, redis.Z{Score: float64(due), Member: gk})
	if _, err := pipe.Exec(ctx); err != nil {
		alertLog().Warn("save group wait pending failed", "error", err, "group_key", gk)
	}
}

func (s *AlertService) loadGroupWaitPending(ctx context.Context, groupKey string) *groupWaitPendingEnvelope {
	if s == nil || s.redis == nil {
		return nil
	}
	gk := strings.TrimSpace(groupKey)
	if gk == "" {
		return nil
	}
	raw, err := s.redis.HGet(ctx, firingGroupTimingRedisKey(gk), groupWaitPendingField).Result()
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}
	var env groupWaitPendingEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		alertLog().Warn("unmarshal group wait pending failed", "error", err, "group_key", gk)
		return nil
	}
	if env.Labels == nil {
		env.Labels = map[string]string{}
	}
	if env.Outgoing == nil {
		env.Outgoing = map[string]interface{}{}
	}
	return &env
}

func (s *AlertService) clearGroupWaitPending(ctx context.Context, groupKey string) {
	if s == nil || s.redis == nil {
		return
	}
	gk := strings.TrimSpace(groupKey)
	if gk == "" {
		return
	}
	pipe := s.redis.TxPipeline()
	pipe.HDel(ctx, firingGroupTimingRedisKey(gk), groupWaitPendingField)
	pipe.ZRem(ctx, groupWaitQueueKey, gk)
	if _, err := pipe.Exec(ctx); err != nil {
		alertLog().Warn("clear group wait pending failed", "error", err, "group_key", gk)
	}
}

func (s *AlertService) listDueGroupWaitKeys(ctx context.Context, now time.Time) []string {
	out := []string{}
	if s == nil || s.redis == nil {
		return out
	}
	members, err := s.redis.ZRangeByScore(ctx, groupWaitQueueKey, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    strconv.FormatInt(now.Unix(), 10),
		Offset: 0,
		Count:  50,
	}).Result()
	if err != nil {
		alertLog().Warn("list due group wait keys failed", "error", err)
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

func cloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneInterfaceMap(in map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (s *AlertService) tryLockGroupSend(ctx context.Context, groupKey string) bool {
	gk := strings.TrimSpace(groupKey)
	if s == nil || s.redis == nil || gk == "" {
		return true
	}
	ok, err := s.redis.SetNX(ctx, groupSendLockKey(gk), "1", 30*time.Second).Result()
	return err == nil && ok
}

func (s *AlertService) unlockGroupSend(ctx context.Context, groupKey string) {
	if s == nil || s.redis == nil {
		return
	}
	gk := strings.TrimSpace(groupKey)
	if gk == "" {
		return
	}
	if err := s.redis.Del(ctx, groupSendLockKey(gk)).Err(); err != nil {
		alertLog().Warn("unlock group send failed", "error", err, "group_key", gk)
	}
}
