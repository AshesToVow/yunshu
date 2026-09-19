package ai

import (
	"context"
	"crypto/cipher"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/llm"

	"gorm.io/gorm"
)

type LLMModelUpsertRequest struct {
	Name          string   `json:"name" binding:"required,max=128"`
	Provider      string   `json:"provider" binding:"required,max=64"`
	BaseURL       string   `json:"base_url"`
	APIKey        string   `json:"api_key"`
	ModelName     string   `json:"model_name" binding:"required,max=128"`
	ModelType     string   `json:"model_type"`
	ModelVersion  string   `json:"model_version"`
	Temperature   *float64 `json:"temperature"`
	MaxTokens     int      `json:"max_tokens"`
	ContextLength int      `json:"context_length"`
	Enabled       *bool    `json:"enabled"`
	IsDefault     bool     `json:"is_default"`
	Remark        string   `json:"remark"`
}

func (s *Service) aead() (cipher.AEAD, error) {
	return cryptox.NewAESGCMFromKeyString(s.encryptionKey)
}

func (s *Service) encryptAPIKey(plain string) (string, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return "", nil
	}
	aead, err := s.aead()
	if err != nil {
		return "", err
	}
	return cryptox.EncryptString(aead, plain)
}

func (s *Service) decryptAPIKey(enc string) (string, error) {
	enc = strings.TrimSpace(enc)
	if enc == "" {
		return "", nil
	}
	aead, err := s.aead()
	if err != nil {
		return "", err
	}
	return cryptox.DecryptString(aead, enc)
}

func normalizeLLMProvider(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	switch p {
	case "claude", "claude_code":
		return config.AIProviderAnthropic
	case "qwen", "tongyi":
		return "qwen"
	default:
		return p
	}
}

func (s *Service) CreateLLMModel(ctx context.Context, req LLMModelUpsertRequest) (*model.AiLLMModel, error) {
	name := strings.TrimSpace(req.Name)
	provider := normalizeLLMProvider(req.Provider)
	if name == "" || provider == "" || strings.TrimSpace(req.ModelName) == "" {
		return nil, constants.ErrBadRequestWithMsg("名称、Provider、模型名必填")
	}
	enc, err := s.encryptAPIKey(req.APIKey)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("加密 API Key 失败: " + err.Error())
	}
	row := model.AiLLMModel{
		Name:          name,
		Provider:      provider,
		BaseURL:       strings.TrimSpace(req.BaseURL),
		APIKeyEnc:     enc,
		ModelName:     strings.TrimSpace(req.ModelName),
		ModelType:     coalesce(req.ModelType, "chat"),
		ModelVersion:  strings.TrimSpace(req.ModelVersion),
		Temperature:   0.2,
		MaxTokens:     req.MaxTokens,
		ContextLength: req.ContextLength,
		Enabled:       true,
		IsDefault:     req.IsDefault,
		Remark:        strings.TrimSpace(req.Remark),
	}
	if req.Temperature != nil {
		row.Temperature = *req.Temperature
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if row.MaxTokens <= 0 {
		row.MaxTokens = 4096
	}
	if row.ContextLength <= 0 {
		row.ContextLength = 128000
	}
	if row.BaseURL == "" {
		row.BaseURL = defaultBaseURL(provider)
	}
	if err := s.repo.CreateLLMModel(ctx, &row, row.IsDefault); err != nil {
		return nil, err
	}
	row.HasAPIKey = row.APIKeyEnc != ""
	row.APIKeyEnc = ""
	return &row, nil
}

func defaultBaseURL(provider string) string {
	switch provider {
	case config.AIProviderDeepSeek:
		return "https://api.deepseek.com/v1"
	case config.AIProviderAnthropic:
		return "https://api.anthropic.com"
	case "qwen":
		return "https://dashscope.aliyuncs.com/compatible-mode/v1"
	default:
		return "https://api.openai.com/v1"
	}
}

