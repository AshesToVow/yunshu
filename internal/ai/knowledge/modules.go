package knowledge

import "strings"

// Module 知识库功能模块标识。
const (
	ModuleAI     = "ai"
	ModuleK8s    = "k8s"
	ModuleCICD   = "cicd"
	ModuleAlert  = "alert"
	ModuleLog    = "log"
	ModuleEsmgmt = "esmgmt"
	ModuleCMDB   = "cmdb"
	ModuleDB     = "dbmgmt"
)

// Doc 内嵌知识文档。
type Doc struct {
	Module  string
	Source  string
	Title   string
	Content string
}

// ModuleDocs 按功能模块整理的平台运维知识（供 RAG 内嵌与 ES 同步）。
func ModuleDocs() []Doc {
	return []Doc{
		{
			Module: ModuleAI,
			Source: "doc:ai",
			Title:  "AI 运维助手",
			Content: `Yunshu AI 模块：字典 ai_* 配置 Provider（openai_compat/deepseek/anthropic）。
API：status/ping/chat/sessions、k8s pod-diagnose、cicd build-fail、alert explain、approvals、knowledge/sync。
对话默认只读工具；写工具（扩缩容/重启/删 Pod）仅创建审批单，需在「AI 操作审批」审核执行。
会话持久化在 MySQL（ai_chat_sessions / ai_chat_messages），按用户隔离。`,
		},
		{
			Module: ModuleK8s,
			Source: "doc:k8s",
			Title:  "Kubernetes 运维",
			Content: `K8s 模块：集群接入、命名空间、Workload（Deployment/StatefulSet 等）、Pod、Event、Helm、节点排水。
排障常用：CrashLoopBackOff（查日志/探针/资源）、ImagePullBackOff（镜像与凭据）、Pending（调度/资源/污点）。
助手工具：list_clusters、list_namespaces、list_pods、get_pod_detail、get_pod_logs、diagnose_pod、run_diagnose_runbook、list_deployments、list_events。
写操作经审批：scale_deployment、restart_deployment、delete_pod。`,
		},
		{
			Module: ModuleCICD,
			Source: "doc:cicd",
			Title:  "CI/CD 构建发布",
			Content: `CI/CD：项目下服务、构建流水线（Jenkins）、构建记录 build-runs、发布记录 release-runs、制品与镜像仓库、审批流。
失败分析：查看构建日志、阶段状态、制品路径；可用场景接口 /ai/cicd/build-fail。
助手工具：list_cicd_builds、get_cicd_build、get_cicd_build_log。参数需 project_id 与 run_id。`,
		},
		{
			Module: ModuleAlert,
			Source: "doc:alert",
			Title:  "告警与通知",
			Content: `告警：数据源、规则、事件、订阅标签链、抑制、渠道（钉钉/企微/邮件）、投递指纹。
排查投递：用 fingerprint 解释为何发送/跳过/抑制。
助手工具：list_alerts、explain_alert。场景接口 /ai/alert/explain。`,
		},
		{
			Module: ModuleLog,
			Source: "doc:log-platform",
			Title:  "日志平台",
			Content: `日志平台双模式：主机 Agent（Loggie）与 K8s DaemonSet。
链路：采集 → Kafka Topic → ES 索引。主机前缀 kafka_topic_prefix / elasticsearch_index_pattern；
集群前缀 kafka_k8s_topic_prefix / elasticsearch_k8s_index_prefix（默认 yunshu-k8s）。
租户隔离：资源名带 projectId；索引/topic 含 clusterId 与 projectId。
检索：项目日志检索 API；助手工具 search_logs（project_id + keyword）。
esmgmt 管 ES 连接/索引，不替代日志平台检索。`,
		},
		{
			Module: ModuleEsmgmt,
			Source: "doc:esmgmt",
			Title:  "Elasticsearch 管理",
			Content: `esmgmt：连接管理、cluster health、索引列表、受限 REST 代理、索引备份恢复（MinIO）。
与日志平台职责分离：日志查询走 log-platform / project-logs；集群运维走 esmgmt。`,
		},
		{
			Module: ModuleCMDB,
			Source: "doc:cmdb",
			Title:  "CMDB 主机资产",
			Content: `CMDB：云厂商账号、主机资产、SSH 连通、与项目/部署关联。
构建机、Loggie Agent 主机等多依赖 CMDB Server 记录。AI 当前以只读说明为主，优先结合项目与集群上下文。`,
		},
		{
			Module: ModuleDB,
			Source: "doc:dbmgmt",
			Title:  "数据库管理",
			Content: `dbmgmt：MySQL 实例纳管、账号授权、查询权限、审批流；另有 mysqlbackup 备份任务。
高危变更走审批，不在 AI 助手直接执行写库。`,
		},
	}
}

// InferModules 根据用户问题关键词推断相关功能模块。
func InferModules(query string) []string {
	q := strings.ToLower(query)
	type rule struct {
		module string
		keys   []string
	}
	rules := []rule{
		{ModuleK8s, []string{"k8s", "kubernetes", "pod", "deployment", "namespace", "命名空间", "集群", "排障", "crashloop", "imagepull", "pending", "helm", "节点"}},
		{ModuleCICD, []string{"cicd", "ci/cd", "jenkins", "构建", "发布", "流水线", "build", "release", "制品", "镜像"}},
		{ModuleAlert, []string{"告警", "alert", "prometheus", "alertmanager", "通知", "抑制", "指纹", "fingerprint", "订阅"}},
		{ModuleLog, []string{"日志", "log", "loggie", "kafka", "elasticsearch", "es 索引", "检索日志"}},
		{ModuleEsmgmt, []string{"esmgmt", "es 管理", "索引备份", "es 连接"}},
		{ModuleCMDB, []string{"cmdb", "主机", "服务器", "资产", "ssh"}},
		{ModuleDB, []string{"数据库", "mysql", "dbmgmt", "授权", "备份库"}},
		{ModuleAI, []string{"ai", "助手", "审批", "tool", "rag", "知识库"}},
	}
	seen := map[string]struct{}{}
	var out []string
	for _, r := range rules {
		for _, k := range r.keys {
			if strings.Contains(q, k) {
				if _, ok := seen[r.module]; !ok {
					seen[r.module] = struct{}{}
					out = append(out, r.module)
				}
				break
			}
		}
	}
	return out
}
