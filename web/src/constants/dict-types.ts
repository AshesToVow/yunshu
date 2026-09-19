export type DictTypeOption = { label: string; value: string };

export type DictCategoryId = "all" | "system" | "alert" | "log" | "k8s" | "cmdb" | "backup" | "dbmgmt" | "cicd" | "ai" | "other";

type DictCategoryMeta = {
  id: Exclude<DictCategoryId, "all">;
  label: string;
  color: string;
  description: string;
};

export const DICT_CATEGORY_TABS: { id: DictCategoryId; label: string }[] = [
  { id: "all", label: "全部" },
  { id: "system", label: "系统" },
  { id: "alert", label: "告警" },
  { id: "log", label: "日志" },
  { id: "k8s", label: "Kubernetes" },
  { id: "cmdb", label: "CMDB / 服务器" },
  { id: "backup", label: "备份 / MinIO" },
  { id: "dbmgmt", label: "数据库管理" },
  { id: "cicd", label: "CI/CD" },
  { id: "ai", label: "AI" },
  { id: "other", label: "其他" },
];

export const DICT_CATEGORY_META: Record<Exclude<DictCategoryId, "all">, DictCategoryMeta> = {
  system: { id: "system", label: "系统", color: "blue", description: "邮件、认证会话、密码策略、通用状态等基础配置" },
  alert: { id: "alert", label: "告警", color: "volcano", description: "告警规则、通道、Prometheus、企微/钉钉" },
  log: { id: "log", label: "日志", color: "cyan", description: "ES / Kafka / Agent、采集源等平台配置" },
  k8s: { id: "k8s", label: "Kubernetes", color: "purple", description: "集群模板、Event 转发、Pod 调试镜像等" },
  cmdb: { id: "cmdb", label: "CMDB / 服务器", color: "geekblue", description: "服务器分组、云厂商凭据模板" },
  backup: { id: "backup", label: "备份 / MinIO", color: "orange", description: "MySQL 备份归档与 MinIO 连接" },
  dbmgmt: { id: "dbmgmt", label: "数据库管理", color: "gold", description: "SQL 查询、审批、goInception 等平台配置" },
  cicd: { id: "cicd", label: "CI/CD", color: "green", description: "Jenkins、流水线、发布枚举" },
  ai: { id: "ai", label: "AI", color: "magenta", description: "大模型 Provider、开关与调用参数" },
  other: { id: "other", label: "其他", color: "default", description: "未归类的自定义 dict_type" },
};

type DictTypeDef = {
  label: string;
  value: string;
  category: Exclude<DictCategoryId, "all">;
};

