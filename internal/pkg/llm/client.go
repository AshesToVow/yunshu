package llm

import (
	"context"
	"fmt"
	"strings"
)

// Message 对话消息。
type Message struct {
	Role       string     `json:"role"` // system|user|assistant|tool
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolDefinition OpenAI 风格工具定义。
type ToolDefinition struct {
	Type     string         `json:"type"` // function
	Function ToolFunctionDef `json:"function"`
}

// ToolFunctionDef 函数描述。
type ToolFunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ToolCall 模型发起的工具调用。
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // function
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ChatRequest 统一聊天请求。
type ChatRequest struct {
	Model       string
	Messages    []Message
	MaxTokens   int
	Temperature float64
	Tools       []ToolDefinition
	ToolChoice  string // auto|none|required；空=auto（有 Tools 时）
}

// Usage token 用量。
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse 统一聊天响应。
type ChatResponse struct {
	Content   string
	Model     string
	Usage     Usage
	ToolCalls []ToolCall
}

// Client LLM 调用接口。
type Client interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	Name() string
}

// Embedder 可选向量嵌入（OpenAI 兼容 /embeddings）。
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
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

// NewFunctionTool 快捷构造 function 工具。
func NewFunctionTool(name, desc string, params map[string]any) ToolDefinition {
	if params == nil {
		params = map[string]any{"type": "object", "properties": map[string]any{}}
	}
	return ToolDefinition{
		Type: "function",
		Function: ToolFunctionDef{
			Name:        name,
			Description: desc,
			Parameters:  params,
		},
	}
}
