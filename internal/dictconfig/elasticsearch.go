package dictconfig

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"yunshu/internal/config"

	"gorm.io/gorm"
)

// ElasticsearchDictTypes 数据字典覆盖 elasticsearch.* 的 dict_type。
type ElasticsearchDictTypes struct {
	Enabled              string
	Addresses            string
	Username             string
	Password             string
	IndexPattern         string
	K8sIndexPrefix       string
	DefaultRetentionDays string
	CleanupCronSpec      string
	// ConnectionID 指向 esmgmt_connections.id；>0 时日志平台使用该连接的地址/认证，字典地址仅作兜底。
	ConnectionID string
}

func DefaultElasticsearchDictTypes() ElasticsearchDictTypes {
	return ElasticsearchDictTypes{
		Enabled:              "elasticsearch_enabled",
		Addresses:            "elasticsearch_addresses",
		Username:             "elasticsearch_username",
		Password:             "elasticsearch_password",
		IndexPattern:         "elasticsearch_index_pattern",
		K8sIndexPrefix:       "elasticsearch_k8s_index_prefix",
		DefaultRetentionDays: "elasticsearch_default_retention_days",
		CleanupCronSpec:      "elasticsearch_cleanup_cron_spec",
		ConnectionID:         "elasticsearch_connection_id",
	}
}

// ResolveElasticsearchConfig 字典优先合并 elasticsearch 配置。
// addresses 支持 JSON 数组或逗号分隔（单节点/集群均可）。
func ResolveElasticsearchConfig(ctx context.Context, db *gorm.DB, base config.ElasticsearchConfig) config.ElasticsearchConfig {
	if db == nil {
		return base.Normalized()
	}
	types := DefaultElasticsearchDictTypes()
	out := base
	if v, ok := FetchEnabledDictValue(ctx, db, types.Enabled); ok {
		out.Enabled = strings.EqualFold(strings.TrimSpace(v), "true") || v == "1"
	}
	if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, types.Addresses); ok {
		if addrs := parseESAddresses(v); len(addrs) > 0 {
			out.Addresses = addrs
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.Username); ok {
		out.Username = strings.TrimSpace(v)
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.Password); ok {
		out.Password = v
	}
	if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, types.IndexPattern); ok {
		out.IndexPattern = strings.TrimSpace(v)
	}
	if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, types.K8sIndexPrefix); ok {
		out.K8sIndexPrefix = strings.TrimSpace(v)
	}
	if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, types.DefaultRetentionDays); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			out.DefaultRetentionDays = n
		}
	}
	if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, types.CleanupCronSpec); ok {
		out.CleanupCronSpec = strings.TrimSpace(v)
	}
	return out.Normalized()
}

func parseESAddresses(raw string) []string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "[") {
		var arr []string
		if err := json.Unmarshal([]byte(s), &arr); err == nil {
			return trimNonEmpty(arr)
		}
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	return trimNonEmpty(parts)
}

func trimNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, a := range in {
		a = strings.TrimSpace(a)
		if a != "" {
			out = append(out, strings.TrimRight(a, "/"))
		}
	}
	return out
}
