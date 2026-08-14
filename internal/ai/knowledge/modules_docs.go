package knowledge

// ModuleDocs 按功能模块整理的平台运维知识（供 RAG 内嵌与 ES 同步）。
func ModuleDocs() []Doc {
	docs := []Doc{
		{
			Module: ModuleAI,
			Source: "doc:ai",
			Title:  "AI 运维助手能力边界",
			Content: `Yunshu AI 运维助手：字典 ai_* 配置 Provider（openai_compat / deepseek / anthropic）。
能力：对话（tools+RAG）、会话落库、Pod/CI/告警场景分析、高危写操作审批。
只读工具可直接查集群/日志/构建/告警；写工具（扩缩容/重启/删 Pod）只创建审批单，需在「AI → 操作审批」审核后执行。
不要指望助手直接改库、改 Jenkins、改告警规则；这类需求引导到对应菜单并说明需人工操作。`,
		},
		{
			Module: ModuleK8s,
			Source: "doc:k8s",
			Title:  "Kubernetes 模块与助手用法",
			Content: `菜单：集群、命名空间、Workload、Pod、Event、Helm、节点等。
助手查集群：先 list_clusters 得到 id/名称，再带 cluster_id 调用 Pod/事件工具。
排障优先级：diagnose_pod（确定性诊断）> get_pod_logs / list_events > run_diagnose_runbook。
常见现象：
- CrashLoopBackOff：查上一容器日志、探针、启动命令、配置/密钥挂载、资源限制。
- ImagePullBackOff：镜像名/tag、镜像仓库凭证、网络策略。
- Pending：资源不足、污点容忍、PVC、调度约束。
写操作：scale_deployment / restart_deployment / delete_pod → 审批后执行。`,
		},
		{
			Module: ModuleCICD,
			Source: "doc:cicd",
			Title:  "CI/CD 构建发布与失败排查",
			Content: `菜单：项目下 CI/CD 服务、构建记录、发布记录、审批流、镜像仓库。
失败排查顺序：list_cicd_builds(project_id) → 定位失败 run_id → get_cicd_build → get_cicd_build_log。
关注：Jenkins job 阶段、编译/单测错误、制品上传、镜像推送 Harbor、审批卡点。
场景接口也可：POST /ai/cicd/build-fail（需 project_id + run_id）。
助手不能直接触发构建或跳过审批。`,
		},
		{
			Module: ModuleAlert,
			Source: "doc:alert",
			Title:  "告警投递与噪音治理",
			Content: `菜单：告警事件、数据源、规则、订阅、抑制、通知渠道（钉钉/企微/邮件）。
「为什么没收到/重复收到」：list_alerts 找到 fingerprint → explain_alert。
常见原因：订阅标签不匹配、抑制规则、静默、渠道失败、指纹聚合、resolved 不投递。
助手不直接改规则；给出应检查的订阅标签链与抑制配置项即可。`,
		},
		{
			Module: ModuleLog,
			Source: "doc:log-platform",
			Title:  "日志平台采集与检索",
			Content: `双模式：主机 Loggie Agent；K8s DaemonSet（按项目租户隔离）。
链路：采集 → Kafka Topic → Elasticsearch。
主机前缀：kafka_topic_prefix / elasticsearch_index_pattern。
集群前缀：kafka_k8s_topic_prefix / elasticsearch_k8s_index_prefix（默认 yunshu-k8s）。
检索：助手 search_logs 需要 project_id + keyword，可选 namespace/pod/cluster_id。
查不到日志时排查：规则是否启用、DaemonSet/Agent 是否部署、Topic/索引是否存在、时间范围与租户前缀。
esmgmt 管 ES 连接与索引运维，不替代项目日志检索。`,
		},
		{
			Module: ModuleEsmgmt,
			Source: "doc:esmgmt",
			Title:  "Elasticsearch 管理（esmgmt）",
			Content: `用于 ES 连接、health、索引列表、受限 REST、备份恢复（MinIO）。
与日志平台分离：业务日志查询走日志平台/search_logs；集群运维走 esmgmt。
助手当前无 esmgmt 专用工具，能力说明即可。`,
		},
		{
			Module: ModuleCMDB,
			Source: "doc:cmdb",
			Title:  "CMDB 主机资产",
			Content: `主机、云账号、SSH 连通；构建机与 Loggie Agent 依赖 CMDB Server。
助手暂无主机列表工具；涉及 Agent 安装/构建机问题时，引导到项目服务器与 CMDB 菜单核对。`,
		},
		{
			Module: ModuleDB,
			Source: "doc:dbmgmt",
			Title:  "数据库管理与备份",
			Content: `MySQL 纳管、授权、查询权限、审批；mysqlbackup 备份任务。
高危变更必须走审批；助手不执行写库或授权。`,
		},
	}
	docs = append(docs, PlaybookDocs()...)
	return docs
}
