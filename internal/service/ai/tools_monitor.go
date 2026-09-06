package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/llm"
	"yunshu/internal/service/alert"
)

func (s *Service) monitorToolDefinitions() []llm.ToolDefinition {
	return []llm.ToolDefinition{
		llm.NewFunctionTool("list_alert_datasources",
			"列出项目 Prometheus/监控数据源（id/名称/类型）。查询监控前必须先调用以拿到 datasource_id。需要 project_id。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer", "description": "必填"},
					"keyword":    map[string]any{"type": "string"},
					"page_size":  map[string]any{"type": "integer", "description": "默认 20，最大 50"},
				},
				"required": []string{"project_id"},
			}),
		llm.NewFunctionTool("query_prometheus",
			"对指定数据源执行 PromQL 即时查询（只读）。用于确认 CPU/内存/错误率等指标现状。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"datasource_id": map[string]any{"type": "integer", "description": "来自 list_alert_datasources"},
					"query":         map[string]any{"type": "string", "description": "PromQL，如 up{job=\"node\"}"},
					"time":          map[string]any{"type": "string", "description": "可选 RFC3339 或 unix 秒"},
				},
				"required": []string{"datasource_id", "query"},
			}),
		llm.NewFunctionTool("query_prometheus_range",
			"对指定数据源执行 PromQL 区间查询（只读）。用于看趋势，step 如 1m/5m。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"datasource_id": map[string]any{"type": "integer"},
					"query":         map[string]any{"type": "string"},
					"start":         map[string]any{"type": "string", "description": "开始时间 RFC3339 或 unix"},
					"end":           map[string]any{"type": "string", "description": "结束时间"},
					"step":          map[string]any{"type": "string", "description": "如 1m、5m"},
				},
				"required": []string{"datasource_id", "query", "start", "end", "step"},
			}),
		llm.NewFunctionTool("list_prometheus_active_alerts",
			"拉取 Prometheus 数据源当前 active alerts（只读）。用于对照「监控里正在 firing 的规则」。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"datasource_id": map[string]any{"type": "integer"},
				},
				"required": []string{"datasource_id"},
			}),
		llm.NewFunctionTool("get_alert_detail",
			"按 fingerprint 拉取平台告警事件详情（标题/级别/标签/状态/最近投递）。排查「告警触发了什么、怎么处理」时先调此工具，再结合日志/监控。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"fingerprint": map[string]any{"type": "string"},
					"project_id":  map[string]any{"type": "integer", "description": "可选，缩小范围"},
					"page_size":   map[string]any{"type": "integer", "description": "默认 10，最大 30"},
				},
				"required": []string{"fingerprint"},
			}),
	}
}

