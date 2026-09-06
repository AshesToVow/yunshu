package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"yunshu/internal/ai/runbooks"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/llm"
	"yunshu/internal/service/alert"
	cicdsvc "yunshu/internal/service/cicd"
	"yunshu/internal/service/k8s"
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
	s.ensureSeed()
	defs := s.builtinToolDefinitions(includeWrite)
	// 追加已启用的脚本工具
	var scripts []model.AiToolDef
	_ = s.db.Where("enabled = ? AND runtime = ?", true, "script").Find(&scripts).Error
	for _, t := range scripts {
		schema := map[string]any{"type": "object", "properties": map[string]any{}}
		if strings.TrimSpace(t.InputSchemaJSON) != "" {
			_ = json.Unmarshal([]byte(t.InputSchemaJSON), &schema)
		}
		if !includeWrite && strings.EqualFold(t.Permission, "WRITE") {
			continue
		}
		defs = append(defs, llm.NewFunctionTool(t.Name, t.Description, schema))
	}
	return defs
}

func (s *Service) builtinToolDefinitions(includeWrite bool) []llm.ToolDefinition {
	defs := []llm.ToolDefinition{
		// --- k8s ---
		llm.NewFunctionTool("list_clusters",
			"列出 Yunshu 已接入的 K8s 集群（id/名称）。用户未选集群或不知道 cluster_id 时必须先调用。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"keyword": map[string]any{"type": "string", "description": "按集群名称模糊过滤"},
				},
			}),
		llm.NewFunctionTool("list_pods",
			"列出集群/命名空间下的 Pod。用于定位异常 Pod 名称；已知准确名称可跳过，直接 diagnose_pod。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster_id": map[string]any{"type": "integer", "description": "必填；可从上下文或 list_clusters 获取"},
					"namespace":  map[string]any{"type": "string", "description": "命名空间，空则按服务默认范围"},
					"keyword":    map[string]any{"type": "string", "description": "按 Pod 名过滤"},
				},
				"required": []string{"cluster_id"},
			}),
		llm.NewFunctionTool("get_pod_detail",
			"获取单个 Pod 详情（规格/状态/容器）。需要完整 YAML 级信息时用；异常排障优先 diagnose_pod。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster_id": map[string]any{"type": "integer"},
					"namespace":  map[string]any{"type": "string"},
					"name":       map[string]any{"type": "string", "description": "Pod 名称"},
				},
				"required": []string{"cluster_id", "namespace", "name"},
			}),
		llm.NewFunctionTool("get_pod_logs",
			"获取 Pod 容器日志尾部（只读）。CrashLoop/启动失败时必查；可指定 container 与 tail_lines。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster_id": map[string]any{"type": "integer"},
					"namespace":  map[string]any{"type": "string"},
					"name":       map[string]any{"type": "string"},
					"container":  map[string]any{"type": "string", "description": "多容器时指定容器名"},
					"tail_lines": map[string]any{"type": "integer", "description": "默认 200，最大 1000"},
				},
				"required": []string{"cluster_id", "namespace", "name"},
			}),
		llm.NewFunctionTool("diagnose_pod",
			"确定性 Pod 排障诊断（规则引擎）。Pod 异常/CrashLoop/Pending/ImagePull 时优先调用，再决定是否看日志或剧本。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster_id": map[string]any{"type": "integer"},
					"namespace":  map[string]any{"type": "string"},
					"name":       map[string]any{"type": "string"},
				},
				"required": []string{"cluster_id", "namespace", "name"},
			}),
		llm.NewFunctionTool("run_diagnose_runbook",
			"先 Diagnose 再套内置排障剧本。在 diagnose_pod 后、原因匹配 CrashLoopBackOff/ImagePullBackOff/PendingUnschedulable 时使用。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster_id":   map[string]any{"type": "integer"},
					"namespace":    map[string]any{"type": "string"},
					"name":         map[string]any{"type": "string"},
					"runbook_name": map[string]any{"type": "string", "description": "CrashLoopBackOff|ImagePullBackOff|PendingUnschedulable，可空自动匹配"},
				},
				"required": []string{"cluster_id", "namespace", "name"},
			}),
		llm.NewFunctionTool("list_deployments",
			"列出 Deployment。用于找工作负载名称、副本数；扩缩容/重启前先确认 name。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster_id": map[string]any{"type": "integer"},
					"namespace":  map[string]any{"type": "string"},
					"keyword":    map[string]any{"type": "string"},
				},
				"required": []string{"cluster_id"},
			}),
		llm.NewFunctionTool("list_namespaces",
			"列出命名空间。用户未提供 namespace 时可先列出再缩小范围。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster_id": map[string]any{"type": "integer"},
					"keyword":    map[string]any{"type": "string"},
				},
				"required": []string{"cluster_id"},
			}),
		llm.NewFunctionTool("list_events",
			"列出 K8s Events。配合 Pod 排障查看 FailedScheduling/Failed/Unhealthy 等事件。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cluster_id": map[string]any{"type": "integer"},
					"namespace":  map[string]any{"type": "string"},
					"keyword":    map[string]any{"type": "string"},
					"limit":      map[string]any{"type": "integer", "description": "默认 50"},
				},
				"required": []string{"cluster_id"},
			}),
		llm.NewFunctionTool("list_runbooks",
			"列出内置排障剧本名称。不清楚有哪些剧本时调用。",
			map[string]any{"type": "object", "properties": map[string]any{}}),
		// --- log ---
		llm.NewFunctionTool("search_logs",
			"检索项目日志平台（ES）原始命中列表。需要 project_id；keyword 建议填写。支持 level/from/to/service_name/namespace/pod 等过滤。复杂排障优先 analyze_logs；与 get_pod_logs（kubectl 实时日志）互补。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id":     map[string]any{"type": "integer", "description": "必填；来自助手页所选项目或用户提供"},
					"keyword":        map[string]any{"type": "string", "description": "检索关键字，如 error、Exception"},
					"level":          map[string]any{"type": "string", "description": "ERROR/WARN/INFO 等"},
					"service_name":   map[string]any{"type": "string"},
					"namespace":      map[string]any{"type": "string"},
					"pod":            map[string]any{"type": "string"},
					"container":      map[string]any{"type": "string"},
					"collector_mode": map[string]any{"type": "string", "description": "host|k8s"},
					"cluster_id":     map[string]any{"type": "integer"},
					"server_id":      map[string]any{"type": "integer"},
					"log_source_id":  map[string]any{"type": "integer"},
					"file_path":      map[string]any{"type": "string"},
					"from":           map[string]any{"type": "string"},
					"to":             map[string]any{"type": "string"},
					"page_size":      map[string]any{"type": "integer", "description": "默认 20，最大 50"},
				},
				"required": []string{"project_id"},
			}),
		// --- cicd ---
		llm.NewFunctionTool("list_cicd_builds",
			"列出项目 CI 构建记录。构建失败排查第一步；需要 project_id。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer", "description": "必填"},
					"keyword":    map[string]any{"type": "string", "description": "按分支/构建名过滤"},
					"page_size":  map[string]any{"type": "integer"},
				},
				"required": []string{"project_id"},
			}),
		llm.NewFunctionTool("get_cicd_build",
			"获取单次构建详情（状态/阶段）。在 list_cicd_builds 选定 run_id 后调用。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer"},
					"run_id":     map[string]any{"type": "integer", "description": "构建记录 ID"},
				},
				"required": []string{"project_id", "run_id"},
			}),
		llm.NewFunctionTool("get_cicd_build_log",
			"获取构建控制台日志（截断）。定位编译/测试/推镜像失败的具体错误行。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer"},
					"run_id":     map[string]any{"type": "integer"},
				},
				"required": []string{"project_id", "run_id"},
			}),
		// --- alert ---
		llm.NewFunctionTool("list_alerts",
			"列出告警事件。排查「未收到/重复告警」时先列出，再取 fingerprint 调用 explain_alert。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer"},
					"status":     map[string]any{"type": "string", "description": "firing|resolved 等"},
					"severity":   map[string]any{"type": "string"},
					"keyword":    map[string]any{"type": "string"},
					"page_size":  map[string]any{"type": "integer"},
				},
			}),
		llm.NewFunctionTool("explain_alert",
			"解释告警指纹投递路径（发送/跳过/抑制）。必须先有 fingerprint（可从 list_alerts 结果取）。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"fingerprint": map[string]any{"type": "string", "description": "告警指纹"},
				},
				"required": []string{"fingerprint"},
			}),
	}
	defs = append(defs, s.monitorToolDefinitions()...)
	defs = append(defs, s.platformToolDefinitions()...)
	defs = append(defs, s.logToolDefinitions()...)
	if includeWrite {
		defs = append(defs,
			llm.NewFunctionTool("scale_deployment",
				"申请扩缩容 Deployment：仅创建审批单，不会立即执行。调用前确认 cluster/namespace/name/replicas。",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"cluster_id": map[string]any{"type": "integer"},
						"namespace":  map[string]any{"type": "string"},
						"name":       map[string]any{"type": "string"},
						"replicas":   map[string]any{"type": "integer"},
						"reason":     map[string]any{"type": "string", "description": "申请原因，便于审批"},
					},
					"required": []string{"cluster_id", "namespace", "name", "replicas"},
				}),
			llm.NewFunctionTool("restart_deployment",
				"申请重启 Deployment：仅创建审批单。",
				map[string]any{
					"type": "object",
					"properties": map[string]any{
						"cluster_id": map[string]any{"type": "integer"},
						"namespace":  map[string]any{"type": "string"},
						"name":       map[string]any{"type": "string"},
						"reason":     map[string]any{"type": "string"},
					},
					"required": []string{"cluster_id", "namespace", "name"},
				}),
			llm.NewFunctionTool("delete_pod",
				"申请删除 Pod：仅创建审批单。高危操作，须说明 reason。",
				map[string]any{
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
	s.ensureSeed()

	// 注册表：禁用 / 脚本工具
	var reg model.AiToolDef
	if err := s.db.WithContext(ctx).Where("name = ?", name).First(&reg).Error; err == nil {
		if !reg.Enabled {
			step.OK = false
			step.Error = "工具已禁用"
			step.Result = step.Error
			s.recordAudit(userID, 0, "tool", name, reg.RiskLevel, false, step.Error)
			return step
		}
		if strings.EqualFold(reg.Runtime, "script") {
			actor := resolveActor(ctx, tc.Actor)
			if actor == nil || actor.ID == 0 {
				step.OK = false
				step.Error = "未登录或用户上下文缺失，拒绝执行工具"
				step.Result = step.Error
				s.recordAudit(userID, 0, "tool", name, reg.RiskLevel, false, step.Error)
				return step
			}
			if scriptToolRequiresApproval(reg.RiskLevel, reg.Permission) {
				err := errScriptNeedsApproval(name)
				step.OK = false
				step.Error = err.Error()
				step.Result = step.Error
				s.recordAudit(userID, 0, "tool", name, reg.RiskLevel, false, step.Error)
				return step
			}
			out, err := s.runScriptTool(ctx, toolDefRow{
				Name: reg.Name, ScriptLang: reg.ScriptLang, ScriptPath: reg.ScriptPath, TimeoutSec: reg.TimeoutSec,
			}, argsJSON)
			if err != nil {
				step.OK = false
				step.Error = err.Error()
				step.Result = err.Error()
				s.recordAudit(userID, 0, "tool", name, reg.RiskLevel, false, err.Error())
				return step
			}
			step.OK = true
			step.Result = out
			s.recordAudit(userID, 0, "tool", name, reg.RiskLevel, true, truncateStr(out, 500))
			return step
		}
	}

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
	actor := resolveActor(ctx, tc.Actor)

	requireActor := func() error {
		if actor == nil || actor.ID == 0 {
			return fmt.Errorf("未登录或用户上下文缺失，拒绝执行工具")
		}
		return nil
	}
	requireProject := func() error {
		if err := requireActor(); err != nil {
			return err
		}
		return s.assertProjectMember(ctx, actor, projectID)
	}
	requireCluster := func() error {
		if err := requireActor(); err != nil {
			return err
		}
		if clusterID == 0 {
			return fmt.Errorf("cluster_id 必填")
		}
		return nil
	}
	requireK8sRead := func() error {
		if err := requireCluster(); err != nil {
			return err
		}
		return s.assertK8sClusterAccess(ctx, actor, clusterID, namespace, k8s.K8sAccessRankReadonly)
	}
	requireK8sAdmin := func() error {
		if err := requireCluster(); err != nil {
			return err
		}
		return s.assertK8sClusterAccess(ctx, actor, clusterID, namespace, k8s.K8sAccessRankAdmin)
	}

	var (
		out any
		err error
	)
	switch name {
	case "list_runbooks":
		out = runbooks.Names()
	case "list_clusters":
		if err = requireActor(); err != nil {
			break
		}
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
				if !auth.IsSuperAdminRole(actor.RoleCodes) {
					if s.accessRepo == nil {
						err = constants.ErrInternal
						break
					}
					rank := s.accessRepo.EffectiveTier(ctx, k8sauth.PackFromCurrentUser(actor), c.ID)
					if rank < k8s.K8sAccessRankReadonly {
						continue
					}
				}
				list = append(list, item{ID: c.ID, Name: c.Name, Status: c.Status})
			}
			if err != nil {
				break
			}
			out = map[string]any{"total": len(list), "list": list}
		}
	case "list_pods":
		if err = requireK8sRead(); err != nil {
			break
		}
		if s.podSvc == nil {
			err = fmt.Errorf("Pod 服务不可用")
			break
		}
		ctx = withActorContext(ctx, actor)
		out, err = s.podSvc.List(ctx, k8s.PodListQuery{ClusterID: clusterID, Namespace: namespace, Keyword: getStr("keyword")})
	case "get_pod_detail":
		if err = requireK8sRead(); err != nil {
			break
		}
		if s.podSvc == nil {
			err = fmt.Errorf("Pod 服务不可用")
			break
		}
		ctx = withActorContext(ctx, actor)
		out, err = s.podSvc.Detail(ctx, k8s.PodDetailQuery{ClusterID: clusterID, Namespace: namespace, Name: getStr("name")})
	case "get_pod_logs":
		if err = requireK8sRead(); err != nil {
			break
		}
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
		ctx = withActorContext(ctx, actor)
		var logs string
		logs, err = s.podSvc.GetLogs(ctx, k8s.PodLogsQuery{
			ClusterID: clusterID, Namespace: namespace, Name: getStr("name"),
			Container: getStr("container"), TailLines: tail,
		})
		out = truncateStr(logs, 12_000)
	case "diagnose_pod":
		if err = requireK8sRead(); err != nil {
			break
		}
		if s.podSvc == nil {
			err = fmt.Errorf("Pod 服务不可用")
			break
		}
		ctx = withActorContext(ctx, actor)
		out, err = s.podSvc.Diagnose(ctx, k8s.PodDiagnoseQuery{ClusterID: clusterID, Namespace: namespace, Name: getStr("name")})
	case "run_diagnose_runbook":
		if err = requireK8sRead(); err != nil {
			break
		}
		ctx = withActorContext(ctx, actor)
		out, err = s.runDiagnoseRunbook(ctx, clusterID, namespace, getStr("name"), getStr("runbook_name"))
	case "list_deployments":
		if err = requireK8sRead(); err != nil {
			break
		}
		if s.workloadSvc == nil {
			err = fmt.Errorf("Workload 服务不可用")
			break
		}
		ctx = withActorContext(ctx, actor)
		out, err = s.workloadSvc.ListDeployments(ctx, k8s.NamespacedListQuery{
			ClusterNamespaceKeywordQuery: k8s.ClusterNamespaceKeywordQuery{
				ClusterID: clusterID, Namespace: namespace, Keyword: getStr("keyword"),
			},
		})
	case "list_namespaces":
		if err = requireK8sRead(); err != nil {
			break
		}
		if s.nsSvc == nil {
			err = fmt.Errorf("Namespace 服务不可用")
			break
		}
		pack := k8sauth.PackFromCurrentUser(actor)
		out, err = s.nsSvc.List(ctx, k8s.NamespaceListQuery{ClusterID: clusterID, Keyword: getStr("keyword")}, &pack)
	case "list_events":
		if err = requireK8sRead(); err != nil {
			break
		}
		if s.eventSvc == nil {
			err = fmt.Errorf("Event 服务不可用")
			break
		}
		ctx = withActorContext(ctx, actor)
		limit := int64(getUint("limit", 50))
		out, err = s.eventSvc.List(ctx, k8s.EventListQuery{ClusterID: clusterID, Namespace: namespace, Keyword: getStr("keyword"), Limit: limit})
	case "search_logs":
		if err = requireProject(); err != nil {
			break
		}
		if s.logSearch == nil {
			err = fmt.Errorf("日志检索服务不可用")
			break
		}
		q := s.buildLogSearchQuery(getUint, getStr, projectID, clusterID)
		out, err = s.logSearch.Search(ctx, q)
	case "analyze_logs", "list_log_sources", "list_loggie_status", "list_cluster_log_rules":
		out, err = s.executeLogTool(ctx, name, getUint, getStr, projectID, clusterID, requireProject, actor)
	case "list_cicd_builds":
		if err = requireProject(); err != nil {
			break
		}
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
		if err = requireProject(); err != nil {
			break
		}
		if s.cicdSvc == nil {
			err = fmt.Errorf("CI/CD 服务不可用")
			break
		}
		out, err = s.cicdSvc.GetBuildRun(ctx, projectID, getUint("run_id", 0), actor)
	case "get_cicd_build_log":
		if err = requireProject(); err != nil {
			break
		}
		if s.cicdSvc == nil {
			err = fmt.Errorf("CI/CD 服务不可用")
			break
		}
		var logs string
		logs, err = s.cicdSvc.GetBuildRunLog(ctx, projectID, getUint("run_id", 0), actor)
		out = truncateStr(logs, 12_000)
	case "list_alerts":
		if err = requireProject(); err != nil {
			break
		}
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
		if err = requireActor(); err != nil {
			break
		}
		if s.alertSvc == nil {
			err = fmt.Errorf("告警服务不可用")
			break
		}
		out, err = s.alertSvc.ExplainFingerprintDelivery(ctx, getStr("fingerprint"))
	case "list_alert_datasources", "query_prometheus", "query_prometheus_range", "list_prometheus_active_alerts", "get_alert_detail":
		out, err = s.executeMonitorTool(ctx, name, getUint, getStr, projectID, actor, requireProject, requireActor)
	case "list_servers", "get_server", "test_server_connectivity", "list_db_instances", "list_es_connections":
		out, err = s.executePlatformTool(ctx, name, args, getUint, getStr, projectID, actor, requireProject, requireActor)
	case "scale_deployment", "restart_deployment", "delete_pod":
		if err = requireK8sAdmin(); err != nil {
			break
		}
		if err = checkWriteToolPolicy(name, argsJSON, namespace, getStr("reason")); err != nil {
			break
		}
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
	if n <= 0 {
		return s
	}
	// n 按 rune 截断，避免中文被截断成非法 UTF-8
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n]) + "…(truncated)"
}
