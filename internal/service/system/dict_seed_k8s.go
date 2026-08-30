package system

// k8sDictSeeds K8s 域内置字典种子：Event 转发、kubeconfig 模板、直连配置模板。
func k8sDictSeeds() []DictEntryCreateRequest {
	return []DictEntryCreateRequest{
		// K8s Event 多集群转发（字典优先，YAML 兜底；入站 /alerts/ingress/k8s-events）
		{DictType: "k8s_event_forward_enabled", Label: "启用 Event 转发", Value: "false", Sort: intRef(1), Status: 0, Remark: "k8s_event_forward.enabled：true/false"},
		{DictType: "k8s_event_forward_watcher_buffer_size", Label: "监听通道缓冲", Value: "1000", Sort: intRef(1), Status: 0, Remark: "k8s_event_forward.watcher_buffer_size"},
		{DictType: "k8s_event_forward_worker_interval_seconds", Label: "批处理周期(秒)", Value: "10", Sort: intRef(1), Status: 0, Remark: "k8s_event_forward.worker_interval_seconds"},
		{DictType: "k8s_event_forward_worker_batch_size", Label: "批大小", Value: "50", Sort: intRef(1), Status: 0, Remark: "k8s_event_forward.worker_batch_size"},
		{DictType: "k8s_event_forward_worker_max_retries", Label: "最大重试", Value: "3", Sort: intRef(1), Status: 0, Remark: "k8s_event_forward.worker_max_retries"},
		// 集群 kubeconfig 模板（请替换 server/token）；「集群管理」表单可一键插入
		{DictType: "k8s_kubeconfig_template", Label: "单集群 kubeconfig 模板", Value: "kubeconfig文件", Sort: intRef(1), Status: 1, Remark: "占位说明：可在字典中维护完整 kubeconfig 供集群管理选择；勿将生产密钥提交到 Git"},
		// 集群直连配置模板：label 作为配置键，value 存直连 JSON（可在集群管理 direct 模式通过 dict_config_key 引用）
		{DictType: "k8s_direct_config", Label: "prod-sa-token", Value: `{"server":"https://10.0.0.10:6443","token":"replace-with-service-account-token","ca_data":"replace-with-base64-ca","insecure_skip_tls_verify":false}`, Sort: intRef(1), Status: 1, Remark: "生产集群直连示例（token 认证）"},
		{DictType: "k8s_direct_config", Label: "staging-basic-auth", Value: `{"server":"https://10.0.0.20:6443","username":"admin","password":"replace-with-password","insecure_skip_tls_verify":true}`, Sort: intRef(2), Status: 1, Remark: "测试集群直连示例（用户名密码认证）"},
	}
}
