package logplatform

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/pkg/esclient"

	"gorm.io/gorm"
)

// ManagedESEndpoint 来自 esmgmt 连接表的可连接端点（地址/认证）。
type ManagedESEndpoint struct {
	ID         uint
	Name       string
	Addresses  []string
	Username   string
	Password   string
	TimeoutSec int
}

// ManagedESConnectionLoader 由 esmgmt.Service 实现，避免 ElasticsearchProvider ↔ esmgmt 构造循环依赖。
type ManagedESConnectionLoader interface {
	LoadManagedESConnection(ctx context.Context, id uint) (*ManagedESEndpoint, error)
}

// ElasticsearchProvider 运行时从数据字典 + YAML 解析 ES 配置并创建客户端。
// 若字典 elasticsearch_connection_id > 0，地址与认证改用 esmgmt 连接；索引模式/保留等仍来自字典。
type ElasticsearchProvider struct {
	db       *gorm.DB
	yamlBase config.ElasticsearchConfig

	mu     sync.Mutex
	cached config.ElasticsearchConfig
	client *esclient.Client

	connLoader ManagedESConnectionLoader
}

func NewElasticsearchProvider(db *gorm.DB, yamlBase config.ElasticsearchConfig) *ElasticsearchProvider {
	return &ElasticsearchProvider{db: db, yamlBase: yamlBase.Normalized()}
}

// SetManagedConnectionLoader 在 Wire 装配 esmgmt 后注入，用于按连接 ID 加载地址与密码。
func (p *ElasticsearchProvider) SetManagedConnectionLoader(l ManagedESConnectionLoader) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connLoader = l
}

func (p *ElasticsearchProvider) Resolve(ctx context.Context) (config.ElasticsearchConfig, error) {
	if p == nil {
		return config.ElasticsearchConfig{}, fmt.Errorf("elasticsearch provider nil")
	}
	cfg := dictconfig.ResolveElasticsearchConfig(ctx, p.db, p.yamlBase)
	connID := p.readManagedConnectionID(ctx)
	if connID == 0 {
		return cfg, nil
	}
	p.mu.Lock()
	loader := p.connLoader
	p.mu.Unlock()
	if loader == nil {
		return cfg, fmt.Errorf("elasticsearch connection_id=%d but managed connection loader not wired", connID)
	}
	ep, err := loader.LoadManagedESConnection(ctx, connID)
	if err != nil {
		return cfg, fmt.Errorf("load esmgmt connection %d: %w", connID, err)
	}
	if ep == nil || len(ep.Addresses) == 0 {
		return cfg, fmt.Errorf("esmgmt connection %d has empty addresses", connID)
	}
	cfg.Addresses = ep.Addresses
	cfg.Username = ep.Username
	cfg.Password = ep.Password
	if ep.TimeoutSec > 0 {
		cfg.TimeoutSeconds = ep.TimeoutSec
	}
	cfg.Enabled = true
	return cfg.Normalized(), nil
}

// ResolveFromDict 仅合并字典/YAML，忽略 elasticsearch_connection_id（供 esmgmt 导入字典连接使用）。
func (p *ElasticsearchProvider) ResolveFromDict(ctx context.Context) (config.ElasticsearchConfig, error) {
	if p == nil {
		return config.ElasticsearchConfig{}, fmt.Errorf("elasticsearch provider nil")
	}
	return dictconfig.ResolveElasticsearchConfig(ctx, p.db, p.yamlBase), nil
}

