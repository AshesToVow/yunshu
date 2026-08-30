package lifecycle

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestGroupWaitConverges(t *testing.T) {
	g := NewGroup()
	var done int32
	ctx, cancel := context.WithCancel(context.Background())

	g.Go("test.worker", func() {
		<-ctx.Done()
		atomic.AddInt32(&done, 1)
	})

	if g.Running() != 1 {
		t.Fatalf("Running() = %d, want 1", g.Running())
	}

	cancel()
	if ok := g.Wait(2 * time.Second); !ok {
		t.Fatalf("Wait() = false, want true (worker should exit after cancel)")
	}
	if atomic.LoadInt32(&done) != 1 {
		t.Fatalf("worker body did not run to completion")
	}
	if g.Running() != 0 {
		t.Fatalf("Running() = %d after Wait, want 0", g.Running())
	}
	if len(g.Pending()) != 0 {
		t.Fatalf("Pending() = %v after Wait, want empty", g.Pending())
	}
}

func TestGroupWaitTimeoutReportsPending(t *testing.T) {
	g := NewGroup()
	release := make(chan struct{})
	defer close(release)

	g.Go("stuck.worker", func() { <-release })

	if ok := g.Wait(100 * time.Millisecond); ok {
		t.Fatalf("Wait() = true, want false (worker is blocked)")
	}
	pending := g.Pending()
	if pending["stuck.worker"] != 1 {
		t.Fatalf("Pending() = %v, want stuck.worker=1", pending)
	}
}

// panic 不得阻塞 Wait：Group.Go 的 defer 中 wg.Done() 必须先于 recover 执行。
func TestGroupPanicDoesNotBlockWait(t *testing.T) {
	g := NewGroup()
	g.Go("panic.worker", func() { panic("boom") })

	if ok := g.Wait(2 * time.Second); !ok {
		t.Fatalf("Wait() = false, want true (panic must not leak wg counter)")
	}
	if g.Running() != 0 {
		t.Fatalf("Running() = %d after panic, want 0", g.Running())
	}
	if len(g.Pending()) != 0 {
		t.Fatalf("Pending() = %v after panic, want empty", g.Pending())
	}
}

func TestGroupGoNilIsSafe(t *testing.T) {
	g := NewGroup()
	g.Go("nil.worker", nil)
	if g.Running() != 0 {
		t.Fatalf("Running() = %d, want 0", g.Running())
	}
	if ok := g.Wait(time.Second); !ok {
		t.Fatalf("Wait() = false, want true for empty group")
	}
}

func TestGroupMultipleInstancesSameName(t *testing.T) {
	g := NewGroup()
	release := make(chan struct{})
	for range 3 {
		g.Go("pool.worker", func() { <-release })
	}
	if g.Running() != 3 {
		t.Fatalf("Running() = %d, want 3", g.Running())
	}
	if p := g.Pending(); p["pool.worker"] != 3 {
		t.Fatalf("Pending() = %v, want pool.worker=3", p)
	}
	close(release)
	if ok := g.Wait(2 * time.Second); !ok {
		t.Fatalf("Wait() = false, want true")
	}
}

func TestWaitDefaultTimeoutOnNonPositive(t *testing.T) {
	g := NewGroup()
	// 空组应立即返回，即使 timeout 非正也不应等待默认的 10s
	start := time.Now()
	if ok := g.Wait(0); !ok {
		t.Fatalf("Wait(0) = false, want true for empty group")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("Wait(0) on empty group took %s, want near-immediate return", elapsed)
	}
}

func TestPackageLevelDefaultGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	Go("pkg.worker", func() { <-ctx.Done() })
	if Running() < 1 {
		t.Fatalf("Running() = %d, want >= 1", Running())
	}
	if p := Pending(); p["pkg.worker"] != 1 {
		t.Fatalf("Pending() = %v, want pkg.worker=1", p)
	}
	cancel()
	if ok := Wait(2 * time.Second); !ok {
		t.Fatalf("Wait() = false, want true")
	}
}