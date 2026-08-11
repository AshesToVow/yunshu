package alert

import (
	"context"
	stderrors "errors"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"yunshu/internal/interfaces"
	bizerrors "yunshu/internal/pkg/errors"
)

// redisStateStore is the authoritative webhook/runtime aggregate state (Redis + DB fallback for firing_delivered).
type redisStateStore struct {
	redis              *redis.Client
	eventRepo          interfaces.AlertEventRepository
	firingDeliveryRepo interfaces.AlertFiringDeliveryRepository
	dedupTTL           time.Duration
	firingDeliveredTTL time.Duration
}

// NewRedisAlertStateService creates a Redis-backed AlertStateService.
// firingDeliveryRepo 可为 nil；非 nil 时 Mark/Was/ClearFiringDelivered 会双写/降级读库。
func NewRedisAlertStateService(
	redisClient *redis.Client,
	eventRepo interfaces.AlertEventRepository,
	firingDeliveryRepo interfaces.AlertFiringDeliveryRepository,
	dedupTTLSeconds int,
	aggregateTTLSeconds int,
) AlertStateService {
	dedup := time.Duration(dedupTTLSeconds) * time.Second
	if dedup <= 0 {
		dedup = 24 * time.Hour
	}
	agg := time.Duration(aggregateTTLSeconds) * time.Second
	if agg <= 0 {
		agg = 7 * 24 * time.Hour
	}
	return &redisStateStore{
		redis:              redisClient,
		eventRepo:          eventRepo,
		firingDeliveryRepo: firingDeliveryRepo,
		dedupTTL:           dedup,
		firingDeliveredTTL: agg,
	}
}

func (s *redisStateStore) redisOK() bool {
	return s != nil && s.redis != nil
}

func (s *redisStateStore) GetOrCreateState(ctx context.Context, fingerprint string) (*AggregateState, error) {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return &AggregateState{Fingerprint: fp, Status: "pending", FirstFiredAt: time.Now()}, nil
	}
	if !s.redisOK() {
		return &AggregateState{Fingerprint: fp, Status: "pending", FirstFiredAt: time.Now()}, nil
	}
	key := fingerprintRedisKey(fp)
	countStr, _ := s.redis.HGet(ctx, key, "count").Result()
	count, _ := strconv.ParseInt(countStr, 10, 64)
	status, _ := s.redis.HGet(ctx, key, "last_status").Result()
	if status == "" {
		status = "pending"
	}
	firstRaw, _ := s.redis.HGet(ctx, key, "first_fired_at").Result()
	first := time.Now()
	if firstRaw != "" {
		if t, err := time.Parse(time.RFC3339, firstRaw); err == nil {
			first = t
		}
	}
	return &AggregateState{
		Fingerprint:  fp,
		Status:       status,
		FirstFiredAt: first,
		FireCount:    count,
	}, nil
}

func (s *redisStateStore) UpdateStatus(ctx context.Context, fingerprint string, newStatus string) error {
	_, err := s.TouchFingerprint(ctx, fingerprint, newStatus)
	if err != nil {
		return err
	}
	if s.eventRepo == nil {
		return nil
	}
	if err := s.eventRepo.UpdateStatus(ctx, fingerprint, newStatus); err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return bizerrors.Pass(ctx, "alert.state", "UpdateStatus", err)
	}
	return nil
}

func (s *redisStateStore) TouchFingerprint(ctx context.Context, fingerprint, status string) (int64, error) {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" || !s.redisOK() {
		return 1, nil
	}
	key := fingerprintRedisKey(fp)
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		status = "firing"
	}
	if strings.EqualFold(status, "firing") {
		count, err := s.redis.HIncrBy(ctx, key, "count", 1).Result()
		if err != nil {
			return 1, bizerrors.Pass(ctx, "alert.state", "TouchFingerprint", err)
		}
		now := time.Now().UTC().Format(time.RFC3339)
		pipe := s.redis.Pipeline()
		pipe.HSet(ctx, key, "last_status", "firing")
		pipe.HSetNX(ctx, key, "first_fired_at", now)
		pipe.Expire(ctx, key, s.dedupTTL)
		if _, err := pipe.Exec(ctx); err != nil {
			return count, bizerrors.Pass(ctx, "alert.state", "TouchFingerprint", err)
		}
		return count, nil
	}
	v, err := s.redis.HGet(ctx, key, "count").Result()
	if err == redis.Nil {
		return 1, nil
	}
	if err != nil {
		return 1, nil
	}
	n, _ := strconv.ParseInt(v, 10, 64)
	if n <= 0 {
		n = 1
	}
	_ = s.redis.HSet(ctx, key, "last_status", status).Err()
	_ = s.redis.Expire(ctx, key, s.dedupTTL).Err()
	return n, nil
}

