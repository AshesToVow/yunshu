package config

import "strings"

// ElasticsearchConfig 日志检索后端（Loggie → ES → Yunshu 代理查询）。
type ElasticsearchConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	Addresses      []string `mapstructure:"addresses"`
	Username       string   `mapstructure:"username"`
	Password       string   `mapstructure:"password"`
	IndexPattern   string   `mapstructure:"index_pattern"`
	MessageFields  []string `mapstructure:"message_fields"`
	TimestampField string   `mapstructure:"timestamp_field"`
	DefaultSize           int      `mapstructure:"default_size"`
	MaxSize               int      `mapstructure:"max_size"`
	TimeoutSeconds        int      `mapstructure:"timeout_seconds"`
	DefaultRetentionDays  int      `mapstructure:"default_retention_days"`
	CleanupCronSpec       string   `mapstructure:"cleanup_cron_spec"`
}

func (c ElasticsearchConfig) Normalized() ElasticsearchConfig {
	out := c
	if out.IndexPattern == "" {
		out.IndexPattern = "yunshu-agent-*"
	}
	if len(out.MessageFields) == 0 {
		out.MessageFields = []string{"message", "body", "log", "content"}
	}
	if out.TimestampField == "" {
		out.TimestampField = "@timestamp"
	}
	if out.DefaultSize <= 0 {
		out.DefaultSize = 100
	}
	if out.MaxSize <= 0 {
		out.MaxSize = 1000
	}
	if out.TimeoutSeconds <= 0 {
		out.TimeoutSeconds = 30
	}
	if out.DefaultRetentionDays <= 0 {
		out.DefaultRetentionDays = 30
	}
	if strings.TrimSpace(out.CleanupCronSpec) == "" {
		out.CleanupCronSpec = "0 3 * * *"
	}
	return out
}
