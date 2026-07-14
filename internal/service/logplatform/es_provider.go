package logplatform

import (
	"context"
	"fmt"
	"sync"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/pkg/esclient"

	"gorm.io/gorm"
)

// ElasticsearchProvider 运行时从数据字典 + YAML 解析 ES 配置并创建客户端。
type ElasticsearchProvider struct {
	db       *gorm.DB
	yamlBase config.ElasticsearchConfig

	mu     sync.Mutex
 cached config.ElasticsearchConfig
 client *esclient.Client
}

func NewElasticsearchProvider(db *gorm.DB, yamlBase config.ElasticsearchConfig) *ElasticsearchProvider {
	return &ElasticsearchProvider{db: db, yamlBase: yamlBase.Normalized()}
}

func (p *ElasticsearchProvider) Resolve(ctx context.Context) (config.ElasticsearchConfig, error) {
	if p == nil {
		return config.ElasticsearchConfig{}, fmt.Errorf("elasticsearch provider nil")
	}
	cfg := dictconfig.ResolveElasticsearchConfig(ctx, p.db, p.yamlBase)
	return cfg, nil
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
	defer p.mu.Unlock()
	if p.client != nil && configsEqual(p.cached, cfg) {
		return p.client, cfg, nil
	}
	cli, err := esclient.New(cfg)
	if err != nil {
		return nil, cfg, err
	}
	p.cached = cfg
	p.client = cli
	return p.client, cfg, nil
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
		a.IndexPattern != b.IndexPattern || a.DefaultRetentionDays != b.DefaultRetentionDays {
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
