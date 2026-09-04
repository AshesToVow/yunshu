package logplatform

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/pkg/constants"
)

// LogHistogramBucket 时间直方图桶。
type LogHistogramBucket struct {
	Time  string `json:"time"`
	Count int64  `json:"count"`
}

// LogOverviewResult 日志概览（P1：直方图 + 级别 + 签名）。
type LogOverviewResult struct {
	Total              int64                `json:"total"`
	Histogram          []LogHistogramBucket `json:"histogram"`
	LevelCounts        map[string]int64     `json:"level_counts"`
	ServiceNameCounts  map[string]int64     `json:"service_name_counts,omitempty"`
	HostCounts         map[string]int64     `json:"host_counts,omitempty"`
	TopErrorSignatures []LogSignatureItem   `json:"top_error_signatures"`
	Summary            *LogSummaryResult    `json:"summary,omitempty"`
}

// Overview 聚合检索概览：ES date_histogram + terms(level) + 采样签名。
//
// 注意：旧索引动态映射下 level 可能是 keyword 或 text+keyword。
// 同一请求里同时聚合 level 与 level.keyword 会在 ES 7 上直接 400，
// 进而被包装成 error_code=50001「operation failed」，而原始 Search 不受影响。
func (s *LogSearchService) Overview(ctx context.Context, q LogSearchQuery) (*LogOverviewResult, error) {
	prep, err := s.prepareSearch(ctx, q)
	if err != nil {
		return nil, err
	}
	cli, cfg, err := s.es.Client(ctx)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	tsField := strings.TrimSpace(cfg.TimestampField)
	if tsField == "" {
		tsField = "@timestamp"
	}

	interval := pickHistogramInterval(q.From, q.To)
	query := prep.boolQuery()

	// 按兼容性从强到弱尝试，避免单个字段映射不兼容拖垮整个概览。
	attempts := []map[string]any{
		overviewAggBody(query, tsField, interval, q, []string{"level", "fields.level"}),
		overviewAggBody(query, tsField, interval, q, []string{"level.keyword", "fields.level.keyword"}),
		overviewAggBody(query, tsField, interval, q, []string{"level"}),
		overviewAggBody(query, tsField, interval, q, nil), // 仅直方图 + total
	}

	var raw map[string]any
	var lastErr error
	for _, body := range attempts {
		raw, lastErr = cli.Search(ctx, prep.indices, body)
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("日志概览聚合失败: %v", lastErr))
	}

	out := &LogOverviewResult{
		LevelCounts:       map[string]int64{},
		ServiceNameCounts: map[string]int64{},
		HostCounts:        map[string]int64{},
		Histogram:         []LogHistogramBucket{},
	}
	out.Total = parseTotalHits(raw)
	out.Histogram = parseDateHistogram(raw, "log_histogram")
	out.LevelCounts = mergeTermsAggs(raw, "level_terms_0", "level_terms_1", "level_terms_2", "level_terms_3")
	out.ServiceNameCounts = s.queryTermsFacet(ctx, prep, []string{
		"service_name.keyword", "service_name", "fields.service_name.keyword", "fields.service_name",
	}, 15)
	out.HostCounts = s.queryTermsFacet(ctx, prep, []string{
		"host.keyword", "host", "server_host.keyword", "server_host", "hostname.keyword", "hostname",
	}, 15)

	// 采样 ERROR/WARN 日志提取签名（复用 SummarizeLogHits）；采样失败不影响概览主体。
	sampleQ := q
	sampleQ.Page = 1
	sampleQ.PageSize = 500
	if sampleQ.Level == "" {
		sampleQ.Level = "ERROR"
	}
	sampleRes, err := s.Search(ctx, sampleQ)
	if err == nil && sampleRes != nil && sampleRes.Total == 0 && q.Level == "" {
		sampleQ.Level = "WARN"
		sampleRes, _ = s.Search(ctx, sampleQ)
	}
	if sampleRes != nil {
		summary := SummarizeLogHits(sampleRes, 8)
		out.TopErrorSignatures = summary.TopErrorSignatures
		out.Summary = summary
		if out.Total == 0 {
			out.Total = summary.Total
		}
		if len(out.LevelCounts) == 0 && len(summary.LevelCounts) > 0 {
			out.LevelCounts = map[string]int64{}
			for k, v := range summary.LevelCounts {
				out.LevelCounts[strings.ToUpper(k)] = int64(v)
			}
		}
	}
	return out, nil
}

func overviewAggBody(query any, tsField, interval string, q LogSearchQuery, levelFields []string) map[string]any {
	dh := map[string]any{
		"field":          tsField,
		"fixed_interval": interval,
		"min_doc_count":  0,
	}
	// 未选时间范围时不要写 extended_bounds:null，ES 7 会 x_content_parse_exception。
	if bounds := extendedBounds(q.From, q.To); bounds != nil {
		dh["extended_bounds"] = bounds
	}
	aggs := map[string]any{
		"log_histogram": map[string]any{
			"date_histogram": dh,
		},
	}
	for i, field := range levelFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		name := fmt.Sprintf("level_terms_%d", i)
		aggs[name] = map[string]any{
			"terms": map[string]any{
				"field": field,
				"size":  20,
			},
		}
	}
	return map[string]any{
		"size":             0,
		"track_total_hits": true,
		"query":            query,
		"aggs":             aggs,
	}
}

