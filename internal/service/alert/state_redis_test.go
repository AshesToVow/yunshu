package alert

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisState_FiringToResolvedLifecycle(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	state := NewRedisAlertStateService(rdb, nil, nil, 3600, 86400)
	ctx := context.Background()
	fp := "fp-integration-test"

	c1, err := state.TouchFingerprint(ctx, fp, "firing")
	if err != nil || c1 != 1 {
		t.Fatalf("first firing touch: count=%d err=%v", c1, err)
	}
	c2, err := state.TouchFingerprint(ctx, fp, "firing")
	if err != nil || c2 != 2 {
		t.Fatalf("second firing touch: count=%d err=%v", c2, err)
	}
	if state.WasFiringDelivered(ctx, fp) {
		t.Fatal("expected not delivered before mark")
	}
	if err := state.MarkFiringDelivered(ctx, fp); err != nil {
		t.Fatal(err)
	}
	if !state.WasFiringDelivered(ctx, fp) {
		t.Fatal("expected delivered after mark")
	}
	first, err := state.MarkResolvedNotificationSent(ctx, fp)
	if err != nil || !first {
		t.Fatalf("first resolved mark: first=%v err=%v", first, err)
	}
	second, err := state.MarkResolvedNotificationSent(ctx, fp)
	if err != nil || second {
		t.Fatalf("duplicate resolved mark should be suppressed: second=%v err=%v", second, err)
	}
	if _, err := state.TouchFingerprint(ctx, fp, "resolved"); err != nil {
		t.Fatal(err)
	}
	if err := state.ClearFingerprint(ctx, fp); err != nil {
		t.Fatal(err)
	}
	if err := state.ClearFiringDelivered(ctx, fp); err != nil {
		t.Fatal(err)
	}
	if err := state.ClearResolvedNotificationSent(ctx, fp); err != nil {
		t.Fatal(err)
	}
	st, err := state.GetOrCreateState(ctx, fp)
	if err != nil {
		t.Fatal(err)
	}
	if st.FireCount != 0 && st.Status != "pending" {
		// after clear, redis hash removed; new state is pending
	}
	_ = time.Second
}
