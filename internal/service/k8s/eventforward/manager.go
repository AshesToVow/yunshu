package eventforward

import (
	"context"
	"fmt"
	"sync"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/repository"
	"yunshu/internal/service/k8s"

	"gorm.io/gorm"
)

// RuntimeConfig 运行期合并字典/YAML 与 DB 规则表全局参数。
type RuntimeConfig struct {
	WatcherBufferSize     int
	WorkerIntervalSeconds int
	WorkerBatchSize       int
	WorkerMaxRetries      int
	// AlertWebhookURL 本机告警平台 K8s Event 入站地址（/alerts/ingress/k8s-events）
	AlertWebhookURL         string
	UseInternalAlertWebhook bool
}

// Manager 协调 Watcher 与 Worker（参考 k8m eventhandler）。
type Manager struct {
	repo     interfaces.K8sEventForwardRepository
	dictDB   *gorm.DB
	watcher  *Watcher
	worker   *Worker
	enabled  bool
	yamlBase config.K8sEventForwardConfig
	appPort  int
	runMu    sync.Mutex
	running  bool
}

func NewManager(
	repo interfaces.K8sEventForwardRepository,
	runtime *k8s.K8sRuntimeService,
	yamlBase config.K8sEventForwardConfig,
	alertCfg config.AlertConfig,
	appPort int,
	dbForDict *gorm.DB,
) (*Manager, error) {
	ctx := context.Background()
	resolved := dictconfig.ResolveK8sEventForwardConfig(ctx, dbForDict, yamlBase, dictconfig.DefaultK8sEventForwardDictTypes())

	if repo == nil {
		repo = repository.NewK8sEventForwardRepository(dbForDict)
	}
	defaults := model.K8sEventForwardSetting{
		ID:                     1,
		ProcessIntervalSeconds: resolved.WorkerIntervalSeconds,
		BatchSize:              resolved.WorkerBatchSize,
		MaxRetries:             resolved.WorkerMaxRetries,
		WatcherBufferSize:      resolved.WatcherBufferSize,
	}
	if err := repo.EnsureDefaultSettings(ctx, defaults); err != nil {
		return nil, err
	}

	rt, err := loadRuntimeConfig(repo, resolved, appPort)
	if err != nil {
		return nil, err
	}

	client := NewWebhookClient(alertCfg.WebhookToken, 0)
	mgr := &Manager{
		repo:     repo,
		watcher:  NewWatcher(repo, runtime, rt),
		worker:   NewWorker(repo, client, rt),
		enabled:  resolved.Enabled,
		yamlBase: yamlBase,
		appPort:  appPort,
	}
	mgr.dictDB = dbForDict
	mgr.worker.onBeforeBatch = mgr.reloadRuntimeConfig
	mgr.worker.isEnabled = func() bool { return mgr.enabled }
	return mgr, nil
}

func (m *Manager) reloadRuntimeConfig() {
	if m == nil || m.dictDB == nil {
		return
	}
	ctx := context.Background()
	resolved := dictconfig.ResolveK8sEventForwardConfig(ctx, m.dictDB, m.yamlBase, dictconfig.DefaultK8sEventForwardDictTypes())
	m.enabled = resolved.Enabled
	rt, err := loadRuntimeConfig(m.repo, resolved, m.appPort)
	if err != nil {
		forwardLog().Warn("Failed to reload K8s event forward config", "error", err)
		return
	}
	m.worker.RefreshSettings(rt)
	m.EnsureRunning()
}

func loadRuntimeConfig(store interfaces.K8sEventForwardRepository, appCfg config.K8sEventForwardConfig, port int) (RuntimeConfig, error) {
	st, err := store.GetSettings(context.Background())
	if err != nil {
		return RuntimeConfig{}, err
	}
	rt := RuntimeConfig{
		WatcherBufferSize:       firstPositive(st.WatcherBufferSize, appCfg.WatcherBufferSize),
		WorkerIntervalSeconds:   firstPositive(st.ProcessIntervalSeconds, appCfg.WorkerIntervalSeconds),
		WorkerBatchSize:         firstPositive(st.BatchSize, appCfg.WorkerBatchSize),
		WorkerMaxRetries:        firstPositive(st.MaxRetries, appCfg.WorkerMaxRetries),
		UseInternalAlertWebhook: true,
	}
	if port <= 0 {
		port = 8080
	}
	rt.AlertWebhookURL = fmt.Sprintf("http://127.0.0.1:%d/api/v1/alerts/ingress/k8s-events", port)
	return rt, nil
}

func firstPositive(a, b int) int {
	if a > 0 {
		return a
	}
	return b
}

func (m *Manager) Start() {
	m.ensureRunning(false)
}

// EnsureRunning 在规则启用/变更后热启动 watcher 与 worker（进程内幂等）。
func (m *Manager) EnsureRunning() {
	m.ensureRunning(true)
}

func (m *Manager) ensureRunning(triggerWatch bool) {
	if m == nil || !m.enabled {
		if m != nil && !m.enabled {
			forwardLog().Info("K8s event forward disabled in config")
		}
		return
	}
	ctx := context.Background()
	ok, err := m.repo.HasEnabledRules(ctx)
	if err != nil {
		forwardLog().Warn("Failed to check K8s event forward rules", "error", err)
		return
	}
	if !ok {
		forwardLog().Info("No enabled K8s event forward rules, watcher and worker not started")
		return
	}
	m.runMu.Lock()
	if !m.running {
		forwardLog().Info("Starting K8s event forward watcher and worker")
		m.watcher.Start()
		m.worker.Start()
		m.running = true
	}
	m.runMu.Unlock()
	if triggerWatch && m.watcher != nil {
		m.watcher.TriggerEnsure()
	}
}

func (m *Manager) Stop() {
	if m.watcher != nil {
		m.watcher.Stop()
	}
	if m.worker != nil {
		m.worker.Stop()
	}
}
