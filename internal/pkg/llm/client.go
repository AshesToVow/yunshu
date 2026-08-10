package llm

import (
	"context"
	"fmt"
	"strings"
)

// Message 对话消息。
type Message struct {
	Role    string `json:"role"` // system|user|assistant
	Content string `json:"content"`
}

// ChatRequest 统一聊天请求。
type ChatRequest struct {
	Model       string
	Messages    []Message
	MaxTokens   int
	Temperature float64
}

// Usage token 用量。
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse 统一聊天响应。
type ChatResponse struct {
	Content string
	Model   string
	Usage   Usage
}

// Client LLM 调用接口。
type Client interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	Name() string
}

// NormalizeBaseURL 去掉尾部斜杠。
func NormalizeBaseURL(u string) string {
	return strings.TrimRight(strings.TrimSpace(u), "/")
}

// ValidateMessages 基础校验。
func ValidateMessages(msgs []Message) error {
	if len(msgs) == 0 {
		return fmt.Errorf("messages 不能为空")
	}
	total := 0
	for _, m := range msgs {
		total += len(m.Content)
		if total > 120_000 {
			return fmt.Errorf("消息总长度过大")
		}
	}
	return nil
}
