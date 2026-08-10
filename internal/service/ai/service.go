package ai

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"yunshu/internal/ai/prompts"
	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/llm"
	"yunshu/internal/service/alert"
	cicdsvc "yunshu/internal/service/cicd"
	"yunshu/internal/service/k8s"

	"gorm.io/gorm"
)

// Service AI 业务服务。
type Service struct {
	db       *gorm.DB
	yamlAI   config.AIConfig
	podSvc   *k8s.K8sPodService
	cicdSvc  *cicdsvc.Service
	alertSvc *alert.AlertService
	rateMu   sync.Mutex
	rateMap  map[uint]time.Time // 简易限流：每用户最短间隔
}

func NewService(
	db *gorm.DB,
	yamlAI config.AIConfig,
	podSvc *k8s.K8sPodService,
	cicdSvc *cicdsvc.Service,
	alertSvc *alert.AlertService,
) *Service {
	return &Service{
		db:       db,
		yamlAI:   yamlAI,
		podSvc:   podSvc,
		cicdSvc:  cicdSvc,
		alertSvc: alertSvc,
		rateMap:  make(map[uint]time.Time),
	}
}

func (s *Service) resolved(ctx context.Context) config.AIConfig {
	return dictconfig.ResolveAIConfig(ctx, s.db, s.yamlAI, dictconfig.DefaultAIDictTypes())
}

func (s *Service) requireEnabled(ctx context.Context) (config.AIConfig, error) {
	cfg := s.resolved(ctx)
	if !cfg.Enabled {
		return cfg, constants.ErrBadRequestWithMsg("AI 未启用，请在数据字典配置 ai_enabled=true 并填写 Provider")
	}
	return cfg, nil
}

func (s *Service) checkRate(userID uint) error {
	if userID == 0 {
		return nil
	}
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	last, ok := s.rateMap[userID]
	if ok && time.Since(last) < 2*time.Second {
		return constants.ErrBadRequestWithMsg("请求过于频繁，请稍后再试")
	}
	s.rateMap[userID] = time.Now()
	return nil
}

func (s *Service) clientFor(cfg config.AIConfig, provider string) (llm.Client, string, config.AIProviderConfig, error) {
	name, pcfg := cfg.ProviderConfig(provider)
	if strings.TrimSpace(pcfg.APIKey) == "" {
		return nil, name, pcfg, constants.ErrBadRequestWithMsg("未配置 API Key，请启用对应 ai_*_api_key 字典项")
	}
	switch name {
	case config.AIProviderAnthropic:
		return llm.NewAnthropicClient(pcfg.BaseURL, pcfg.APIKey, pcfg.Model, cfg.TimeoutSec), name, pcfg, nil
	case config.AIProviderDeepSeek:
		return llm.NewOpenAICompatClient(name, pcfg.BaseURL, pcfg.APIKey, pcfg.Model, cfg.TimeoutSec), name, pcfg, nil
	default:
		return llm.NewOpenAICompatClient(config.AIProviderOpenAICompat, pcfg.BaseURL, pcfg.APIKey, pcfg.Model, cfg.TimeoutSec), config.AIProviderOpenAICompat, pcfg, nil
	}
}

// StatusResponse AI 状态（脱敏）。
type StatusResponse struct {
	Enabled         bool   `json:"enabled"`
	DefaultProvider string `json:"default_provider"`
	TimeoutSec      int    `json:"timeout_sec"`
	MaxTokens       int    `json:"max_tokens"`
	Providers       []ProviderStatus `json:"providers"`
}

type ProviderStatus struct {
	Name      string `json:"name"`
	Configured bool  `json:"configured"`
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
}

