package store

import (
	"context"
	"errors"
	"testing"
)

func TestPublishCasbinPolicyChangeRequiresRedis(t *testing.T) {
	t.Parallel()
	if err := PublishCasbinPolicyChange(context.Background(), nil, "unit"); !errors.Is(err, ErrRedisRequired) {
		t.Fatalf("err = %v, want ErrRedisRequired", err)
	}
}

func TestWatchCasbinPolicyChangesWithoutRedisIsNoop(t *testing.T) {
	t.Parallel()
	called := 0
	stop := WatchCasbinPolicyChanges(context.Background(), nil, nil, "unit", func() error {
		called++
		return nil
	})
	if stop == nil {
		t.Fatal("stop func must not be nil")
	}
	stop()
	stop() // 幂等：重复停止不应 panic
	if called != 0 {
		t.Fatalf("reload called %d times without redis", called)
	}
}
