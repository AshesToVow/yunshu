package system

// notifyDictSeeds 通知域内置字典种子：企业微信、钉钉、邮件。
func notifyDictSeeds() []DictEntryCreateRequest {
	return []DictEntryCreateRequest{
		{DictType: "wecom_webhook_url", Label: "企业微信机器人 URL 示例", Value: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=replace-me", Sort: intRef(1), Status: 0, Remark: "企业微信 Webhook URL"},
		{DictType: "dingtalk_webhook_url", Label: "钉钉机器人 URL 示例", Value: "https://oapi.dingtalk.com/robot/send?access_token=replace-me", Sort: intRef(1), Status: 0, Remark: "钉钉 Webhook URL"},
		{DictType: "wecom_notify_mode", Label: "群机器人(robot)", Value: "robot", Sort: intRef(1), Status: 1, Remark: "企业微信通知模式"},
		{DictType: "wecom_notify_mode", Label: "企业应用(app)", Value: "app", Sort: intRef(2), Status: 1, Remark: "企业微信通知模式"},
		{DictType: "dingtalk_notify_mode", Label: "群机器人(robot)", Value: "robot", Sort: intRef(1), Status: 1, Remark: "钉钉通知模式"},
		{DictType: "dingtalk_notify_mode", Label: "应用会话(app_chat)", Value: "app_chat", Sort: intRef(2), Status: 1, Remark: "钉钉通知模式"},
		// 企业微信 / 钉钉专用字典（用于通道配置自动填充）
		{DictType: "wecom_corp_id", Label: "企业微信 CorpID 示例", Value: "wwxxxxxxxxxxxxxxxx", Sort: intRef(1), Status: 0, Remark: "企业微信应用模式 corpID"},
		{DictType: "wecom_corp_secret", Label: "企业微信 CorpSecret 示例", Value: "", Sort: intRef(1), Status: 0, Remark: "企业微信应用模式 corpSecret，敏感信息建议仅在生产维护"},
		{DictType: "wecom_agent_id", Label: "企业微信 AgentID 示例", Value: "1000002", Sort: intRef(1), Status: 0, Remark: "企业微信应用模式 agentId"},
		{DictType: "dingtalk_app_key", Label: "钉钉 AppKey 示例", Value: "dingxxxxxxxx", Sort: intRef(1), Status: 0, Remark: "钉钉 app_chat 模式 appKey"},
		{DictType: "dingtalk_app_secret", Label: "钉钉 AppSecret 示例", Value: "", Sort: intRef(1), Status: 0, Remark: "钉钉 app_chat 模式 appSecret，敏感信息建议仅在生产维护"},
		{DictType: "dingtalk_chat_id", Label: "钉钉 ChatID 示例", Value: "chatxxxxxxxx", Sort: intRef(1), Status: 0, Remark: "钉钉 app_chat 模式 chatId"},
		// 须为启用：告警渠道「从字典填充 signSecret」走 /dict/options，仅返回 status=1 的条目
		{DictType: "dingtalk_sign_secret", Label: "钉钉 SignSecret 示例", Value: "", Sort: intRef(1), Status: 1, Remark: "钉钉 robot 模式加签 SEC（在字典中填写真实值；与 app_chat 的 dingtalk_app_secret 不同）"},
		// Mail（作为字典权威来源，覆盖 config.yaml）
		{DictType: "mail_host", Label: "163 SMTP", Value: "smtp.163.com", Sort: intRef(1), Status: 1, Remark: "mail.host：字典存在则覆盖 config.yaml"},
		{DictType: "mail_port", Label: "SMTP 端口(SSL)", Value: "465", Sort: intRef(1), Status: 1, Remark: "mail.port：字典存在则覆盖 config.yaml"},
		{DictType: "mail_use_tls", Label: "启用 TLS", Value: "true", Sort: intRef(1), Status: 1, Remark: "mail.use_tls：true/false"},
	}
}
