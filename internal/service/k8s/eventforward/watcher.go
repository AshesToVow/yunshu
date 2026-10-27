package eventforward

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/service/k8s"

	"github.com/robfig/cron/v3"
	"github.com/weibaohui/kom/kom"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type Watcher struct {
	repo      interfaces.K8sEventForwardRepository
	runtime   *k8s.K8sRuntimeService
	cfg       RuntimeConfig
	eventCh   chan *model.K8sForwardedEvent
	ctx       context.Context
	cancel    context.CancelFunc
	activeMu  sync.Mutex
	active    map[string]context.CancelFunc // clusterID -> cancel watch
	startOnce sync.Once
}

func NewWatcher(repo interfaces.K8sEventForwardRepository, runtime *k8s.K8sRuntimeService, cfg RuntimeConfig) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	buf := cfg.WatcherBufferSize
	if buf <= 0 {
		buf = 1000
	}
	return &Watcher{
		repo:    repo,
		runtime: runtime,
		cfg:     cfg,
		eventCh: make(chan *model.K8sForwardedEvent, buf),
		ctx:     ctx,
		cancel:  cancel,
		active:  make(map[string]context.CancelFunc),
	}
}

func (w *Watcher) Start() {
	w.startOnce.Do(func() {
		go w.persistLoop()
		go w.scheduleLoop()
	})
}

func (w *Watcher) TriggerEnsure() {
	go w.ensureWatches()
}

func (w *Watcher) Stop() {
	w.StopAllClusters()
	if w.cancel != nil {
		w.cancel()
	}
}

// StopAllClusters 停止各集群 watch，保留 persist/schedule 循环，便于禁用后再次启用。
func (w *Watcher) StopAllClusters() {
	if w == nil {
		return
	}
	w.activeMu.Lock()
	defer w.activeMu.Unlock()
	for cid, cancel := range w.active {
		cancel()
		delete(w.active, cid)
	}
}

func (w *Watcher) scheduleLoop() {
	inst := cron.New()
	_, _ = inst.AddFunc("@every 1m", func() {
		w.ensureWatches()
	})
	inst.Start()
	<-w.ctx.Done()
	inst.Stop()
}

func (w *Watcher) ensureWatches() {
	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()

	ids, err := w.repo.ListEnabledClusterIDs(ctx)
	if err != nil {
		forwardLog().Warn("Failed to list K8s event forward clusters", "error", err)
		return
	}
	wanted := map[string]uint{}
	for _, id := range ids {
		cid := strconv.FormatUint(uint64(id), 10)
		wanted[cid] = id
	}

	w.activeMu.Lock()
	for cid, stop := range w.active {
		if _, ok := wanted[cid]; ok {
			continue
		}
		stop()
		delete(w.active, cid)
		forwardLog().Info("Stopped K8s event watch for removed cluster", "cluster_id", cid)
	}
	for cid, id := range wanted {
		if _, ok := w.active[cid]; ok {
			continue
		}
		clusterCtx, clusterCancel := context.WithCancel(w.ctx)
		w.active[cid] = clusterCancel
		go w.watchCluster(clusterCtx, cid, id)
	}
	w.activeMu.Unlock()
}

func (w *Watcher) watchCluster(ctx context.Context, clusterID string, id uint) {
	defer func() {
		w.activeMu.Lock()
		delete(w.active, clusterID)
		w.activeMu.Unlock()
	}()

	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		started := w.runClusterWatch(ctx, clusterID, id)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if started {
			// 连接建立成功过：短退避，避免断线后长时间漏事件。
			backoff = time.Second
		} else if backoff < 30*time.Second {
			backoff *= 2
		} else {
			backoff = 30 * time.Second
		}
	}
}

// runClusterWatch 启动一次事件 Watch；返回是否成功建立连接（用于重置退避）。
func (w *Watcher) runClusterWatch(ctx context.Context, clusterID string, id uint) bool {
	// EnsureClusterRegistered 返回已注册到稳定 bare ID 的 kubectl；
	// 勿再 kom.Cluster(clusterID)，控制台意图注册用的是 :r/:w 后缀，裸 ID 可能为 nil。
	kubectl, err := w.runtime.EnsureClusterRegistered(ctx, id)
	if err != nil {
		forwardLog().Warn("Failed to register cluster for event watch", "cluster_id", clusterID, "error", err)
		return false
	}
	if kubectl == nil {
		forwardLog().Warn("Failed to register cluster for event watch: kubectl is nil", "cluster_id", clusterID)
		return false
	}

	var watcher watch.Interface
	useCoreV1 := false
	if err := startEventWatch(ctx, kubectl, false, &watcher); err != nil {
		forwardLog().Warn("events.k8s.io/v1 watch failed, falling back to core/v1 Event",
			"cluster_id", clusterID, "error", err)
		if err2 := startEventWatch(ctx, kubectl, true, &watcher); err2 != nil {
			forwardLog().Warn("Failed to start K8s event watch", "cluster_id", clusterID, "error", err2)
			return false
		}
		useCoreV1 = true
	}
	defer watcher.Stop()

	api := "events.k8s.io/v1"
	if useCoreV1 {
		api = "core/v1"
	}
	forwardLog().Info("Started watching K8s events", "cluster_id", clusterID, "api", api)
	for {
		select {
		case <-ctx.Done():
			return true
		case e, ok := <-watcher.ResultChan():
			if !ok {
				forwardLog().Info("K8s event watch channel closed", "cluster_id", clusterID)
				return true
			}
			if e.Type == watch.Bookmark {
				continue
			}
			if e.Type == watch.Error {
				forwardLog().Warn("K8s event watch returned error", "cluster_id", clusterID, "api", api)
				return true
			}
			var m *model.K8sForwardedEvent
			if useCoreV1 {
				var typed corev1.Event
				if err := kubectl.WithContext(ctx).Tools().ConvertRuntimeObjectToTypedObject(e.Object, &typed); err != nil {
					forwardLog().Warn("Failed to convert core/v1 K8s event", "cluster_id", clusterID, "error", err)
					continue
				}
				m = w.fromCoreV1Event(clusterID, &typed)
			} else {
				var typed eventsv1.Event
				if err := kubectl.WithContext(ctx).Tools().ConvertRuntimeObjectToTypedObject(e.Object, &typed); err != nil {
					forwardLog().Warn("Failed to convert K8s event", "cluster_id", clusterID, "error", err)
					continue
				}
				m = w.fromK8sEvent(clusterID, &typed)
			}
			if m == nil || !m.ShouldForward() {
				continue
			}
			if err := w.enqueue(m); err != nil {
				forwardLog().Warn("Failed to enqueue K8s event", "evt_key", m.EvtKey, "error", err)
			}
		}
	}
}

