package alert

import (
	"context"
	"testing"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/model"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestAlertServiceWithRedis(t *testing.T) (*AlertService, *miniredis.Miniredis) {
	t.Helper()
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
			GroupWaitSeconds:    15,
			AggregateTTLSeconds: 3600,
		},
	}
	return s, mr
}

func TestGroupWaitPending_saveDueAndClear(t *testing.T) {
	t.Parallel()
	s, _ := newTestAlertServiceWithRedis(t)
	ctx := context.Background()
	firstSeen := time.Now().UTC().Add(-20 * time.Second).Format(time.RFC3339)
	s.saveGroupWaitPending(ctx, groupWaitPendingEnvelope{
		GroupKey:     "gk-disk",
		Fingerprint:  "fp-1",
		LabelsDigest: "d1",
		Source:       "platform_monitor",
		Title:        "Disk",
		Severity:     "warning",
		Status:       "firing",
		Labels:       map[string]string{"alertname": "Disk"},
		Outgoing:     map[string]interface{}{"summary": "full"},
	}, firstSeen)

	got := s.loadGroupWaitPending(ctx, "gk-disk")
	if got == nil || got.Fingerprint != "fp-1" || got.Title != "Disk" {
		t.Fatalf("pending envelope: %+v", got)
	}
	due := s.listDueGroupWaitKeys(ctx, time.Now().UTC())
	if len(due) != 1 || due[0] != "gk-disk" {
		t.Fatalf("due keys: %v", due)
	}

	s.clearGroupWaitPending(ctx, "gk-disk")
	if s.loadGroupWaitPending(ctx, "gk-disk") != nil {
		t.Fatal("expected pending cleared")
	}
	if keys := s.listDueGroupWaitKeys(ctx, time.Now().UTC()); len(keys) != 0 {
		t.Fatalf("queue should be empty, got %v", keys)
	}
}

func TestGroupWaitPending_notDueYet(t *testing.T) {
	t.Parallel()
	s, _ := newTestAlertServiceWithRedis(t)
	ctx := context.Background()
	firstSeen := time.Now().UTC().Format(time.RFC3339)
	s.saveGroupWaitPending(ctx, groupWaitPendingEnvelope{
		GroupKey:    "gk-wait",
		Fingerprint: "fp-w",
		Title:       "Port",
		Status:      "firing",
		Labels:      map[string]string{},
		Outgoing:    map[string]interface{}{},
	}, firstSeen)
	if keys := s.listDueGroupWaitKeys(ctx, time.Now().UTC()); len(keys) != 0 {
		t.Fatalf("should not be due yet: %v", keys)
	}
}

func TestTryLockGroupSend(t *testing.T) {
	t.Parallel()
	s, _ := newTestAlertServiceWithRedis(t)
	ctx := context.Background()
	if !s.tryLockGroupSend(ctx, "gk-lock") {
		t.Fatal("first lock should succeed")
	}
	if s.tryLockGroupSend(ctx, "gk-lock") {
		t.Fatal("second lock should fail")
	}
	s.unlockGroupSend(ctx, "gk-lock")
	if !s.tryLockGroupSend(ctx, "gk-lock") {
		t.Fatal("lock after unlock should succeed")
	}
}

func TestAcquireTimingLeader_refreshSameToken(t *testing.T) {
	t.Parallel()
	s, _ := newTestAlertServiceWithRedis(t)
	s.timingLeaderToken = "tok-a"
	ctx := context.Background()
	if !s.acquireTimingLeader(ctx) {
		t.Fatal("first acquire should succeed")
	}
	if !s.acquireTimingLeader(ctx) {
		t.Fatal("same token refresh should succeed")
	}
	other := &AlertService{redis: s.redis, cfg: s.cfg, timingLeaderToken: "tok-b"}
	if other.acquireTimingLeader(ctx) {
		t.Fatal("other token should not steal leader")
	}
}

func TestGroupTimingAlreadySent(t *testing.T) {
	t.Parallel()
	s, _ := newTestAlertServiceWithRedis(t)
	ctx := context.Background()
	if s.groupTimingAlreadySent(ctx, "gk-sent") {
		t.Fatal("empty last_sent should be false")
	}
	if err := s.redis.HSet(ctx, firingGroupTimingRedisKey("gk-sent"), "last_sent", "2026-01-01T00:00:00Z").Err(); err != nil {
		t.Fatal(err)
	}
	if !s.groupTimingAlreadySent(ctx, "gk-sent") {
		t.Fatal("expected last_sent true")
	}
}

func TestFlushOneGroupWait_skipsWhenLastSent(t *testing.T) {
	t.Parallel()
	s, _ := newTestAlertServiceWithRedis(t)
	ctx := context.Background()
	s.saveGroupWaitPending(ctx, groupWaitPendingEnvelope{
		GroupKey:    "gk-sent",
		Fingerprint: "fp-sent",
		Title:       "Disk",
		Status:      "firing",
		Labels:      map[string]string{},
		Outgoing:    map[string]interface{}{},
	}, time.Now().UTC().Add(-30*time.Second).Format(time.RFC3339))
	if err := s.redis.HSet(ctx, firingGroupTimingRedisKey("gk-sent"), "last_sent", "2026-01-01T00:00:00Z").Err(); err != nil {
		t.Fatal(err)
	}
	host := &pipelineTestHost{
		peekShouldSend: true,
		deliverCode:    200,
		route: ChannelRoute{
			ChannelIDs: map[uint]struct{}{1: {}},
		},
		channels: []model.AlertChannel{{ID: 1, Name: "mail", Type: "email", Enabled: true}},
	}
	env := s.loadGroupWaitPending(ctx, "gk-sent")
	if env == nil {
		t.Fatal("expected pending envelope")
	}
	s.flushOneGroupWait(ctx, host, host.channels, *env)
	if s.loadGroupWaitPending(ctx, "gk-sent") != nil {
		t.Fatal("last_sent flush should clear pending")
	}
	if len(host.sends) != 0 {
		t.Fatalf("should not deliver when last_sent set, got %v", host.sends)
	}
}

func TestFlushOneGroupWait_skipsWhenAcked(t *testing.T) {
	t.Parallel()
	s, _ := newTestAlertServiceWithRedis(t)
	ctx := context.Background()
	host := &pipelineTestHost{peekShouldSend: true, deliverCode: 200}
	host.channels = nil
	s.saveGroupWaitPending(ctx, groupWaitPendingEnvelope{
		GroupKey:    "gk-ack",
		Fingerprint: "fp-ack",
		Title:       "Disk",
		Status:      "firing",
		Labels:      map[string]string{},
		Outgoing:    map[string]interface{}{},
	}, time.Now().UTC().Add(-30*time.Second).Format(time.RFC3339))
	ackedHost := &ackActiveHost{pipelineTestHost: *host, acked: true}
	s.flushOneGroupWait(ctx, ackedHost, nil, *s.loadGroupWaitPending(ctx, "gk-ack"))
	if s.loadGroupWaitPending(ctx, "gk-ack") == nil {
		t.Fatal("acked flush should keep pending")
	}
}

type ackActiveHost struct {
	pipelineTestHost
	acked bool
}

func (h *ackActiveHost) IsAckActive(context.Context, string) bool { return h.acked }

