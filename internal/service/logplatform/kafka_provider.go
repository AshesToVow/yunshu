package logplatform

import (
	"context"
	"fmt"
	"sync"

	"yunshu/internal/config"
)

// KafkaConfigResolver 解析运行期字典覆盖后的 Kafka 配置（由 Wire 注入，避免 Provider 持有 *gorm.DB）。
type KafkaConfigResolver func(ctx context.Context) config.KafkaConfig

// KafkaProvider 运行时从数据字典 + YAML 解析 Kafka 配置。
type KafkaProvider struct {
	resolve  KafkaConfigResolver
	yamlBase config.KafkaConfig

	mu     sync.RWMutex
	cached config.KafkaConfig
	has    bool
}

func NewKafkaProvider(resolve KafkaConfigResolver, yamlBase config.KafkaConfig) *KafkaProvider {
	return &KafkaProvider{resolve: resolve, yamlBase: yamlBase.Normalized()}
}

func (p *KafkaProvider) Resolve(ctx context.Context) (config.KafkaConfig, error) {
	if p == nil {
		return config.KafkaConfig{}, fmt.Errorf("kafka provider nil")
	}
	var cfg config.KafkaConfig
	if p.resolve != nil {
		cfg = p.resolve(ctx)
	} else {
		cfg = p.yamlBase
	}
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
