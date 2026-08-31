package ai

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

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
		return summarizeLogHits(res, q), nil
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

var (
	reDigits   = regexp.MustCompile(`\d+`)
	reUUIDLike = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	reHexLong  = regexp.MustCompile(`\b[0-9a-fA-F]{16,}\b`)
	reIPv4     = regexp.MustCompile(`\b\d{1,3}(?:\.\d{1,3}){3}\b`)
)

type countEntry struct {
	Key   string
	Count int
}

func summarizeLogHits(res *pagination.Result[logplatform.LogSearchItem], q logplatform.LogSearchQuery) map[string]any {
	if res == nil {
		return map[string]any{"total": 0, "summary": "无命中", "query": q}
	}
	list := res.List
	levelCnt := map[string]int{}
	svcCnt := map[string]int{}
	podCnt := map[string]int{}
	sigCnt := map[string]int{}
	sigSample := map[string]string{}
	samples := make([]map[string]any, 0, 8)

	for i, it := range list {
		lv := strings.TrimSpace(it.Level)
		if lv == "" {
			lv = inferLevelFromMessage(it.Message)
		}
		if lv == "" {
			lv = "(unknown)"
		}
		levelCnt[lv]++
		svc := strings.TrimSpace(it.ServiceName)
		if svc == "" {
			svc = "(unknown)"
		}
		svcCnt[svc]++
		pod := firstNonEmpty(it.Pod, it.PodName)
		if pod == "" {
			pod = "(unknown)"
		}
		podCnt[pod]++

		sig := normalizeLogSignature(it.Message)
		if sig != "" {
			sigCnt[sig]++
			if _, ok := sigSample[sig]; !ok {
				sigSample[sig] = truncateRunes(it.Message, 240)
			}
		}
		if i < 8 {
			samples = append(samples, map[string]any{
				"timestamp":    it.Timestamp,
				"level":        lv,
				"service_name": it.ServiceName,
				"namespace":    it.Namespace,
				"pod":          firstNonEmpty(it.Pod, it.PodName),
				"message":      truncateRunes(it.Message, 300),
			})
		}
	}

	topSigs := topCountEntries(sigCnt, 10)
	topSigOut := make([]map[string]any, 0, len(topSigs))
	for _, e := range topSigs {
		topSigOut = append(topSigOut, map[string]any{
			"signature": e.Key,
			"count":     e.Count,
			"sample":    sigSample[e.Key],
		})
	}

	hint := "已整理采样日志；请结合 top_error_signatures 与 samples 给出根因假设与下一步。"
	if res.Total == 0 || len(list) == 0 {
		hint = "无命中：请放宽 keyword、调整 from/to，或调用 list_log_sources / list_loggie_status / list_cluster_log_rules 排查采集。"
	}

	return map[string]any{
		"total":                res.Total,
		"sampled":              len(list),
		"page":                 res.Page,
		"page_size":            res.PageSize,
		"query":                q,
		"level_counts":         levelCnt,
		"service_counts":       topCountMap(svcCnt, 8),
		"pod_counts":           topCountMap(podCnt, 8),
		"top_error_signatures": topSigOut,
		"samples":              samples,
		"analysis_hint":        hint,
	}
}

func normalizeLogSignature(msg string) string {
	s := strings.TrimSpace(msg)
	if s == "" {
		return ""
	}
	s = reUUIDLike.ReplaceAllString(s, "<uuid>")
	s = reHexLong.ReplaceAllString(s, "<hex>")
	s = reIPv4.ReplaceAllString(s, "<ip>")
	s = reDigits.ReplaceAllString(s, "N")
	s = strings.Join(strings.Fields(s), " ")
	return truncateRunes(s, 160)
}

func inferLevelFromMessage(msg string) string {
	u := strings.ToUpper(msg)
	for _, lv := range []string{"FATAL", "ERROR", "WARN", "WARNING", "INFO", "DEBUG", "TRACE"} {
		if strings.Contains(u, lv) {
			if lv == "WARNING" {
				return "WARN"
			}
			return lv
		}
	}
	return ""
}

func topCountEntries(m map[string]int, n int) []countEntry {
	out := make([]countEntry, 0, len(m))
	for k, v := range m {
		out = append(out, countEntry{Key: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Key < out[j].Key
		}
		return out[i].Count > out[j].Count
	})
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

func topCountMap(m map[string]int, n int) map[string]int {
	entries := topCountEntries(m, n)
	out := make(map[string]int, len(entries))
	for _, e := range entries {
		out[e.Key] = e.Count
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