const DICT_TYPE_DEFS: DictTypeDef[] = [
  { label: "通用状态（common_status）", value: "common_status", category: "system" },
  { label: "邮件主机（mail_host）", value: "mail_host", category: "system" },
  { label: "邮件端口（mail_port）", value: "mail_port", category: "system" },
  { label: "邮件用户名（mail_username）", value: "mail_username", category: "system" },
  { label: "邮件密码（mail_password）", value: "mail_password", category: "system" },
  { label: "发件邮箱（mail_from_email）", value: "mail_from_email", category: "system" },
  { label: "发件名称（mail_from_name）", value: "mail_from_name", category: "system" },
  { label: "是否启用 TLS（mail_use_tls）", value: "mail_use_tls", category: "system" },
  { label: "密码最小长度（password_min_length）", value: "password_min_length", category: "system" },
  { label: "密码最大长度（password_max_length）", value: "password_max_length", category: "system" },
  { label: "须含大写字母（password_require_upper）", value: "password_require_upper", category: "system" },
  { label: "须含小写字母（password_require_lower）", value: "password_require_lower", category: "system" },
  { label: "须含数字（password_require_digit）", value: "password_require_digit", category: "system" },
  { label: "须含特殊字符（password_require_special）", value: "password_require_special", category: "system" },
  { label: "密码过期天数（password_expiry_days）", value: "password_expiry_days", category: "system" },
  { label: "禁止包含用户名（password_forbid_username）", value: "password_forbid_username", category: "system" },
  { label: "JWT 签名密钥（auth_jwt_secret）", value: "auth_jwt_secret", category: "system" },
  { label: "Access Token 有效期分钟（auth_access_token_ttl_minutes）", value: "auth_access_token_ttl_minutes", category: "system" },
  { label: "Refresh Token 有效期小时（auth_refresh_token_ttl_hours）", value: "auth_refresh_token_ttl_hours", category: "system" },
  { label: "Cookie Secure（auth_cookie_secure）", value: "auth_cookie_secure", category: "system" },
  { label: "Cookie Domain（auth_cookie_domain）", value: "auth_cookie_domain", category: "system" },
  { label: "启用 CSP（auth_csp_enabled）", value: "auth_csp_enabled", category: "system" },
  { label: "邮箱验证码有效期秒（auth_email_code_ttl_seconds）", value: "auth_email_code_ttl_seconds", category: "system" },
  { label: "邮箱验证码冷却秒（auth_email_code_cooldown_seconds）", value: "auth_email_code_cooldown_seconds", category: "system" },
  { label: "登录失败锁定次数（auth_login_max_fail_attempts）", value: "auth_login_max_fail_attempts", category: "system" },
  { label: "登录锁定时长秒（auth_login_lock_seconds）", value: "auth_login_lock_seconds", category: "system" },

  { label: "告警通道类型（alert_channel_type）", value: "alert_channel_type", category: "alert" },
  { label: "告警 Webhook URL（alert_webhook_url）", value: "alert_webhook_url", category: "alert" },
  { label: "告警级别（alert_severity）", value: "alert_severity", category: "alert" },
  { label: "告警数据源地址（alert_datasource_base_url）", value: "alert_datasource_base_url", category: "alert" },
  { label: "告警数据源账号（alert_datasource_basic_user）", value: "alert_datasource_basic_user", category: "alert" },
  { label: "PromQL 标签键（alert_promql_label_key）", value: "alert_promql_label_key", category: "alert" },
  { label: "阈值单位（alert_threshold_unit）", value: "alert_threshold_unit", category: "alert" },
  { label: "告警认领时长分钟（alert_ack_ttl_minutes）", value: "alert_ack_ttl_minutes", category: "alert" },
  { label: "告警规则文案预设（alert_rule_template_preset）", value: "alert_rule_template_preset", category: "alert" },
  { label: "告警 Webhook Token（alert_webhook_token）", value: "alert_webhook_token", category: "alert" },
  { label: "告警 Prometheus URL（alert_enrich_prometheus_url）", value: "alert_enrich_prometheus_url", category: "alert" },
  { label: "告警 Prometheus Token（alert_enrich_prometheus_token）", value: "alert_enrich_prometheus_token", category: "alert" },
  { label: "企业微信 Webhook URL（wecom_webhook_url）", value: "wecom_webhook_url", category: "alert" },
  { label: "企业微信通知模式（wecom_notify_mode）", value: "wecom_notify_mode", category: "alert" },
  { label: "企业微信 corpID（wecom_corp_id）", value: "wecom_corp_id", category: "alert" },
  { label: "企业微信 corpSecret（wecom_corp_secret）", value: "wecom_corp_secret", category: "alert" },
  { label: "企业微信 agentId（wecom_agent_id）", value: "wecom_agent_id", category: "alert" },
  { label: "钉钉 Webhook URL（dingtalk_webhook_url）", value: "dingtalk_webhook_url", category: "alert" },
  { label: "钉钉通知模式（dingtalk_notify_mode）", value: "dingtalk_notify_mode", category: "alert" },
  { label: "钉钉 appKey（dingtalk_app_key）", value: "dingtalk_app_key", category: "alert" },
  { label: "钉钉 appSecret（dingtalk_app_secret）", value: "dingtalk_app_secret", category: "alert" },
  { label: "钉钉 chatId（dingtalk_chat_id）", value: "dingtalk_chat_id", category: "alert" },
  { label: "钉钉 signSecret（dingtalk_sign_secret）", value: "dingtalk_sign_secret", category: "alert" },

  { label: "日志源类型（log_source_type）", value: "log_source_type", category: "log" },
  { label: "启用 ES（elasticsearch_enabled）", value: "elasticsearch_enabled", category: "log" },
  { label: "ES 地址（elasticsearch_addresses）", value: "elasticsearch_addresses", category: "log" },
  { label: "日志平台 ES 连接 ID（elasticsearch_connection_id）", value: "elasticsearch_connection_id", category: "log" },
  { label: "ES 索引模式-主机（elasticsearch_index_pattern）", value: "elasticsearch_index_pattern", category: "log" },
  { label: "ES 索引前缀-集群（elasticsearch_k8s_index_prefix）", value: "elasticsearch_k8s_index_prefix", category: "log" },
  { label: "启用 Kafka 中转（kafka_enabled）", value: "kafka_enabled", category: "log" },
  { label: "Kafka Brokers（kafka_brokers）", value: "kafka_brokers", category: "log" },
  { label: "Kafka Topic 前缀-主机（kafka_topic_prefix）", value: "kafka_topic_prefix", category: "log" },
  { label: "Kafka Topic 前缀-集群（kafka_k8s_topic_prefix）", value: "kafka_k8s_topic_prefix", category: "log" },
  { label: "Kafka Topic（kafka_topic）", value: "kafka_topic", category: "log" },
  { label: "Kafka 消费组（kafka_consumer_group）", value: "kafka_consumer_group", category: "log" },
  { label: "Kafka 密码（kafka_password）", value: "kafka_password", category: "log" },
  { label: "Kafka 用户名（kafka_username）", value: "kafka_username", category: "log" },
  { label: "Kafka SASL 机制（kafka_sasl_mechanism）", value: "kafka_sasl_mechanism", category: "log" },
  { label: "Kafka 消费批大小（kafka_batch_size）", value: "kafka_batch_size", category: "log" },
  { label: "Kafka 消费并发（kafka_workers）", value: "kafka_workers", category: "log" },
  { label: "ES 用户名（elasticsearch_username）", value: "elasticsearch_username", category: "log" },
  { label: "ES 密码（elasticsearch_password）", value: "elasticsearch_password", category: "log" },
  { label: "ES 默认保留天数（elasticsearch_default_retention_days）", value: "elasticsearch_default_retention_days", category: "log" },
  { label: "ES 清理 Cron（elasticsearch_cleanup_cron_spec）", value: "elasticsearch_cleanup_cron_spec", category: "log" },

  { label: "K8s Event 转发开关（k8s_event_forward_enabled）", value: "k8s_event_forward_enabled", category: "k8s" },
  { label: "K8s Event 缓冲（k8s_event_forward_watcher_buffer_size）", value: "k8s_event_forward_watcher_buffer_size", category: "k8s" },
  { label: "K8s Event 批处理周期秒（k8s_event_forward_worker_interval_seconds）", value: "k8s_event_forward_worker_interval_seconds", category: "k8s" },
  { label: "K8s Event 批大小（k8s_event_forward_worker_batch_size）", value: "k8s_event_forward_worker_batch_size", category: "k8s" },
  { label: "K8s Event 最大重试（k8s_event_forward_worker_max_retries）", value: "k8s_event_forward_worker_max_retries", category: "k8s" },
  { label: "Pod 调试镜像（k8s_pod_debug_image）", value: "k8s_pod_debug_image", category: "k8s" },
  { label: "Kubeconfig 模板（k8s_kubeconfig_template）", value: "k8s_kubeconfig_template", category: "k8s" },
  { label: "K8s 直连配置键（k8s_direct_config）", value: "k8s_direct_config", category: "k8s" },

  { label: "服务器分组分类（server_group_category）", value: "server_group_category", category: "cmdb" },
  { label: "服务器系统类型（server_os_type）", value: "server_os_type", category: "cmdb" },
  { label: "服务器认证类型（server_auth_type）", value: "server_auth_type", category: "cmdb" },
  { label: "服务器 SSH 用户名模板（server_ssh_username）", value: "server_ssh_username", category: "cmdb" },
  { label: "服务器 SSH 密码模板（server_ssh_password）", value: "server_ssh_password", category: "cmdb" },
  { label: "服务器端口模板（server_port）", value: "server_port", category: "cmdb" },
  { label: "阿里云 AK（cloud_alibaba_ak）", value: "cloud_alibaba_ak", category: "cmdb" },
  { label: "阿里云 SK（cloud_alibaba_sk）", value: "cloud_alibaba_sk", category: "cmdb" },
  { label: "腾讯云 AK（cloud_tencent_ak）", value: "cloud_tencent_ak", category: "cmdb" },
  { label: "腾讯云 SK（cloud_tencent_sk）", value: "cloud_tencent_sk", category: "cmdb" },
  { label: "京东云 AK（cloud_jd_ak）", value: "cloud_jd_ak", category: "cmdb" },
  { label: "京东云 SK（cloud_jd_sk）", value: "cloud_jd_sk", category: "cmdb" },
  { label: "阿里云服务器用户名（server_cloud_alibaba_username）", value: "server_cloud_alibaba_username", category: "cmdb" },
  { label: "阿里云服务器密码（server_cloud_alibaba_password）", value: "server_cloud_alibaba_password", category: "cmdb" },
  { label: "阿里云服务器私钥（server_cloud_alibaba_private_key）", value: "server_cloud_alibaba_private_key", category: "cmdb" },
  { label: "阿里云服务器端口（server_cloud_alibaba_port）", value: "server_cloud_alibaba_port", category: "cmdb" },
  { label: "腾讯云服务器用户名（server_cloud_tencent_username）", value: "server_cloud_tencent_username", category: "cmdb" },
  { label: "腾讯云服务器密码（server_cloud_tencent_password）", value: "server_cloud_tencent_password", category: "cmdb" },
  { label: "腾讯云服务器私钥（server_cloud_tencent_private_key）", value: "server_cloud_tencent_private_key", category: "cmdb" },
  { label: "腾讯云服务器端口（server_cloud_tencent_port）", value: "server_cloud_tencent_port", category: "cmdb" },
  { label: "京东云服务器用户名（server_cloud_jd_username）", value: "server_cloud_jd_username", category: "cmdb" },
  { label: "京东云服务器密码（server_cloud_jd_password）", value: "server_cloud_jd_password", category: "cmdb" },
  { label: "京东云服务器私钥（server_cloud_jd_private_key）", value: "server_cloud_jd_private_key", category: "cmdb" },
  { label: "京东云服务器端口（server_cloud_jd_port）", value: "server_cloud_jd_port", category: "cmdb" },
  { label: "服务器文件传输上限 MB（cmdb_max_transfer_file_mb）", value: "cmdb_max_transfer_file_mb", category: "cmdb" },

  { label: "MinIO Endpoint（minio_endpoint）", value: "minio_endpoint", category: "backup" },
  { label: "MinIO AccessKey（minio_access_key）", value: "minio_access_key", category: "backup" },
  { label: "MinIO SecretKey（minio_secret_key）", value: "minio_secret_key", category: "backup" },
  { label: "MinIO Bucket（minio_bucket）", value: "minio_bucket", category: "backup" },
  { label: "MinIO SSL（minio_use_ssl）", value: "minio_use_ssl", category: "backup" },
  { label: "MinIO Region（minio_region）", value: "minio_region", category: "backup" },
  { label: "MinIO 备份前缀（minio_backup_prefix）", value: "minio_backup_prefix", category: "backup" },
  { label: "MySQL 备份 Worker 开关（mysql_backup_scheduler_enabled）", value: "mysql_backup_scheduler_enabled", category: "backup" },
  { label: "MySQL 备份调度 Cron（mysql_backup_scheduler_tick_spec）", value: "mysql_backup_scheduler_tick_spec", category: "backup" },
  { label: "ES 备份调度开关（esmgmt_backup_scheduler_enabled）", value: "esmgmt_backup_scheduler_enabled", category: "backup" },
  { label: "ES 备份调度 Cron（esmgmt_backup_scheduler_tick_spec）", value: "esmgmt_backup_scheduler_tick_spec", category: "backup" },

  { label: "SQL 查询超时秒（dbmgmt_query_timeout_seconds）", value: "dbmgmt_query_timeout_seconds", category: "dbmgmt" },
  { label: "查询最大行数（dbmgmt_max_rows）", value: "dbmgmt_max_rows", category: "dbmgmt" },
  { label: "SQL 导入上限 MB（dbmgmt_max_import_file_mb）", value: "dbmgmt_max_import_file_mb", category: "dbmgmt" },
  { label: "生产强制审批（dbmgmt_prod_force_approval）", value: "dbmgmt_prod_force_approval", category: "dbmgmt" },
  { label: "禁止自审自批（dbmgmt_forbid_self_approve）", value: "dbmgmt_forbid_self_approve", category: "dbmgmt" },
  { label: "审批超时小时（dbmgmt_approval_sla_hours）", value: "dbmgmt_approval_sla_hours", category: "dbmgmt" },
  { label: "审批提醒间隔小时（dbmgmt_approval_reminder_interval_hours）", value: "dbmgmt_approval_reminder_interval_hours", category: "dbmgmt" },
  { label: "实例探活间隔秒（dbmgmt_ping_interval_seconds）", value: "dbmgmt_ping_interval_seconds", category: "dbmgmt" },
  { label: "单实例最大并发（dbmgmt_max_concurrent_per_instance）", value: "dbmgmt_max_concurrent_per_instance", category: "dbmgmt" },
  { label: "启用 goInception（dbmgmt_goinception_enabled）", value: "dbmgmt_goinception_enabled", category: "dbmgmt" },
  { label: "goInception 地址（dbmgmt_goinception_host）", value: "dbmgmt_goinception_host", category: "dbmgmt" },
  { label: "goInception 端口（dbmgmt_goinception_port）", value: "dbmgmt_goinception_port", category: "dbmgmt" },
  { label: "goInception 备份（dbmgmt_goinception_backup）", value: "dbmgmt_goinception_backup", category: "dbmgmt" },

  { label: "CI/CD 启用（cicd_enabled）", value: "cicd_enabled", category: "cicd" },
  { label: "Jenkins 地址（cicd_jenkins_base_url）", value: "cicd_jenkins_base_url", category: "cicd" },
  { label: "Jenkins 用户（cicd_jenkins_username）", value: "cicd_jenkins_username", category: "cicd" },
  { label: "Jenkins Token（cicd_jenkins_api_token）", value: "cicd_jenkins_api_token", category: "cicd" },
  { label: "Jenkins Job 目录（cicd_jenkins_job_folder）", value: "cicd_jenkins_job_folder", category: "cicd" },
  { label: "Jenkinsfile 仓库（cicd_jenkinsfile_repo）", value: "cicd_jenkinsfile_repo", category: "cicd" },
  { label: "Jenkinsfile 分支（cicd_jenkinsfile_branch）", value: "cicd_jenkinsfile_branch", category: "cicd" },
  { label: "前端 Jenkinsfile（cicd_jenkinsfile_front）", value: "cicd_jenkinsfile_front", category: "cicd" },
  { label: "后端 Jenkinsfile（cicd_jenkinsfile_backend）", value: "cicd_jenkinsfile_backend", category: "cicd" },
  { label: "K8s Jenkinsfile（cicd_jenkinsfile_k8s）", value: "cicd_jenkinsfile_k8s", category: "cicd" },
  { label: "共享库名称（cicd_shared_library_name）", value: "cicd_shared_library_name", category: "cicd" },
  { label: "Git 凭据 ID（cicd_git_credential_id）", value: "cicd_git_credential_id", category: "cicd" },
  { label: "SSH 部署凭据 ID（cicd_ssh_deploy_credential_id）", value: "cicd_ssh_deploy_credential_id", category: "cicd" },
  { label: "MinIO 凭据 ID（cicd_minio_credential_id）", value: "cicd_minio_credential_id", category: "cicd" },
  { label: "Harbor 凭据 ID（cicd_harbor_credential_id）", value: "cicd_harbor_credential_id", category: "cicd" },
  { label: "Harbor 地址（cicd_harbor_url）", value: "cicd_harbor_url", category: "cicd" },
  { label: "Harbor 解析 IP（cicd_harbor_host_ip）", value: "cicd_harbor_host_ip", category: "cicd" },
  { label: "Harbor 项目（cicd_harbor_project_group）", value: "cicd_harbor_project_group", category: "cicd" },
  { label: "Harbor 用户名（cicd_harbor_username）", value: "cicd_harbor_username", category: "cicd" },
  { label: "Harbor 密码（cicd_harbor_password）", value: "cicd_harbor_password", category: "cicd" },
  { label: "MinIO S3 地址（cicd_minio_endpoint）", value: "cicd_minio_endpoint", category: "cicd" },
  { label: "前端制品桶（cicd_minio_bucket_frontend）", value: "cicd_minio_bucket_frontend", category: "cicd" },
  { label: "后端制品桶（cicd_minio_bucket_backend）", value: "cicd_minio_bucket_backend", category: "cicd" },
  { label: "mc 路径（cicd_mc_bin）", value: "cicd_mc_bin", category: "cicd" },
  { label: "mc 别名（cicd_mc_alias）", value: "cicd_mc_alias", category: "cicd" },
  { label: "手动发布超时（cicd_default_wait_mins）", value: "cicd_default_wait_mins", category: "cicd" },
  { label: "制品保留数（cicd_default_artifact_retain_count）", value: "cicd_default_artifact_retain_count", category: "cicd" },
  { label: "Run 同步间隔（cicd_run_sync_interval_seconds）", value: "cicd_run_sync_interval_seconds", category: "cicd" },
  { label: "审批超时阈值（cicd_approval_sla_hours）", value: "cicd_approval_sla_hours", category: "cicd" },
  { label: "审批提醒间隔（cicd_approval_reminder_interval_hours）", value: "cicd_approval_reminder_interval_hours", category: "cicd" },
  { label: "禁止自审自批（cicd_forbid_self_approve）", value: "cicd_forbid_self_approve", category: "cicd" },
  { label: "生产强制审批（cicd_prod_force_audit）", value: "cicd_prod_force_audit", category: "cicd" },
  { label: "启用 SonarQube（cicd_sonar_enabled）", value: "cicd_sonar_enabled", category: "cicd" },
  { label: "SonarQube 地址（cicd_sonar_url）", value: "cicd_sonar_url", category: "cicd" },
  { label: "SonarQube Token（cicd_sonar_token）", value: "cicd_sonar_token", category: "cicd" },
  { label: "门禁失败拦截（cicd_sonar_gate_block）", value: "cicd_sonar_gate_block", category: "cicd" },
  { label: "Jenkins 回调 HMAC（cicd_jenkins_callback_hmac_secret）", value: "cicd_jenkins_callback_hmac_secret", category: "cicd" },
  { label: "Jenkins 回调 URL（cicd_jenkins_callback_url）", value: "cicd_jenkins_callback_url", category: "cicd" },
  { label: "MinIO AccessKey Yunshu（cicd_minio_access_key）", value: "cicd_minio_access_key", category: "cicd" },
  { label: "MinIO SecretKey Yunshu（cicd_minio_secret_key）", value: "cicd_minio_secret_key", category: "cicd" },
  { label: "发布模式（cicd_publish_mode）", value: "cicd_publish_mode", category: "cicd" },
  { label: "CI/CD 环境（cicd_tenv）", value: "cicd_tenv", category: "cicd" },
  { label: "应用类型（cicd_pipeline_type）", value: "cicd_pipeline_type", category: "cicd" },
  { label: "前端发布操作（cicd_release_op_frontend）", value: "cicd_release_op_frontend", category: "cicd" },
  { label: "后端发布操作（cicd_release_op_backend）", value: "cicd_release_op_backend", category: "cicd" },
  { label: "前端构建类型（cicd_build_type_frontend）", value: "cicd_build_type_frontend", category: "cicd" },
  { label: "后端构建类型（cicd_build_type_backend）", value: "cicd_build_type_backend", category: "cicd" },
  { label: "npm 安装模式（cicd_npm_install_mode）", value: "cicd_npm_install_mode", category: "cicd" },
  { label: "发布方式（cicd_deploy_kind）", value: "cicd_deploy_kind", category: "cicd" },
  { label: "发布策略（cicd_deploy_strategy）", value: "cicd_deploy_strategy", category: "cicd" },
  { label: "部署动作（cicd_deploy_action）", value: "cicd_deploy_action", category: "cicd" },
  { label: "启动脚本类型（cicd_start_script_type）", value: "cicd_start_script_type", category: "cicd" },
  { label: "K8s 凭据（cicd_k8s_credential）", value: "cicd_k8s_credential", category: "cicd" },
  { label: "重要级别（cicd_importance_level）", value: "cicd_importance_level", category: "cicd" },

  { label: "AI 启用（ai_enabled）", value: "ai_enabled", category: "ai" },
  { label: "AI 默认 Provider（ai_default_provider）", value: "ai_default_provider", category: "ai" },
  { label: "AI 超时秒（ai_timeout_sec）", value: "ai_timeout_sec", category: "ai" },
  { label: "AI Max Tokens（ai_max_tokens）", value: "ai_max_tokens", category: "ai" },
  { label: "OpenAI Base URL（ai_openai_base_url）", value: "ai_openai_base_url", category: "ai" },
  { label: "OpenAI API Key（ai_openai_api_key）", value: "ai_openai_api_key", category: "ai" },
  { label: "OpenAI Model（ai_openai_model）", value: "ai_openai_model", category: "ai" },
  { label: "DeepSeek Base URL（ai_deepseek_base_url）", value: "ai_deepseek_base_url", category: "ai" },
  { label: "DeepSeek API Key（ai_deepseek_api_key）", value: "ai_deepseek_api_key", category: "ai" },
  { label: "DeepSeek Model（ai_deepseek_model）", value: "ai_deepseek_model", category: "ai" },
  { label: "Anthropic Base URL（ai_anthropic_base_url）", value: "ai_anthropic_base_url", category: "ai" },
  { label: "Anthropic API Key（ai_anthropic_api_key）", value: "ai_anthropic_api_key", category: "ai" },
  { label: "Anthropic Model（ai_anthropic_model）", value: "ai_anthropic_model", category: "ai" },
];

