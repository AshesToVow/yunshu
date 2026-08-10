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

// OpenAICompatClient OpenAI Chat Completions 兼容客户端（DeepSeek / 通义 / vLLM 等）。
type OpenAICompatClient struct {
	name    string
	baseURL string
	apiKey  string
	model   string
	timeout time.Duration
	http    *http.Client
}

func NewOpenAICompatClient(name, baseURL, apiKey, defaultModel string, timeoutSec int) *OpenAICompatClient {
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	return &OpenAICompatClient{
		name:    name,
		baseURL: NormalizeBaseURL(baseURL),
		apiKey:  strings.TrimSpace(apiKey),
		model:   strings.TrimSpace(defaultModel),
		timeout: time.Duration(timeoutSec) * time.Second,
		http:    &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

func (c *OpenAICompatClient) Name() string { return c.name }

func (c *OpenAICompatClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if err := ValidateMessages(req.Messages); err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, fmt.Errorf("未配置 API Key")
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.model
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 2048
	}
	temp := req.Temperature
	if temp <= 0 {
		temp = 0.2
	}

	msgs := make([]map[string]any, 0, len(req.Messages))
	for _, m := range req.Messages {
		item := map[string]any{
			"role":    m.Role,
			"content": m.Content,
		}
		if m.Name != "" {
			item["name"] = m.Name
		}
		if m.ToolCallID != "" {
			item["tool_call_id"] = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			item["tool_calls"] = m.ToolCalls
		}
		msgs = append(msgs, item)
	}

	body := map[string]any{
		"model":       model,
		"messages":    msgs,
		"max_tokens":  maxTokens,
		"temperature": temp,
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
		choice := strings.TrimSpace(req.ToolChoice)
		if choice == "" {
			choice = "auto"
		}
		body["tool_choice"] = choice
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := c.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("LLM HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 400))
	}
	var parsed openaiChatResp
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	content := ""
	var toolCalls []ToolCall
	if len(parsed.Choices) > 0 {
		content = parsed.Choices[0].Message.Content
		toolCalls = parsed.Choices[0].Message.ToolCalls
	}
	return &ChatResponse{
		Content:   content,
		Model:     parsed.Model,
		ToolCalls: toolCalls,
		Usage: Usage{
			PromptTokens:     parsed.Usage.PromptTokens,
			CompletionTokens: parsed.Usage.CompletionTokens,
			TotalTokens:      parsed.Usage.TotalTokens,
		},
	}, nil
}

// Embed 调用 /embeddings（不支持时返回错误，由上层降级 BM25）。
func (c *OpenAICompatClient) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, fmt.Errorf("未配置 API Key")
	}
	body := map[string]any{
		"model": "text-embedding-3-small",
		"input": texts,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embeddings", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embeddings HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 300))
	}
	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	out := make([][]float64, 0, len(parsed.Data))
	for _, d := range parsed.Data {
		out = append(out, d.Embedding)
	}
	return out, nil
}

type openaiChatResp struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