func (s *Service) executeMonitorTool(
	ctx context.Context,
	name string,
	getUint func(string, uint) uint,
	getStr func(string) string,
	projectID uint,
	actor *auth.CurrentUser,
	requireProject, requireActor func() error,
) (any, error) {
	_ = actor
	switch name {
	case "list_alert_datasources":
		if err := requireProject(); err != nil {
			return nil, err
		}
		if s.alertDSSvc == nil {
			return nil, fmt.Errorf("监控数据源服务不可用")
		}
		ps := int(getUint("page_size", 20))
		if ps <= 0 || ps > 50 {
			ps = 20
		}
		list, total, page, pageSize, err := s.alertDSSvc.List(ctx, alert.AlertDatasourceListQuery{
			ProjectID: projectID,
			Keyword:   getStr("keyword"),
			Page:      1,
			PageSize:  ps,
		})
		if err != nil {
			return nil, err
		}
		type item struct {
			ID        uint   `json:"id"`
			Name      string `json:"name"`
			Type      string `json:"type"`
			BaseURL   string `json:"base_url"`
			Enabled   bool   `json:"enabled"`
			ProjectID uint   `json:"project_id"`
			Remark    string `json:"remark,omitempty"`
		}
		out := make([]item, 0, len(list))
		for _, d := range list {
			out = append(out, item{
				ID: d.ID, Name: d.Name, Type: d.Type, BaseURL: d.BaseURL,
				Enabled: d.Enabled, ProjectID: d.ProjectID, Remark: d.Remark,
			})
		}
		return map[string]any{"list": out, "total": total, "page": page, "page_size": pageSize}, nil

	case "query_prometheus":
		if err := requireActor(); err != nil {
			return nil, err
		}
		if s.alertDSSvc == nil {
			return nil, fmt.Errorf("监控数据源服务不可用")
		}
		id := getUint("datasource_id", 0)
		q := strings.TrimSpace(getStr("query"))
		if id == 0 || q == "" {
			return nil, fmt.Errorf("datasource_id 与 query 必填")
		}
		if err := s.assertDatasourceProjectAccess(ctx, actor, id, projectID); err != nil {
			return nil, err
		}
		raw, err := s.alertDSSvc.PromQuery(ctx, id, alert.PromQueryRequest{Query: q, Time: getStr("time")})
		if err != nil {
			return nil, err
		}
		return truncatePromJSON(raw, 20_000), nil

	case "query_prometheus_range":
		if err := requireActor(); err != nil {
			return nil, err
		}
		if s.alertDSSvc == nil {
			return nil, fmt.Errorf("监控数据源服务不可用")
		}
		id := getUint("datasource_id", 0)
		q := strings.TrimSpace(getStr("query"))
		start, end, step := getStr("start"), getStr("end"), getStr("step")
		if id == 0 || q == "" || start == "" || end == "" || step == "" {
			return nil, fmt.Errorf("datasource_id/query/start/end/step 必填")
		}
		if err := s.assertDatasourceProjectAccess(ctx, actor, id, projectID); err != nil {
			return nil, err
		}
		raw, err := s.alertDSSvc.PromQueryRange(ctx, id, alert.PromQueryRangeRequest{
			Query: q, Start: start, End: end, Step: step,
		})
		if err != nil {
			return nil, err
		}
		return truncatePromJSON(raw, 24_000), nil

	case "list_prometheus_active_alerts":
		if err := requireActor(); err != nil {
			return nil, err
		}
		if s.alertDSSvc == nil {
			return nil, fmt.Errorf("监控数据源服务不可用")
		}
		id := getUint("datasource_id", 0)
		if id == 0 {
			return nil, fmt.Errorf("datasource_id 必填")
		}
		if err := s.assertDatasourceProjectAccess(ctx, actor, id, projectID); err != nil {
			return nil, err
		}
		raw, err := s.alertDSSvc.PrometheusActiveAlerts(ctx, id)
		if err != nil {
			return nil, err
		}
		return truncatePromJSON(raw, 24_000), nil

	case "get_alert_detail":
		if err := requireActor(); err != nil {
			return nil, err
		}
		if s.alertSvc == nil {
			return nil, fmt.Errorf("告警服务不可用")
		}
		fp := strings.TrimSpace(getStr("fingerprint"))
		if fp == "" {
			return nil, fmt.Errorf("fingerprint 必填")
		}
		pid := getUint("project_id", projectID)
		if pid > 0 {
			if err := s.assertProjectMember(ctx, actor, pid); err != nil {
				return nil, err
			}
		}
		ps := int(getUint("page_size", 10))
		if ps <= 0 || ps > 30 {
			ps = 10
		}
		list, total, page, pageSize, err := s.alertSvc.ListEvents(ctx, alert.AlertEventListQuery{
			Fingerprint: fp,
			ProjectID:   pid,
			Page:        1,
			PageSize:    ps,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"fingerprint": fp,
			"list":        list,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"hint":        "结合 explain_alert 看投递；结合 analyze_logs / query_prometheus 查根因与处理建议",
		}, nil

	default:
		return nil, fmt.Errorf("未知监控工具: %s", name)
	}
}

func (s *Service) assertDatasourceProjectAccess(ctx context.Context, actor *auth.CurrentUser, datasourceID, fallbackProjectID uint) error {
	if s.alertDSSvc == nil {
		return fmt.Errorf("监控数据源服务不可用")
	}
	ds, err := s.alertDSSvc.Get(ctx, datasourceID)
	if err != nil {
		return err
	}
	pid := ds.ProjectID
	if pid == 0 {
		pid = fallbackProjectID
	}
	if pid == 0 {
		return fmt.Errorf("数据源未绑定项目，请在上下文选择 project_id")
	}
	return s.assertProjectMember(ctx, actor, pid)
}

func truncatePromJSON(raw json.RawMessage, max int) any {
	if len(raw) == 0 {
		return map[string]any{"raw": nil}
	}
	s := string(raw)
	if max > 0 && len(s) > max {
		s = s[:max] + "…(truncated)"
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err == nil {
		return v
	}
	return map[string]any{"raw": s}
}
