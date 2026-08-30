package system

// storageDictSeeds 存储与日志管道域内置字典种子：MinIO、MySQL/ES 备份调度、Elasticsearch、Kafka。
func storageDictSeeds() []DictEntryCreateRequest {
	return []DictEntryCreateRequest{
		// MinIO（MySQL 备份归档，字典权威来源）
		{DictType: "minio_endpoint", Label: "MinIO Endpoint", Value: "127.0.0.1:9000", Sort: intRef(1), Status: 0, Remark: "S3 API 端口，填 9000（勿填 9001 控制台端口）；如 127.0.0.1:9000"},
		{DictType: "minio_access_key", Label: "MinIO AccessKey", Value: "", Sort: intRef(1), Status: 0, Remark: "MinIO 访问密钥"},
		{DictType: "minio_secret_key", Label: "MinIO SecretKey", Value: "", Sort: intRef(1), Status: 0, Remark: "MinIO 秘密密钥"},
		{DictType: "minio_bucket", Label: "MinIO Bucket", Value: "yunshu-mysql-backup", Sort: intRef(1), Status: 0, Remark: "备份归档桶名"},
		{DictType: "minio_use_ssl", Label: "MinIO 使用 SSL", Value: "false", Sort: intRef(1), Status: 0, Remark: "true/false"},
		{DictType: "minio_region", Label: "MinIO Region", Value: "", Sort: intRef(1), Status: 0, Remark: "可选"},
		{DictType: "minio_backup_prefix", Label: "对象前缀", Value: "mysql-backups", Sort: intRef(1), Status: 0, Remark: "对象键前缀，如 mysql-backups"},
		// 备份调度 Worker
		{DictType: "mysql_backup_scheduler_enabled", Label: "启用 MySQL 定时备份 Worker", Value: "true", Sort: intRef(1), Status: 0, Remark: "后台 Cron 调度总开关"},
		{DictType: "mysql_backup_scheduler_tick_spec", Label: "调度轮询 Cron", Value: "*/30 * * * * *", Sort: intRef(1), Status: 0, Remark: "六段式 Cron，用于轮询各实例 cron_spec 是否到点"},
		{DictType: "esmgmt_backup_scheduler_enabled", Label: "启用 ES 索引定时备份 Worker", Value: "true", Sort: intRef(1), Status: 0, Remark: "esmgmt 后台 Cron 调度总开关"},
		{DictType: "esmgmt_backup_scheduler_tick_spec", Label: "ES 备份调度轮询 Cron", Value: "*/30 * * * * *", Sort: intRef(1), Status: 0, Remark: "六段式 Cron，轮询 esmgmt 备份规则"},
		// Elasticsearch（字典优先，config.yaml 兜底）
		{DictType: "elasticsearch_enabled", Label: "启用 ES 日志检索", Value: "false", Sort: intRef(1), Status: 1, Remark: "true/false；Loggie 采集写入 ES 后开启"},
		{DictType: "elasticsearch_addresses", Label: "ES 地址列表", Value: "http://127.0.0.1:9200", Sort: intRef(1), Status: 1, Remark: "单节点填一个；集群可 JSON 数组或逗号分隔"},
		{DictType: "elasticsearch_username", Label: "ES 用户名", Value: "", Sort: intRef(1), Status: 0, Remark: "Basic 认证用户名"},
		{DictType: "elasticsearch_password", Label: "ES 密码", Value: "", Sort: intRef(1), Status: 0, Remark: "敏感：Basic 认证密码"},
		{DictType: "elasticsearch_index_pattern", Label: "ES 索引模式（主机）", Value: "yunshu-agent-*", Sort: intRef(1), Status: 1, Remark: "主机 Agent 检索/保留通配"},
		{DictType: "elasticsearch_k8s_index_prefix", Label: "ES 索引前缀（集群）", Value: "yunshu-k8s", Sort: intRef(1), Status: 1, Remark: "集群采集：{prefix}-{clusterId}-p{projectId}-YYYY.MM.DD"},
		{DictType: "elasticsearch_default_retention_days", Label: "默认保留天数", Value: "30", Sort: intRef(1), Status: 1, Remark: "elasticsearch.default_retention_days"},
		{DictType: "elasticsearch_cleanup_cron_spec", Label: "清理 Cron", Value: "0 3 * * *", Sort: intRef(1), Status: 1, Remark: "elasticsearch.cleanup_cron_spec"},
		// Kafka 日志中转（字典优先；enabled=true 时 Loggie→Kafka→Yunshu 写 ES）
		{DictType: "kafka_enabled", Label: "启用 Kafka 中转", Value: "false", Sort: intRef(1), Status: 1, Remark: "true/false；关闭则 Loggie 直写 ES"},
		{DictType: "kafka_brokers", Label: "Kafka Broker 列表", Value: "127.0.0.1:9092", Sort: intRef(1), Status: 1, Remark: "单节点一个地址；集群 JSON 数组或逗号分隔 host:port"},
		{DictType: "kafka_topic_prefix", Label: "Kafka Topic 前缀（主机）", Value: "yunshu-agent", Sort: intRef(1), Status: 1, Remark: "主机 Agent Topic：{prefix}-{服务器IP}-YYYY.MM.DD"},
		{DictType: "kafka_k8s_topic_prefix", Label: "Kafka Topic 前缀（集群）", Value: "yunshu-k8s", Sort: intRef(1), Status: 1, Remark: "集群采集 Topic：{prefix}-{clusterId}-p{projectId}-YYYY.MM.DD"},
		{DictType: "kafka_topic", Label: "Kafka Topic 前缀（兼容）", Value: "yunshu-agent", Sort: intRef(2), Status: 0, Remark: "兼容旧键；优先使用 kafka_topic_prefix"},
		{DictType: "kafka_consumer_group", Label: "Kafka 消费组", Value: "yunshu-log-es", Sort: intRef(1), Status: 1, Remark: "Yunshu 写 ES 消费者组"},
		{DictType: "kafka_username", Label: "Kafka 用户名", Value: "", Sort: intRef(1), Status: 0, Remark: "SASL 用户名，可选"},
		{DictType: "kafka_password", Label: "Kafka 密码", Value: "", Sort: intRef(1), Status: 0, Remark: "敏感：SASL 密码"},
		{DictType: "kafka_sasl_mechanism", Label: "Kafka SASL 机制", Value: "plain", Sort: intRef(1), Status: 0, Remark: "plain / scram-sha-256 / scram-sha-512"},
		{DictType: "kafka_batch_size", Label: "Kafka 消费批大小", Value: "200", Sort: intRef(1), Status: 1, Remark: "写入 ES 的批量条数"},
		{DictType: "kafka_workers", Label: "Kafka 消费并发", Value: "1", Sort: intRef(1), Status: 1, Remark: "消费者并发数"},
	}
}