func (s *Service) UpdateLLMModel(ctx context.Context, id uint, req LLMModelUpsertRequest) (*model.AiLLMModel, error) {
	row, err := s.repo.GetLLMModelByID(ctx, id)
	if err != nil {
		return nil, constants.ErrNotFoundWithMsg("模型不存在")
	}
	updates := map[string]any{
		"name":          strings.TrimSpace(req.Name),
		"provider":      normalizeLLMProvider(req.Provider),
		"base_url":      strings.TrimSpace(req.BaseURL),
		"model_name":    strings.TrimSpace(req.ModelName),
		"model_type":    coalesce(req.ModelType, row.ModelType),
		"model_version": strings.TrimSpace(req.ModelVersion),
		"remark":        strings.TrimSpace(req.Remark),
		"is_default":    req.IsDefault,
	}
	if req.Temperature != nil {
		updates["temperature"] = *req.Temperature
	}
	if req.MaxTokens > 0 {
		updates["max_tokens"] = req.MaxTokens
	}
	if req.ContextLength > 0 {
		updates["context_length"] = req.ContextLength
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if strings.TrimSpace(req.APIKey) != "" {
		enc, err := s.encryptAPIKey(req.APIKey)
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg("加密 API Key 失败: " + err.Error())
		}
		updates["api_key_enc"] = enc
	}
	if err := s.repo.UpdateLLMModel(ctx, id, updates, req.IsDefault); err != nil {
		return nil, err
	}
	row, err = s.repo.GetLLMModelByID(ctx, id)
	if err != nil {
		return nil, err
	}
	row.HasAPIKey = row.APIKeyEnc != ""
	row.APIKeyEnc = ""
	return row, nil
}

func (s *Service) DeleteLLMModel(ctx context.Context, id uint) error {
	if err := s.repo.DeleteLLMModel(ctx, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFoundWithMsg("模型不存在")
		}
		return err
	}
	return nil
}

func (s *Service) SetDefaultLLMModel(ctx context.Context, id uint) error {
	if _, err := s.repo.GetLLMModelByID(ctx, id); err != nil {
		return constants.ErrNotFoundWithMsg("模型不存在")
	}
	return s.repo.SetDefaultLLMModel(ctx, id)
}

func (s *Service) findLLMModelForChat(ctx context.Context, selector string) (*model.AiLLMModel, error) {
	selector = strings.TrimSpace(selector)
	if selector != "" {
		row, err := s.repo.FindLLMModelByName(ctx, selector)
		if err == nil {
			return row, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		prov := normalizeLLMProvider(selector)
		row, err = s.repo.FindLLMModelByProvider(ctx, prov)
		if err == nil {
			return row, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	row, err := s.repo.FindDefaultLLMModel(ctx)
	if err == nil {
		return row, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	row, err = s.repo.FindFirstEnabledLLMModel(ctx)
	if err == nil {
		return row, nil
	}
	return nil, err
}

func (s *Service) clientFromDBModel(row *model.AiLLMModel, timeoutSec int) (llm.Client, string, config.AIProviderConfig, error) {
	key, err := s.decryptAPIKey(row.APIKeyEnc)
	if err != nil {
		return nil, row.Name, config.AIProviderConfig{}, constants.ErrBadRequestWithMsg("解密 API Key 失败")
	}
	if strings.TrimSpace(key) == "" {
		return nil, row.Name, config.AIProviderConfig{}, constants.ErrBadRequestWithMsg("模型未配置 API Key")
	}
	pcfg := config.AIProviderConfig{BaseURL: row.BaseURL, APIKey: key, Model: row.ModelName}
	prov := normalizeLLMProvider(row.Provider)
	switch prov {
	case config.AIProviderAnthropic:
		return llm.NewAnthropicClient(pcfg.BaseURL, pcfg.APIKey, pcfg.Model, timeoutSec), row.Name, pcfg, nil
	case config.AIProviderDeepSeek:
		return llm.NewOpenAICompatClient(config.AIProviderDeepSeek, pcfg.BaseURL, pcfg.APIKey, pcfg.Model, timeoutSec), row.Name, pcfg, nil
	default:
		name := prov
		if name == "" || name == "qwen" {
			name = config.AIProviderOpenAICompat
		}
		return llm.NewOpenAICompatClient(name, pcfg.BaseURL, pcfg.APIKey, pcfg.Model, timeoutSec), row.Name, pcfg, nil
	}
}
