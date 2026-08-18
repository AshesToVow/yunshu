package eventforward

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
)

const unmatchedEventMaxAge = 24 * time.Hour

type Worker struct {
	repo          interfaces.K8sEventForwardRepository
	client        *WebhookClient
	cfg           RuntimeConfig
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	interval      time.Duration
	batch         int
	maxRetry      int
	onBeforeBatch func()
	isEnabled     func() bool
	startOnce     sync.Once
}

func NewWorker(repo interfaces.K8sEventForwardRepository, client *WebhookClient, cfg RuntimeConfig) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		repo:     repo,
		client:   client,
		cfg:      cfg,
		ctx:      ctx,
		cancel:   cancel,
		interval: time.Duration(cfg.WorkerIntervalSeconds) * time.Second,
		batch:    cfg.WorkerBatchSize,
		maxRetry: cfg.WorkerMaxRetries,
	}
}

func (w *Worker) Start() {
	w.startOnce.Do(func() {
		w.wg.Add(1)
		go w.loop()
	})
}

func (w *Worker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
}

func (w *Worker) RefreshSettings(cfg RuntimeConfig) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.cfg = cfg
	if cfg.WorkerIntervalSeconds > 0 {
		w.interval = time.Duration(cfg.WorkerIntervalSeconds) * time.Second
	}
	if cfg.WorkerBatchSize > 0 {
		w.batch = cfg.WorkerBatchSize
	}
	if cfg.WorkerMaxRetries > 0 {
		w.maxRetry = cfg.WorkerMaxRetries
	}
}

func (w *Worker) currentSettings() (interval time.Duration, batch, maxRetry int, cfg RuntimeConfig) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	interval = w.interval
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return interval, w.batch, w.maxRetry, w.cfg
}

func (w *Worker) loop() {
	defer w.wg.Done()
	interval, _, _, _ := w.currentSettings()
	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-timer.C:
			if err := w.processBatch(); err != nil {
				forwardLog().Warn("Failed to process K8s event forward batch", "error", err)
			}
			next, _, _, _ := w.currentSettings()
			timer.Reset(next)
		}
	}
}

func (w *Worker) processBatch() error {
	if w.onBeforeBatch != nil {
		w.onBeforeBatch()
	}
	if w.isEnabled != nil && !w.isEnabled() {
		return nil
	}
	_, batch, maxRetry, cfg := w.currentSettings()
	ctx, cancel := context.WithTimeout(w.ctx, 2*time.Minute)
	defer cancel()

	rules, err := w.repo.ListEnabledRules(ctx)
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}

	events, err := w.repo.ClaimUnprocessedEvents(ctx, batch)
	if err != nil || len(events) == 0 {
		return err
	}

	processedIDs := make(map[int64]bool)
	matchedIDs := make(map[int64]bool)

	for _, rule := range rules {
		clusters := ParseClusterIDSet(rule.ClusterIDs)
		filter := ParseRuleFilter(rule)
		webhookURL := w.resolveWebhookURL(rule.WebhookURL, cfg.UseInternalAlertWebhook, cfg.AlertWebhookURL)

		grouped := make(map[string][]model.K8sForwardedEvent)
		for i := range events {
			ev := events[i]
			if processedIDs[ev.ID] {
				continue
			}
			if ev.Attempts >= maxRetry {
				forwardLog().Warn("K8s forwarded event exceeded max retries, already claimed",
					"event_id", ev.ID, "cluster_id", ev.ClusterID, "attempts", ev.Attempts)
				processedIDs[ev.ID] = true
				continue
			}
			if _, ok := clusters[ev.ClusterID]; !ok {
				continue
			}
			if !filter.Match(&ev) {
				continue
			}
			matchedIDs[ev.ID] = true
			grouped[ev.ClusterID] = append(grouped[ev.ClusterID], ev)
		}

		for clusterID, batchEvents := range grouped {
			if err := w.push(ctx, webhookURL, rule.Name, clusterID, batchEvents); err != nil {
				forwardLog().Warn("Failed to push K8s event forward webhook",
					"rule", rule.Name,
					"cluster_id", clusterID,
					"error", err)
				for _, e := range batchEvents {
					_ = w.repo.IncrementEventAttempts(ctx, e.ID)
					_ = w.repo.MarkEventProcessed(ctx, e.ID, false)
				}
				continue
			}
			for _, e := range batchEvents {
				processedIDs[e.ID] = true
			}
		}
	}

	now := time.Now()
	for _, ev := range events {
		if processedIDs[ev.ID] || matchedIDs[ev.ID] {
			continue
		}
		// 未匹配规则：放回队列，便于后续新增规则命中；过旧事件直接丢弃。
		if !ev.Timestamp.IsZero() && now.Sub(ev.Timestamp) > unmatchedEventMaxAge {
			forwardLog().Debug("K8s forwarded event unmatched and stale, keeping processed",
				"event_id", ev.ID, "cluster_id", ev.ClusterID, "namespace", ev.Namespace)
			processedIDs[ev.ID] = true
			continue
		}
		forwardLog().Debug("K8s forwarded event matched no rule, requeue",
			"event_id", ev.ID, "cluster_id", ev.ClusterID, "namespace", ev.Namespace)
		_ = w.repo.MarkEventProcessed(ctx, ev.ID, false)
	}
	return nil
}

func (w *Worker) resolveWebhookURL(ruleURL string, useInternal bool, alertURL string) string {
	u := strings.TrimSpace(ruleURL)
	if useInternal && (u == "" || strings.EqualFold(u, "internal") || strings.EqualFold(u, "alertmanager")) {
		return alertURL
	}
	return u
}

func (w *Worker) push(ctx context.Context, url, ruleName, clusterID string, events []model.K8sForwardedEvent) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("no webhook url for rule %s", ruleName)
	}
	var cid uint
	if id, err := strconv.ParseUint(clusterID, 10, 64); err == nil {
		cid = uint(id)
	}
	clusterName := w.repo.GetClusterName(ctx, cid)
	projectID := w.repo.GetClusterOwningProjectID(ctx, cid)
	payload := buildAlertManagerPayload(ruleName, clusterID, clusterName, projectID, events)
	return w.client.PostAlertmanager(ctx, url, payload)
}
