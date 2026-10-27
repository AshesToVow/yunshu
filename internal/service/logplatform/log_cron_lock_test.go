package logplatform

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newLogCronTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestAcquireLogCronLease(t *testing.T) {
	ctx := context.Background()
	rdb := newLogCronTestRedis(t)

	release, ok := acquireLogCronLease(ctx, rdb, "logs:test:lock", time.Minute)
	if !ok {
		t.Fatal("first acquire should succeed")
	}
	if _, ok := acquireLogCronLease(ctx, rdb, "logs:test:lock", time.Minute); ok {
		t.Fatal("second acquire should fail while lease held")
	}
	release()
	release() // 幂等

	releaseAgain, ok := acquireLogCronLease(ctx, rdb, "logs:test:lock", time.Minute)
	if !ok {
		t.Fatal("acquire after release should succeed")
	}
	releaseAgain()

	noopRelease, ok := acquireLogCronLease(ctx, nil, "logs:test:lock", time.Minute)
	if !ok || noopRelease == nil {
		t.Fatal("nil redis should degrade to always-held no-op lease")
	}
	noopRelease()
}

func TestRunLogCronTaskSkipsOverlapAndForeignLease(t *testing.T) {
	ctx := context.Background()
	rdb := newLogCronTestRedis(t)

	var running atomic.Bool
	started := make(chan struct{})
	unblock := make(chan struct{})
	done := make(chan bool, 1)
	go func() {
		done <- runLogCronTask(ctx, rdb, &running, "logs:test:task", time.Minute, func() {
			close(started)
			<-unblock
		})
	}()
	<-started

	if runLogCronTask(ctx, rdb, &running, "logs:test:task", time.Minute, func() {}) {
		t.Fatal("same-instance overlapping run should be skipped")
	}
	var otherInstance atomic.Bool
	if runLogCronTask(ctx, rdb, &otherInstance, "logs:test:task", time.Minute, func() {}) {
		t.Fatal("foreign replica should be skipped while lease held")
	}

	close(unblock)
	if !<-done {
		t.Fatal("first run should complete successfully")
	}

	var after atomic.Bool
	if !runLogCronTask(ctx, rdb, &after, "logs:test:task", time.Minute, func() {}) {
		t.Fatal("run after lease release should succeed")
	}
}
