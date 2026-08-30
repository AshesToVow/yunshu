package system

// alertDictSeeds 告警域内置字典种子：Webhook、增强查询、数据源、PromQL 标签键、阈值单位、认领时长。
func alertDictSeeds() []DictEntryCreateRequest {
	return []DictEntryCreateRequest{
		{DictType: "alert_webhook_url", Label: "企业微信机器人 URL 示例", Value: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=replace-me", Sort: intRef(1), Status: 0, Remark: "Webhook 通道 URL 候选"},
		// Alert 运行配置（字典优先，YAML 兜底）
		{DictType: "alert_webhook_token", Label: "Webhook Token 示例", Value: "change-me-alert-token", Sort: intRef(1), Status: 0, Remark: "alert.webhook_token：K8s Event 等内部入站鉴权（非 Alertmanager）"},
		{DictType: "alert_enrich_prometheus_url", Label: "Prometheus 地址示例", Value: "http://127.0.0.1:9090", Sort: intRef(1), Status: 0, Remark: "alert.prometheus_url：用于告警增强查询"},
		{DictType: "alert_enrich_prometheus_token", Label: "Prometheus Token（可选）", Value: "", Sort: intRef(1), Status: 0, Remark: "alert.prometheus_token：敏感信息建议仅在生产库维护"},
		{DictType: "alert_datasource_base_url", Label: "Prometheus 根地址示例", Value: "http://prometheus:9090", Sort: intRef(1), Status: 1, Remark: "告警数据源 Base URL，可在数据字典增删"},
		{DictType: "alert_datasource_basic_user", Label: "Basic 用户示例", Value: "prometheus", Sort: intRef(1), Status: 1, Remark: "告警数据源 Basic 用户，可在数据字典增删"},
		// PromQL 标签键候选（规则编辑页下拉）
		{DictType: "alert_promql_label_key", Label: "instance", Value: "instance", Sort: intRef(1), Status: 1, Remark: "PromQL 标签键"},
		{DictType: "alert_promql_label_key", Label: "job", Value: "job", Sort: intRef(2), Status: 1, Remark: "PromQL 标签键"},
		{DictType: "alert_promql_label_key", Label: "cluster", Value: "cluster", Sort: intRef(3), Status: 1, Remark: "PromQL 标签键"},
		{DictType: "alert_promql_label_key", Label: "namespace", Value: "namespace", Sort: intRef(4), Status: 1, Remark: "PromQL 标签键"},
		{DictType: "alert_promql_label_key", Label: "pod", Value: "pod", Sort: intRef(5), Status: 1, Remark: "PromQL 标签键"},
		{DictType: "alert_promql_label_key", Label: "service", Value: "service", Sort: intRef(6), Status: 1, Remark: "PromQL 标签键"},
		{DictType: "alert_promql_label_key", Label: "node", Value: "node", Sort: intRef(7), Status: 1, Remark: "PromQL 标签键"},
		{DictType: "alert_promql_label_key", Label: "severity", Value: "severity", Sort: intRef(8), Status: 1, Remark: "PromQL 标签键"},
		{DictType: "alert_promql_label_key", Label: "alertname", Value: "alertname", Sort: intRef(9), Status: 1, Remark: "PromQL 标签键"},
		{DictType: "alert_promql_label_key", Label: "path", Value: "path", Sort: intRef(10), Status: 1, Remark: "PromQL 标签键"},
		{DictType: "alert_promql_label_key", Label: "device", Value: "device", Sort: intRef(11), Status: 1, Remark: "PromQL 标签键"},
		{DictType: "alert_promql_label_key", Label: "fstype", Value: "fstype", Sort: intRef(12), Status: 1, Remark: "PromQL 标签键"},
		{DictType: "alert_promql_label_key", Label: "mountpoint", Value: "mountpoint", Sort: intRef(13), Status: 1, Remark: "PromQL 标签键"},
		// 阈值单位候选
		{DictType: "alert_threshold_unit", Label: "原始值", Value: "raw", Sort: intRef(1), Status: 1, Remark: "不指定单位"},
		{DictType: "alert_threshold_unit", Label: "百分比(%)", Value: "percent", Sort: intRef(2), Status: 1, Remark: "用于使用率类指标"},
		{DictType: "alert_threshold_unit", Label: "字节(bytes)", Value: "bytes", Sort: intRef(3), Status: 1, Remark: "用于容量类指标"},
		{DictType: "alert_threshold_unit", Label: "毫秒(ms)", Value: "ms", Sort: intRef(4), Status: 1, Remark: "用于耗时类指标"},
		{DictType: "alert_threshold_unit", Label: "计数(count)", Value: "count", Sort: intRef(5), Status: 1, Remark: "用于请求数/错误数"},
		// 告警认领时长（分钟）；排序第一项为默认，前端下拉与后端 TTL 都读此字典
		{DictType: "alert_ack_ttl_minutes", Label: "15 分钟", Value: "15", Sort: intRef(1), Status: 1, Remark: "认领时长；排序第一项为默认"},
		{DictType: "alert_ack_ttl_minutes", Label: "30 分钟", Value: "30", Sort: intRef(2), Status: 1, Remark: "认领时长（分钟）"},
		{DictType: "alert_ack_ttl_minutes", Label: "1 小时", Value: "60", Sort: intRef(3), Status: 1, Remark: "认领时长（分钟）"},
		{DictType: "alert_ack_ttl_minutes", Label: "2 小时", Value: "120", Sort: intRef(4), Status: 1, Remark: "认领时长（分钟）"},
		{DictType: "alert_ack_ttl_minutes", Label: "4 小时", Value: "240", Sort: intRef(5), Status: 1, Remark: "认领时长（分钟）"},
	}
}