func pickHistogramInterval(from, to string) string {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" {
		return "1h"
	}
	tf, err1 := time.Parse(time.RFC3339, from)
	tt, err2 := time.Parse(time.RFC3339, to)
	if err1 != nil || err2 != nil {
		return "1h"
	}
	d := tt.Sub(tf)
	switch {
	case d <= 30*time.Minute:
		return "1m"
	case d <= 2*time.Hour:
		return "5m"
	case d <= 24*time.Hour:
		return "30m"
	case d <= 7*24*time.Hour:
		return "2h"
	default:
		return "1d"
	}
}

// extendedBounds 仅在 from/to 都有有效值时返回；任一为空则返回 nil（调用方勿写入 JSON）。
func extendedBounds(from, to string) map[string]any {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" {
		return nil
	}
	return map[string]any{"min": from, "max": to}
}

func parseTotalHits(raw map[string]any) int64 {
	hitsRoot, _ := raw["hits"].(map[string]any)
	if tv, ok := hitsRoot["total"].(map[string]any); ok {
		if v, ok := tv["value"].(float64); ok {
			return int64(v)
		}
	}
	if v, ok := hitsRoot["total"].(float64); ok {
		return int64(v)
	}
	return 0
}

func parseDateHistogram(raw map[string]any, aggName string) []LogHistogramBucket {
	aggs, _ := raw["aggregations"].(map[string]any)
	root, _ := aggs[aggName].(map[string]any)
	buckets, _ := root["buckets"].([]any)
	out := make([]LogHistogramBucket, 0, len(buckets))
	for _, b := range buckets {
		bm, ok := b.(map[string]any)
		if !ok {
			continue
		}
		key, _ := bm["key_as_string"].(string)
		if key == "" {
			if kv, ok := bm["key"].(float64); ok {
				key = time.UnixMilli(int64(kv)).UTC().Format(time.RFC3339)
			}
		}
		cnt := int64(0)
		if v, ok := bm["doc_count"].(float64); ok {
			cnt = int64(v)
		}
		out = append(out, LogHistogramBucket{Time: key, Count: cnt})
	}
	return out
}

func mergeTermsAggs(raw map[string]any, names ...string) map[string]int64 {
	out := map[string]int64{}
	for _, name := range names {
		for k, v := range parseTermsAgg(raw, name) {
			key := strings.ToUpper(strings.TrimSpace(k))
			if key == "" || key == "(UNKNOWN)" {
				continue
			}
			out[key] += v
		}
	}
	return out
}

func (s *LogSearchService) queryTermsFacet(ctx context.Context, prep *preparedSearch, fieldCandidates []string, size int) map[string]int64 {
	if prep == nil || size <= 0 {
		return map[string]int64{}
	}
	cli, _, err := s.es.Client(ctx)
	if err != nil {
		return map[string]int64{}
	}
	for _, field := range fieldCandidates {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		body := map[string]any{
			"size":  0,
			"query": prep.boolQuery(),
			"aggs": map[string]any{
				"facet": map[string]any{
					"terms": map[string]any{
						"field": field,
						"size":  size,
					},
				},
			},
		}
		raw, err := cli.Search(ctx, prep.indices, body)
		if err != nil {
			continue
		}
		if counts := parseTermsAggPreserveCase(raw, "facet"); len(counts) > 0 {
			return counts
		}
	}
	return map[string]int64{}
}

func parseTermsAggPreserveCase(raw map[string]any, aggName string) map[string]int64 {
	aggs, _ := raw["aggregations"].(map[string]any)
	root, _ := aggs[aggName].(map[string]any)
	buckets, _ := root["buckets"].([]any)
	out := map[string]int64{}
	for _, b := range buckets {
		bm, ok := b.(map[string]any)
		if !ok {
			continue
		}
		key := termsBucketKey(bm)
		if key == "" {
			continue
		}
		cnt := int64(0)
		if v, ok := bm["doc_count"].(float64); ok {
			cnt = int64(v)
		}
		out[key] = cnt
	}
	return out
}

func termsBucketKey(bm map[string]any) string {
	if key, ok := bm["key"].(string); ok {
		return strings.TrimSpace(key)
	}
	if kv, ok := bm["key"].(float64); ok {
		return fmt.Sprintf("%v", int64(kv))
	}
	return ""
}

func parseTermsAgg(raw map[string]any, aggName string) map[string]int64 {
	aggs, _ := raw["aggregations"].(map[string]any)
	root, _ := aggs[aggName].(map[string]any)
	buckets, _ := root["buckets"].([]any)
	out := map[string]int64{}
	for _, b := range buckets {
		bm, ok := b.(map[string]any)
		if !ok {
			continue
		}
		key := termsBucketKey(bm)
		cnt := int64(0)
		if v, ok := bm["doc_count"].(float64); ok {
			cnt = int64(v)
		}
		out[strings.ToUpper(key)] = cnt
	}
	return out
}
