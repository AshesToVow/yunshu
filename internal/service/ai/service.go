package ai

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/llm"
	"yunshu/internal/service/alert"
	cicdsvc "yunshu/internal/service/cicd"
	"yunshu/internal/service/k8s"
	"yunshu/internal/service/logplatform"

	"gorm.io/gorm"
)

// Service AI 业务服务。
type Service struct {
	db            *gorm.DB
	yamlAI        config.AIConfig
	encryptionKey string
	dataDir       string
	memberRepo    interfaces.ProjectMemberRepository
	accessRepo    interfaces.K8sClusterAccessRepository
	nsDenyRepo    interfaces.K8sNamespaceDenyRepository
	nsAllowRepo   interfaces.K8sNamespaceAllowRepository
	clusterSvc    *k8s.K8sClusterService
	podSvc        *k8s.K8sPodService
	workloadSvc   *k8s.K8sWorkloadService
	nsSvc         *k8s.K8sNamespaceService
	eventSvc      *k8s.K8sEventService
	logSearch     *logplatform.LogSearchService
	esProvider    *logplatform.ElasticsearchProvider
	cicdSvc       *cicdsvc.Service
	alertSvc      *alert.AlertService
	rateMu        sync.Mutex
	rateMap       map[uint]time.Time // 简易限流：每用户最短间隔
	seedOnce      sync.Once
}

func NewService(
	db *gorm.DB,
	yamlAI config.AIConfig,
	encryptionKey string,
	memberRepo interfaces.ProjectMemberRepository,
	accessRepo interfaces.K8sClusterAccessRepository,
	nsDenyRepo interfaces.K8sNamespaceDenyRepository,
	nsAllowRepo interfaces.K8sNamespaceAllowRepository,
	clusterSvc *k8s.K8sClusterService,
	podSvc *k8s.K8sPodService,
	workloadSvc *k8s.K8sWorkloadService,
	nsSvc *k8s.K8sNamespaceService,
	eventSvc *k8s.K8sEventService,
	logSearch *logplatform.LogSearchService,
	esProvider *logplatform.ElasticsearchProvider,
	cicdSvc *cicdsvc.Service,
	alertSvc *alert.AlertService,
) *Service {
	return &Service{
		db:            db,
		yamlAI:        yamlAI,
		encryptionKey: encryptionKey,
		dataDir:       filepath.Join("data", "ai"),
		memberRepo:    memberRepo,
		accessRepo:    accessRepo,
		nsDenyRepo:    nsDenyRepo,
		nsAllowRepo:   nsAllowRepo,
		clusterSvc:    clusterSvc,
		podSvc:        podSvc,
		workloadSvc:   workloadSvc,
		nsSvc:         nsSvc,
		eventSvc:      eventSvc,
		logSearch:     logSearch,
		esProvider:    esProvider,
		cicdSvc:       cicdSvc,
		alertSvc:      alertSvc,
		rateMap:       make(map[uint]time.Time),
	}
}

func (s *Service) ensureSeed() {
	s.seedOnce.Do(func() {
		if err := s.EnsureCenterSeed(context.Background()); err != nil {
			slog.Warn("ai center seed", "err", err)
		}
	})
}

func (s *Service) resolved(ctx context.Context) config.AIConfig {
	return dictconfig.ResolveAIConfig(ctx, s.db, s.yamlAI, dictconfig.DefaultAIDictTypes())
}

