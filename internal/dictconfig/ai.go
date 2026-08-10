package dictconfig

import (
	"context"
	"strings"

	"yunshu/internal/config"

	"gorm.io/gorm"
)

// AIDictTypes 数据字典中覆盖 ai.* 的 dict_type。
type AIDictTypes struct {
	Enabled         string
	DefaultProvider string
	TimeoutSec      string
	MaxTokens       string

	OpenAIBaseURL string
	OpenAIAPIKey  string
	OpenAIModel   string

	DeepSeekBaseURL string
	DeepSeekAPIKey  string
	DeepSeekModel   string

	AnthropicBaseURL string
	AnthropicAPIKey  string
	AnthropicModel   string
}

func DefaultAIDictTypes() AIDictTypes {
	return AIDictTypes{
		Enabled:          "ai_enabled",
		DefaultProvider:  "ai_default_provider",
		TimeoutSec:       "ai_timeout_sec",
		MaxTokens:        "ai_max_tokens",
		OpenAIBaseURL:    "ai_openai_base_url",
		OpenAIAPIKey:     "ai_openai_api_key",
		OpenAIModel:      "ai_openai_model",
		DeepSeekBaseURL:  "ai_deepseek_base_url",
		DeepSeekAPIKey:   "ai_deepseek_api_key",
		DeepSeekModel:    "ai_deepseek_model",
		AnthropicBaseURL: "ai_anthropic_base_url",
		AnthropicAPIKey:  "ai_anthropic_api_key",
		AnthropicModel:   "ai_anthropic_model",
	}
}

// ResolveAIConfig 以 yamlBase 为底，用已启用的数据字典项覆盖。
func ResolveAIConfig(ctx context.Context, db *gorm.DB, yamlBase config.AIConfig, types AIDictTypes) config.AIConfig {
	cfg := yamlBase
	cfg.ApplyDefaults()
	if db == nil {
		return cfg
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.Enabled); ok {
		if bv, ok2 := parseBoolLoose(v); ok2 {
			cfg.Enabled = bv
		}
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.DefaultProvider); ok {
		if s := strings.TrimSpace(v); s != "" {
			cfg.DefaultProvider = s
		}
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.TimeoutSec); ok {
		if n, ok2 := parseInt(v); ok2 && n > 0 {
			cfg.TimeoutSec = n
		}
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.MaxTokens); ok {
		if n, ok2 := parseInt(v); ok2 && n > 0 {
			cfg.MaxTokens = n
		}
	}
	overrideProvider(&cfg.OpenAI, db, ctx, types.OpenAIBaseURL, types.OpenAIAPIKey, types.OpenAIModel)
	overrideProvider(&cfg.DeepSeek, db, ctx, types.DeepSeekBaseURL, types.DeepSeekAPIKey, types.DeepSeekModel)
	overrideProvider(&cfg.Anthropic, db, ctx, types.AnthropicBaseURL, types.AnthropicAPIKey, types.AnthropicModel)
	return cfg
}

func overrideProvider(p *config.AIProviderConfig, db *gorm.DB, ctx context.Context, baseURLType, apiKeyType, modelType string) {
	if v, ok := fetchEnabledDictValue(ctx, db, baseURLType); ok {
		if s := strings.TrimSpace(v); s != "" {
			p.BaseURL = strings.TrimRight(s, "/")
		}
	}
	if v, ok := fetchEnabledDictValue(ctx, db, apiKeyType); ok {
		if s := strings.TrimSpace(v); s != "" {
			p.APIKey = s
		}
	}
	if v, ok := fetchEnabledDictValue(ctx, db, modelType); ok {
		if s := strings.TrimSpace(v); s != "" {
			p.Model = s
		}
	}
}
