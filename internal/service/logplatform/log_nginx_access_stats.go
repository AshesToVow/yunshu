package logplatform

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/pkg/constants"
)

// NginxAccessStatsQuery Nginx 访问统计查询（复用日志检索过滤字段）。
type NginxAccessStatsQuery struct {
	LogSearchQuery
	TopN int `form:"top_n"`
}

// NginxAccessURIStat TOP URI 条目。
type NginxAccessURIStat struct {
	URI     string  `json:"uri"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

// NginxAccessStatsResult Nginx 访问统计结果。
type NginxAccessStatsResult struct {
	From               string                `json:"from"`
	To                 string                `json:"to"`
	PV                 int64                 `json:"pv"`
	UV                 int64                 `json:"uv"`
	AvgQPS             float64               `json:"avg_qps"`
	PeakQPS            float64               `json:"peak_qps"`
	Status2xxRate      float64               `json:"status_2xx_rate"`
	Status3xxRate      float64               `json:"status_3xx_rate"`
	Status4xxRate      float64               `json:"status_4xx_rate"`
	Status5xxRate      float64               `json:"status_5xx_rate"`
	LatencyAvailable   bool                  `json:"latency_available"`
	AvgLatencyMs       *float64              `json:"avg_latency_ms,omitempty"`
	P95LatencyMs       *float64              `json:"p95_latency_ms,omitempty"`
	P99LatencyMs       *float64              `json:"p99_latency_ms,omitempty"`
	Histogram          []LogHistogramBucket  `json:"histogram"`
	TopURIs            []NginxAccessURIStat  `json:"top_uris"`
	LatencyHint        string                `json:"latency_hint,omitempty"`
}

// NginxAccessStats 聚合 Nginx access 日志：PV/UV/QPS/状态码/延迟/TOP URI。
func (s *LogSearchService) NginxAccessStats(ctx context.Context, q NginxAccessStatsQuery) (*NginxAccessStatsResult, error) {
	from := strings.TrimSpace(q.From)
	to := strings.TrimSpace(q.To)
	if from == "" || to == "" {
		return nil, constants.ErrBadRequestWithMsg("from/to 时间范围必填")
	}
	topN := q.TopN
	if topN <= 0 {
		topN = 5
	}
	if topN > 50 {
		topN = 50
	}

	prep, err := s.prepareSearch(ctx, q.LogSearchQuery)
	if err != nil {
		return nil, err
	}
	prep.filters = append(prep.filters, nginxAccessDocFilter())

	cli, cfg, err := s.es.Client(ctx)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	tsField := strings.TrimSpace(cfg.TimestampField)
	if tsField == "" {
		tsField = "@timestamp"
	}
	interval := pickHistogramInterval(from, to)
	intervalSec := histogramIntervalSeconds(interval)
	query := prep.boolQuery()

	attempts := []map[string]any{
		nginxAccessStatsAggBody(query, tsField, interval, topN,
			[]string{"remote.keyword", "remote", "fields.remote.keyword", "fields.remote"},
			[]string{"status.keyword", "status", "fields.status.keyword", "fields.status"},
			[]string{"uri.keyword", "uri", "fields.uri.keyword", "fields.uri"},
			[]string{"request.keyword", "request", "fields.request.keyword", "fields.request"},
			[]string{"request_time", "fields.request_time"},
		),
		nginxAccessStatsAggBody(query, tsField, interval, topN,
			[]string{"remote"},
			[]string{"status"},
			[]string{"uri"},
			[]string{"request"},
			[]string{"request_time"},
		),
		nginxAccessStatsAggBody(query, tsField, interval, topN, nil, nil, nil, nil, nil),
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
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("Nginx 访问统计聚合失败: %v", lastErr))
	}

	pv := parseTotalHits(raw)
	uv := parseCardinalityAgg(raw, "uv")
	hist := parseDateHistogram(raw, "access_histogram")
	statusCounts := mergeTermsAggs(raw, "status_terms_0", "status_terms_1", "status_terms_2", "status_terms_3")
	uriTerms := mergeTermsAggs(raw, "uri_terms_0", "uri_terms_1", "uri_terms_2", "uri_terms_3")
	reqTerms := mergeTermsAggs(raw, "request_terms_0", "request_terms_1", "request_terms_2", "request_terms_3")

	classCounts := classifyHTTPStatusCounts(statusCounts)
	rates := statusClassRates(classCounts, pv)

	durationSec := timeRangeSeconds(from, to)
	avgQPS := float64(pv) / math.Max(1, durationSec)
	peakQPS := peakQPSFromHistogram(hist, intervalSec)

	topURIs := buildTopURIs(uriTerms, reqTerms, topN, pv)

	out := &NginxAccessStatsResult{
		From:          from,
		To:            to,
		PV:            pv,
		UV:            uv,
		AvgQPS:        roundFloat(avgQPS, 3),
		PeakQPS:       roundFloat(peakQPS, 3),
		Status2xxRate: rates[2],
		Status3xxRate: rates[3],
		Status4xxRate: rates[4],
		Status5xxRate: rates[5],
		Histogram:     hist,
		TopURIs:       topURIs,
	}

	avgSec, p95Sec, p99Sec, ok := parseLatencyAggs(raw)
	if ok {
		out.LatencyAvailable = true
		avgMs := roundFloat(avgSec*1000, 2)
		p95Ms := roundFloat(p95Sec*1000, 2)
		p99Ms := roundFloat(p99Sec*1000, 2)
		out.AvgLatencyMs = &avgMs
		out.P95LatencyMs = &p95Ms
		out.P99LatencyMs = &p99Ms
	} else {
		out.LatencyHint = "未解析到 request_time。请在 Nginx log_format 末尾增加 $request_time，并使用 nginx_access 解析模板后重新下发 Agent。"
	}
	return out, nil
}

func nginxAccessDocFilter() map[string]any {
	return map[string]any{
		"bool": map[string]any{
			"must": []map[string]any{
				{"bool": map[string]any{
					"should": []map[string]any{
						{"exists": map[string]any{"field": "status"}},
						{"exists": map[string]any{"field": "fields.status"}},
					},
					"minimum_should_match": 1,
				}},
				{"bool": map[string]any{
					"should": []map[string]any{
						{"exists": map[string]any{"field": "request"}},
						{"exists": map[string]any{"field": "uri"}},
						{"exists": map[string]any{"field": "fields.request"}},
						{"exists": map[string]any{"field": "fields.uri"}},
					},
					"minimum_should_match": 1,
				}},
			},
		},
	}
}

func nginxAccessStatsAggBody(
	query any,
	tsField, interval string,
	topN int,
	remoteFields, statusFields, uriFields, requestFields, latencyFields []string,
) map[string]any {
	dh := map[string]any{
		"field":          tsField,
		"fixed_interval": interval,
		"min_doc_count":  0,
	}
	aggs := map[string]any{
		"access_histogram": map[string]any{
			"date_histogram": dh,
		},
	}
	for _, field := range remoteFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		aggs["uv"] = map[string]any{
			"cardinality": map[string]any{"field": field},
		}
		break
	}
	for i, field := range statusFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		aggs[fmt.Sprintf("status_terms_%d", i)] = map[string]any{
			"terms": map[string]any{"field": field, "size": 50},
		}
	}
	for i, field := range uriFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		aggs[fmt.Sprintf("uri_terms_%d", i)] = map[string]any{
			"terms": map[string]any{"field": field, "size": topN * 3},
		}
	}
	for i, field := range requestFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		aggs[fmt.Sprintf("request_terms_%d", i)] = map[string]any{
			"terms": map[string]any{"field": field, "size": topN * 5},
		}
	}
	if len(latencyFields) > 0 {
		lf := strings.TrimSpace(latencyFields[0])
		if lf != "" {
			aggs["latency_avg"] = map[string]any{
				"avg": map[string]any{"field": lf},
			}
			aggs["latency_pct"] = map[string]any{
				"percentiles": map[string]any{
					"field":    lf,
					"percents": []float64{95, 99},
				},
			}
		}
	}
	return map[string]any{
		"size":             0,
		"track_total_hits": true,
		"query":            query,
		"aggs":             aggs,
	}
}

func parseCardinalityAgg(raw map[string]any, name string) int64 {
	aggs, _ := raw["aggregations"].(map[string]any)
	node, _ := aggs[name].(map[string]any)
	if v, ok := node["value"].(float64); ok {
		return int64(v)
	}
	return 0
}

func parseLatencyAggs(raw map[string]any) (avg, p95, p99 float64, ok bool) {
	aggs, _ := raw["aggregations"].(map[string]any)
	avgNode, _ := aggs["latency_avg"].(map[string]any)
	pctNode, _ := aggs["latency_pct"].(map[string]any)
	avgVal, avgOK := avgNode["value"].(float64)
	if !avgOK || math.IsNaN(avgVal) {
		return 0, 0, 0, false
	}
	values, _ := pctNode["values"].(map[string]any)
	p95Val, p95OK := percentileValue(values, "95.0", "95")
	p99Val, p99OK := percentileValue(values, "99.0", "99")
	if !p95OK || !p99OK {
		return avgVal, 0, 0, true
	}
	return avgVal, p95Val, p99Val, true
}

func percentileValue(values map[string]any, keys ...string) (float64, bool) {
	for _, k := range keys {
		if v, ok := values[k].(float64); ok && !math.IsNaN(v) {
			return v, true
		}
	}
	return 0, false
}

func histogramIntervalSeconds(interval string) float64 {
	switch strings.TrimSpace(interval) {
	case "1m":
		return 60
	case "5m":
		return 300
	case "30m":
		return 1800
	case "2h":
		return 7200
	case "1d":
		return 86400
	default:
		return 3600
	}
}

func timeRangeSeconds(from, to string) float64 {
	tf, err1 := time.Parse(time.RFC3339, strings.TrimSpace(from))
	tt, err2 := time.Parse(time.RFC3339, strings.TrimSpace(to))
	if err1 != nil || err2 != nil || !tt.After(tf) {
		return 1
	}
	sec := tt.Sub(tf).Seconds()
	if sec < 1 {
		return 1
	}
	return sec
}

func peakQPSFromHistogram(hist []LogHistogramBucket, intervalSec float64) float64 {
	if intervalSec <= 0 {
		intervalSec = 1
	}
	var maxCount int64
	for _, b := range hist {
		if b.Count > maxCount {
			maxCount = b.Count
		}
	}
	return float64(maxCount) / intervalSec
}

// classifyHTTPStatusCounts 将 status terms 按首位数字归入 2/3/4/5。
func classifyHTTPStatusCounts(statusCounts map[string]int64) map[int]int64 {
	out := map[int]int64{2: 0, 3: 0, 4: 0, 5: 0}
	for k, c := range statusCounts {
		k = strings.TrimSpace(k)
		if k == "" || c <= 0 {
			continue
		}
		code, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		class := code / 100
		if class < 2 || class > 5 {
			continue
		}
		out[class] += c
	}
	return out
}

func statusClassRates(classCounts map[int]int64, total int64) map[int]float64 {
	out := map[int]float64{2: 0, 3: 0, 4: 0, 5: 0}
	if total <= 0 {
		return out
	}
	for class, c := range classCounts {
		out[class] = roundFloat(float64(c)/float64(total)*100, 2)
	}
	return out
}

// parseURIFromRequest 从 "METHOD /path HTTP/1.1" 或裸 path 提取 URI。
func parseURIFromRequest(request string) string {
	request = strings.TrimSpace(request)
	if request == "" || request == "-" {
		return ""
	}
	parts := strings.Fields(request)
	switch len(parts) {
	case 0:
		return ""
	case 1:
		if strings.HasPrefix(parts[0], "/") {
			return parts[0]
		}
		return ""
	default:
		// METHOD URI [PROTO]
		if strings.HasPrefix(parts[1], "/") {
			return parts[1]
		}
		if strings.HasPrefix(parts[0], "/") {
			return parts[0]
		}
		return ""
	}
}

func buildTopURIs(uriTerms, reqTerms map[string]int64, topN int, pv int64) []NginxAccessURIStat {
	merged := map[string]int64{}
	for uri, c := range uriTerms {
		uri = strings.TrimSpace(uri)
		if uri == "" || uri == "-" {
			continue
		}
		merged[uri] += c
	}
	if len(merged) == 0 {
		for req, c := range reqTerms {
			uri := parseURIFromRequest(req)
			if uri == "" {
				continue
			}
			merged[uri] += c
		}
	}
	type kv struct {
		uri string
		c   int64
	}
	list := make([]kv, 0, len(merged))
	for u, c := range merged {
		list = append(list, kv{uri: u, c: c})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].c == list[j].c {
			return list[i].uri < list[j].uri
		}
		return list[i].c > list[j].c
	})
	if len(list) > topN {
		list = list[:topN]
	}
	out := make([]NginxAccessURIStat, 0, len(list))
	for _, item := range list {
		pct := 0.0
		if pv > 0 {
			pct = roundFloat(float64(item.c)/float64(pv)*100, 2)
		}
		out = append(out, NginxAccessURIStat{URI: item.uri, Count: item.c, Percent: pct})
	}
	return out
}

func roundFloat(v float64, places int) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}
