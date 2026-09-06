package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/llm"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/service/alert"
	cmdbsvc "yunshu/internal/service/cmdb"
	"yunshu/internal/service/k8s"
	"yunshu/internal/service/logplatform"
	projectsvc "yunshu/internal/service/project"
)

// InvestigationReport 结构化调查报告。
type InvestigationReport struct {
	Summary    string           `json:"summary"`
	RootCauses []map[string]any `json:"root_causes"`
	Actions    []map[string]any `json:"actions"`
	Evidence   []map[string]any `json:"evidence,omitempty"`
	Provider   string           `json:"provider,omitempty"`
	Model      string           `json:"model,omitempty"`
	RawReply   string           `json:"raw_reply,omitempty"`
}

type StartInvestigationRequest struct {
	Kind        string `json:"kind" binding:"required"` // alert|pod|cicd|chat|incident
	Title       string `json:"title"`
	Provider    string `json:"provider"`
	ProjectID   uint   `json:"project_id"`
	ClusterID   uint   `json:"cluster_id"`
	Namespace   string `json:"namespace"`
	Resource    string `json:"resource"` // pod name / run id as string / server id
	Fingerprint string `json:"fingerprint"`
	RunID       uint   `json:"run_id"`
	ServerID    uint   `json:"server_id"`
	Keyword     string `json:"keyword"` // incident/log keyword
	SessionID   *uint  `json:"session_id"`
	Query       string `json:"query"` // chat kind
}