export const PROJECT_DICT_TYPE_OPTIONS: DictTypeOption[] = DICT_TYPE_DEFS.map(({ label, value }) => ({ label, value }));

const DICT_TYPE_CATEGORY_MAP = new Map<string, Exclude<DictCategoryId, "all">>(
  DICT_TYPE_DEFS.map((item) => [item.value, item.category]),
);

/** 按命名规则推断分类；未命中注册表时用前缀规则兜底。 */
export function resolveDictCategory(dictType: string): Exclude<DictCategoryId, "all"> {
  const key = String(dictType || "").trim().toLowerCase();
  if (!key) return "other";
  const registered = DICT_TYPE_CATEGORY_MAP.get(key);
  if (registered) return registered;

  // 前缀规则（与插件 / 业务域对齐；顺序：更具体前缀优先）
  if (key.startsWith("cicd_")) return "cicd";
  if (key.startsWith("ai_")) return "ai";
  if (key.startsWith("dbmgmt_")) return "dbmgmt";
  if (key.startsWith("cmdb_") || key.startsWith("server_") || key.startsWith("cloud_")) return "cmdb";
  if (key.startsWith("alert_") || key.startsWith("wecom_") || key.startsWith("dingtalk_")) return "alert";
  if (
    key.startsWith("log_") ||
    key.startsWith("loggie_") ||
    key.startsWith("elasticsearch_") ||
    key.startsWith("kafka_")
  ) {
    return "log";
  }
  if (key.startsWith("k8s_")) return "k8s";
  if (
    key.startsWith("minio_") ||
    key.startsWith("mysql_backup_") ||
    key.startsWith("esmgmt_")
  ) {
    return "backup";
  }
  if (
    key.startsWith("mail_") ||
    key.startsWith("password_") ||
    key.startsWith("auth_") ||
    key === "common_status"
  ) {
    return "system";
  }
  return "other";
}

