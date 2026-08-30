package system

// aiDictSeeds AI 域内置字典种子（字典优先，config.yaml 兜底）。
func aiDictSeeds() []DictEntryCreateRequest {
	return []DictEntryCreateRequest{
		{DictType: "ai_enabled", Label: "启用 AI", Value: "false", Sort: intRef(1), Status: 1, Remark: "AI 插件总开关"},
		{DictType: "ai_default_provider", Label: "默认 Provider", Value: "openai_compat", Sort: intRef(1), Status: 1, Remark: "openai_compat | deepseek | anthropic"},
		{DictType: "ai_timeout_sec", Label: "调用超时(秒)", Value: "60", Sort: intRef(1), Status: 1, Remark: "LLM HTTP 超时"},
		{DictType: "ai_max_tokens", Label: "Max Tokens", Value: "2048", Sort: intRef(1), Status: 1, Remark: "单次回复 token 上限"},
		{DictType: "ai_openai_base_url", Label: "OpenAI 兼容 Base URL", Value: "https://api.openai.com/v1", Sort: intRef(1), Status: 0, Remark: "也可填 LiteLLM/通义/vLLM 兼容地址"},
		{DictType: "ai_openai_api_key", Label: "OpenAI API Key", Value: "", Sort: intRef(1), Status: 0, Remark: "敏感"},
		{DictType: "ai_openai_model", Label: "OpenAI Model", Value: "gpt-4o-mini", Sort: intRef(1), Status: 0, Remark: "默认模型名"},
		{DictType: "ai_deepseek_base_url", Label: "DeepSeek Base URL", Value: "https://api.deepseek.com/v1", Sort: intRef(1), Status: 0, Remark: "DeepSeek OpenAI 兼容"},
		{DictType: "ai_deepseek_api_key", Label: "DeepSeek API Key", Value: "", Sort: intRef(1), Status: 0, Remark: "敏感"},
		{DictType: "ai_deepseek_model", Label: "DeepSeek Model", Value: "deepseek-chat", Sort: intRef(1), Status: 0, Remark: ""},
		{DictType: "ai_anthropic_base_url", Label: "Anthropic Base URL", Value: "https://api.anthropic.com", Sort: intRef(1), Status: 0, Remark: "Claude Messages API"},
		{DictType: "ai_anthropic_api_key", Label: "Anthropic API Key", Value: "", Sort: intRef(1), Status: 0, Remark: "敏感"},
		{DictType: "ai_anthropic_model", Label: "Anthropic Model", Value: "claude-sonnet-4-20250514", Sort: intRef(1), Status: 0, Remark: ""},
	}
}

// dbmgmtDictSeeds 数据库管理平台域内置字典种子（字典优先，config.yaml 兜底）。
func dbmgmtDictSeeds() []DictEntryCreateRequest {
	return []DictEntryCreateRequest{
		{DictType: "dbmgmt_query_timeout_seconds", Label: "SQL 查询超时(秒)", Value: "30", Sort: intRef(1), Status: 1, Remark: "dbmgmt.query_timeout_seconds"},
		{DictType: "dbmgmt_max_rows", Label: "查询最大返回行数", Value: "1000", Sort: intRef(1), Status: 1, Remark: "dbmgmt.max_result_rows"},
		{DictType: "dbmgmt_max_import_file_mb", Label: "SQL 文件导入上限(MB)", Value: "10", Sort: intRef(1), Status: 1, Remark: "dbmgmt.max_import_file_mb"},
		{DictType: "cmdb_max_transfer_file_mb", Label: "服务器文件传输上限(MB)", Value: "50", Sort: intRef(1), Status: 1, Remark: "服务器操作台 SFTP 上传/下载单文件上限"},
		{DictType: "dbmgmt_prod_force_approval", Label: "生产环境强制审批", Value: "true", Sort: intRef(1), Status: 1, Remark: "dbmgmt.prod_force_approval：true/false"},
		{DictType: "dbmgmt_forbid_self_approve", Label: "禁止自审自批", Value: "true", Sort: intRef(1), Status: 1, Remark: "dbmgmt.forbid_self_approve：true/false"},
		{DictType: "dbmgmt_approval_sla_hours", Label: "审批超时阈值(小时)", Value: "24", Sort: intRef(1), Status: 1, Remark: "dbmgmt.approval_sla_hours"},
		{DictType: "dbmgmt_approval_reminder_interval_hours", Label: "审批提醒间隔(小时)", Value: "4", Sort: intRef(1), Status: 1, Remark: "dbmgmt.approval_reminder_interval_hours"},
		{DictType: "dbmgmt_ping_interval_seconds", Label: "实例探活间隔(秒)", Value: "300", Sort: intRef(1), Status: 1, Remark: "dbmgmt.ping_interval_seconds"},
		{DictType: "dbmgmt_max_concurrent_per_instance", Label: "单实例最大并发", Value: "5", Sort: intRef(1), Status: 1, Remark: "dbmgmt.max_concurrent_per_instance"},
		{DictType: "dbmgmt_goinception_enabled", Label: "启用 goInception 审核", Value: "true", Sort: intRef(1), Status: 1, Remark: "dbmgmt.goinception_enabled：true/false"},
		{DictType: "dbmgmt_goinception_host", Label: "goInception 地址", Value: "10.10.10.103", Sort: intRef(1), Status: 1, Remark: "dbmgmt.goinception_host"},
		{DictType: "dbmgmt_goinception_port", Label: "goInception 端口", Value: "4000", Sort: intRef(1), Status: 1, Remark: "dbmgmt.goinception_port"},
		{DictType: "dbmgmt_goinception_backup", Label: "goInception 执行前备份", Value: "true", Sort: intRef(1), Status: 1, Remark: "dbmgmt.goinception_backup：true/false"},
	}
}

// securityDictSeeds 安全域内置字典种子：密码策略（系统分类；可随时在数据字典调整）。
func securityDictSeeds() []DictEntryCreateRequest {
	return []DictEntryCreateRequest{
		{DictType: "password_min_length", Label: "密码最小长度", Value: "8", Sort: intRef(1), Status: 1, Remark: "password_min_length"},
		{DictType: "password_max_length", Label: "密码最大长度", Value: "64", Sort: intRef(1), Status: 1, Remark: "password_max_length"},
		{DictType: "password_require_upper", Label: "须含大写字母", Value: "true", Sort: intRef(1), Status: 1, Remark: "true/false"},
		{DictType: "password_require_lower", Label: "须含小写字母", Value: "true", Sort: intRef(1), Status: 1, Remark: "true/false"},
		{DictType: "password_require_digit", Label: "须含数字", Value: "true", Sort: intRef(1), Status: 1, Remark: "true/false"},
		{DictType: "password_require_special", Label: "须含特殊字符", Value: "true", Sort: intRef(1), Status: 1, Remark: "true/false"},
		{DictType: "password_expiry_days", Label: "密码过期天数", Value: "90", Sort: intRef(1), Status: 1, Remark: "默认 90 天（约 3 个月）；0=不过期"},
		{DictType: "password_forbid_username", Label: "禁止包含用户名", Value: "true", Sort: intRef(1), Status: 1, Remark: "true/false"},
	}
}
