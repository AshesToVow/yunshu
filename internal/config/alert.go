package config

import "strings"

// DefaultAlertConfig 告警模块缺省值（group_by / digest_by / 节流与平台长度限制）。
func DefaultAlertConfig() AlertConfig {
	return AlertConfig{
		DefaultTimeoutMS:             5000,
		MaxPayloadChars:              8000,
		DedupTTLSeconds:              86400,
		PromQueryTimeout:             5,
		PrometheusEnrichQueueSize:    1024,
		PrometheusEnrichWorkers:      4,
		GroupBy:                      []string{"alertname", "cluster", "namespace", "severity", "receiver"},
		DigestBy:                     []string{"instance", "pod", "node", "host", "mountpoint", "device", "fqdn", "job"},
		GroupWaitSeconds:             0,
		GroupIntervalSeconds:         60,
		RepeatIntervalSeconds:        300,
		AggregateTTLSeconds:          86400,
		WebhookQueueMaxLen:           10000,
		MonitorEvalLeaderLockSeconds: 30,
		MonitorEvalCronSpec:          "*/5 * * * * *",
		PlatformLimits: AlertPlatformLimits{
			DingdingMaxChars: 4500,
			WeComMaxChars:    3500,
			GenericMaxChars:  8000,
		},
	}
}

// ApplyDefaults 填充 Alert 零值字段（不含 webhook_skip_group_timing 的 IsSet 语义）。
func (c *AlertConfig) ApplyDefaults() {
	def := DefaultAlertConfig()
	if c.DefaultTimeoutMS <= 0 {
		c.DefaultTimeoutMS = def.DefaultTimeoutMS
	}
	if c.MaxPayloadChars <= 0 {
		c.MaxPayloadChars = def.MaxPayloadChars
	}
	if c.DedupTTLSeconds <= 0 {
		c.DedupTTLSeconds = def.DedupTTLSeconds
	}
	if c.PromQueryTimeout <= 0 {
		c.PromQueryTimeout = def.PromQueryTimeout
	}
	if c.PrometheusEnrichQueueSize <= 0 {
		c.PrometheusEnrichQueueSize = def.PrometheusEnrichQueueSize
	}
	if c.PrometheusEnrichWorkers <= 0 {
		c.PrometheusEnrichWorkers = def.PrometheusEnrichWorkers
	}
	if c.GroupWaitSeconds < 0 {
		c.GroupWaitSeconds = def.GroupWaitSeconds
	}
	if c.GroupIntervalSeconds <= 0 {
		c.GroupIntervalSeconds = def.GroupIntervalSeconds
	}
	if c.RepeatIntervalSeconds <= 0 {
		c.RepeatIntervalSeconds = def.RepeatIntervalSeconds
	}
	if c.AggregateTTLSeconds <= 0 {
		c.AggregateTTLSeconds = def.AggregateTTLSeconds
	}
	if c.WebhookQueueMaxLen <= 0 {
		c.WebhookQueueMaxLen = def.WebhookQueueMaxLen
	}
	if c.MonitorEvalLeaderLockSeconds <= 0 {
		c.MonitorEvalLeaderLockSeconds = def.MonitorEvalLeaderLockSeconds
	}
	if strings.TrimSpace(c.MonitorEvalCronSpec) == "" {
		c.MonitorEvalCronSpec = def.MonitorEvalCronSpec
	}
	if len(c.GroupBy) == 0 {
		c.GroupBy = append([]string(nil), def.GroupBy...)
	}
	if len(c.DigestBy) == 0 {
		c.DigestBy = append([]string(nil), def.DigestBy...)
	}
	if c.PlatformLimits.DingdingMaxChars <= 0 {
		c.PlatformLimits.DingdingMaxChars = def.PlatformLimits.DingdingMaxChars
	}
	if c.PlatformLimits.WeComMaxChars <= 0 {
		c.PlatformLimits.WeComMaxChars = def.PlatformLimits.WeComMaxChars
	}
	if c.PlatformLimits.GenericMaxChars <= 0 {
		c.PlatformLimits.GenericMaxChars = def.PlatformLimits.GenericMaxChars
	}
}
