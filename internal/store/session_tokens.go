package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// UserTokensKey 用户活跃 access token 索引（SET 成员为 tokenID）。
func UserTokensKey(userID uint) string {
	return fmt.Sprintf("auth:user_tokens:%d", userID)
}

// RegisterUserAccessToken 记录用户新签发的 access token，便于按用户批量失效。
func RegisterUserAccessToken(ctx context.Context, rdb *redis.Client, userID uint, tokenID string, ttl time.Duration) error {
	if rdb == nil || userID == 0 || strings.TrimSpace(tokenID) == "" {
		return nil
	}
	key := UserTokensKey(userID)
	pipe := rdb.Pipeline()
	pipe.SAdd(ctx, key, strings.TrimSpace(tokenID))
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl+time.Hour)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// UnregisterUserAccessToken 登出时从用户索引移除 token。
func UnregisterUserAccessToken(ctx context.Context, rdb *redis.Client, userID uint, tokenID string) error {
	if rdb == nil || userID == 0 || strings.TrimSpace(tokenID) == "" {
		return nil
	}
	return rdb.SRem(ctx, UserTokensKey(userID), strings.TrimSpace(tokenID)).Err()
}

// InvalidateAllUserAccessTokens 使指定用户全部 access token 失效（改密等场景）。
func InvalidateAllUserAccessTokens(ctx context.Context, rdb *redis.Client, userID uint) error {
	if rdb == nil || userID == 0 {
		return nil
	}
	setKey := UserTokensKey(userID)
	tokenIDs, err := rdb.SMembers(ctx, setKey).Result()
	if err != nil {
		return err
	}
	if len(tokenIDs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(tokenIDs))
	for _, tid := range tokenIDs {
		if tid = strings.TrimSpace(tid); tid != "" {
			keys = append(keys, AccessTokenKey(tid))
		}
	}
	pipe := rdb.Pipeline()
	if len(keys) > 0 {
		pipe.Del(ctx, keys...)
	}
	pipe.Del(ctx, setKey)
	_, err = pipe.Exec(ctx)
	return err
}
