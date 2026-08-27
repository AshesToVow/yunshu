package alert

import (
	"encoding/json"
	"testing"
	"time"

	"yunshu/internal/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestNextEscalationDelayAmongGroups(t *testing.T) {
	groups := []*CachedReceiverGroup{
		{EscalationLevel: 0, EscalationDelaySeconds: 0},
		{EscalationLevel: 1, EscalationDelaySeconds: 600},
		{EscalationLevel: 1, EscalationDelaySeconds: 1200},
		{EscalationLevel: 2, EscalationDelaySeconds: 60},
	}
	d, ok := nextEscalationDelayAmongGroups(groups, 1)
	if !ok || d != 600 {
		t.Fatalf("level1 delay got %d ok=%v want 600", d, ok)
	}
	d2, ok2 := nextEscalationDelayAmongGroups(groups, 2)
	if !ok2 || d2 != 60 {
		t.Fatalf("level2 delay got %d ok=%v want 60", d2, ok2)
	}
	if _, ok3 := nextEscalationDelayAmongGroups(groups, 3); ok3 {
		t.Fatal("expected no level 3")
	}
	d0, ok0 := nextEscalationDelayAmongGroups([]*CachedReceiverGroup{
		{EscalationLevel: 1, EscalationDelaySeconds: 0},
	}, 1)
	if !ok0 || d0 != defaultEscalationDelaySecs {
		t.Fatalf("default delay got %d ok=%v", d0, ok0)
	}
}

func TestEscalationScheduleAndClear(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	s := &AlertService{
		redis: rdb,
		cfg: config.AlertConfig{
			AggregateTTLSeconds: 3600,
		},
	}
	ctx := t.Context()
	fp := "fp-esc-1"

	s.setEscalationLevel(ctx, fp, 1)
	if got := s.currentEscalationLevel(ctx, fp); got != 1 {
		t.Fatalf("level=%d", got)
	}

	raw, _ := json.Marshal(escalationPendingEnvelope{
		Fingerprint: fp,
		TargetLevel: 1,
		Status:      "firing",
		Labels:      map[string]string{},
		Outgoing:    map[string]interface{}{},
	})
	due := time.Now().UTC().Add(-time.Second).Unix()
	_ = s.redis.Set(ctx, escalationPendingKey(fp), raw, time.Hour).Err()
	_ = s.redis.ZAdd(ctx, escalationQueueKey, redis.Z{Score: float64(due), Member: fp}).Err()
	if len(s.listDueEscalationFingerprints(ctx, time.Now().UTC())) != 1 {
		t.Fatal("expected due fingerprint")
	}

	s.clearEscalationState(ctx, fp)
	if got := s.currentEscalationLevel(ctx, fp); got != 0 {
		t.Fatalf("cleared level=%d", got)
	}
	if len(s.listDueEscalationFingerprints(ctx, time.Now().UTC())) != 0 {
		t.Fatal("queue should be cleared")
	}
}

func TestEffectiveEscalationDelaySeconds(t *testing.T) {
	if effectiveEscalationDelaySeconds(0, 100) != 0 {
		t.Fatal("level0 should ignore delay")
	}
	if effectiveEscalationDelaySeconds(1, 0) != defaultEscalationDelaySecs {
		t.Fatal("level1 zero config should default")
	}
	if effectiveEscalationDelaySeconds(1, 120) != 120 {
		t.Fatal("level1 custom delay")
	}
}
