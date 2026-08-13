package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"yunshu/internal/ai/runbooks"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/llm"
	"yunshu/internal/service/alert"
	cicdsvc "yunshu/internal/service/cicd"
	"yunshu/internal/service/k8s"
	"yunshu/internal/service/logplatform"
)

type toolContext struct {
	ClusterID uint
	ProjectID uint
	Namespace string
	Actor     *auth.CurrentUser
}

type toolStep struct {
	Name   string `json:"name"`
	Args   string `json:"args"`
	Result string `json:"result"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
}

func (s *Service) toolDefinitions(includeWrite bool) []llm.ToolDefinition {
	defs := []llm.ToolDefinition{
		// --- k8s ---
		llm.NewFunctionTool("list_clusters", "列出 Yunshu 已接入的 Kubernetes 集群（id/名称）", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"keyword": map[string]any{"type": "string"},
			},
		}),
		llm.NewFunctionTool("list_pods", "列出指定集群/命名空间的 Pod", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"cluster_id": map[string]any{"type": "integer"},
				"namespace":  map[string]any{"type": "string"},
				"keyword":    map[string]any{"type": "string"},
			},
			"required": []string{"cluster_id"},
		}),
		llm.NewFunctionTool("get_pod_detail", "获取 Pod 详情", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"cluster_id": map[string]any{"type": "integer"},
				"namespace":  map[string]any{"type": "string"},
				"name":       map[string]any{"type": "string"},
			},
			"required": []string{"cluster_id", "namespace", "name"},
		}),
		llm.NewFunctionTool("get_pod_logs", "获取 Pod 日志尾部（只读）", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"cluster_id": map[string]any{"type": "integer"},
				"namespace":  map[string]any{"type": "string"},
				"name":       map[string]any{"type": "string"},
				"container":  map[string]any{"type": "string"},
				"tail_lines": map[string]any{"type": "integer"},
			},
			"required": []string{"cluster_id", "namespace", "name"},
		}),
		llm.NewFunctionTool("diagnose_pod", "确定性 Pod 排障诊断", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"cluster_id": map[string]any{"type": "integer"},
				"namespace":  map[string]any{"type": "string"},
				"name":       map[string]any{"type": "string"},
			},
			"required": []string{"cluster_id", "namespace", "name"},
		}),
		llm.NewFunctionTool("run_diagnose_runbook", "按排障剧本分析 Pod（先 Diagnose 再套剧本）", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"cluster_id":   map[string]any{"type": "integer"},
				"namespace":    map[string]any{"type": "string"},
				"name":         map[string]any{"type": "string"},
				"runbook_name": map[string]any{"type": "string", "description": "CrashLoopBackOff|ImagePullBackOff|PendingUnschedulable，可空自动匹配"},
			},
			"required": []string{"cluster_id", "namespace", "name"},
		}),
		llm.NewFunctionTool("list_deployments", "列出 Deployment", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"cluster_id": map[string]any{"type": "integer"},
				"namespace":  map[string]any{"type": "string"},
				"keyword":    map[string]any{"type": "string"},
			},
			"required": []string{"cluster_id"},
		}),
		llm.NewFunctionTool("list_namespaces", "列出命名空间", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"cluster_id": map[string]any{"type": "integer"},
				"keyword":    map[string]any{"type": "string"},
			},
			"required": []string{"cluster_id"},
		}),
		llm.NewFunctionTool("list_events", "列出 K8s Events", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"cluster_id": map[string]any{"type": "integer"},
				"namespace":  map[string]any{"type": "string"},
				"keyword":    map[string]any{"type": "string"},
				"limit":      map[string]any{"type": "integer"},
			},
			"required": []string{"cluster_id"},
		}),
		llm.NewFunctionTool("list_runbooks", "列出内置排障剧本名称", map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}),
		// --- log ---
		llm.NewFunctionTool("search_logs", "按项目检索平台日志（ES，支持 namespace/pod）", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_id": map[string]any{"type": "integer"},
				"keyword":    map[string]any{"type": "string"},
				"namespace":  map[string]any{"type": "string"},
				"pod":        map[string]any{"type": "string"},
				"cluster_id": map[string]any{"type": "integer"},
				"page_size":  map[string]any{"type": "integer"},
			},
			"required": []string{"project_id", "keyword"},
		}),
		// --- cicd ---
		llm.NewFunctionTool("list_cicd_builds", "列出项目 CI 构建记录", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_id": map[string]any{"type": "integer"},
				"keyword":    map[string]any{"type": "string"},
				"page_size":  map[string]any{"type": "integer"},
			},
			"required": []string{"project_id"},
		}),
		llm.NewFunctionTool("get_cicd_build", "获取单次构建详情", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_id": map[string]any{"type": "integer"},
				"run_id":     map[string]any{"type": "integer"},
			},
			"required": []string{"project_id", "run_id"},
		}),
		llm.NewFunctionTool("get_cicd_build_log", "获取构建控制台日志（截断）", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_id": map[string]any{"type": "integer"},
				"run_id":     map[string]any{"type": "integer"},
			},
			"required": []string{"project_id", "run_id"},
		}),
		// --- alert ---
		llm.NewFunctionTool("list_alerts", "列出告警事件（可按状态/严重级别/关键词）", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_id": map[string]any{"type": "integer"},
				"status":     map[string]any{"type": "string", "description": "firing|resolved 等"},
				"severity":   map[string]any{"type": "string"},
				"keyword":    map[string]any{"type": "string"},
				"page_size":  map[string]any{"type": "integer"},
			},
		}),
		llm.NewFunctionTool("explain_alert", "解释告警指纹投递路径（发送/跳过/抑制）", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"fingerprint": map[string]any{"type": "string"},
			},
			"required": []string{"fingerprint"},
		}),
	}
	if includeWrite {
		defs = append(defs,
			llm.NewFunctionTool("scale_deployment", "扩缩容 Deployment（需审批，不会立即执行）", map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster_id": map[string]any{"type": "integer"},
					"namespace":  map[string]any{"type": "string"},
					"name":       map[string]any{"type": "string"},
					"replicas":   map[string]any{"type": "integer"},
					"reason":     map[string]any{"type": "string"},
				},
				"required": []string{"cluster_id", "namespace", "name", "replicas"},
			}),
			llm.NewFunctionTool("restart_deployment", "重启 Deployment（需审批）", map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster_id": map[string]any{"type": "integer"},
					"namespace":  map[string]any{"type": "string"},
					"name":       map[string]any{"type": "string"},
					"reason":     map[string]any{"type": "string"},
				},
				"required": []string{"cluster_id", "namespace", "name"},
			}),
			llm.NewFunctionTool("delete_pod", "删除 Pod（需审批）", map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster_id": map[string]any{"type": "integer"},
					"namespace":  map[string]any{"type": "string"},
					"name":       map[string]any{"type": "string"},
					"reason":     map[string]any{"type": "string"},
				},
				"required": []string{"cluster_id", "namespace", "name"},
			}),
		)
	}
	return defs
}

func (s *Service) executeTool(ctx context.Context, userID uint, name, argsJSON string, tc toolContext) toolStep {
	step := toolStep{Name: name, Args: truncateStr(argsJSON, 2000)}
	var args map[string]any
	_ = json.Unmarshal([]byte(argsJSON), &args)
	if args == nil {
		args = map[string]any{}
	}
	getUint := func(key string, fallback uint) uint {
		if v, ok := args[key]; ok {
			switch n := v.(type) {
			case float64:
				return uint(n)
			case int:
				return uint(n)
			case json.Number:
				i, _ := n.Int64()
				return uint(i)
			}
		}
		return fallback
	}
	getStr := func(key string) string {
		if v, ok := args[key].(string); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}

	clusterID := getUint("cluster_id", tc.ClusterID)
	namespace := getStr("namespace")
	if namespace == "" {
		namespace = tc.Namespace
	}
	projectID := getUint("project_id", tc.ProjectID)
	actor := tc.Actor
	if actor == nil && userID > 0 {
		actor = &auth.CurrentUser{ID: userID}
	}

	var (
		out any
		err error
	)
	switch name {
	case "list_runbooks":
		out = runbooks.Names()
	case "list_clusters":
		if s.clusterSvc == nil {
			err = fmt.Errorf("集群服务不可用")
			break
		}
		var res *k8s.K8sClusterListResponse
		res, err = s.clusterSvc.List(ctx, k8s.K8sClusterListQuery{Keyword: getStr("keyword"), Page: 1, PageSize: 100})
		if err == nil && res != nil {
			type item struct {
				ID     uint   `json:"id"`
				Name   string `json:"name"`
				Status int    `json:"status"`
			}
			list := make([]item, 0, len(res.List))
			for _, c := range res.List {
				list = append(list, item{ID: c.ID, Name: c.Name, Status: c.Status})
			}
			out = map[string]any{"total": res.Total, "list": list}
		}
	case "list_pods":
		if s.podSvc == nil {
			err = fmt.Errorf("Pod 服务不可用")
			break
		}
		out, err = s.podSvc.List(ctx, k8s.PodListQuery{ClusterID: clusterID, Namespace: namespace, Keyword: getStr("keyword")})
	case "get_pod_detail":
		if s.podSvc == nil {
			err = fmt.Errorf("Pod 服务不可用")
			break
		}
		out, err = s.podSvc.Detail(ctx, k8s.PodDetailQuery{ClusterID: clusterID, Namespace: namespace, Name: getStr("name")})
	case "get_pod_logs":
		if s.podSvc == nil {
			err = fmt.Errorf("Pod 服务不可用")
			break
		}
		tail := int64(getUint("tail_lines", 200))
		if tail <= 0 {
			tail = 200
		}
		if tail > 1000 {
			tail = 1000
		}
		var logs string
		logs, err = s.podSvc.GetLogs(ctx, k8s.PodLogsQuery{
			ClusterID: clusterID, Namespace: namespace, Name: getStr("name"),
			Container: getStr("container"), TailLines: tail,
		})
		out = truncateStr(logs, 12_000)
	case "diagnose_pod":
		if s.podSvc == nil {
			err = fmt.Errorf("Pod 服务不可用")
			break
		}
		out, err = s.podSvc.Diagnose(ctx, k8s.PodDiagnoseQuery{ClusterID: clusterID, Namespace: namespace, Name: getStr("name")})
	case "run_diagnose_runbook":
		out, err = s.runDiagnoseRunbook(ctx, clusterID, namespace, getStr("name"), getStr("runbook_name"))
	case "list_deployments":
		if s.workloadSvc == nil {
			err = fmt.Errorf("Workload 服务不可用")
			break
		}
		out, err = s.workloadSvc.ListDeployments(ctx, k8s.NamespacedListQuery{
			ClusterNamespaceKeywordQuery: k8s.ClusterNamespaceKeywordQuery{
				ClusterID: clusterID, Namespace: namespace, Keyword: getStr("keyword"),
			},
		})
	case "list_namespaces":
		if s.nsSvc == nil {
			err = fmt.Errorf("Namespace 服务不可用")
			break
		}
		out, err = s.nsSvc.List(ctx, k8s.NamespaceListQuery{ClusterID: clusterID, Keyword: getStr("keyword")}, nil)
	case "list_events":
		if s.eventSvc == nil {
			err = fmt.Errorf("Event 服务不可用")
			break
		}
		limit := int64(getUint("limit", 50))
		out, err = s.eventSvc.List(ctx, k8s.EventListQuery{ClusterID: clusterID, Namespace: namespace, Keyword: getStr("keyword"), Limit: limit})
	case "search_logs":
		if s.logSearch == nil {
			err = fmt.Errorf("日志检索服务不可用")
			break
		}
		ps := int(getUint("page_size", 20))
		if ps <= 0 || ps > 50 {
			ps = 20
		}
		q := logplatform.LogSearchQuery{
			ProjectID: projectID,
			Keyword:   getStr("keyword"),
			Namespace: getStr("namespace"),
			Pod:       getStr("pod"),
			Page:      1,
			PageSize:  ps,
		}
		if clusterID > 0 {
			cid := clusterID
			q.ClusterID = &cid
		}
		out, err = s.logSearch.Search(ctx, q)
	case "list_cicd_builds":
		if s.cicdSvc == nil {
			err = fmt.Errorf("CI/CD 服务不可用")
			break
		}
		ps := int(getUint("page_size", 20))
		if ps <= 0 || ps > 50 {
			ps = 20
		}
		out, err = s.cicdSvc.ListBuildRuns(ctx, cicdsvc.BuildRunListQuery{
			ProjectID: projectID,
			Keyword:   getStr("keyword"),
			Page:      1,
			PageSize:  ps,
			Actor:     actor,
		})
	case "get_cicd_build":
		if s.cicdSvc == nil {
			err = fmt.Errorf("CI/CD 服务不可用")
			break
		}
		out, err = s.cicdSvc.GetBuildRun(ctx, projectID, getUint("run_id", 0), actor)
	case "get_cicd_build_log":
		if s.cicdSvc == nil {
			err = fmt.Errorf("CI/CD 服务不可用")
			break
		}
		var logs string
		logs, err = s.cicdSvc.GetBuildRunLog(ctx, projectID, getUint("run_id", 0), actor)
		out = truncateStr(logs, 12_000)
	case "list_alerts":
		if s.alertSvc == nil {
			err = fmt.Errorf("告警服务不可用")
			break
		}
		ps := int(getUint("page_size", 20))
		if ps <= 0 || ps > 50 {
			ps = 20
		}
		list, total, page, pageSize, e := s.alertSvc.ListEvents(ctx, alert.AlertEventListQuery{
			ProjectID: projectID,
			Status:    getStr("status"),
			Severity:  getStr("severity"),
			Keyword:   getStr("keyword"),
			Page:      1,
			PageSize:  ps,
		})
		err = e
		if err == nil {
			out = map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}
		}
	case "explain_alert":
		if s.alertSvc == nil {
			err = fmt.Errorf("告警服务不可用")
			break
		}
		out, err = s.alertSvc.ExplainFingerprintDelivery(ctx, getStr("fingerprint"))
	case "scale_deployment", "restart_deployment", "delete_pod":
		out, err = s.createToolApproval(ctx, userID, name, argsJSON, clusterID, namespace, getStr("name"), getStr("reason"))
	default:
		err = fmt.Errorf("未知工具: %s", name)
	}

	if err != nil {
		step.OK = false
		step.Error = err.Error()
		step.Result = err.Error()
		return step
	}
	raw, _ := json.Marshal(out)
	step.OK = true
	step.Result = truncateStr(string(raw), 24_000)
	return step
}

func (s *Service) runDiagnoseRunbook(ctx context.Context, clusterID uint, ns, name, rbName string) (map[string]any, error) {
	if s.podSvc == nil {
		return nil, fmt.Errorf("Pod 服务不可用")
	}
	diag, err := s.podSvc.Diagnose(ctx, k8s.PodDiagnoseQuery{ClusterID: clusterID, Namespace: ns, Name: name})
	if err != nil {
		return nil, err
	}
	reason := ""
	for _, c := range diag.Containers {
		if c.Reason != "" {
			reason = c.Reason
			break
		}
	}
	if strings.TrimSpace(rbName) == "" {
		rbName = runbooks.MatchByReason(reason, diag.Summary)
	}
	body, err := runbooks.Load(rbName)
	if err != nil {
		body = ""
		rbName = "(missing)"
	}
	return map[string]any{
		"runbook":  rbName,
		"playbook": body,
		"diagnose": diag,
	}, nil
}

func truncateStr(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "…(truncated)"
}
