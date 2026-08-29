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

// RefreshFamilyKey SET of active refresh tokens in a rotation family.
func RefreshFamilyKey(familyID string) string {
	return "auth:refresh_family:" + strings.TrimSpace(familyID)
}

// RefreshSpentKey marks a consumed refresh token for reuse detection (value = spent payload).
func RefreshSpentKey(token string) string {
	return "auth:refresh_spent:" + strings.TrimSpace(token)
}

// ErrRefreshTokenReuse 表示已轮换过的 refresh 再次出现（疑似盗用）。
var ErrRefreshTokenReuse = errors.New("refresh token reuse detected")

// RefreshReuseError 携带被重放 token 所属 family，便于整链吊销。
type RefreshReuseError struct {
	FamilyID string
	UserID   uint
}

func (e *RefreshReuseError) Error() string { return ErrRefreshTokenReuse.Error() }
func (e *RefreshReuseError) Unwrap() error { return ErrRefreshTokenReuse }

// RefreshSession Redis payload for a refresh token.
type RefreshSession struct {
	UserID        uint   `json:"user_id"`
	AccessTokenID string `json:"access_token_id"`
	FamilyID      string `json:"family_id"`
}

type refreshSpentMeta struct {
	FamilyID string `json:"family_id"`
	UserID   uint   `json:"user_id"`
}

// SaveRefreshToken stores opaque refresh token and indexes it under the user + family.
func SaveRefreshToken(ctx context.Context, rdb *redis.Client, token string, sess RefreshSession, ttl time.Duration) error {
	if rdb == nil || strings.TrimSpace(token) == "" || sess.UserID == 0 || ttl <= 0 {
		return fmt.Errorf("invalid refresh token save args")
	}
	token = strings.TrimSpace(token)
	familyID := strings.TrimSpace(sess.FamilyID)
	if familyID == "" {
		return fmt.Errorf("invalid refresh token save args: empty family_id")
	}
	bs, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	key := RefreshTokenKey(token)
	pipe := rdb.Pipeline()
	pipe.Set(ctx, key, bs, ttl)
	ukey := UserRefreshTokensKey(sess.UserID)
	pipe.SAdd(ctx, ukey, token)
	pipe.Expire(ctx, ukey, ttl+time.Hour)
	fkey := RefreshFamilyKey(familyID)
	pipe.SAdd(ctx, fkey, token)
	pipe.Expire(ctx, fkey, ttl+time.Hour)
	_, err = pipe.Exec(ctx)
	return err
}

// ConsumeRefreshToken loads and deletes the refresh token (rotation).
// 若 token 已被消费过（spent 标记存在），返回 *RefreshReuseError。
func ConsumeRefreshToken(ctx context.Context, rdb *redis.Client, token string) (*RefreshSession, error) {
	if rdb == nil {
		return nil, ErrRedisRequired
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrSessionNotFound
	}
	key := RefreshTokenKey(token)
	ttl, _ := rdb.TTL(ctx, key).Result()
	bs, err := rdb.GetDel(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			if reuse := lookupRefreshSpent(ctx, rdb, token); reuse != nil {
				return nil, reuse
			}
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("%w: %w", ErrRedisUnavailable, err)
	}
	var sess RefreshSession
	if err := json.Unmarshal(bs, &sess); err != nil {
		return nil, err
	}
	spentTTL := ttl
	if spentTTL <= 0 {
		spentTTL = 7 * 24 * time.Hour
	}
	_ = markRefreshSpent(ctx, rdb, token, sess, spentTTL)

	pipe := rdb.Pipeline()
	if sess.UserID != 0 {
		pipe.SRem(ctx, UserRefreshTokensKey(sess.UserID), token)
	}
	if fam := strings.TrimSpace(sess.FamilyID); fam != "" {
		pipe.SRem(ctx, RefreshFamilyKey(fam), token)
	}
	_, _ = pipe.Exec(ctx)
	return &sess, nil
}

func markRefreshSpent(ctx context.Context, rdb *redis.Client, token string, sess RefreshSession, ttl time.Duration) error {
	meta := refreshSpentMeta{FamilyID: strings.TrimSpace(sess.FamilyID), UserID: sess.UserID}
	if meta.FamilyID == "" {
		return nil
	}
	bs, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, RefreshSpentKey(token), bs, ttl).Err()
}

func lookupRefreshSpent(ctx context.Context, rdb *redis.Client, token string) *RefreshReuseError {
	bs, err := rdb.Get(ctx, RefreshSpentKey(token)).Bytes()
	if err != nil {
		return nil
	}
	var meta refreshSpentMeta
	if json.Unmarshal(bs, &meta) != nil || strings.TrimSpace(meta.FamilyID) == "" {
		return nil
	}
	return &RefreshReuseError{FamilyID: strings.TrimSpace(meta.FamilyID), UserID: meta.UserID}
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
		if json.Unmarshal(bs, &sess) == nil {
			pipe := rdb.Pipeline()
			if sess.UserID != 0 {
				pipe.SRem(ctx, UserRefreshTokensKey(sess.UserID), token)
			}
			if fam := strings.TrimSpace(sess.FamilyID); fam != "" {
				pipe.SRem(ctx, RefreshFamilyKey(fam), token)
			}
			pipe.Del(ctx, key)
			_, _ = pipe.Exec(ctx)
			return nil
		}
	}
	return rdb.Del(ctx, key).Err()
}

// InvalidateRefreshFamily 吊销同一轮换链上全部 refresh，并失效其关联 access。
func InvalidateRefreshFamily(ctx context.Context, rdb *redis.Client, familyID string) error {
	if rdb == nil {
		return nil
	}
	familyID = strings.TrimSpace(familyID)
	if familyID == "" {
		return nil
	}
	fkey := RefreshFamilyKey(familyID)
	tokens, err := rdb.SMembers(ctx, fkey).Result()
	if err != nil {
		return err
	}
	accessIDs := make([]string, 0, len(tokens))
	userIDs := map[uint]struct{}{}
	pipe := rdb.Pipeline()
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		rkey := RefreshTokenKey(t)
		if bs, gerr := rdb.Get(ctx, rkey).Bytes(); gerr == nil {
			var sess RefreshSession
			if json.Unmarshal(bs, &sess) == nil {
				if aid := strings.TrimSpace(sess.AccessTokenID); aid != "" {
					accessIDs = append(accessIDs, aid)
				}
				if sess.UserID != 0 {
					userIDs[sess.UserID] = struct{}{}
					pipe.SRem(ctx, UserRefreshTokensKey(sess.UserID), t)
				}
			}
		}
		pipe.Del(ctx, rkey)
	}
	pipe.Del(ctx, fkey)
	for _, aid := range accessIDs {
		pipe.Del(ctx, AccessTokenKey(aid))
	}
	for uid := range userIDs {
		for _, aid := range accessIDs {
			pipe.SRem(ctx, UserTokensKey(uid), aid)
		}
	}
	_, err = pipe.Exec(ctx)
	return err
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
	families := map[string]struct{}{}
	keys := make([]string, 0, len(tokens)+1)
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		keys = append(keys, RefreshTokenKey(t))
		if bs, gerr := rdb.Get(ctx, RefreshTokenKey(t)).Bytes(); gerr == nil {
			var sess RefreshSession
			if json.Unmarshal(bs, &sess) == nil {
				if fam := strings.TrimSpace(sess.FamilyID); fam != "" {
					families[fam] = struct{}{}
				}
			}
		}
	}
	keys = append(keys, setKey)
	for fam := range families {
		keys = append(keys, RefreshFamilyKey(fam))
	}
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
