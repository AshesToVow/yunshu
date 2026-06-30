package bootstrap

import (
	"context"
	"strings"

	"yunshu/internal/dictconfig"
	"yunshu/internal/pkg/logutil"
)

type dictConfigOverrides struct {
	// Alert
	AlertWebhookTokenType    string
	AlertPrometheusURLType   string
	AlertPrometheusTokenType string

	// K8s Event Forward
	K8sEventForwardEnabledType               string
	K8sEventForwardWatcherBufferSizeType     string
	K8sEventForwardWorkerIntervalSecondsType string
	K8sEventForwardWorkerBatchSizeType       string
	K8sEventForwardWorkerMaxRetriesType      string

	// Mail
	MailHostType      string
	MailPortType      string
	MailUsernameType  string
	MailPasswordType  string
	MailFromEmailType string
	MailFromNameType  string
	MailUseTLSType    string
}

func defaultDictConfigOverrides() dictConfigOverrides {
	return dictConfigOverrides{
		AlertWebhookTokenType:    "alert_webhook_token",
		AlertPrometheusURLType:   "alert_enrich_prometheus_url",
		AlertPrometheusTokenType: "alert_enrich_prometheus_token",

		K8sEventForwardEnabledType:               "k8s_event_forward_enabled",
		K8sEventForwardWatcherBufferSizeType:     "k8s_event_forward_watcher_buffer_size",
		K8sEventForwardWorkerIntervalSecondsType: "k8s_event_forward_worker_interval_seconds",
		K8sEventForwardWorkerBatchSizeType:       "k8s_event_forward_worker_batch_size",
		K8sEventForwardWorkerMaxRetriesType:      "k8s_event_forward_worker_max_retries",

		MailHostType:      "mail_host",
		MailPortType:      "mail_port",
		MailUsernameType:  "mail_username",
		MailPasswordType:  "mail_password",
		MailFromEmailType: "mail_from_email",
		MailFromNameType:  "mail_from_name",
		MailUseTLSType:    "mail_use_tls",
	}
}

// applyDictConfigOverrides best-effort 覆盖 app.Config 中的 alert/mail 配置项。
// 约束：MySQL 连接已建立（可读 dict_entries 表）。
func (b *Builder) applyDictConfigOverrides(ctx context.Context, ov dictConfigOverrides) {
	if b == nil || b.app == nil || b.app.Config == nil || b.app.DB == nil {
		return
	}

	logf := func(msg string, kv ...any) {
		logutil.Worker("config").Infow(msg, kv...)
	}

	// Alert: webhook_token
	if v, ok := dictconfig.FetchEnabledDictValue(ctx, b.app.DB, ov.AlertWebhookTokenType); ok {
		b.app.Config.Alert.WebhookToken = v
		logf("config override from dict", "key", "alert.webhook_token", "dict_type", ov.AlertWebhookTokenType)
	}
	// Alert: prometheus_url
	if v, ok := dictconfig.FetchEnabledDictValue(ctx, b.app.DB, ov.AlertPrometheusURLType); ok {
		b.app.Config.Alert.PrometheusURL = v
		logf("config override from dict", "key", "alert.prometheus_url", "dict_type", ov.AlertPrometheusURLType)
	}
	// Alert: prometheus_token (sensitive) - allow empty string override
	if v, ok := dictconfig.FetchEnabledDictValue(ctx, b.app.DB, ov.AlertPrometheusTokenType); ok {
		b.app.Config.Alert.PrometheusToken = v
		logf("config override from dict", "key", "alert.prometheus_token", "dict_type", ov.AlertPrometheusTokenType, "sensitive", true)
	}

	// K8s Event Forward: 字典优先，YAML 兜底
	b.app.Config.K8sEventForward = dictconfig.ResolveK8sEventForwardConfig(
		ctx, b.app.DB, b.yamlK8sEventForwardBase, dictconfig.DefaultK8sEventForwardDictTypes(),
	)
	logf("k8s event forward config resolved (dict overrides yaml)",
		"enabled", b.app.Config.K8sEventForward.Enabled,
		"worker_interval_seconds", b.app.Config.K8sEventForward.WorkerIntervalSeconds,
	)

	// Mail: 字典优先，YAML 兜底（与发信时 DynamicSender 解析规则一致）
	types := dictconfig.MailDictTypes{
		Host:      ov.MailHostType,
		Port:      ov.MailPortType,
		Username:  ov.MailUsernameType,
		Password:  ov.MailPasswordType,
		FromEmail: ov.MailFromEmailType,
		FromName:  ov.MailFromNameType,
		UseTLS:    ov.MailUseTLSType,
	}
	b.app.Config.Mail = dictconfig.ResolveMailConfig(ctx, b.app.DB, b.yamlMailBase, types)
	if strings.TrimSpace(b.app.Config.Mail.Host) != "" {
		logf("mail config resolved (dict overrides yaml)",
			"host", b.app.Config.Mail.Host,
			"port", b.app.Config.Mail.Port,
			"from", b.app.Config.Mail.FromEmail,
		)
	}
}
