package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AnthropicClient Anthropic Messages API（Claude）。
type AnthropicClient struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

func NewAnthropicClient(baseURL, apiKey, defaultModel string, timeoutSec int) *AnthropicClient {
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	return &AnthropicClient{
		baseURL: NormalizeBaseURL(baseURL),
		apiKey:  strings.TrimSpace(apiKey),
		model:   strings.TrimSpace(defaultModel),
		http:    &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

func (c *AnthropicClient) Name() string { return "anthropic" }

func (c *AnthropicClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if err := ValidateMessages(req.Messages); err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, fmt.Errorf("未配置 Anthropic API Key")
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.model
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 2048
	}

	var system string
	var msgs []map[string]string
	for _, m := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		switch role {
		case "system":
			if system == "" {
				system = m.Content
			} else {
				system += "\n\n" + m.Content
			}
		case "assistant":
			msgs = append(msgs, map[string]string{"role": "assistant", "content": m.Content})
		default:
			msgs = append(msgs, map[string]string{"role": "user", "content": m.Content})
		}
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("至少需要一条 user/assistant 消息")
	}

	body := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   msgs,
	}
	if system != "" {
		body["system"] = system
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := c.baseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Anthropic HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 400))
	}
	var parsed anthropicResp
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("解析 Anthropic 响应失败: %w", err)
	}
	var b strings.Builder
	for _, block := range parsed.Content {
		if block.Type == "text" {
			b.WriteString(block.Text)
		}
	}
	return &ChatResponse{
		Content: b.String(),
		Model:   parsed.Model,
		Usage: Usage{
			PromptTokens:     parsed.Usage.InputTokens,
			CompletionTokens: parsed.Usage.OutputTokens,
			TotalTokens:      parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		},
	}, nil
}

type anthropicResp struct {
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}
