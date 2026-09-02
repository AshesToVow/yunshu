package ai

import (
	"context"
	"fmt"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/llm"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/service/logplatform"
	projectsvc "yunshu/internal/service/project"
)

func (s *Service) logToolDefinitions() []llm.ToolDefinition {
	return []llm.ToolDefinition{
		llm.NewFunctionTool("analyze_logs",
			"检索并整理项目日志：按级别/服务/Pod 统计，提取高频错误签名与样例行，便于快速排障。需要 project_id；建议传 keyword 或 level（如 ERROR）。与 search_logs（原始命中列表）互补。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id":     map[string]any{"type": "integer", "description": "必填"},
					"keyword":        map[string]any{"type": "string", "description": "关键字，如 Exception、timeout"},
					"level":          map[string]any{"type": "string", "description": "日志级别，如 ERROR/WARN/INFO"},
					"service_name":   map[string]any{"type": "string"},
					"namespace":      map[string]any{"type": "string"},
					"pod":            map[string]any{"type": "string"},
					"container":      map[string]any{"type": "string"},
					"collector_mode": map[string]any{"type": "string", "description": "host|k8s"},
					"cluster_id":     map[string]any{"type": "integer"},
					"from":           map[string]any{"type": "string", "description": "开始时间"},
					"to":             map[string]any{"type": "string", "description": "结束时间"},
					"page_size":      map[string]any{"type": "integer", "description": "采样条数，默认 50，最大 50"},
				},
				"required": []string{"project_id"},
			}),
		llm.NewFunctionTool("list_log_sources",
			"列出项目日志源配置（路径/服务绑定）。检索为空或怀疑未采集时调用。需要 project_id。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer"},
					"service_id": map[string]any{"type": "integer"},
					"page_size":  map[string]any{"type": "integer"},
				},
				"required": []string{"project_id"},
			}),
		llm.NewFunctionTool("list_loggie_status",
			"查看主机侧 Loggie Agent 健康/流水线状态。日志为空且 collector_mode=host 时优先调用。需要 project_id。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer"},
				},
				"required": []string{"project_id"},
			}),
		llm.NewFunctionTool("list_cluster_log_rules",
			"列出 K8s 集群日志采集规则。日志为空且 collector_mode=k8s 时调用。需要 project_id 与 cluster_id。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer"},
					"cluster_id": map[string]any{"type": "integer"},
				},
				"required": []string{"project_id", "cluster_id"},
			}),
	}
}

func (s *Service) executeLogTool(
	ctx context.Context,
	name string,
	getUint func(string, uint) uint,
	getStr func(string) string,
	projectID, clusterID uint,
	requireProject func() error,
	_ *auth.CurrentUser,
) (any, error) {
	switch name {
	case "analyze_logs":
		if err := requireProject(); err != nil {
			return nil, err
		}
		if s.logSearch == nil {
			return nil, fmt.Errorf("日志检索服务不可用")
		}
		q := s.buildLogSearchQuery(getUint, getStr, projectID, clusterID)
		if q.PageSize <= 0 || q.PageSize > 50 {
			q.PageSize = 50
		}
		res, err := s.logSearch.Search(ctx, q)
		if err != nil {
			return nil, err
		}
		return summarizeLogHitsForTool(res, q), nil
	case "list_log_sources":
		if err := requireProject(); err != nil {
			return nil, err
		}
		if s.projectMgmt == nil {
			return nil, fmt.Errorf("项目日志源服务不可用")
		}
		ps := int(getUint("page_size", 50))
		if ps <= 0 || ps > 100 {
			ps = 50
		}
		q := projectsvc.LogSourceListQuery{ProjectID: projectID, Page: 1, PageSize: ps}
		if sid := getUint("service_id", 0); sid > 0 {
			q.ServiceID = &sid
		}
		return s.projectMgmt.ListLogSources(ctx, q)
	case "list_loggie_status":
		if err := requireProject(); err != nil {
			return nil, err
		}
		if s.loggieAgent == nil {
			return nil, fmt.Errorf("Loggie 服务不可用")
		}
		return s.loggieAgent.ListStatus(ctx, projectID)
	case "list_cluster_log_rules":
		if err := requireProject(); err != nil {
			return nil, err
		}
		cid := getUint("cluster_id", clusterID)
		if cid == 0 {
			return nil, fmt.Errorf("cluster_id 必填")
		}
		if s.clusterLog == nil {
			return nil, fmt.Errorf("集群日志服务不可用")
		}
		return s.clusterLog.ListRules(ctx, projectID, cid)
	default:
		return nil, fmt.Errorf("未知日志工具: %s", name)
	}
}

func (s *Service) buildLogSearchQuery(
	getUint func(string, uint) uint,
	getStr func(string) string,
	projectID, clusterID uint,
) logplatform.LogSearchQuery {
	ps := int(getUint("page_size", 20))
	if ps <= 0 || ps > 50 {
		ps = 20
	}
	q := logplatform.LogSearchQuery{
		ProjectID:     projectID,
		Keyword:       getStr("keyword"),
		Level:         getStr("level"),
		ServiceName:   getStr("service_name"),
		Namespace:     getStr("namespace"),
		Pod:           getStr("pod"),
		Container:     getStr("container"),
		CollectorMode: getStr("collector_mode"),
		FilePath:      getStr("file_path"),
		From:          getStr("from"),
		To:            getStr("to"),
		Page:          1,
		PageSize:      ps,
	}
	if clusterID > 0 {
		cid := clusterID
		q.ClusterID = &cid
	} else if c := getUint("cluster_id", 0); c > 0 {
		q.ClusterID = &c
	}
	if sid := getUint("server_id", 0); sid > 0 {
		q.ServerID = &sid
	}
	if lsid := getUint("log_source_id", 0); lsid > 0 {
		q.LogSourceID = &lsid
	}
	return q
}

func summarizeLogHitsForTool(res *pagination.Result[logplatform.LogSearchItem], q logplatform.LogSearchQuery) map[string]any {
	summary := logplatform.SummarizeLogHits(res, 8)
	if summary == nil {
		return map[string]any{"total": 0, "summary": "无命中", "query": q}
	}
	hint := "已整理采样日志；请结合 top_error_signatures 与 samples 给出根因假设与下一步。"
	if summary.Total == 0 || summary.Sampled == 0 {
		hint = "无命中：请放宽 keyword、调整 from/to，或调用 list_log_sources / list_loggie_status / list_cluster_log_rules 排查采集。"
	}
	return map[string]any{
		"total":                summary.Total,
		"sampled":              summary.Sampled,
		"query":                q,
		"level_counts":         summary.LevelCounts,
		"service_counts":       summary.ServiceCounts,
		"pod_counts":           summary.PodCounts,
		"top_error_signatures": summary.TopErrorSignatures,
		"samples":              summary.Samples,
		"analysis_hint":        hint,
	}
}