func (p *ElasticsearchProvider) readManagedConnectionID(ctx context.Context) uint {
	types := dictconfig.DefaultElasticsearchDictTypes()
	v, ok := dictconfig.FetchEnabledDictValue(ctx, p.db, types.ConnectionID)
	if !ok {
		return 0
	}
	n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

// ManagedConnectionID 返回日志平台当前绑定的 esmgmt 连接 ID（0 表示使用字典地址）。
func (p *ElasticsearchProvider) ManagedConnectionID(ctx context.Context) uint {
	if p == nil {
		return 0
	}
	return p.readManagedConnectionID(ctx)
}

// LookupManagedConnection 按 ID 加载托管连接（仅展示用，可为 nil）。
func (p *ElasticsearchProvider) LookupManagedConnection(ctx context.Context, id uint) (*ManagedESEndpoint, error) {
	if p == nil || id == 0 {
		return nil, fmt.Errorf("invalid connection lookup")
	}
	p.mu.Lock()
	loader := p.connLoader
	p.mu.Unlock()
	if loader == nil {
		return nil, fmt.Errorf("managed connection loader not wired")
	}
	return loader.LoadManagedESConnection(ctx, id)
}

// SetManagedConnectionID 写入字典 elasticsearch_connection_id；id=0 表示回退字典地址。
func (p *ElasticsearchProvider) SetManagedConnectionID(ctx context.Context, id uint) error {
	if p == nil {
		return fmt.Errorf("elasticsearch provider nil")
	}
	if id > 0 {
		p.mu.Lock()
		loader := p.connLoader
		p.mu.Unlock()
		if loader == nil {
			return fmt.Errorf("managed connection loader not wired")
		}
		if _, err := loader.LoadManagedESConnection(ctx, id); err != nil {
			return fmt.Errorf("validate esmgmt connection %d: %w", id, err)
		}
	}
	types := dictconfig.DefaultElasticsearchDictTypes()
	if err := dictconfig.UpsertEnabledDictValue(
		ctx, p.db, types.ConnectionID,
		"日志平台 ES 连接 ID",
		strconv.FormatUint(uint64(id), 10),
		"esmgmt_connections.id；>0 时日志检索使用该连接",
	); err != nil {
		return err
	}
	// 绑定真实连接时同步开启检索开关，避免选了连接却仍因 enabled=false 不可用。
	if id > 0 {
		_ = dictconfig.UpsertEnabledDictValue(
			ctx, p.db, types.Enabled,
			"启用 ES 日志检索",
			"true",
			"由日志平台选择 ES 连接时自动开启",
		)
	}
	p.InvalidateCache()
	return nil
}

func (p *ElasticsearchProvider) Client(ctx context.Context) (*esclient.Client, config.ElasticsearchConfig, error) {
	cfg, err := p.Resolve(ctx)
	if err != nil {
		return nil, cfg, err
	}
	if !cfg.Enabled {
		return nil, cfg, fmt.Errorf("elasticsearch disabled")
	}
	p.mu.Lock()
	if p.client != nil && configsEqual(p.cached, cfg) {
		cli := p.client
		p.mu.Unlock()
		return cli, cfg, nil
	}
	p.mu.Unlock()

	cli, err := esclient.New(cfg)
	if err != nil {
		return nil, cfg, err
	}

	p.mu.Lock()
	// 并发下可能已有其它 goroutine 建好客户端
	if p.client != nil && configsEqual(p.cached, cfg) {
		existing := p.client
		p.mu.Unlock()
		return existing, cfg, nil
	}
	p.cached = cfg
	p.client = cli
	p.mu.Unlock()

	// 必须在释放锁之后写模板，且不得再调用 Client()（否则重入互斥锁死锁）
	ensureLogIndexTemplateWithClient(ctx, cli)
	return cli, cfg, nil
}

func (p *ElasticsearchProvider) InvalidateCache() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.client = nil
}

func configsEqual(a, b config.ElasticsearchConfig) bool {
	if a.Enabled != b.Enabled || a.Username != b.Username || a.Password != b.Password ||
		a.IndexPattern != b.IndexPattern || a.DefaultRetentionDays != b.DefaultRetentionDays ||
		a.TimeoutSeconds != b.TimeoutSeconds {
		return false
	}
	if len(a.Addresses) != len(b.Addresses) {
		return false
	}
	for i := range a.Addresses {
		if a.Addresses[i] != b.Addresses[i] {
			return false
		}
	}
	return true
}