// startEventWatch 先 List 取 resourceVersion，再从该版本开始 Watch：
// 避免 Watch 建立瞬间的事件缺口，并支持 Bookmark 心跳。
func startEventWatch(ctx context.Context, kube *kom.Kubectl, core bool, dest *watch.Interface) error {
	listOpts := metav1.ListOptions{Limit: 1}
	watchOpts := metav1.ListOptions{AllowWatchBookmarks: true}
	if core {
		var list corev1.EventList
		if err := kube.WithContext(ctx).Resource(&corev1.Event{}).AllNamespace().List(&list, listOpts).Error; err != nil {
			return err
		}
		watchOpts.ResourceVersion = list.ResourceVersion
		var evt corev1.Event
		return kube.WithContext(ctx).Resource(&evt).AllNamespace().Watch(dest, watchOpts).Error
	}
	var list eventsv1.EventList
	if err := kube.WithContext(ctx).Resource(&eventsv1.Event{}).AllNamespace().List(&list, listOpts).Error; err != nil {
		return err
	}
	watchOpts.ResourceVersion = list.ResourceVersion
	var evt eventsv1.Event
	return kube.WithContext(ctx).Resource(&evt).AllNamespace().Watch(dest, watchOpts).Error
}

func (w *Watcher) fromK8sEvent(clusterID string, evt *eventsv1.Event) *model.K8sForwardedEvent {
	ts := time.Now()
	if !evt.EventTime.IsZero() {
		ts = evt.EventTime.Time
	} else if !evt.CreationTimestamp.IsZero() {
		ts = evt.CreationTimestamp.Time
	}
	key := string(evt.UID)
	if key == "" {
		key = fmt.Sprintf("%s/%s/%s/%s/%d", clusterID, evt.Regarding.Namespace, evt.Regarding.Name, evt.Reason, ts.UnixNano())
	}
	return &model.K8sForwardedEvent{
		EvtKey:    key,
		ClusterID: clusterID,
		Namespace: evt.Regarding.Namespace,
		Name:      evt.Regarding.Name,
		Type:      evt.Type,
		Reason:    evt.Reason,
		Level:     evt.Type,
		Message:   evt.Note,
		Timestamp: ts,
		Processed: false,
		Status:    model.K8sFwdStatusPending,
	}
}

func (w *Watcher) fromCoreV1Event(clusterID string, evt *corev1.Event) *model.K8sForwardedEvent {
	ts := time.Now()
	if !evt.LastTimestamp.IsZero() {
		ts = evt.LastTimestamp.Time
	} else if !evt.EventTime.IsZero() {
		ts = evt.EventTime.Time
	} else if !evt.CreationTimestamp.IsZero() {
		ts = evt.CreationTimestamp.Time
	}
	key := string(evt.UID)
	if key == "" {
		key = fmt.Sprintf("%s/%s/%s/%s/%d", clusterID, evt.Namespace, evt.InvolvedObject.Name, evt.Reason, ts.UnixNano())
	}
	ns := evt.InvolvedObject.Namespace
	if ns == "" {
		ns = evt.Namespace
	}
	return &model.K8sForwardedEvent{
		EvtKey:    key,
		ClusterID: clusterID,
		Namespace: ns,
		Name:      evt.InvolvedObject.Name,
		Type:      evt.Type,
		Reason:    evt.Reason,
		Level:     evt.Type,
		Message:   evt.Message,
		Timestamp: ts,
		Processed: false,
		Status:    model.K8sFwdStatusPending,
	}
}

func (w *Watcher) enqueue(ev *model.K8sForwardedEvent) error {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-w.ctx.Done():
		return fmt.Errorf("watcher stopped")
	case w.eventCh <- ev:
		return nil
	case <-timer.C:
		return fmt.Errorf("enqueue timeout: buffer full")
	}
}

func (w *Watcher) persistLoop() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case ev, ok := <-w.eventCh:
			if !ok {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := w.repo.SaveForwardedEvent(ctx, ev)
			cancel()
			if err != nil {
				forwardLog().Warn("Failed to save K8s forwarded event", "error", err)
			}
		}
	}
}