type InvestigationListQuery struct {
	Kind     string `form:"kind"`
	Status   string `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

func (s *Service) StartInvestigation(
	ctx context.Context,
	userID uint,
	actor *auth.CurrentUser,
	req StartInvestigationRequest,
) (*model.AiInvestigation, error) {
	if _, err := s.requireEnabled(ctx); err != nil {
		return nil, err
	}
	if err := s.checkRate(userID); err != nil {
		return nil, err
	}
	actor = resolveActor(ctx, actor)
	if actor == nil || actor.ID == 0 {
		return nil, constants.ErrUnauthorized
	}
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	switch kind {
	case "alert", "pod", "cicd", "chat", "incident":
	default:
		return nil, constants.ErrBadRequestWithMsg("kind 须为 alert|pod|cicd|chat|incident")
	}
	if req.ProjectID > 0 {
		if err := s.assertProjectMember(ctx, actor, req.ProjectID); err != nil {
			return nil, err
		}
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = defaultInvestigationTitle(kind, req)
	}
	input, _ := json.Marshal(req)
	row := model.AiInvestigation{
		UserID:      userID,
		Kind:        kind,
		Title:       truncateStr(title, 250),
		Status:      "collecting",
		ProjectID:   req.ProjectID,
		ClusterID:   req.ClusterID,
		Namespace:   strings.TrimSpace(req.Namespace),
		Resource:    strings.TrimSpace(req.Resource),
		Fingerprint: strings.TrimSpace(req.Fingerprint),
		InputJSON:   string(input),
		SessionID:   req.SessionID,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}

	fail := func(msg string) (*model.AiInvestigation, error) {
		row.Status = "failed"
		row.ErrorMsg = truncateStr(msg, 1000)
		_ = s.db.WithContext(ctx).Save(&row).Error
		return &row, constants.ErrBadRequestWithMsg(msg)
	}

	collect, evidence, err := s.collectInvestigation(ctx, userID, actor, kind, req)
	if err != nil {
		return fail("采集失败: " + err.Error())
	}
	collectRaw, _ := json.Marshal(collect)
	row.CollectJSON = scrubNonBMPForMySQL(string(truncateBytes(collectRaw, 500_000)))
	row.Status = "analyzing"
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return fail("保存采集失败: " + err.Error())
	}

	query := investigationQuery(kind, req, collect)
	ragHits := s.retrieveKnowledge(ctx, query, 8)
	ragEvidence := make([]map[string]any, 0, len(ragHits))
	for _, h := range ragHits {
		ragEvidence = append(ragEvidence, map[string]any{
			"source": h.Source, "module": h.Module, "score": h.Score,
			"content": truncateStr(h.Content, 400),
		})
	}
	evidence = append(evidence, ragEvidence...)

	report, err := s.analyzeInvestigation(ctx, userID, actor, kind, req, collect, ragHits)
	if err != nil {
		return fail("分析失败: " + err.Error())
	}
	report.Evidence = evidence
	analysisRaw, _ := json.Marshal(map[string]any{
		"summary":     report.Summary,
		"root_causes": report.RootCauses,
		"actions":     report.Actions,
		"provider":    report.Provider,
		"model":       report.Model,
	})
	reportRaw, _ := json.Marshal(report)
	row.AnalysisJSON = scrubNonBMPForMySQL(string(analysisRaw))
	row.ReportJSON = scrubNonBMPForMySQL(string(reportRaw))
	row.Status = "done"
	row.UpdatedAt = time.Now()
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return fail("保存报告失败: " + err.Error())
	}
	return &row, nil
}

func (s *Service) ListInvestigations(
	ctx context.Context,
	userID uint,
	q InvestigationListQuery,
) (*pagination.Result[model.AiInvestigation], error) {
	if userID == 0 {
		return nil, constants.ErrUnauthorized
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	db := s.db.WithContext(ctx).Model(&model.AiInvestigation{}).Where("user_id = ?", userID)
	if k := strings.TrimSpace(q.Kind); k != "" {
		db = db.Where("kind = ?", k)
	}
	if st := strings.TrimSpace(q.Status); st != "" {
		db = db.Where("status = ?", st)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []model.AiInvestigation
	if err := db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return &pagination.Result[model.AiInvestigation]{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) GetInvestigation(ctx context.Context, userID, id uint) (*model.AiInvestigation, error) {
	if userID == 0 {
		return nil, constants.ErrUnauthorized
	}
	var row model.AiInvestigation
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&row).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	return &row, nil
}

func defaultInvestigationTitle(kind string, req StartInvestigationRequest) string {
	switch kind {
	case "alert":
		return "告警调查 " + strings.TrimSpace(req.Fingerprint)
	case "pod":
		return "Pod 调查 " + strings.TrimSpace(req.Namespace) + "/" + strings.TrimSpace(req.Resource)
	case "cicd":
		return "CI 构建调查"
	case "incident":
		t := strings.TrimSpace(req.Title)
		if t == "" {
			t = strings.TrimSpace(req.Keyword)
		}
		if t == "" {
			t = strings.TrimSpace(req.Query)
		}
		if t == "" {
			return "综合故障调查"
		}
		return "综合调查: " + truncateStr(t, 80)
	default:
		q := strings.TrimSpace(req.Query)
		if q == "" {
			return "对话调查"
		}
		return "对话调查: " + truncateStr(q, 80)
	}
}

func investigationQuery(kind string, req StartInvestigationRequest, collect map[string]any) string {
	parts := []string{kind, req.Title, req.Fingerprint, req.Namespace, req.Resource, req.Query}
	if collect != nil {
		if v, ok := collect["summary"].(string); ok {
			parts = append(parts, v)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func (s *Service) collectInvestigation(
	ctx context.Context,
	userID uint,
	actor *auth.CurrentUser,
	kind string,
	req StartInvestigationRequest,
) (map[string]any, []map[string]any, error) {
	evidence := []map[string]any{}
	switch kind {
	case "alert":
		fp := strings.TrimSpace(req.Fingerprint)
		if fp == "" {
			return nil, nil, constants.ErrBadRequestWithMsg("alert 调查需要 fingerprint")
		}
		if s.alertSvc == nil {
			return nil, nil, constants.ErrBadRequestWithMsg("告警服务不可用")
		}
		events, total, _, _, err := s.alertSvc.ListEvents(ctx, alert.AlertEventListQuery{
			Fingerprint: fp,
			ProjectID:   req.ProjectID,
			Page:        1,
			PageSize:    10,
		})
		if err != nil {
			return nil, nil, err
		}
		explain, err := s.alertSvc.ExplainFingerprintDelivery(ctx, fp)
		if err != nil {
			return nil, nil, err
		}
		var logSummary any
		if req.ProjectID > 0 && s.logSearch != nil {
			kw := strings.TrimSpace(req.Keyword)
			if kw == "" && len(events) > 0 {
				kw = strings.TrimSpace(events[0].Title)
			}
			if kw != "" {
				if res, e := s.logSearch.Search(ctx, logplatform.LogSearchQuery{
					ProjectID: req.ProjectID,
					Keyword:   kw,
					Namespace: strings.TrimSpace(req.Namespace),
					Page:      1,
					PageSize:  30,
				}); e == nil {
					logSummary = summarizeLogHitsForTool(res, logplatform.LogSearchQuery{Keyword: kw, ProjectID: req.ProjectID})
					evidence = append(evidence, map[string]any{"type": "logs", "keyword": kw})
				}
			}
		}
		bundle := map[string]any{
			"fingerprint":             explain.Fingerprint,
			"events_total":            total,
			"events":                  events,
			"firing_delivered":        explain.FiringDelivered,
			"firing_delivered_source": explain.FiringDeliveredSource,
			"skip_summary":            explain.SkipSummary,
			"delivery_events_count":   len(explain.Events),
			"log_summary":             logSummary,
			"summary":                 fmt.Sprintf("告警指纹 %s，事件 %d 条", fp, total),
		}
		evidence = append(evidence, map[string]any{
			"type": "alert_detail", "fingerprint": fp, "events_total": total,
		})
		evidence = append(evidence, map[string]any{
			"type": "alert_explain", "fingerprint": fp, "firing_delivered": explain.FiringDelivered,
		})

		// 变更时间线：告警前后 2 小时
		if req.ProjectID > 0 && s.changeEventSvc != nil {
			to := time.Now()
			from := to.Add(-2 * time.Hour)
			if len(events) > 0 && !events[0].CreatedAt.IsZero() {
				from = events[0].CreatedAt.Add(-2 * time.Hour)
				to = events[0].CreatedAt.Add(30 * time.Minute)
				if to.After(time.Now()) {
					to = time.Now()
				}
			}
			if ch, e := s.changeEventSvc.List(ctx, projectsvc.ChangeEventListQuery{
				ProjectID: req.ProjectID,
				From:      from.Format(time.RFC3339),
				To:        to.Format(time.RFC3339),
				Page:      1,
				PageSize:  30,
			}); e == nil && ch != nil {
				bundle["recent_changes"] = ch.List
				bundle["recent_changes_total"] = ch.Total
				evidence = append(evidence, map[string]any{
					"type": "change_events", "total": ch.Total, "window": map[string]string{
						"from": from.Format(time.RFC3339), "to": to.Format(time.RFC3339),
					},
				})
			}
		}

		// 可选：指定 server_id 或从告警标题/载荷猜测主机后 SSH 只读探测
		serverID := req.ServerID
		if serverID == 0 && req.ProjectID > 0 && s.cmdbSvc != nil && len(events) > 0 {
			hint := events[0].Title + " " + events[0].Cluster + " " + events[0].RequestPayload
			serverID = s.matchServerFromAlertHint(ctx, req.ProjectID, hint)
		}
		if serverID > 0 && req.ProjectID > 0 && s.cmdbSvc != nil {
			if probe, e := s.cmdbSvc.ProbeHostMetrics(ctx, cmdbsvc.HostProbeRequest{
				ProjectID: req.ProjectID,
				ServerID:  serverID,
				Kind:      cmdbsvc.HostProbeAll,
				Actor:     actor,
			}); e == nil {
				bundle["host_probe"] = probe
				evidence = append(evidence, map[string]any{"type": "host_probe", "server_id": serverID})
			} else {
				bundle["host_probe_error"] = e.Error()
			}
		}

		bundle["recommended_actions"] = []map[string]any{
			{"action": "silence", "hint": "可在告警页快捷静默，或助手调用 create_alert_silence", "fingerprint": fp},
			{"action": "check_changes", "hint": "核对 recent_changes 是否与告警同源"},
			{"action": "conclude", "hint": "调查完成后在 AI 调查记录中归档结论"},
		}
		return bundle, evidence, nil
	case "incident":
		return s.collectIncidentInvestigation(ctx, actor, req)
	case "pod":
		ns := strings.TrimSpace(req.Namespace)
		name := strings.TrimSpace(req.Resource)
		if req.ClusterID == 0 || ns == "" || name == "" {
			return nil, nil, constants.ErrBadRequestWithMsg("pod 调查需要 cluster_id/namespace/resource")
		}
		if s.podSvc == nil {
			return nil, nil, constants.ErrBadRequestWithMsg("K8s Pod 服务不可用")
		}
		if err := s.assertK8sClusterAccess(ctx, actor, req.ClusterID, ns, k8s.K8sAccessRankReadonly); err != nil {
			return nil, nil, err
		}
		ctx = withActorContext(ctx, actor)
		diag, err := s.podSvc.Diagnose(ctx, k8s.PodDiagnoseQuery{
			ClusterID: req.ClusterID, Namespace: ns, Name: name,
		})
		if err != nil {
			return nil, nil, err
		}
		bundle := map[string]any{"diagnose": diag, "summary": diag.Summary}
		evidence = append(evidence, map[string]any{
			"type": "pod_diagnose", "cluster_id": req.ClusterID, "namespace": ns, "name": name, "summary": diag.Summary,
		})
		return bundle, evidence, nil
	case "cicd":
		if req.ProjectID == 0 || req.RunID == 0 {
			return nil, nil, constants.ErrBadRequestWithMsg("cicd 调查需要 project_id/run_id")
		}
		if s.cicdSvc == nil {
			return nil, nil, constants.ErrBadRequestWithMsg("CI/CD 服务不可用")
		}
		run, err := s.cicdSvc.GetBuildRun(ctx, req.ProjectID, req.RunID, actor)
		if err != nil {
			return nil, nil, err
		}
		stages, _ := s.cicdSvc.ListBuildRunStages(ctx, req.ProjectID, req.RunID, actor)
		logTail, _ := s.cicdSvc.GetBuildRunLog(ctx, req.ProjectID, req.RunID, actor)
		bundle := map[string]any{
			"build":            run,
			"stages":           stages,
			"console_log_tail": truncateTail(logTail, 20_000),
			"summary":          "构建结果: " + strings.TrimSpace(run.BuildResult),
		}
		evidence = append(evidence, map[string]any{
			"type": "cicd_build", "project_id": req.ProjectID, "run_id": req.RunID, "result": run.BuildResult,
		})
		_ = userID
		return bundle, evidence, nil
	default: // chat
		q := strings.TrimSpace(req.Query)
		if q == "" {
			q = strings.TrimSpace(req.Title)
		}
		bundle := map[string]any{
			"query":      q,
			"project_id": req.ProjectID,
			"cluster_id": req.ClusterID,
			"namespace":  req.Namespace,
			"summary":    q,
		}
		evidence = append(evidence, map[string]any{"type": "chat_query", "query": truncateStr(q, 200)})
		return bundle, evidence, nil
	}
}

func (s *Service) collectIncidentInvestigation(
	ctx context.Context,
	actor *auth.CurrentUser,
	req StartInvestigationRequest,
) (map[string]any, []map[string]any, error) {
	if req.ProjectID == 0 {
		return nil, nil, constants.ErrBadRequestWithMsg("incident 调查需要 project_id")
	}
	if err := s.assertProjectMember(ctx, actor, req.ProjectID); err != nil {
		return nil, nil, err
	}
	evidence := []map[string]any{}
	bundle := map[string]any{
		"project_id": req.ProjectID,
		"cluster_id": req.ClusterID,
		"namespace":  strings.TrimSpace(req.Namespace),
		"keyword":    strings.TrimSpace(req.Keyword),
		"query":      strings.TrimSpace(req.Query),
	}
	parts := []string{"综合采集"}

	// 主机
	if s.cmdbSvc != nil {
		if req.ServerID > 0 {
			if r, err := s.cmdbSvc.TestServerConnectivity(ctx, cmdbsvc.ServerTestRequest{ServerID: req.ServerID}); err == nil {
				bundle["server_probe"] = r
				evidence = append(evidence, map[string]any{"type": "server_probe", "server_id": req.ServerID, "ok": r.OK})
				parts = append(parts, fmt.Sprintf("主机探测 ok=%v", r.OK))
			}
			if sv, err := s.cmdbSvc.GetServer(ctx, req.ServerID); err == nil {
				bundle["server"] = sv
			}
		} else {
			if list, err := s.cmdbSvc.ListServers(ctx, cmdbsvc.ServerListQuery{
				ProjectID: req.ProjectID, Keyword: strings.TrimSpace(req.Keyword), Page: 1, PageSize: 10, Actor: actor,
			}); err == nil {
				bundle["servers"] = list
				cnt := 0
				if list != nil {
					cnt = int(list.Total)
				}
				evidence = append(evidence, map[string]any{"type": "servers", "count": cnt})
			}
		}
	}

	// 日志
	kw := strings.TrimSpace(req.Keyword)
	if kw == "" {
		kw = strings.TrimSpace(req.Query)
	}
	if kw == "" {
		kw = strings.TrimSpace(req.Title)
	}
	if s.logSearch != nil && kw != "" {
		lq := logplatform.LogSearchQuery{
			ProjectID: req.ProjectID,
			Keyword:   kw,
			Namespace: strings.TrimSpace(req.Namespace),
			Page:      1,
			PageSize:  40,
		}
		if req.ClusterID > 0 {
			cid := req.ClusterID
			lq.ClusterID = &cid
		}
		if res, err := s.logSearch.Search(ctx, lq); err == nil {
			bundle["log_summary"] = summarizeLogHitsForTool(res, logplatform.LogSearchQuery{Keyword: kw, ProjectID: req.ProjectID})
			evidence = append(evidence, map[string]any{"type": "logs", "keyword": kw})
			parts = append(parts, "已采集日志摘要")
		}
	}

	// 告警
	if s.alertSvc != nil {
		if fp := strings.TrimSpace(req.Fingerprint); fp != "" {
			if events, total, _, _, err := s.alertSvc.ListEvents(ctx, alert.AlertEventListQuery{
				Fingerprint: fp, ProjectID: req.ProjectID, Page: 1, PageSize: 10,
			}); err == nil {
				bundle["alert_events"] = events
				bundle["alert_events_total"] = total
				evidence = append(evidence, map[string]any{"type": "alert_detail", "fingerprint": fp, "total": total})
				parts = append(parts, fmt.Sprintf("告警事件 %d", total))
			}
			if explain, err := s.alertSvc.ExplainFingerprintDelivery(ctx, fp); err == nil {
				bundle["alert_delivery"] = map[string]any{
					"firing_delivered": explain.FiringDelivered,
					"skip_summary":     explain.SkipSummary,
				}
			}
		} else {
			if events, total, _, _, err := s.alertSvc.ListEvents(ctx, alert.AlertEventListQuery{
				ProjectID: req.ProjectID, Keyword: kw, Status: "firing", Page: 1, PageSize: 15,
			}); err == nil {
				bundle["recent_firing_alerts"] = events
				bundle["recent_firing_total"] = total
				evidence = append(evidence, map[string]any{"type": "alerts_firing", "total": total})
				parts = append(parts, fmt.Sprintf("firing 告警 %d", total))
			}
		}
	}

	bundle["summary"] = strings.Join(parts, "；")
	return bundle, evidence, nil
}

func (s *Service) analyzeInvestigation(
	ctx context.Context,
	userID uint,
	_ *auth.CurrentUser,
	kind string,
	req StartInvestigationRequest,
	collect map[string]any,
	ragHits []ragHit,
) (*InvestigationReport, error) {
	cfg, err := s.requireEnabled(ctx)
	if err != nil {
		return nil, err
	}
	promptCode := ""
	summaryFallback := ""
	var ctxPayload any
	switch kind {
	case "alert":
		// 含事件详情 + 可选日志时走综合处置；仅投递解释场景仍可用 alert-explain
		if _, hasEvents := collect["events"]; hasEvents {
			promptCode = "diagnosis/incident-remediate"
		} else {
			promptCode = "diagnosis/alert-explain"
		}
		summaryFallback = "告警调查"
		ctxPayload = collect
	case "incident":
		promptCode = "diagnosis/incident-remediate"
		summaryFallback = "综合故障调查"
		ctxPayload = collect
		if v, ok := collect["summary"].(string); ok {
			summaryFallback = v
		}
	case "pod":
		promptCode = "diagnosis/k8s-pod"
		if d, ok := collect["diagnose"]; ok {
			ctxPayload = d
		} else {
			ctxPayload = collect
		}
		if v, ok := collect["summary"].(string); ok {
			summaryFallback = v
		}
	case "cicd":
		promptCode = "diagnosis/cicd-build-fail"
		ctxPayload = collect
		if v, ok := collect["summary"].(string); ok {
			summaryFallback = v
		}
	default:
		return s.analyzeChatInvestigation(ctx, userID, req, collect, ragHits)
	}
	ctxJSON, _ := json.Marshal(ctxPayload)
	ctxJSON = truncateBytes(ctxJSON, 60_000)
	if len(ragHits) > 0 {
		var b strings.Builder
		b.Write(ctxJSON)
		b.WriteString("\n\n## RAG\n")
		for _, h := range ragHits {
			b.WriteString("- ")
			b.WriteString(truncateStr(h.Content, 400))
			b.WriteString("\n")
		}
		ctxJSON = truncateBytes([]byte(b.String()), 60_000)
	}
	prompt, err := s.loadPromptContent(ctx, promptCode, map[string]string{"context_json": string(ctxJSON)})
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
	s.logUsage(userID, "investigation_"+kind, name, resp)
	out := &InvestigationReport{
		Summary:  summaryFallback,
		Provider: name,
		Model:    resp.Model,
		RawReply: resp.Content,
	}
	applyParsedAnalysis(&out.Summary, &out.RootCauses, &out.Actions, resp.Content)
	if kind == "alert" {
		if rec, ok := collect["recommended_actions"].([]map[string]any); ok && len(rec) > 0 {
			out.Actions = append(out.Actions, rec...)
		}
	}
	return out, nil
}

func (s *Service) analyzeChatInvestigation(
	ctx context.Context,
	userID uint,
	req StartInvestigationRequest,
	collect map[string]any,
	ragHits []ragHit,
) (*InvestigationReport, error) {
	cfg, err := s.requireEnabled(ctx)
	if err != nil {
		return nil, err
	}
	bundle := map[string]any{"collect": collect, "rag": ragHits}
	ctxJSON, _ := json.Marshal(bundle)
	ctxJSON = truncateBytes(ctxJSON, 60_000)
	prompt := "你是运维助手。根据以下采集与知识检索，输出 JSON：" +
		`{"ai_summary":"...","root_causes":[{"cause":"...","confidence":0.8}],"actions":[{"action":"...","priority":"high"}]}` +
		"\n\n" + string(ctxJSON)
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
	s.logUsage(userID, "investigation_chat", name, resp)
	summary := "对话调查完成"
	if v, ok := collect["summary"].(string); ok && v != "" {
		summary = truncateStr(v, 200)
	}
	out := &InvestigationReport{
		Summary:  summary,
		Provider: name,
		Model:    resp.Model,
		RawReply: resp.Content,
	}
	applyParsedAnalysis(&out.Summary, &out.RootCauses, &out.Actions, resp.Content)
	return out, nil
}

// matchServerFromAlertHint 从告警标题/集群名/载荷中匹配 CMDB 主机（host 或 name 子串）。
func (s *Service) matchServerFromAlertHint(ctx context.Context, projectID uint, hint string) uint {
	hint = strings.ToLower(strings.TrimSpace(hint))
	if projectID == 0 || hint == "" || s.cmdbSvc == nil {
		return 0
	}
	res, err := s.cmdbSvc.ListServers(ctx, cmdbsvc.ServerListQuery{
		ProjectID: projectID, Page: 1, PageSize: 100,
	})
	if err != nil || res == nil {
		return 0
	}
	for _, sv := range res.List {
		host := strings.ToLower(strings.TrimSpace(sv.Host))
		name := strings.ToLower(strings.TrimSpace(sv.Name))
		if host != "" && strings.Contains(hint, host) {
			return sv.ID
		}
		if name != "" && len(name) >= 3 && strings.Contains(hint, name) {
			return sv.ID
		}
	}
	return 0
}
