package config

import "strings"

// AIProvider 名称常量。
const (
	AIProviderOpenAICompat = "openai_compat"
	AIProviderDeepSeek     = "deepseek"
	AIProviderAnthropic    = "anthropic"
)

// AIConfig AI 模块配置（数据字典 ai_* 优先覆盖）。
type AIConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	DefaultProvider string `mapstructure:"default_provider"`
	TimeoutSec      int    `mapstructure:"timeout_sec"`
	MaxTokens       int    `mapstructure:"max_tokens"`

	OpenAI    AIProviderConfig `mapstructure:"openai"`
	DeepSeek  AIProviderConfig `mapstructure:"deepseek"`
	Anthropic AIProviderConfig `mapstructure:"anthropic"`
}

// AIProviderConfig 单个 LLM Provider 连接参数。
type AIProviderConfig struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
}

// DefaultAIConfig YAML / 缺省兜底。
func DefaultAIConfig() AIConfig {
	return AIConfig{
		Enabled:         false,
		DefaultProvider: AIProviderOpenAICompat,
		TimeoutSec:      60,
		MaxTokens:       2048,
		OpenAI: AIProviderConfig{
			BaseURL: "https://api.openai.com/v1",
			Model:   "gpt-4o-mini",
		},
		DeepSeek: AIProviderConfig{
			BaseURL: "https://api.deepseek.com/v1",
			Model:   "deepseek-chat",
		},
		Anthropic: AIProviderConfig{
			BaseURL: "https://api.anthropic.com",
			Model:   "claude-sonnet-4-20250514",
		},
	}
}

// ApplyDefaults 填充零值字段。
func (c *AIConfig) ApplyDefaults() {
	def := DefaultAIConfig()
	if strings.TrimSpace(c.DefaultProvider) == "" {
		c.DefaultProvider = def.DefaultProvider
	}
	if c.TimeoutSec <= 0 {
		c.TimeoutSec = def.TimeoutSec
	}
	if c.MaxTokens <= 0 {
		c.MaxTokens = def.MaxTokens
	}
	if strings.TrimSpace(c.OpenAI.BaseURL) == "" {
		c.OpenAI.BaseURL = def.OpenAI.BaseURL
	}
	if strings.TrimSpace(c.OpenAI.Model) == "" {
		c.OpenAI.Model = def.OpenAI.Model
	}
	if strings.TrimSpace(c.DeepSeek.BaseURL) == "" {
		c.DeepSeek.BaseURL = def.DeepSeek.BaseURL
	}
	if strings.TrimSpace(c.DeepSeek.Model) == "" {
		c.DeepSeek.Model = def.DeepSeek.Model
	}
	if strings.TrimSpace(c.Anthropic.BaseURL) == "" {
		c.Anthropic.BaseURL = def.Anthropic.BaseURL
	}
	if strings.TrimSpace(c.Anthropic.Model) == "" {
		c.Anthropic.Model = def.Anthropic.Model
	}
}

// ProviderConfig 按名称取 Provider；未知名回退 DefaultProvider。
func (c AIConfig) ProviderConfig(name string) (string, AIProviderConfig) {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		n = strings.ToLower(strings.TrimSpace(c.DefaultProvider))
	}
	switch n {
	case AIProviderDeepSeek:
		return AIProviderDeepSeek, c.DeepSeek
	case AIProviderAnthropic, "claude", "claude_code":
		return AIProviderAnthropic, c.Anthropic
	default:
		return AIProviderOpenAICompat, c.OpenAI
	}
}
