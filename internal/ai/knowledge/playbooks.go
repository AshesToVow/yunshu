package knowledge

// PlaybookDocs 常见运维场景步骤（RAG 用，偏「怎么查」而非空泛概念）。
func PlaybookDocs() []Doc {
	return []Doc{
		{
			Module: ModuleK8s,
			Source: "playbook:pod-crashloop",
			Title:  "Pod CrashLoopBackOff 排查步骤",
			Content: `步骤：1) list_pods/diagnose_pod 确认 Restart 与 Reason；2) get_pod_logs（含上一容器若支持）找 panic/exit；3) list_events 看 Failed/Unhealthy；4) get_pod_detail 看探针/环境变量/挂载；5) run_diagnose_runbook(CrashLoopBackOff)。
常见根因：应用配置错误、依赖不可达、探针过严、OOMKilled、错误命令。
输出应对用户给出：错误日志关键行、建议改探针/配置/资源、是否需要重启（写工具需审批）。`,
		},
		{
			Module: ModuleK8s,
			Source: "playbook:pod-imagepull",
			Title:  "ImagePullBackOff 排查步骤",
			Content: `步骤：diagnose_pod/list_events 确认 ErrImagePull；核对镜像地址与 tag；检查 imagePullSecrets / 仓库连通；Harbor 项目权限。
不要建议「随便改镜像」而不核对仓库凭证。`,
		},
		{
			Module: ModuleK8s,
			Source: "playbook:pod-pending",
			Title:  "Pod Pending 排查步骤",
			Content: `步骤：diagnose_pod + list_events 看 FailedScheduling；检查 CPU/内存请求、节点污点、亲和、PVC Bound、ResourceQuota。
输出：调度失败原文 + 建议调整 requests 或扩容节点（写操作需审批）。`,
		},
		{
			Module: ModuleCICD,
			Source: "playbook:cicd-build-fail",
			Title:  "构建失败排查步骤",
			Content: `必须有 project_id。list_cicd_builds 找失败记录 → get_cicd_build 看状态/阶段 → get_cicd_build_log 定位首个 ERROR。
区分：编译失败、测试失败、推镜像失败、回调超时。给出日志关键摘录与修复建议；不要编造 Jenkins 控制台内容。`,
		},
		{
			Module: ModuleAlert,
			Source: "playbook:alert-not-received",
			Title:  "告警未收到排查步骤",
			Content: `list_alerts 确认事件存在与 status → 取 fingerprint → explain_alert 看投递/跳过/抑制原因。
对照：订阅是否启用、标签是否匹配、渠道是否成功、是否被抑制/静默。`,
		},
		{
			Module: ModuleLog,
			Source: "playbook:log-empty",
			Title:  "项目日志查不到排查步骤",
			Content: `确认 project_id；search_logs 用明确 keyword；缩小 namespace/pod。
仍为空：检查采集规则是否启用、Agent/DaemonSet 状态、Kafka Topic 与 ES 索引前缀是否匹配租户（含 clusterId/projectId）。`,
		},
		{
			Module: ModuleLinux,
			Source: "playbook:linux-disk-full",
			Title:  "磁盘打满排查步骤",
			Content: `先 linux.disk.check(path) 看 used_ratio；可选 linux.mem.check / linux.load.check。
说明：脚本工具探测 AI 运行环境本机。远端纳管主机请到服务器操作台 SSH/SFTP。清理前必须确认路径，禁止建议 rm -rf /。`,
		},
		{
			Module: ModuleAI,
			Source: "playbook:ask-user-for-ids",
			Title:  "向用户索要上下文的话术",
			Content: `缺 cluster_id：先 list_clusters，或请用户在助手页选择集群。
缺 project_id：请用户选择项目（日志/CI/告警工具必需）。
缺资源名：请用户提供 namespace + Pod/Deployment 名称，或先 list_pods/list_deployments。`,
		},
	}
}