export function getDictCategoryLabel(category: Exclude<DictCategoryId, "all">): string {
  return DICT_CATEGORY_META[category]?.label ?? category;
}

export function matchesDictCategory(dictType: string, category: DictCategoryId): boolean {
  if (category === "all") return true;
  return resolveDictCategory(dictType) === category;
}

type SelectGroupOption = { label: string; options: DictTypeOption[] };

/** 构建分组下拉；可按当前 Tab 分类收窄选项。 */
export function buildGroupedDictTypeSelectOptions(
  categoryFilter: DictCategoryId = "all",
  extraTypes: string[] = [],
): SelectGroupOption[] {
  const known = new Map<string, DictTypeOption>();
  for (const item of DICT_TYPE_DEFS) {
    if (categoryFilter !== "all" && item.category !== categoryFilter) continue;
    known.set(item.value, { label: item.label, value: item.value });
  }
  for (const value of extraTypes) {
    const v = String(value || "").trim();
    if (!v || known.has(v)) continue;
    if (categoryFilter !== "all" && resolveDictCategory(v) !== categoryFilter) continue;
    known.set(v, { label: `${v}（现有）`, value: v });
  }

  const grouped = new Map<Exclude<DictCategoryId, "all">, DictTypeOption[]>();
  for (const opt of known.values()) {
    const cat = resolveDictCategory(opt.value);
    const list = grouped.get(cat) ?? [];
    list.push(opt);
    grouped.set(cat, list);
  }

  const order: Exclude<DictCategoryId, "all">[] = [
    "system",
    "alert",
    "log",
    "k8s",
    "cmdb",
    "backup",
    "dbmgmt",
    "cicd",
    "ai",
    "other",
  ];
  return order
    .filter((id) => grouped.has(id))
    .map((id) => ({
      label: DICT_CATEGORY_META[id].label,
      options: (grouped.get(id) ?? []).sort((a, b) => a.value.localeCompare(b.value)),
    }));
}

export function flattenGroupedDictTypeOptions(groups: SelectGroupOption[]): DictTypeOption[] {
  return groups.flatMap((group) => group.options);
}
