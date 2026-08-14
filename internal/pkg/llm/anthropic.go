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
	var msgs []map[string]any
	for _, m := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		switch role {
		case "system":
			if system == "" {
				system = m.Content
			} else {
				system += "\n\n" + m.Content
			}
		case "tool":
			msgs = append(msgs, map[string]any{
				"role": "user",
				"content": []map[string]any{{
					"type":        "tool_result",
					"tool_use_id": m.ToolCallID,
					"content":     m.Content,
				}},
			})
		case "assistant":
			if len(m.ToolCalls) > 0 {
				blocks := make([]map[string]any, 0, len(m.ToolCalls)+1)
				if strings.TrimSpace(m.Content) != "" {
					blocks = append(blocks, map[string]any{"type": "text", "text": m.Content})
				}
				for _, tc := range m.ToolCalls {
					var input any
					_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
					if input == nil {
						input = map[string]any{}
					}
					blocks = append(blocks, map[string]any{
						"type":  "tool_use",
						"id":    tc.ID,
						"name":  tc.Function.Name,
						"input": input,
					})
				}
				msgs = append(msgs, map[string]any{"role": "assistant", "content": blocks})
			} else {
				msgs = append(msgs, map[string]any{"role": "assistant", "content": m.Content})
			}
		default:
			msgs = append(msgs, map[string]any{"role": "user", "content": m.Content})
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
	temp := req.Temperature
	if temp <= 0 {
		temp = 0.2
	}
	body["temperature"] = temp
	if system != "" {
		body["system"] = system
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, map[string]any{
				"name":         t.Function.Name,
				"description":  t.Function.Description,
				"input_schema": t.Function.Parameters,
			})
		}
		body["tools"] = tools
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
	var toolCalls []ToolCall
	for _, block := range parsed.Content {
		switch block.Type {
		case "text":
			b.WriteString(block.Text)
		case "tool_use":
			args, _ := json.Marshal(block.Input)
			tc := ToolCall{ID: block.ID, Type: "function"}
			tc.Function.Name = block.Name
			tc.Function.Arguments = string(args)
			toolCalls = append(toolCalls, tc)
		}
	}
	return &ChatResponse{
		Content:   b.String(),
		Model:     parsed.Model,
		ToolCalls: toolCalls,
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
		Type  string         `json:"type"`
		Text  string         `json:"text"`
		ID    string         `json:"id"`
		Name  string         `json:"name"`
		Input map[string]any `json:"input"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}
