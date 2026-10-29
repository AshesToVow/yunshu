package logplatform

import (
	"context"
	"fmt"
	"log/slog"
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

	// 聚合只能落在可聚合类型上：任一 text 字段参与 terms/percentiles 都会让整个请求 400 分片失败，
	// 所以不能把同一指标的候选字段一起发出去。先查 _field_caps，为每个指标挑一个在所有索引里
	// 类型都可聚合的字段；同一字段老索引 text、新索引 keyword 时必须退回 .keyword 子字段。
	caps, capsErr := cli.FieldCaps(ctx, prep.indices, nginxAccessFieldCandidates())
	if capsErr != nil {
		slog.Default().With("component", "lognginx").Warn(
			"nginx access field caps failed, using keyword-only candidates", "error", capsErr)
	}
	choice := resolveNginxAccessFields(caps)
	choices := []nginxAccessFieldChoice{choice}
	if caps == nil {
		// _field_caps 不可用：动态映射索引靠 .keyword 子字段，模板索引的 status 是裸 keyword，
		// 两种约定各试一次，取第一个真正产出分桶的结果。
		choices = append(choices, nginxAccessFieldChoice{
			Remote: "remote", Status: "status", URI: "uri", Request: "request", Latency: "request_time",
		})
	}

	attempts := make([]map[string]any, 0, len(choices)+2)
	for _, c := range choices {
		attempts = append(attempts, nginxAccessStatsAggBody(query, tsField, interval, topN, c))
		if c.Latency != "" {
			// 延迟聚合仍可能因个别索引的类型/权限失败，去掉它再试一次，保住 PV/UV/状态码/TOP URI。
			noLatency := c
			noLatency.Latency, noLatency.RuntimeLatency = "", false
			attempts = append(attempts, nginxAccessStatsAggBody(query, tsField, interval, topN, noLatency))
		}
	}
	// 最后兜底：只出 PV 与时序，保证页面不至于全空。
	attempts = append(attempts, nginxAccessStatsAggBody(query, tsField, interval, topN, nginxAccessFieldChoice{}))

	var raw map[string]any
	var lastErr error
	for _, body := range attempts {
		resp, err := cli.Search(ctx, prep.indices, body)
		if err != nil {
			lastErr = err
			continue
		}
		raw, lastErr = resp, nil
		if reason := shardFailureReason(resp); reason != "" {
			// 部分分片失败时 ES 仍返回 200，失败分片的数据既不计入 hits.total 也不进聚合，
			// 照单全收只会得到"全 0"的正常响应。
			slog.Default().With("component", "lognginx").Warn(
				"nginx access stats aggregation partially failed, falling back", "reason", reason)
			continue
		}
		if caps == nil && parseTotalHits(resp) > 0 && nginxAccessAggsEmpty(resp) {
			// 字段名选错时聚合空但无报错；换另一种映射约定再试。
			continue
		}
		break
	}
	if raw == nil {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("Nginx 访问统计聚合失败: %v", lastErr))
	}

	pv := parseTotalHits(raw)
	uv := parseCardinalityAgg(raw, "uv")
	hist := parseDateHistogram(raw, "access_histogram")
	statusCounts := parseTermsAgg(raw, "status_terms_0")
	// URI 区分大小写，不能走会统一转大写的 parseTermsAgg
	uriTerms := parseTermsAggPreserveCase(raw, "uri_terms_0")
	reqTerms := parseTermsAggPreserveCase(raw, "request_terms_0")

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

// nginxAccessFieldChoice 每个指标最终只选一个可聚合字段；空串表示该指标不可用。
type nginxAccessFieldChoice struct {
	Remote         string
	Status         string
	URI            string
	Request        string
	Latency        string
	RuntimeLatency bool
}

// nginxAccessFieldCandidates 参与 _field_caps 探测的候选字段（固定列表，不含用户输入）。
func nginxAccessFieldCandidates() []string {
	return []string{
		"remote.keyword", "remote", "fields.remote.keyword", "fields.remote",
		"status.keyword", "status", "fields.status.keyword", "fields.status",
		"uri.keyword", "uri", "fields.uri.keyword", "fields.uri",
		"request.keyword", "request", "fields.request.keyword", "fields.request",
		"request_time", "fields.request_time",
		"request_time.keyword", "fields.request_time.keyword",
	}
}

// nginxAggregatableTypes 能安全用于 terms/cardinality/avg/percentiles 的类型。
var nginxAggregatableTypes = map[string]struct{}{
	"keyword": {}, "constant_keyword": {}, "wildcard": {}, "ip": {},
	"integer": {}, "long": {}, "short": {}, "byte": {}, "unsigned_long": {},
	"double": {}, "float": {}, "half_float": {}, "scaled_float": {},
	"boolean": {}, "date": {},
}

func capTypes(caps map[string][]string, field string) []string {
	return caps[field]
}

// fieldAggregatable 要求字段在所有索引里的类型都可聚合：
// 同一字段老索引是 text、新索引是 keyword 时，仍会有一部分分片失败。
func fieldAggregatable(types []string) bool {
	if len(types) == 0 {
		return false
	}
	for _, t := range types {
		if _, ok := nginxAggregatableTypes[t]; !ok {
			return false
		}
	}
	return true
}

func pickAggregatableField(caps map[string][]string, candidates ...string) string {
	for _, field := range candidates {
		if fieldAggregatable(capTypes(caps, field)) {
			return field
		}
	}
	return ""
}

