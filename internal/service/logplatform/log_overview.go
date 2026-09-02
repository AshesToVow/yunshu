package logplatform

import (
	"context"
	"fmt"
	"strings"
	"time"

	bizerrors "yunshu/internal/pkg/errors"
)

// LogHistogramBucket 时间直方图桶。
type LogHistogramBucket struct {
	Time  string `json:"time"`
	Count int64  `json:"count"`
}

// LogOverviewResult 日志概览（P1：直方图 + 级别 + 签名）。
type LogOverviewResult struct {
	Total              int64              `json:"total"`
	Histogram          []LogHistogramBucket `json:"histogram"`
	LevelCounts        map[string]int64   `json:"level_counts"`
	TopErrorSignatures []LogSignatureItem `json:"top_error_signatures"`
	Summary            *LogSummaryResult  `json:"summary,omitempty"`
}

// Overview 聚合检索概览：ES date_histogram + terms(level) + 采样签名。
func (s *LogSearchService) Overview(ctx context.Context, q LogSearchQuery) (*LogOverviewResult, error) {
	prep, err := s.prepareSearch(ctx, q)
	if err != nil {
		return nil, err
	}
	cli, cfg, err := s.es.Client(ctx)
	if err != nil {
		return nil, err
	}
	tsField := strings.TrimSpace(cfg.TimestampField)
	if tsField == "" {
		tsField = "@timestamp"
	}

	interval := pickHistogramInterval(q.From, q.To)
	body := map[string]any{
		"size":             0,
		"track_total_hits": true,
		"query":            prep.boolQuery(),
		"aggs": map[string]any{
			"log_histogram": map[string]any{
				"date_histogram": map[string]any{
					"field":          tsField,
					"fixed_interval": interval,
					"min_doc_count":  0,
					"extended_bounds": extendedBounds(q.From, q.To),
				},
			},
			"level_terms": map[string]any{
				"terms": map[string]any{
					"field": "level",
					"size":  20,
					"missing": "(unknown)",
				},
			},
			"level_kw_terms": map[string]any{
				"terms": map[string]any{
					"field": "level.keyword",
					"size":  20,
				},
			},
			"fields_level_terms": map[string]any{
				"terms": map[string]any{
					"field": "fields.level",
					"size":  20,
				},
			},
		},
	}
	raw, err := cli.Search(ctx, prep.indices, body)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "logsearch", "Overview", err)
	}

	out := &LogOverviewResult{
		LevelCounts: map[string]int64{},
		Histogram:   []LogHistogramBucket{},
	}
	out.Total = parseTotalHits(raw)
	out.Histogram = parseDateHistogram(raw, "log_histogram")
	out.LevelCounts = mergeTermsAggs(raw, "level_kw_terms", "level_terms", "fields_level_terms")

	// 采样 ERROR/WARN 日志提取签名（复用 SummarizeLogHits）
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

func extendedBounds(from, to string) map[string]any {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" && to == "" {
		return nil
	}
	b := map[string]any{}
	if from != "" {
		b["min"] = from
	}
	if to != "" {
		b["max"] = to
	}
	if len(b) == 0 {
		return nil
	}
	return b
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
		key, _ := bm["key"].(string)
		if key == "" {
			if kv, ok := bm["key"].(float64); ok {
				key = fmt.Sprintf("%v", int64(kv))
			}
		}
		cnt := int64(0)
		if v, ok := bm["doc_count"].(float64); ok {
			cnt = int64(v)
		}
		out[strings.ToUpper(key)] = cnt
	}
	return out
}