func (s *Service) Status(ctx context.Context) StatusResponse {
	cfg := s.resolved(ctx)
	mk := func(name string, p config.AIProviderConfig) ProviderStatus {
		return ProviderStatus{
			Name:       name,
			Configured: strings.TrimSpace(p.APIKey) != "",
			BaseURL:    p.BaseURL,
			Model:      p.Model,
		}
	}
	return StatusResponse{
		Enabled:         cfg.Enabled,
		DefaultProvider: cfg.DefaultProvider,
		TimeoutSec:      cfg.TimeoutSec,
		MaxTokens:       cfg.MaxTokens,
		Providers: []ProviderStatus{
			mk(config.AIProviderOpenAICompat, cfg.OpenAI),
			mk(config.AIProviderDeepSeek, cfg.DeepSeek),
			mk(config.AIProviderAnthropic, cfg.Anthropic),
		},
	}
}

type PingRequest struct {
	Provider string `json:"provider"`
}

type PingResponse struct {
	OK       bool   `json:"ok"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Reply    string `json:"reply"`
	Message  string `json:"message,omitempty"`
}

func (s *Service) Ping(ctx context.Context, userID uint, req PingRequest) (*PingResponse, error) {
	cfg, err := s.requireEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.checkRate(userID); err != nil {
		return nil, err
	}
	cli, name, pcfg, err := s.clientFor(cfg, req.Provider)
	if err != nil {
		return nil, err
	}
	resp, err := cli.Chat(ctx, llm.ChatRequest{
		Model: pcfg.Model,
		Messages: []llm.Message{
			{Role: "user", Content: "Reply with exactly: pong"},
		},
		MaxTokens:   16,
		Temperature: 0,
	})
	if err != nil {
		slog.Warn("ai ping failed", "provider", name, "err", err)
		return &PingResponse{OK: false, Provider: name, Model: pcfg.Model, Message: err.Error()}, nil
	}
	s.logUsage(userID, "ping", name, resp)
	return &PingResponse{OK: true, Provider: name, Model: resp.Model, Reply: strings.TrimSpace(resp.Content)}, nil
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Provider  string        `json:"provider"`
	Messages  []ChatMessage `json:"messages" binding:"required"`
	ProjectID uint          `json:"project_id"`
	ClusterID uint          `json:"cluster_id"`
}

type ChatResponse struct {
	Reply    string   `json:"reply"`
	Provider string   `json:"provider"`
	Model    string   `json:"model"`
	Usage    llm.Usage `json:"usage"`
}

func (s *Service) Chat(ctx context.Context, userID uint, req ChatRequest) (*ChatResponse, error) {
	cfg, err := s.requireEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.checkRate(userID); err != nil {
		return nil, err
	}
	if len(req.Messages) == 0 {
		return nil, constants.ErrBadRequestWithMsg("messages 不能为空")
	}
	for _, m := range req.Messages {
		if len(m.Content) > 20_000 {
			return nil, constants.ErrBadRequestWithMsg("单条消息过长")
		}
	}

	ctxJSON, _ := json.Marshal(map[string]any{
		"project_id": req.ProjectID,
		"cluster_id": req.ClusterID,
		"note":       "只读运维上下文；无密钥",
	})
	sys, err := prompts.Load("system_ops_assistant", map[string]string{"context_json": string(ctxJSON)})
	if err != nil {
		return nil, err
	}

	msgs := []llm.Message{{Role: "system", Content: sys}}
	for _, m := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if role == "system" {
			continue // 禁止用户覆盖 system
		}
		if role != "assistant" {
			role = "user"
		}
		msgs = append(msgs, llm.Message{Role: role, Content: strings.TrimSpace(m.Content)})
	}

	cli, name, pcfg, err := s.clientFor(cfg, req.Provider)
	if err != nil {
		return nil, err
	}
	resp, err := cli.Chat(ctx, llm.ChatRequest{
		Model:     pcfg.Model,
		Messages:  msgs,
		MaxTokens: cfg.MaxTokens,
	})
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("AI 调用失败: " + err.Error())
	}
	s.logUsage(userID, "chat", name, resp)
	return &ChatResponse{Reply: resp.Content, Provider: name, Model: resp.Model, Usage: resp.Usage}, nil
}

type PodDiagnoseAIRequest struct {
	Provider  string `json:"provider"`
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
}

type PodDiagnoseAIResponse struct {
	Diagnose   *k8s.PodDiagnoseResult `json:"diagnose"`
	AISummary  string                 `json:"ai_summary"`
	RootCauses []map[string]any       `json:"root_causes"`
	Actions    []map[string]any       `json:"actions"`
	RawReply   string                 `json:"raw_reply,omitempty"`
	Provider   string                 `json:"provider"`
	Model      string                 `json:"model"`
}

func (s *Service) AnalyzePodDiagnose(ctx context.Context, userID uint, req PodDiagnoseAIRequest) (*PodDiagnoseAIResponse, error) {
	cfg, err := s.requireEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.checkRate(userID); err != nil {
		return nil, err
	}
	if s.podSvc == nil {
		return nil, constants.ErrBadRequestWithMsg("K8s Pod 服务不可用")
	}
	diag, err := s.podSvc.Diagnose(ctx, k8s.PodDiagnoseQuery{
		ClusterID: req.ClusterID,
		Namespace: strings.TrimSpace(req.Namespace),
		Name:      strings.TrimSpace(req.Name),
	})
	if err != nil {
		return nil, err
	}

	ctxJSON, _ := json.Marshal(diag)
	if len(ctxJSON) > 60_000 {
		ctxJSON = ctxJSON[:60_000]
	}
	prompt, err := prompts.Load("k8s_pod_diagnose", map[string]string{"context_json": string(ctxJSON)})
	if err != nil {
		return nil, err
	}
	cli, name, pcfg, err := s.clientFor(cfg, req.Provider)
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
	s.logUsage(userID, "k8s_pod_diagnose", name, resp)

	out := &PodDiagnoseAIResponse{
		Diagnose:  diag,
		RawReply:  resp.Content,
		Provider:  name,
		Model:     resp.Model,
		AISummary: diag.Summary,
	}
	applyParsedAnalysis(&out.AISummary, &out.RootCauses, &out.Actions, resp.Content)
	return out, nil
}

type CicdBuildFailAIRequest struct {
	Provider  string `json:"provider"`
	ProjectID uint   `json:"project_id" binding:"required"`
	RunID     uint   `json:"run_id" binding:"required"`
}

type CicdBuildFailAIResponse struct {
	Build      *cicdsvc.BuildRunItem `json:"build"`
	AISummary  string                `json:"ai_summary"`
	RootCauses []map[string]any      `json:"root_causes"`
	Actions    []map[string]any      `json:"actions"`
	RawReply   string                `json:"raw_reply,omitempty"`
	Provider   string                `json:"provider"`
	Model      string                `json:"model"`
}

func (s *Service) AnalyzeCicdBuildFail(ctx context.Context, userID uint, actor *auth.CurrentUser, req CicdBuildFailAIRequest) (*CicdBuildFailAIResponse, error) {
	cfg, err := s.requireEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.checkRate(userID); err != nil {
		return nil, err
	}
	if s.cicdSvc == nil {
		return nil, constants.ErrBadRequestWithMsg("CI/CD 服务不可用")
	}
	run, err := s.cicdSvc.GetBuildRun(ctx, req.ProjectID, req.RunID, actor)
	if err != nil {
		return nil, err
	}
	stages, _ := s.cicdSvc.ListBuildRunStages(ctx, req.ProjectID, req.RunID, actor)
	stageBrief := make([]map[string]any, 0, len(stages))
	for _, st := range stages {
		stageBrief = append(stageBrief, map[string]any{
			"stage_name":    st.StageName,
			"stage_type":    st.StageType,
			"status":        st.Status,
			"duration_sec":  st.DurationSec,
			"error_message": st.ErrorMessage,
			"logs_tail":     truncateTail(st.Logs, 2500),
		})
	}
	consoleLog, logErr := s.cicdSvc.GetBuildRunLog(ctx, req.ProjectID, req.RunID, actor)
	logNote := ""
	if logErr != nil {
		logNote = logErr.Error()
		consoleLog = ""
	}
	bundle := map[string]any{
		"build": map[string]any{
			"id":                 run.ID,
			"project_id":         run.ProjectID,
			"service_id":         run.ServiceID,
			"service_name":       run.ServiceName,
			"service_identifier": run.ServiceIdentifier,
			"branch_name":        run.BranchName,
			"version":            run.Version,
			"tenv":               run.Tenv,
			"publish_mode":       run.PublishMode,
			"build_result":       run.BuildResult,
			"build_number":       run.BuildNumber,
			"jenkins_build_url":  run.JenkinsBuildURL,
			"package_path":       run.PackagePath,
			"image_address":      run.ImageAddress,
			"sonar_project_key":  run.SonarProjectKey,
			"started_at":         run.StartedAt,
			"finished_at":        run.FinishedAt,
			"quality_gate":       run.QualityGateStatus,
			"git_commit":         run.GitCommit,
		},
		"stages":            stageBrief,
		"console_log_tail":  truncateTail(consoleLog, 40_000),
		"console_log_error": logNote,
	}
	ctxJSON, _ := json.Marshal(bundle)
	ctxJSON = truncateBytes(ctxJSON, 60_000)

	prompt, err := prompts.Load("cicd_build_fail", map[string]string{"context_json": string(ctxJSON)})
	if err != nil {
		return nil, err
	}
	cli, name, pcfg, err := s.clientFor(cfg, req.Provider)
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
	s.logUsage(userID, "cicd_build_fail", name, resp)

	summary := "构建结果: " + strings.TrimSpace(run.BuildResult)
	for _, st := range stages {
		if (st.Status == "failure" || st.Status == "failed") && strings.TrimSpace(st.ErrorMessage) != "" {
			summary = strings.TrimSpace(st.ErrorMessage)
			break
		}
	}
	out := &CicdBuildFailAIResponse{
		Build:     run,
		RawReply:  resp.Content,
		Provider:  name,
		Model:     resp.Model,
		AISummary: summary,
	}
	applyParsedAnalysis(&out.AISummary, &out.RootCauses, &out.Actions, resp.Content)
	return out, nil
}

type AlertExplainAIRequest struct {
	Provider    string `json:"provider"`
	Fingerprint string `json:"fingerprint" binding:"required"`
	ProjectID   uint   `json:"project_id"`
	WindowHours int    `json:"window_hours"`
}

type AlertExplainAIResponse struct {
	Explain    *alert.FingerprintDeliveryExplain `json:"explain"`
	AISummary  string                            `json:"ai_summary"`
	RootCauses []map[string]any                  `json:"root_causes"`
	Actions    []map[string]any                  `json:"actions"`
	RawReply   string                            `json:"raw_reply,omitempty"`
	Provider   string                            `json:"provider"`
	Model      string                            `json:"model"`
}

func (s *Service) AnalyzeAlertExplain(ctx context.Context, userID uint, req AlertExplainAIRequest) (*AlertExplainAIResponse, error) {
	cfg, err := s.requireEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.checkRate(userID); err != nil {
		return nil, err
	}
	if s.alertSvc == nil {
		return nil, constants.ErrBadRequestWithMsg("告警服务不可用")
	}
	fp := strings.TrimSpace(req.Fingerprint)
	explain, err := s.alertSvc.ExplainFingerprintDelivery(ctx, fp)
	if err != nil {
		return nil, err
	}

	eventsBrief := make([]map[string]any, 0, len(explain.Events))
	for i, ev := range explain.Events {
		if i >= 40 {
			break
		}
		eventsBrief = append(eventsBrief, map[string]any{
			"created_at":    ev.CreatedAt,
			"status":        ev.Status,
			"title":         ev.Title,
			"channel_name":  ev.ChannelName,
			"success":       ev.Success,
			"error_message": ev.ErrorMessage,
			"category":      ev.Category,
			"reason_hint":   ev.ReasonHint,
		})
	}

	bundle := map[string]any{
		"fingerprint":             explain.Fingerprint,
		"firing_delivered":        explain.FiringDelivered,
		"firing_delivered_source": explain.FiringDeliveredSource,
		"skip_summary":            explain.SkipSummary,
		"events":                  eventsBrief,
	}
	window := req.WindowHours
	if window <= 0 {
		window = 24
	}
	if q, qerr := s.alertSvc.QualityReport(ctx, window, req.ProjectID); qerr == nil && q != nil {
		// 仅带与指纹相关的质量摘要，避免整包过大
		repeatHit := false
		for _, it := range q.RepeatFingerprints {
			if it.Fingerprint == fp {
				repeatHit = true
				bundle["quality_repeat"] = it
				break
			}
		}
		bundle["quality"] = map[string]any{
			"window_hours":     q.WindowHours,
			"total_events":     q.TotalEvents,
			"notify_fail_rate": q.NotifyFailRate,
			"quality_score":    q.QualityScore,
			"noise_top":        takeNoiseTop(q.NoiseTop, 5),
			"repeat_matched":   repeatHit,
		}
	}

	ctxJSON, _ := json.Marshal(bundle)
	ctxJSON = truncateBytes(ctxJSON, 60_000)
	prompt, err := prompts.Load("alert_explain", map[string]string{"context_json": string(ctxJSON)})
	if err != nil {
		return nil, err
	}
	cli, name, pcfg, err := s.clientFor(cfg, req.Provider)
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
	s.logUsage(userID, "alert_explain", name, resp)

	summary := "指纹投递追溯"
	if explain.FiringDelivered {
		summary = "已有成功 firing 投递"
	} else if len(explain.SkipSummary) > 0 {
		summary = "尚未成功 firing；存在跳过/失败原因"
	}
	out := &AlertExplainAIResponse{
		Explain:   explain,
		RawReply:  resp.Content,
		Provider:  name,
		Model:     resp.Model,
		AISummary: summary,
	}
	applyParsedAnalysis(&out.AISummary, &out.RootCauses, &out.Actions, resp.Content)
	return out, nil
}

func applyParsedAnalysis(summary *string, causes *[]map[string]any, actions *[]map[string]any, raw string) {
	parsed := parseAIJSON(raw)
	if parsed == nil {
		return
	}
	if v, ok := parsed["ai_summary"].(string); ok && strings.TrimSpace(v) != "" {
		*summary = v
	}
	if arr, ok := parsed["root_causes"].([]any); ok {
		*causes = toMapSlice(arr)
	}
	if arr, ok := parsed["actions"].([]any); ok {
		*actions = toMapSlice(arr)
	}
}

func takeNoiseTop(items []alert.AlertNoiseItem, n int) []alert.AlertNoiseItem {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func truncateTail(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	s = s[len(s)-maxBytes:]
	// 避免截断在 UTF-8 中间
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[1:]
	}
	return "…(truncated)\n" + s
}

func truncateBytes(b []byte, max int) []byte {
	if max <= 0 || len(b) <= max {
		return b
	}
	return b[:max]
}

func parseAIJSON(raw string) map[string]any {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			s = s[i : j+1]
		}
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}

func toMapSlice(arr []any) []map[string]any {
	out := make([]map[string]any, 0, len(arr))
	for _, it := range arr {
		if m, ok := it.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func (s *Service) logUsage(userID uint, scene, provider string, resp *llm.ChatResponse) {
	if resp == nil {
		return
	}
	slog.Info("ai call",
		"user_id", userID,
		"scene", scene,
		"provider", provider,
		"model", resp.Model,
		"prompt_tokens", resp.Usage.PromptTokens,
		"completion_tokens", resp.Usage.CompletionTokens,
		"total_tokens", resp.Usage.TotalTokens,
	)
}
