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

var ErrWSTicketInvalid = errors.New("ws ticket invalid or expired")

const wsTicketKeyPrefix = "auth:ws_ticket:"

// WSTicketKey WebSocket 一次性握手票据缓存键。
func WSTicketKey(ticket string) string {
	return wsTicketKeyPrefix + strings.TrimSpace(ticket)
}

// SaveWSTicket 写入短效 WS 票据，value 格式：userID:tokenID[:scope]。
func SaveWSTicket(ctx context.Context, rdb *redis.Client, ticket string, userID uint, tokenID, scope string, ttlSeconds int) error {
	if rdb == nil {
		return ErrRedisRequired
	}
	ticket = strings.TrimSpace(ticket)
	tokenID = strings.TrimSpace(tokenID)
	scope = strings.TrimSpace(scope)
	if ticket == "" || userID == 0 || tokenID == "" {
		return ErrWSTicketInvalid
	}
	if ttlSeconds <= 0 {
		ttlSeconds = 30
	}
	payload := fmt.Sprintf("%d:%s", userID, tokenID)
	if scope != "" {
		payload += ":" + scope
	}
	return rdb.Set(ctx, WSTicketKey(ticket), payload, time.Duration(ttlSeconds)*time.Second).Err()
}

// ConsumeWSTicket 原子读取并删除票据（一次性）。
func ConsumeWSTicket(ctx context.Context, rdb *redis.Client, ticket string) (userID uint, tokenID string, err error) {
	if rdb == nil {
		return 0, "", ErrRedisRequired
	}
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return 0, "", ErrWSTicketInvalid
	}
	raw, err := rdb.GetDel(ctx, WSTicketKey(ticket)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, "", ErrWSTicketInvalid
		}
		return 0, "", fmt.Errorf("%w: %w", ErrRedisUnavailable, err)
	}
	parts := strings.SplitN(raw, ":", 3)
	if len(parts) < 2 {
		return 0, "", ErrWSTicketInvalid
	}
	uid, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || uid == 0 {
		return 0, "", ErrWSTicketInvalid
	}
	tokenID = strings.TrimSpace(parts[1])
	if tokenID == "" {
		return 0, "", ErrWSTicketInvalid
	}
	return uint(uid), tokenID, nil
}
