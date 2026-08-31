package ai

import (
	"context"
	"strings"

	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/llm"
)

// GenerateK8sYAMLRequest 根据自然语言生成 K8s 资源 YAML。
type GenerateK8sYAMLRequest struct {
	Provider     string `json:"provider"`
	ResourceKind string `json:"resource_kind"` // Deployment / Service / Pod ...
	Namespace    string `json:"namespace"`
	Description  string `json:"description" binding:"required"`
	HintYAML     string `json:"hint_yaml"` // 可选：当前模板或编辑器内容
	ClusterID    uint   `json:"cluster_id"`
}

// GenerateK8sYAMLResponse AI 生成结果。
type GenerateK8sYAMLResponse struct {
	YAML     string `json:"yaml"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	RawReply string `json:"raw_reply,omitempty"`
}

// GenerateK8sYAML 按描述生成 Kubernetes YAML（只读，不直接 apply）。
func (s *Service) GenerateK8sYAML(
	ctx context.Context,
	userID uint,
	req GenerateK8sYAMLRequest,
) (*GenerateK8sYAMLResponse, error) {
	cfg, err := s.requireEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.checkRate(userID); err != nil {
		return nil, err
	}
	desc := strings.TrimSpace(req.Description)
	if desc == "" {
		return nil, constants.ErrBadRequestWithMsg("请填写资源需求描述")
	}
	kind := strings.TrimSpace(req.ResourceKind)
	if kind == "" {
		kind = "Resource"
	}
	ns := strings.TrimSpace(req.Namespace)
	if ns == "" {
		ns = "default"
	}
	hint := strings.TrimSpace(req.HintYAML)
	if hint == "" {
		hint = "（无）"
	}
	if len(hint) > 12_000 {
		hint = hint[:12_000]
	}

	prompt, err := s.loadPromptContent(ctx, "generation/k8s-yaml", map[string]string{
		"resource_kind": kind,
		"namespace":     ns,
		"description":   desc,
		"hint_yaml":     hint,
	})
	if err != nil {
		return nil, err
	}
	cli, name, pcfg, err := s.clientFor(ctx, &cfg, req.Provider)
	if err != nil {
		return nil, err
	}
	resp, err := cli.Chat(ctx, llm.ChatRequest{
		Model: pcfg.Model,
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens: cfg.MaxTokens,
	})
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("AI 调用失败: " + err.Error())
	}
	s.logUsage(userID, "k8s_generate_yaml", name, resp)

	yamlText := extractYAMLDocument(resp.Content)
	if strings.TrimSpace(yamlText) == "" {
		return nil, constants.ErrBadRequestWithMsg("AI 未返回有效 YAML，请重试或调整描述")
	}
	return &GenerateK8sYAMLResponse{
		YAML:     yamlText,
		Provider: name,
		Model:    resp.Model,
		RawReply: resp.Content,
	}, nil
}

// extractYAMLDocument 去掉 Markdown 围栏与前后说明文字，保留 YAML 主体。
func extractYAMLDocument(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "```"); i >= 0 {
		rest := s[i+3:]
		rest = strings.TrimLeft(rest, " \t")
		if strings.HasPrefix(strings.ToLower(rest), "yaml") {
			rest = rest[4:]
		} else if strings.HasPrefix(strings.ToLower(rest), "yml") {
			rest = rest[3:]
		}
		rest = strings.TrimLeft(rest, "\r\n")
		if j := strings.Index(rest, "```"); j >= 0 {
			rest = rest[:j]
		}
		s = strings.TrimSpace(rest)
	}
	// 若仍混有说明，尽量从 apiVersion 起截取；无 apiVersion 时再尝试 kind
	if i := strings.Index(s, "apiVersion:"); i > 0 {
		s = strings.TrimSpace(s[i:])
	} else if !strings.Contains(s, "apiVersion:") {
		if i := strings.Index(s, "kind:"); i > 0 {
			s = strings.TrimSpace(s[i:])
		}
	}
	return s
}
