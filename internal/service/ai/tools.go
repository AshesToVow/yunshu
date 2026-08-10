package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"yunshu/internal/ai/runbooks"
	"yunshu/internal/pkg/llm"
	"yunshu/internal/service/k8s"
	"yunshu/internal/service/logplatform"
)

type toolContext struct {
	ClusterID uint
	ProjectID uint
	Namespace string
}

type toolStep struct {
	Name    string `json:"name"`
	Args    string `json:"args"`
	Result  string `json:"result"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
}

func (s *Service) toolDefinitions(includeWrite bool) []llm.ToolDefinition {
	defs := []llm.ToolDefinition{
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
				"cluster_id":    map[string]any{"type": "integer"},
				"namespace":     map[string]any{"type": "string"},
				"name":          map[string]any{"type": "string"},
				"runbook_name":  map[string]any{"type": "string", "description": "CrashLoopBackOff|ImagePullBackOff|PendingUnschedulable，可空自动匹配"},
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
		llm.NewFunctionTool("search_logs", "按项目检索平台日志（ES）", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_id": map[string]any{"type": "integer"},
				"keyword":    map[string]any{"type": "string"},
				"namespace":  map[string]any{"type": "string"},
				"pod":        map[string]any{"type": "string"},
				"page_size":  map[string]any{"type": "integer"},
			},
			"required": []string{"project_id", "keyword"},
		}),
		llm.NewFunctionTool("list_runbooks", "列出内置排障剧本名称", map[string]any{
			"type":       "object",
			"properties": map[string]any{},
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

	var (
		out any
		err error
	)
	switch name {
	case "list_runbooks":
		out = runbooks.Names()
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
		out, err = s.logSearch.Search(ctx, logplatform.LogSearchQuery{
			ProjectID: projectID, Keyword: getStr("keyword"), Page: 1, PageSize: ps,
		})
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