// pickLatencyField 优先用数值型主字段；主字段被动态映射成 text 时，
// 退回 keyword 子字段并经 runtime field 转 double（text 上做 percentiles 会分片失败）。
func pickLatencyField(caps map[string][]string) (string, bool) {
	for _, field := range []string{"request_time", "fields.request_time"} {
		if fieldAggregatable(capTypes(caps, field)) {
			return field, false
		}
	}
	for _, field := range []string{"request_time.keyword", "fields.request_time.keyword"} {
		if fieldAggregatable(capTypes(caps, field)) {
			return field, true
		}
	}
	return "", false
}

func resolveNginxAccessFields(caps map[string][]string) nginxAccessFieldChoice {
	if caps == nil {
		// _field_caps 不可用时按映射约定退化：status 由索引模板显式建成 keyword，
		// remote/uri/request 走动态映射的 .keyword 子字段，request_time 只能取 keyword。
		return nginxAccessFieldChoice{
			Remote:         "remote.keyword",
			Status:         "status",
			URI:            "uri.keyword",
			Request:        "request.keyword",
			Latency:        "request_time.keyword",
			RuntimeLatency: true,
		}
	}
	choice := nginxAccessFieldChoice{
		Remote:  pickAggregatableField(caps, "remote.keyword", "fields.remote.keyword", "remote", "fields.remote"),
		Status:  pickAggregatableField(caps, "status", "status.keyword", "fields.status", "fields.status.keyword"),
		URI:     pickAggregatableField(caps, "uri.keyword", "fields.uri.keyword", "uri", "fields.uri"),
		Request: pickAggregatableField(caps, "request.keyword", "fields.request.keyword", "request", "fields.request"),
	}
	choice.Latency, choice.RuntimeLatency = pickLatencyField(caps)
	return choice
}

func nginxAccessStatsAggBody(
	query any,
	tsField, interval string,
	topN int,
	choice nginxAccessFieldChoice,
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
	if choice.Remote != "" {
		aggs["uv"] = map[string]any{
			"cardinality": map[string]any{"field": choice.Remote},
		}
	}
	if choice.Status != "" {
		aggs["status_terms_0"] = map[string]any{
			"terms": map[string]any{"field": choice.Status, "size": 50},
		}
	}
	if choice.URI != "" {
		aggs["uri_terms_0"] = map[string]any{
			"terms": map[string]any{"field": choice.URI, "size": topN * 3},
		}
	}
	if choice.Request != "" {
		aggs["request_terms_0"] = map[string]any{
			"terms": map[string]any{"field": choice.Request, "size": topN * 5},
		}
	}
	latencyField := choice.Latency
	runtime := map[string]any{}
	if latencyField != "" && choice.RuntimeLatency {
		latencyField = nginxAccessLatencyRuntimeField
		runtime = nginxAccessLatencyRuntimeMapping(choice.Latency)
	}
	if latencyField != "" {
		aggs["latency_avg"] = map[string]any{
			"avg": map[string]any{"field": latencyField},
		}
		aggs["latency_pct"] = map[string]any{
			"percentiles": map[string]any{
				"field":    latencyField,
				"percents": []float64{95, 99},
			},
		}
	}
	body := map[string]any{
		"size":             0,
		"track_total_hits": true,
		"query":            query,
		"aggs":             aggs,
	}
	if len(runtime) > 0 {
		body["runtime_mappings"] = runtime
	}
	return body
}

const nginxAccessLatencyRuntimeField = "request_time_num"

// nginxAccessLatencyRuntimeMapping 用 runtime field 把 keyword 形态的 request_time 转成 double。
// 老索引按动态映射把该字段建成 text + keyword 子字段，percentiles 只能走 keyword。
func nginxAccessLatencyRuntimeMapping(keywordField string) map[string]any {
	keywordField = strings.TrimSpace(keywordField)
	if keywordField == "" {
		keywordField = "request_time.keyword"
	}
	source := fmt.Sprintf(
		"if (doc.containsKey('%s') && !doc['%s'].empty) { try { emit(Double.parseDouble(doc['%s'].value)); } catch (Exception e) { } }",
		keywordField, keywordField, keywordField)
	return map[string]any{
		nginxAccessLatencyRuntimeField: map[string]any{
			"type":   "double",
			"script": map[string]any{"source": source},
		},
	}
}

// shardFailureReason 提取 _shards.failed 的首条失败原因；全部分片成功时返回空串。
func shardFailureReason(raw map[string]any) string {
	shards, _ := raw["_shards"].(map[string]any)
	failed, _ := shards["failed"].(float64)
	if failed <= 0 {
		return ""
	}
	failures, _ := shards["failures"].([]any)
	if len(failures) > 0 {
		if fm, ok := failures[0].(map[string]any); ok {
			if reason, ok := fm["reason"].(map[string]any); ok {
				if text, ok := reason["reason"].(string); ok && text != "" {
					return truncateString(text, 300)
				}
			}
		}
	}
	return fmt.Sprintf("%d shard(s) failed", int(failed))
}

// nginxAccessAggsEmpty 所有指标聚合都没有分桶（字段名选错时 ES 不报错，只返回空）。
func nginxAccessAggsEmpty(raw map[string]any) bool {
	aggs, _ := raw["aggregations"].(map[string]any)
	for _, name := range []string{"uv", "status_terms_0", "uri_terms_0", "request_terms_0"} {
		node, _ := aggs[name].(map[string]any)
		if buckets, ok := node["buckets"].([]any); ok && len(buckets) > 0 {
			return false
		}
		if v, ok := node["value"].(float64); ok && v > 0 {
			return false
		}
	}
	return true
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
