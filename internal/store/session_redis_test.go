package store

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestValidateAccessTokenSession_NilClient(t *testing.T) {
	err := ValidateAccessTokenSession(context.Background(), nil, "tid")
	if !errors.Is(err, ErrRedisRequired) {
		t.Fatalf("want ErrRedisRequired, got %v", err)
	}
}

func TestValidateAccessTokenSession_EmptyTokenRequiresRedis(t *testing.T) {
	// Without a Redis client, validation fails before session lookup.
	err := ValidateAccessTokenSession(context.Background(), nil, "")
	if !errors.Is(err, ErrRedisRequired) {
		t.Fatalf("want ErrRedisRequired when client is nil, got %v", err)
	}
	_ = redis.Nil
}