func (s *Service) requireEnabled(ctx context.Context) (config.AIConfig, error) {
	cfg := s.resolved(ctx)
	if cfg.Enabled {
		return cfg, nil
	}
	var n int64
	_ = s.db.WithContext(ctx).Model(&model.AiLLMModel{}).Where("enabled = ?", true).Count(&n)
	if n > 0 {
		cfg.Enabled = true
		return cfg, nil
	}
	return cfg, constants.ErrBadRequestWithMsg("AI 未启用：请在能力中心录入模型，或配置字典 ai_enabled=true")
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

func (s *Service) clientFor(ctx context.Context, cfg *config.AIConfig, provider string) (llm.Client, string, config.AIProviderConfig, error) {
	if cfg == nil {
		return nil, "", config.AIProviderConfig{}, constants.ErrBadRequestWithMsg("AI 配置无效")
	}
	row, err := s.findLLMModelForChat(ctx, provider)
	if err == nil && row != nil {
		timeout := cfg.TimeoutSec
		if timeout <= 0 {
			timeout = 60
		}
		if row.MaxTokens > 0 {
			cfg.MaxTokens = row.MaxTokens
		}
		return s.clientFromDBModel(row, timeout)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", config.AIProviderConfig{}, err
	}
	name, pcfg := cfg.ProviderConfig(provider)
	if strings.TrimSpace(pcfg.APIKey) == "" {
		return nil, name, pcfg, constants.ErrBadRequestWithMsg("未配置 API Key：请在能力中心录入模型，或启用对应 ai_*_api_key 字典项")
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
	Enabled         bool             `json:"enabled"`
	DefaultProvider string           `json:"default_provider"`
	TimeoutSec      int              `json:"timeout_sec"`
	MaxTokens       int              `json:"max_tokens"`
	Providers       []ProviderStatus `json:"providers"`
}

type ProviderStatus struct {
	Name       string `json:"name"`
	Configured bool   `json:"configured"`
	BaseURL    string `json:"base_url"`
	Model      string `json:"model"`
	Source     string `json:"source,omitempty"` // db | dict
}

func (s *Service) Status(ctx context.Context) StatusResponse {
	cfg := s.resolved(ctx)
	var providers []ProviderStatus
	var dbRows []model.AiLLMModel
	_ = s.db.WithContext(ctx).Where("enabled = ?", true).Order("is_default DESC, id ASC").Find(&dbRows).Error
	defaultName := cfg.DefaultProvider
	for _, row := range dbRows {
		providers = append(providers, ProviderStatus{
			Name:       row.Name,
			Configured: strings.TrimSpace(row.APIKeyEnc) != "",
			BaseURL:    row.BaseURL,
			Model:      row.ModelName,
			Source:     "db",
		})
		if row.IsDefault {
			defaultName = row.Name
		}
	}
	mk := func(name string, p config.AIProviderConfig) ProviderStatus {
		return ProviderStatus{
			Name:       name,
			Configured: strings.TrimSpace(p.APIKey) != "",
			BaseURL:    p.BaseURL,
			Model:      p.Model,
			Source:     "dict",
		}
	}
	providers = append(providers,
		mk(config.AIProviderOpenAICompat, cfg.OpenAI),
		mk(config.AIProviderDeepSeek, cfg.DeepSeek),
		mk(config.AIProviderAnthropic, cfg.Anthropic),
	)
	enabled := cfg.Enabled || len(dbRows) > 0
	return StatusResponse{
		Enabled:         enabled,
		DefaultProvider: defaultName,
		TimeoutSec:      cfg.TimeoutSec,
		MaxTokens:       cfg.MaxTokens,
		Providers:       providers,
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
	cli, name, pcfg, err := s.clientFor(ctx, &cfg, req.Provider)
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
	Provider    string        `json:"provider"`
	Messages    []ChatMessage `json:"messages" binding:"required"`
	SessionID   uint          `json:"session_id"`
	ProjectID   uint          `json:"project_id"`
	ClusterID   uint          `json:"cluster_id"`
	Namespace   string        `json:"namespace"`
	EnableTools *bool         `json:"enable_tools"`
	EnableWrite bool          `json:"enable_write_tools"`
	DisableRAG  bool          `json:"disable_rag"`
}

type ChatResponse struct {
	Reply     string     `json:"reply"`
	Provider  string     `json:"provider"`
	Model     string     `json:"model"`
	Usage     llm.Usage  `json:"usage"`
	SessionID uint       `json:"session_id,omitempty"`
	ToolSteps []toolStep `json:"tool_steps,omitempty"`
	RAGHits   []ragHit   `json:"rag_hits,omitempty"`
}

func (s *Service) Chat(ctx context.Context, userID uint, actor *auth.CurrentUser, req ChatRequest) (*ChatResponse, error) {
	cfg, err := s.requireEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.checkRate(userID); err != nil {
		return nil, err
	}
	actor = resolveActor(ctx, actor)
	if actor == nil && userID > 0 {
		return nil, constants.ErrUnauthorized
	}
	if req.ProjectID > 0 {
		if err := s.assertProjectMember(ctx, actor, req.ProjectID); err != nil {
			return nil, err
		}
	}
	if len(req.Messages) == 0 {
		return nil, constants.ErrBadRequestWithMsg("messages 不能为空")
	}
	for _, m := range req.Messages {
		if len(m.Content) > 20_000 {
			return nil, constants.ErrBadRequestWithMsg("单条消息过长")
		}
	}
	if req.SessionID > 0 {
		if _, err := s.getOwnedSession(ctx, userID, req.SessionID); err != nil {
			return nil, err
		}
	}

	enableTools := true
	if req.EnableTools != nil {
		enableTools = *req.EnableTools
	}

	lastUser := ""
	for _, v := range slices.Backward(req.Messages) {
		if strings.EqualFold(v.Role, "user") {
			lastUser = v.Content
			break
		}
	}

	ctxJSON, _ := json.Marshal(map[string]any{
		"project_id": req.ProjectID,
		"cluster_id": req.ClusterID,
		"namespace":  req.Namespace,
		"session_id": req.SessionID,
		"note":       "优先使用工具获取真实平台/集群数据；写操作仅创建审批单",
	})
	s.ensureSeed()
	sys, err := s.loadPromptContent(ctx, "system/ops-agent", map[string]string{"context_json": string(ctxJSON)})
	if err != nil {
		return nil, err
	}

	var ragHits []ragHit
	if !req.DisableRAG {
		ragHits = s.retrieveKnowledge(ctx, lastUser, 8)
		if len(ragHits) > 0 {
			var b strings.Builder
			b.WriteString("\n\n## 知识库检索片段\n")
			for _, h := range ragHits {
				b.WriteString("- [")
				b.WriteString(h.Source)
				if h.Module != "" {
					b.WriteString("|")
					b.WriteString(h.Module)
				}
				b.WriteString("] ")
				b.WriteString(truncateStr(h.Content, 800))
				b.WriteString("\n")
			}
			sys += b.String()
		}
	}

	msgs := []llm.Message{{Role: "system", Content: sys}}
	for _, m := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if role == "system" || role == "tool" {
			continue
		}
		if role != "assistant" {
			role = "user"
		}
		msgs = append(msgs, llm.Message{Role: role, Content: strings.TrimSpace(m.Content)})
	}

	cli, name, pcfg, err := s.clientFor(ctx, &cfg, req.Provider)
	if err != nil {
		return nil, err
	}

	tools := []llm.ToolDefinition(nil)
	if enableTools {
		tools = s.toolDefinitions(req.EnableWrite)
	}
	tc := toolContext{
		ClusterID: req.ClusterID,
		ProjectID: req.ProjectID,
		Namespace: strings.TrimSpace(req.Namespace),
		Actor:     actor,
	}
	var steps []toolStep
	var last *llm.ChatResponse
	usage := llm.Usage{}

	finish := func(reply, model string) (*ChatResponse, error) {
		sid, perr := s.persistChatTurn(ctx, userID, req, lastUser, reply, steps, ragHits, name, model)
		if perr != nil {
			slog.Warn("ai chat persist failed", "user_id", userID, "err", perr)
		}
		return &ChatResponse{
			Reply:     reply,
			Provider:  name,
			Model:     model,
			Usage:     usage,
			SessionID: sid,
			ToolSteps: steps,
			RAGHits:   ragHits,
		}, nil
	}

	const maxRounds = 6
	for range maxRounds {
		resp, err := cli.Chat(ctx, llm.ChatRequest{
			Model:     pcfg.Model,
			Messages:  msgs,
			MaxTokens: cfg.MaxTokens,
			Tools:     tools,
		})
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg("AI 调用失败: " + err.Error())
		}
		last = resp
		usage.PromptTokens += resp.Usage.PromptTokens
		usage.CompletionTokens += resp.Usage.CompletionTokens
		usage.TotalTokens += resp.Usage.TotalTokens

		if len(resp.ToolCalls) == 0 || !enableTools {
			s.logUsage(userID, "chat", name, resp)
			return finish(resp.Content, resp.Model)
		}

		msgs = append(msgs, llm.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})
		for _, call := range resp.ToolCalls {
			step := s.executeTool(ctx, userID, call.Function.Name, call.Function.Arguments, tc)
			steps = append(steps, step)
			msgs = append(msgs, llm.Message{
				Role:       "tool",
				Content:    step.Result,
				ToolCallID: call.ID,
				Name:       call.Function.Name,
			})
		}
	}
	if last == nil {
		return nil, constants.ErrBadRequestWithMsg("AI 无响应")
	}
	s.logUsage(userID, "chat", name, last)
	return finish(last.Content, last.Model)
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
	actor := resolveActor(ctx, nil)
	if actor == nil || actor.ID == 0 {
		return nil, constants.ErrUnauthorized
	}
	ns := strings.TrimSpace(req.Namespace)
	if err := s.assertK8sClusterAccess(ctx, actor, req.ClusterID, ns, k8s.K8sAccessRankReadonly); err != nil {
		return nil, err
	}
	ctx = withActorContext(ctx, actor)
	diag, err := s.podSvc.Diagnose(ctx, k8s.PodDiagnoseQuery{
		ClusterID: req.ClusterID,
		Namespace: ns,
		Name:      strings.TrimSpace(req.Name),
	})
	if err != nil {
		return nil, err
	}

	ctxJSON, _ := json.Marshal(diag)
	if len(ctxJSON) > 60_000 {
		ctxJSON = ctxJSON[:60_000]
	}
	prompt, err := s.loadPromptContent(ctx, "diagnosis/k8s-pod", map[string]string{"context_json": string(ctxJSON)})
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

	prompt, err := s.loadPromptContent(ctx, "diagnosis/cicd-build-fail", map[string]string{"context_json": string(ctxJSON)})
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
	prompt, err := s.loadPromptContent(ctx, "diagnosis/alert-explain", map[string]string{"context_json": string(ctxJSON)})
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
