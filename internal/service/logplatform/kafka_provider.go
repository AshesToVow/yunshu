package logplatform

import (
	"context"
	"fmt"
	"sync"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"

	"gorm.io/gorm"
)

// KafkaProvider 运行时从数据字典 + YAML 解析 Kafka 配置。
type KafkaProvider struct {
	db       *gorm.DB
	yamlBase config.KafkaConfig

	mu     sync.RWMutex
	cached config.KafkaConfig
	has    bool
}

func NewKafkaProvider(db *gorm.DB, yamlBase config.KafkaConfig) *KafkaProvider {
	return &KafkaProvider{db: db, yamlBase: yamlBase.Normalized()}
}

func (p *KafkaProvider) Resolve(ctx context.Context) (config.KafkaConfig, error) {
	if p == nil {
		return config.KafkaConfig{}, fmt.Errorf("kafka provider nil")
	}
	cfg := dictconfig.ResolveKafkaConfig(ctx, p.db, p.yamlBase)
	p.mu.Lock()
	p.cached = cfg
	p.has = true
	p.mu.Unlock()
	return cfg, nil
}

func (p *KafkaProvider) Cached() (config.KafkaConfig, bool) {
	if p == nil {
		return config.KafkaConfig{}, false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cached, p.has
}

func (p *KafkaProvider) InvalidateCache() {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.has = false
	p.mu.Unlock()
}
