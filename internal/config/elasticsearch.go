package config

import "strings"

// ElasticsearchConfig 日志检索后端（Loggie → ES → Yunshu 代理查询）。
type ElasticsearchConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	Addresses      []string `mapstructure:"addresses"`
	Username       string   `mapstructure:"username"`
	Password       string   `mapstructure:"password"`
	IndexPattern   string   `mapstructure:"index_pattern"`
	// K8sIndexPrefix 集群采集索引名前缀，默认 yunshu-k8s → yunshu-k8s-{clusterId}-p{projectId}-日期
	K8sIndexPrefix string   `mapstructure:"k8s_index_prefix"`
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
	out.IndexPattern = strings.TrimSpace(out.IndexPattern)
	// 空或历史共用索引模式迁移为现行 Agent 分索引通配
	if out.IndexPattern == "" || out.IndexPattern == "yunshu-logs-*" || out.IndexPattern == "yunshu-logs" {
		out.IndexPattern = "yunshu-agent-*"
	}
	if out.IndexPattern == "*" {
		out.IndexPattern = "yunshu-agent-*"
	}
	out.K8sIndexPrefix = strings.Trim(strings.TrimSpace(out.K8sIndexPrefix), "-")
	if out.K8sIndexPrefix == "" {
		out.K8sIndexPrefix = "yunshu-k8s"
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
		// 日志检索可能扫多日索引；过短会导致前端误报超时
		out.TimeoutSeconds = 90
	}
	if out.DefaultRetentionDays <= 0 {
		out.DefaultRetentionDays = 30
	}
	if strings.TrimSpace(out.CleanupCronSpec) == "" {
		out.CleanupCronSpec = "0 3 * * *"
	}
	return out
}
