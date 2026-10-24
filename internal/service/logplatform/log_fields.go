package logplatform

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"yunshu/internal/pkg/constants"
)

// LogFieldStat 日志字段观测项（观测云风格字段发现）。
type LogFieldStat struct {
	Name         string   `json:"name"`
	Count        int      `json:"count"`
	SampleValues []string `json:"sample_values,omitempty"`
	Kind         string   `json:"kind,omitempty"` // keyword|text|number|other
}

// LogFieldsResult 字段发现结果。
type LogFieldsResult struct {
	TotalSamples int            `json:"total_samples"`
	Fields       []LogFieldStat `json:"fields"`
}

var preferredFieldOrder = []string{
	"level", "status", "service_name", "host", "server_host", "namespace", "pod", "podname",
	"container", "containername", "trace_id", "span_id", "file_path", "collector_mode",
	"message", "route", "path", "method",
}

// DiscoverFields 从当前筛选条件下采样文档，提取可观察字段及样例值。
func (s *LogSearchService) DiscoverFields(ctx context.Context, q LogSearchQuery) (*LogFieldsResult, error) {
	prep, err := s.prepareSearch(ctx, q)
	if err != nil {
		return nil, err
	}
	cli, cfg, err := s.es.Client(ctx)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	size := 80
	if q.PageSize > 0 && q.PageSize < size {
		size = q.PageSize
	}
	body := map[string]any{
		"size":    size,
		"sort":    searchSort(cfg.TimestampField),
		"query":   prep.boolQuery(),
		"_source": true,
	}
	raw, err := cli.Search(ctx, prep.indices, body)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("字段发现失败: %v", err))
	}
	hitsRoot, _ := raw["hits"].(map[string]any)
	hits, _ := hitsRoot["hits"].([]any)

	type acc struct {
		count   int
		samples map[string]struct{}
	}
	stats := map[string]*acc{}
	for _, h := range hits {
		hm, ok := h.(map[string]any)
		if !ok {
			continue
		}
		src, _ := hm["_source"].(map[string]any)
		if src == nil {
			continue
		}
		flat := flattenLogSource(src, "", 0)
		for k, v := range flat {
			a := stats[k]
			if a == nil {
				a = &acc{samples: map[string]struct{}{}}
				stats[k] = a
			}
			a.count++
			if len(a.samples) < 5 && v != "" {
				a.samples[v] = struct{}{}
			}
		}
	}

	out := &LogFieldsResult{TotalSamples: len(hits), Fields: make([]LogFieldStat, 0, len(stats))}
	for name, a := range stats {
		samples := make([]string, 0, len(a.samples))
		for sv := range a.samples {
			samples = append(samples, sv)
		}
		sort.Strings(samples)
		out.Fields = append(out.Fields, LogFieldStat{
			Name:         name,
			Count:        a.count,
			SampleValues: samples,
			Kind:         guessFieldKind(samples),
		})
	}
	sort.Slice(out.Fields, func(i, j int) bool {
		pi, pj := preferredFieldRank(out.Fields[i].Name), preferredFieldRank(out.Fields[j].Name)
		if pi != pj {
			return pi < pj
		}
		if out.Fields[i].Count != out.Fields[j].Count {
			return out.Fields[i].Count > out.Fields[j].Count
		}
		return out.Fields[i].Name < out.Fields[j].Name
	})
	return out, nil
}

func preferredFieldRank(name string) int {
	for i, n := range preferredFieldOrder {
		if name == n || strings.HasSuffix(name, "."+n) {
			return i
		}
	}
	return len(preferredFieldOrder) + 1
}

func guessFieldKind(samples []string) string {
	if len(samples) == 0 {
		return "other"
	}
	numish := 0
	for _, s := range samples {
		if _, err := parseLooseNumber(s); err == nil {
			numish++
		}
	}
	if numish == len(samples) {
		return "number"
	}
	avg := 0
	for _, s := range samples {
		avg += len(s)
	}
	avg /= len(samples)
	if avg > 80 {
		return "text"
	}
	return "keyword"
}

func parseLooseNumber(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%f", &f)
	return f, err
}

// flattenLogSource 扁平化 _source，便于字段观测与堆叠列表展示。
func flattenLogSource(src map[string]any, prefix string, depth int) map[string]string {
	out := map[string]string{}
	if src == nil || depth > 3 {
		return out
	}
	for k, v := range src {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if prefix == "" && (k == "@timestamp" || k == "timestamp") {
			continue
		}
		switch t := v.(type) {
		case map[string]any:
			for ck, cv := range flattenLogSource(t, key, depth+1) {
				out[ck] = cv
			}
		case []any:
			if len(t) == 0 {
				continue
			}
			if s := anyToLogString(t[0]); s != "" {
				out[key] = truncateFieldValue(s)
			}
		default:
			if s := anyToLogString(v); s != "" {
				out[key] = truncateFieldValue(s)
			}
		}
	}
	return out
}

func truncateFieldValue(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 240 {
		return s[:240] + "…"
	}
	return s
}