func (s *redisStateStore) ClearFingerprint(ctx context.Context, fingerprint string) error {
	if !s.redisOK() || strings.TrimSpace(fingerprint) == "" {
		return nil
	}
	return s.redis.Del(ctx, fingerprintRedisKey(fingerprint)).Err()
}

func (s *redisStateStore) IsDuplicate(ctx context.Context, fingerprint string) (bool, error) {
	st, err := s.GetOrCreateState(ctx, fingerprint)
	if err != nil {
		return false, err
	}
	return st.Status == "firing" || st.Status == "resolved", nil
}

func (s *redisStateStore) CleanupExpiredStates(ctx context.Context, ttl time.Duration) (int64, error) {
	if !s.redisOK() {
		return 0, nil
	}
	_ = ttl // 指纹键本身带 Expire；此处清理无 TTL 的残留键
	var deleted int64
	for _, pattern := range []string{
		"alert:fingerprint:*",
		"alert:resolved:sent:*",
		"alert:firing_delivered:*",
		"alert:current:*",
	} {
		n, err := s.scanDeleteNoTTL(ctx, pattern)
		if err != nil {
			return deleted, err
		}
		deleted += n
	}
	return deleted, nil
}

func (s *redisStateStore) scanDeleteNoTTL(ctx context.Context, pattern string) (int64, error) {
	var deleted int64
	var cursor uint64
	for {
		keys, next, err := s.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return deleted, bizerrors.Pass(ctx, "alert.state", "CleanupExpiredStates", err)
		}
		for _, key := range keys {
			t, err := s.redis.TTL(ctx, key).Result()
			if err != nil {
				continue
			}
			// 已设 TTL 的交给 Redis 过期；仅删无 TTL 的残留
			if t >= 0 {
				continue
			}
			if n, err := s.redis.Del(ctx, key).Result(); err == nil {
				deleted += n
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return deleted, nil
}

func (s *redisStateStore) MarkResolvedNotificationSent(ctx context.Context, fingerprint string) (bool, error) {
	if !s.redisOK() || strings.TrimSpace(fingerprint) == "" {
		// Redis 不可用：返回 false 允许调用方重试发送，避免「当已发送」导致漏通知
		return false, nil
	}
	ok, err := s.redis.SetNX(ctx, resolvedSentRedisKey(fingerprint), "1", s.dedupTTL).Result()
	if err != nil {
		return false, bizerrors.Pass(ctx, "alert.state", "MarkResolvedNotificationSent", err)
	}
	return ok, nil
}

func (s *redisStateStore) ClearResolvedNotificationSent(ctx context.Context, fingerprint string) error {
	if !s.redisOK() || strings.TrimSpace(fingerprint) == "" {
		return nil
	}
	return s.redis.Del(ctx, resolvedSentRedisKey(fingerprint)).Err()
}

func (s *redisStateStore) MarkFiringDelivered(ctx context.Context, fingerprint string) error {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return nil
	}
	if s.firingDeliveryRepo != nil {
		if err := s.firingDeliveryRepo.Mark(ctx, fp); err != nil {
			return bizerrors.Pass(ctx, "alert.state", "MarkFiringDelivered", err)
		}
	}
	if s.redisOK() {
		if err := s.redis.Set(ctx, firingDeliveredRedisKey(fp), "1", s.firingDeliveredTTL).Err(); err != nil {
			// DB 已写入时 Redis 失败不阻断：恢复判定可降级读库
			return nil
		}
	}
	return nil
}

func (s *redisStateStore) WasFiringDelivered(ctx context.Context, fingerprint string) bool {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return false
	}
	if s.redisOK() {
		v, err := s.redis.Get(ctx, firingDeliveredRedisKey(fp)).Result()
		if err == nil && strings.TrimSpace(v) == "1" {
			return true
		}
	}
	if s.firingDeliveryRepo != nil {
		ok, _ := s.firingDeliveryRepo.Exists(ctx, fp)
		return ok
	}
	return false
}

func (s *redisStateStore) ClearFiringDelivered(ctx context.Context, fingerprint string) error {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return nil
	}
	if s.redisOK() {
		_ = s.redis.Del(ctx, firingDeliveredRedisKey(fp)).Err()
	}
	if s.firingDeliveryRepo != nil {
		return s.firingDeliveryRepo.Delete(ctx, fp)
	}
	return nil
}

// ClearCurrentMetric removes cached Prometheus current value for a fingerprint.
func (s *redisStateStore) ClearCurrentMetric(ctx context.Context, fingerprint string) error {
	if !s.redisOK() || strings.TrimSpace(fingerprint) == "" {
		return nil
	}
	return s.redis.Del(ctx, currentMetricRedisKey(fingerprint)).Err()
}

var _ AlertStateService = (*redisStateStore)(nil)
