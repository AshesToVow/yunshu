package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrMFAChallengeInvalid 表示 MFA 挑战不存在、已过期或已被消费。
var ErrMFAChallengeInvalid = errors.New("mfa challenge invalid or expired")

const (
	mfaChallengeKeyPrefix = "auth:mfa_challenge:"
	mfaAttemptsKeyPrefix  = "auth:mfa_challenge_attempts:"
)

// MFAChallengeKey 登录二次验证挑战缓存键。
func MFAChallengeKey(token string) string {
	return mfaChallengeKeyPrefix + strings.TrimSpace(token)
}

func mfaAttemptsKey(token string) string {
	return mfaAttemptsKeyPrefix + strings.TrimSpace(token)
}

// SaveMFAChallenge 在通过第一因子校验后暂存「已完成密码校验」的用户，等待 TOTP 校验。
// value 格式 userID[:source]，source 用于第二因子通过后写入登录日志。
func SaveMFAChallenge(ctx context.Context, rdb *redis.Client, token string, userID uint, source string, ttl time.Duration) error {
	if rdb == nil {
		return ErrRedisRequired
	}
	token = strings.TrimSpace(token)
	if token == "" || userID == 0 {
		return ErrMFAChallengeInvalid
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	payload := strconv.FormatUint(uint64(userID), 10)
	if source = strings.TrimSpace(source); source != "" {
		payload += ":" + source
	}
	return rdb.Set(ctx, MFAChallengeKey(token), payload, ttl).Err()
}

// GetMFAChallenge 读取挑战对应的用户 ID 与登录来源（挑战在成功校验后才删除，允许用户输错重试）。
func GetMFAChallenge(ctx context.Context, rdb *redis.Client, token string) (uint, string, error) {
	if rdb == nil {
		return 0, "", ErrRedisRequired
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, "", ErrMFAChallengeInvalid
	}
	raw, err := rdb.Get(ctx, MFAChallengeKey(token)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, "", ErrMFAChallengeInvalid
		}
		return 0, "", fmt.Errorf("%w: %w", ErrRedisUnavailable, err)
	}
	parts := strings.SplitN(raw, ":", 2)
	uid, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || uid == 0 {
		return 0, "", ErrMFAChallengeInvalid
	}
	source := ""
	if len(parts) == 2 {
		source = strings.TrimSpace(parts[1])
	}
	return uint(uid), source, nil
}

// ConsumeMFAChallenge 删除挑战（成功校验或失败次数超限后调用）。
func ConsumeMFAChallenge(ctx context.Context, rdb *redis.Client, token string) error {
	if rdb == nil {
		return ErrRedisRequired
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return rdb.Del(ctx, MFAChallengeKey(token), mfaAttemptsKey(token)).Err()
}

// IncrMFAChallengeAttempts 累计挑战的失败次数；返回是否已超限（超限时同步销毁挑战）。
func IncrMFAChallengeAttempts(ctx context.Context, rdb *redis.Client, token string, max int64) (bool, error) {
	if rdb == nil {
		return false, ErrRedisRequired
	}
	if max <= 0 {
		max = 5
	}
	key := mfaAttemptsKey(token)
	n, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrRedisUnavailable, err)
	}
	if n == 1 {
		rdb.Expire(ctx, key, 5*time.Minute)
	}
	if n >= max {
		_ = ConsumeMFAChallenge(ctx, rdb, token)
		return true, nil
	}
	return false, nil
}
