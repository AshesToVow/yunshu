package ai

import (
	"context"
	"encoding/json"
	"strings"

	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/llm"
	"yunshu/internal/service/logplatform"
)

// PipelineAdjustRequest AI 调整 Loggie Pipeline / 解析规则。
type PipelineAdjustRequest struct {
	Provider     string   `json:"provider"`
	ProjectID    uint     `json:"project_id"`
	Kind         string   `json:"kind"` // host|k8s
	Goal         string   `json:"goal"` // 用户目标，如「抽出 status/service/trace_id」
	SampleLogs   []string `json:"sample_logs"`
	CurrentYAML  string   `json:"current_yml"`
	ParseProfile string   `json:"parse_profile"`
}

// PipelineAdjustResponse AI 调整建议。
type PipelineAdjustResponse struct {
	Summary        string   `json:"summary"`
	ParseProfile   string   `json:"parse_profile"`
	SuggestedYAML  string   `json:"suggested_yml"`
	ExtractedFields []string `json:"extracted_fields"`
	Notes          []string `json:"notes"`
	Provider       string   `json:"provider"`
	Model          string   `json:"model"`
	RawReply       string   `json:"raw_reply,omitempty"`
}

// AdjustLoggiePipeline 根据样例日志与当前 YAML，建议 Loggie pipelines.yml / 解析档。
func (s *Service) AdjustLoggiePipeline(ctx context.Context, userID uint, req PipelineAdjustRequest) (*PipelineAdjustResponse, error) {
	cfg, err := s.requireEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.checkRate(userID); err != nil {
		return nil, err
	}
	samples := make([]string, 0, len(req.SampleLogs))
	for _, line := range req.SampleLogs {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > 2000 {
			line = line[:2000]
		}
		samples = append(samples, line)
		if len(samples) >= 12 {
			break
		}
	}
	if len(samples) == 0 && strings.TrimSpace(req.CurrentYAML) == "" {
		return nil, constants.ErrBadRequestWithMsg("请提供样例日志或当前 Pipeline YAML")
	}
	profiles := logplatform.ListParseProfileOptions()
	profileNames := make([]string, 0, len(profiles))
	for _, p := range profiles {
		profileNames = append(profileNames, p.Value)
	}
	payload := map[string]any{
		"goal":               strings.TrimSpace(req.Goal),
		"kind":               strings.TrimSpace(req.Kind),
		"current_yml":        truncateForPrompt(req.CurrentYAML, 12000),
		"parse_profile":      strings.TrimSpace(req.ParseProfile),
		"sample_logs":        samples,
		"available_profiles": profileNames,
	}
	ctxJSON, _ := json.Marshal(payload)
	prompt, err := s.loadPromptContent(ctx, "generation/loggie-pipeline", map[string]string{
		"context_json": string(ctxJSON),
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
	s.logUsage(userID, "loggie_pipeline_adjust", name, resp)

	out := &PipelineAdjustResponse{
		Provider: name,
		Model:    resp.Model,
		RawReply: resp.Content,
	}
	if parsed := parsePipelineAdjustJSON(resp.Content); parsed != nil {
		out.Summary = parsed.Summary
		out.ParseProfile = parsed.ParseProfile
		out.SuggestedYAML = parsed.SuggestedYAML
		out.ExtractedFields = parsed.ExtractedFields
		out.Notes = parsed.Notes
	}
	if strings.TrimSpace(out.SuggestedYAML) == "" {
		if y := extractYAMLDocument(resp.Content); strings.Contains(y, "pipelines:") {
			out.SuggestedYAML = y
		}
	}
	if strings.TrimSpace(out.SuggestedYAML) == "" && strings.TrimSpace(out.Summary) == "" {
		return nil, constants.ErrBadRequestWithMsg("AI 未返回有效 Pipeline 建议，请补充样例或目标后重试")
	}
	if out.Summary == "" {
		out.Summary = "已生成 Pipeline 调整建议"
	}
	return out, nil
}

func truncateForPrompt(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "\n# ... truncated ..."
}

func parsePipelineAdjustJSON(raw string) *PipelineAdjustResponse {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	// 尝试从 Markdown 代码块取出 JSON
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			raw = raw[i : j+1]
		}
	}
	var m struct {
		Summary         string   `json:"summary"`
		ParseProfile    string   `json:"parse_profile"`
		SuggestedYAML   string   `json:"suggested_yml"`
		SuggestedYml    string   `json:"suggested_yaml"`
		ExtractedFields []string `json:"extracted_fields"`
		Notes           []string `json:"notes"`
	}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}
	yml := strings.TrimSpace(m.SuggestedYAML)
	if yml == "" {
		yml = strings.TrimSpace(m.SuggestedYml)
	}
	return &PipelineAdjustResponse{
		Summary:         m.Summary,
		ParseProfile:    m.ParseProfile,
		SuggestedYAML:   yml,
		ExtractedFields: m.ExtractedFields,
		Notes:           m.Notes,
	}
}
