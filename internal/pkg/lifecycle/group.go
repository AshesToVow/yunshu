// Package lifecycle 提供后台 goroutine 的统一登记、panic 兜底与优雅收敛能力。
//
// 背景：进程原先仅在退出前 cancel 根 context 就立即返回，后台 worker
// （Kafka 消费、告警投递、备份、巡检等）可能在写库或投递中途被强制终止。
// 本包让这些 worker 登记到一个 WaitGroup，进程退出时可带超时等待其收敛。
//
// 用法：把 `go func() { ... }()` 换成 `lifecycle.Go("worker-name", func() { ... })`。
package lifecycle

import (
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// Group 一组后台 goroutine 的登记表。
type Group struct {
	wg      sync.WaitGroup
	running int64

	mu    sync.Mutex
	names map[string]int
}

// NewGroup 创建独立的 Group（测试或需要隔离时使用）。
func NewGroup() *Group {
	return &Group{names: make(map[string]int)}
}

// Go 启动一个受管 goroutine：自动登记到 WaitGroup，并对 panic 做兜底，
// 避免单个 worker 崩溃导致整个进程退出。
func (g *Group) Go(name string, fn func()) {
	if fn == nil {
		return
	}
	g.wg.Add(1)
	atomic.AddInt64(&g.running, 1)
	g.track(name, 1)

	go func() {
		defer func() {
			g.track(name, -1)
			atomic.AddInt64(&g.running, -1)
			g.wg.Done()
			if r := recover(); r != nil {
				slog.Error("background worker panicked",
					"worker", name,
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()
		fn()
	}()
}

// GoDetached 启动一个仅有 panic 兜底、不纳入 Wait 的 goroutine。
//
// 适用于任务级/请求级短生命周期协程（重试入队、租约续期、日志轮询、探测等）：
// 其父 context 已受控，纳入进程级 Wait 只会拖长关闭时间；但仍需 recover，
// 否则单个 panic 会直接终止整个进程。
func GoDetached(name string, fn func()) {
	if fn == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("detached goroutine panicked",
					"worker", name,
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()
		fn()
	}()
}

func (g *Group) track(name string, delta int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.names == nil {
		g.names = make(map[string]int)
	}
	g.names[name] += delta
	if g.names[name] <= 0 {
		delete(g.names, name)
	}
}

// Running 返回当前仍在运行的受管 goroutine 数量。
func (g *Group) Running() int { return int(atomic.LoadInt64(&g.running)) }

// Pending 返回当前仍未退出的 worker 名称及其实例数，用于超时后定位卡点。
func (g *Group) Pending() map[string]int {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make(map[string]int, len(g.names))
	for k, v := range g.names {
		out[k] = v
	}
	return out
}

// Wait 等待全部受管 goroutine 退出，最多等待 timeout。
// 返回 true 表示全部收敛；false 表示超时，此时 Pending 可给出卡住的 worker。
//
// 调用方必须先 cancel 传给 worker 的 context，否则必然超时。
func (g *Group) Wait(timeout time.Duration) bool {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	done := make(chan struct{})
	go func() {
		g.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// defaultGroup 进程级默认组。后台 worker 分散在各 service 包中，
// 使用包级函数可让调用点以最小改动接入，无需层层传递 Group。
var defaultGroup = NewGroup()

// Go 在进程级默认组中启动受管 goroutine。
func Go(name string, fn func()) { defaultGroup.Go(name, fn) }

// Running 进程级默认组中仍在运行的 goroutine 数量。
func Running() int { return defaultGroup.Running() }

// Pending 进程级默认组中未退出的 worker 明细。
func Pending() map[string]int { return defaultGroup.Pending() }

// Wait 等待进程级默认组收敛，语义同 (*Group).Wait。
func Wait(timeout time.Duration) bool { return defaultGroup.Wait(timeout) }

// WaitAndLog 等待默认组收敛并记录结果，供进程退出路径直接调用。
func WaitAndLog(logger *slog.Logger, timeout time.Duration) {
	if logger == nil {
		logger = slog.Default()
	}
	n := Running()
	if n == 0 {
		return
	}
	logger.Info("waiting for background workers to finish", "count", n, "timeout", timeout.String())
	start := time.Now()
	if Wait(timeout) {
		logger.Info("background workers stopped", "elapsed", time.Since(start).String())
		return
	}
	logger.Warn("background workers did not stop in time; forcing exit",
		"timeout", timeout.String(),
		"pending", Pending(),
	)
}
