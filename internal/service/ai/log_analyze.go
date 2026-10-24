package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/llm"
	"yunshu/internal/service/logplatform"
)

type LogAnalyzeAIRequest struct {
	Provider      string `json:"provider"`
	ProjectID     uint   `json:"project_id" binding:"required"`
	Keyword       string `json:"keyword"`
	Level         string `json:"level"`
	ServiceName   string `json:"service_name"`
	Namespace     string `json:"namespace"`
	Pod           string `json:"pod"`
	Container     string `json:"container"`
	CollectorMode string `json:"collector_mode"`
	ClusterID     uint   `json:"cluster_id"`
	From          string `json:"from"`
	To            string `json:"to"`
}

type LogAnalyzeAIResponse struct {
	Overview   *logplatform.LogOverviewResult `json:"overview"`
	AISummary  string                         `json:"ai_summary"`
	RootCauses []map[string]any               `json:"root_causes"`
	Actions    []map[string]any               `json:"actions"`
	RawReply   string                         `json:"raw_reply,omitempty"`
	Provider   string                         `json:"provider"`
	Model      string                         `json:"model"`
}

func (s *Service) AnalyzeLogs(ctx context.Context, userID uint, req LogAnalyzeAIRequest) (*LogAnalyzeAIResponse, error) {
	cfg, err := s.requireEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.checkRate(userID); err != nil {
		return nil, err
	}
	if s.logSearch == nil {
		return nil, constants.ErrBadRequestWithMsg("日志检索服务不可用")
	}

	sq := logplatform.LogSearchQuery{
		ProjectID:     req.ProjectID,
		Keyword:       req.Keyword,
		Level:         req.Level,
		ServiceName:   req.ServiceName,
		Namespace:     req.Namespace,
		Pod:           req.Pod,
		Container:     req.Container,
		CollectorMode: req.CollectorMode,
		From:          req.From,
		To:            req.To,
	}
	if req.ClusterID > 0 {
		cid := req.ClusterID
		sq.ClusterID = &cid
	}
	if strings.TrimSpace(sq.Level) == "" {
		sq.Level = "ERROR"
	}

	overview, err := s.logSearch.Overview(ctx, sq)
	if err != nil {
		return nil, err
	}
	ctxPayload := map[string]any{
		"query":    sq,
		"overview": overview,
	}
	if overview != nil && overview.Summary != nil {
		ctxPayload["level_counts"] = overview.Summary.LevelCounts
		ctxPayload["top_error_signatures"] = overview.Summary.TopErrorSignatures
		ctxPayload["samples"] = overview.Summary.Samples
		ctxPayload["total"] = overview.Total
	}
	ctxJSON, _ := json.Marshal(ctxPayload)
	if len(ctxJSON) > 60_000 {
		ctxJSON = ctxJSON[:60_000]
	}
	prompt, err := s.loadPromptContent(ctx, "diagnosis/log-analyze", map[string]string{"context_json": string(ctxJSON)})
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
	s.logUsage(userID, "log_analyze", name, resp)

	summary := "已分析日志采样"
	if overview != nil && overview.Total > 0 {
		summary = fmt.Sprintf("共 %d 条命中，已提取错误签名", overview.Total)
	}
	out := &LogAnalyzeAIResponse{
		Overview:  overview,
		RawReply:  resp.Content,
		Provider:  name,
		Model:     resp.Model,
		AISummary: summary,
	}
	applyParsedAnalysis(&out.AISummary, &out.RootCauses, &out.Actions, resp.Content)
	return out, nil
}
