package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RefreshTokenKey opaque refresh token → session payload.
func RefreshTokenKey(token string) string {
	return "auth:refresh_token:" + strings.TrimSpace(token)
}

// UserRefreshTokensKey SET of refresh token ids for a user (revoke-all).
func UserRefreshTokensKey(userID uint) string {
	return fmt.Sprintf("auth:user_refresh_tokens:%d", userID)
}

// RefreshSession Redis payload for a refresh token.
type RefreshSession struct {
	UserID        uint   `json:"user_id"`
	AccessTokenID string `json:"access_token_id"`
	FamilyID      string `json:"family_id"`
}

// SaveRefreshToken stores opaque refresh token and indexes it under the user.
func SaveRefreshToken(ctx context.Context, rdb *redis.Client, token string, sess RefreshSession, ttl time.Duration) error {
	if rdb == nil || strings.TrimSpace(token) == "" || sess.UserID == 0 || ttl <= 0 {
		return fmt.Errorf("invalid refresh token save args")
	}
	bs, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	key := RefreshTokenKey(token)
	pipe := rdb.Pipeline()
	pipe.Set(ctx, key, bs, ttl)
	ukey := UserRefreshTokensKey(sess.UserID)
	pipe.SAdd(ctx, ukey, strings.TrimSpace(token))
	pipe.Expire(ctx, ukey, ttl+time.Hour)
	_, err = pipe.Exec(ctx)
	return err
}

// ConsumeRefreshToken loads and deletes the refresh token (rotation).
func ConsumeRefreshToken(ctx context.Context, rdb *redis.Client, token string) (*RefreshSession, error) {
	if rdb == nil {
		return nil, ErrRedisRequired
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrSessionNotFound
	}
	key := RefreshTokenKey(token)
	bs, err := rdb.GetDel(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("%w: %w", ErrRedisUnavailable, err)
	}
	var sess RefreshSession
	if err := json.Unmarshal(bs, &sess); err != nil {
		return nil, err
	}
	if sess.UserID != 0 {
		_ = rdb.SRem(ctx, UserRefreshTokensKey(sess.UserID), token).Err()
	}
	return &sess, nil
}

// DeleteRefreshToken removes a refresh token without rotation.
func DeleteRefreshToken(ctx context.Context, rdb *redis.Client, token string) error {
	if rdb == nil || strings.TrimSpace(token) == "" {
		return nil
	}
	token = strings.TrimSpace(token)
	key := RefreshTokenKey(token)
	bs, err := rdb.Get(ctx, key).Bytes()
	if err == nil {
		var sess RefreshSession
		if json.Unmarshal(bs, &sess) == nil && sess.UserID != 0 {
			_ = rdb.SRem(ctx, UserRefreshTokensKey(sess.UserID), token).Err()
		}
	}
	return rdb.Del(ctx, key).Err()
}

// InvalidateAllUserRefreshTokens revokes every refresh token for a user.
func InvalidateAllUserRefreshTokens(ctx context.Context, rdb *redis.Client, userID uint) error {
	if rdb == nil || userID == 0 {
		return nil
	}
	setKey := UserRefreshTokensKey(userID)
	tokens, err := rdb.SMembers(ctx, setKey).Result()
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return rdb.Del(ctx, setKey).Err()
	}
	keys := make([]string, 0, len(tokens)+1)
	for _, t := range tokens {
		if t = strings.TrimSpace(t); t != "" {
			keys = append(keys, RefreshTokenKey(t))
		}
	}
	keys = append(keys, setKey)
	return rdb.Del(ctx, keys...).Err()
}

// ParseUintUserID parses redis string user id.
func ParseUintUserID(s string) (uint, error) {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}
