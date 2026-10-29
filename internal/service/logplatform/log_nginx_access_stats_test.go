package logplatform

import (
	"strings"
	"testing"
)

func TestParseURIFromRequest(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"GET /api/user/list HTTP/1.1", "/api/user/list"},
		{"POST /api/login HTTP/1.0", "/api/login"},
		{"/api/only", "/api/only"},
		{"-", ""},
		{"", ""},
		{"GET", ""},
	}
	for _, tc := range cases {
		if got := parseURIFromRequest(tc.in); got != tc.want {
			t.Fatalf("parseURIFromRequest(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestClassifyHTTPStatusCounts(t *testing.T) {
	t.Parallel()
	got := classifyHTTPStatusCounts(map[string]int64{
		"200": 80,
		"301": 5,
		"404": 10,
		"500": 5,
		"ok":  3,
	})
	if got[2] != 80 || got[3] != 5 || got[4] != 10 || got[5] != 5 {
		t.Fatalf("unexpected class counts: %#v", got)
	}
}

func TestStatusClassRatesAndQPS(t *testing.T) {
	t.Parallel()
	rates := statusClassRates(map[int]int64{2: 90, 3: 0, 4: 10, 5: 0}, 100)
	if rates[2] != 90.0 || rates[4] != 10.0 {
		t.Fatalf("unexpected rates: %#v", rates)
	}
	peak := peakQPSFromHistogram([]LogHistogramBucket{{Count: 120}, {Count: 60}}, 60)
	if peak != 2.0 {
		t.Fatalf("peak QPS=%v want 2", peak)
	}
	if timeRangeSeconds("bad", "worse") != 1.0 {
		t.Fatal("invalid time range should fallback to 1s")
	}
}

func TestBuildTopURIsFallbackRequest(t *testing.T) {
	t.Parallel()
	out := buildTopURIs(nil, map[string]int64{
		"GET /api/user/list HTTP/1.1":  50,
		"GET /api/order/list HTTP/1.1": 30,
		"POST /api/login HTTP/1.1":     20,
	}, 5, 100)
	if len(out) != 3 {
		t.Fatalf("len=%d want 3", len(out))
	}
	if out[0].URI != "/api/user/list" || out[0].Count != 50 || out[0].Percent != 50.0 {
		t.Fatalf("unexpected top[0]: %#v", out[0])
	}
}

func TestNginxAccessStatsAggBodyHasKeys(t *testing.T) {
	t.Parallel()
	body := nginxAccessStatsAggBody(
		map[string]any{"match_all": map[string]any{}},
		"@timestamp", "1h", 5,
		nginxAccessFieldChoice{
			Remote:  "remote.keyword",
			Status:  "status",
			URI:     "uri.keyword",
			Request: "request.keyword",
			Latency: "request_time",
		},
	)
	if _, exists := body["runtime_mappings"]; exists {
		t.Fatal("numeric latency field must not need runtime_mappings")
	}
	aggs, ok := body["aggs"].(map[string]any)
	if !ok {
		t.Fatal("missing aggs")
	}
	for _, key := range []string{"access_histogram", "uv", "status_terms_0", "uri_terms_0", "request_terms_0", "latency_avg", "latency_pct"} {
		if _, exists := aggs[key]; !exists {
			t.Fatalf("missing agg key %s", key)
		}
	}
}

func TestNginxAccessStatsRuntimeLatencyFallback(t *testing.T) {
	t.Parallel()
	body := nginxAccessStatsAggBody(
		map[string]any{"match_all": map[string]any{}},
		"@timestamp", "1h", 5,
		nginxAccessFieldChoice{
			Remote:         "remote.keyword",
			Status:         "status",
			URI:            "uri.keyword",
			Request:        "request.keyword",
			Latency:        "request_time.keyword",
			RuntimeLatency: true,
		},
	)
	runtime, ok := body["runtime_mappings"].(map[string]any)
	if !ok {
		t.Fatal("missing runtime_mappings")
	}
	field, ok := runtime["request_time_num"].(map[string]any)
	if !ok || field["type"] != "double" {
		t.Fatalf("unexpected runtime field: %#v", runtime)
	}
	aggs, _ := body["aggs"].(map[string]any)
	pct, _ := aggs["latency_pct"].(map[string]any)
	percentiles, _ := pct["percentiles"].(map[string]any)
	if percentiles["field"] != "request_time_num" {
		t.Fatalf("latency field=%v want request_time_num", percentiles["field"])
	}
}

// 真实索引里 remote/uri/request 是动态映射的 text + keyword，
// status 是模板里的 keyword，request_time 老索引是 text + keyword。
// 选择结果必须全部落在 keyword/numeric 上，否则 terms/percentiles 会让整单 400。
func TestResolveNginxAccessFieldsAvoidsText(t *testing.T) {
	t.Parallel()
	caps := map[string][]string{
		"remote":                {"text"},
		"remote.keyword":        {"keyword"},
		"status":                {"keyword"},
		"uri":                   {"text"},
		"uri.keyword":           {"keyword"},
		"request":               {"text"},
		"request.keyword":       {"keyword"},
		"request_time":          {"text"},
		"request_time.keyword":  {"keyword"},
		"fields.request_time":   {"double"},
		"fields.remote":         {"text"},
		"fields.remote.keyword": {"keyword"},
	}
	choice := resolveNginxAccessFields(caps)
	if choice.Remote != "remote.keyword" || choice.Status != "status" ||
		choice.URI != "uri.keyword" || choice.Request != "request.keyword" {
		t.Fatalf("unexpected choice: %#v", choice)
	}
	if choice.Latency != "fields.request_time" || choice.RuntimeLatency {
		t.Fatalf("should prefer numeric latency field: %#v", choice)
	}

	// 混合类型（老索引 text、新索引 keyword）同样不可选
	mixed := map[string][]string{
		"status":      {"keyword"},
		"uri":         {"text", "keyword"},
		"uri.keyword": {"keyword"},
	}
	choice = resolveNginxAccessFields(mixed)
	if choice.URI != "uri.keyword" {
		t.Fatalf("mixed-type field must not be aggregated directly: %#v", choice)
	}

	// 只有 text 的 request_time 必须走 runtime field
	textOnly := map[string][]string{
		"status":               {"keyword"},
		"request_time":         {"text"},
		"request_time.keyword": {"keyword"},
	}
	choice = resolveNginxAccessFields(textOnly)
	if choice.Latency != "request_time.keyword" || !choice.RuntimeLatency {
		t.Fatalf("text latency field should fall back to runtime mapping: %#v", choice)
	}

	// _field_caps 不可用时按映射约定退化：status 取模板裸字段，其余取 .keyword 子字段
	choice = resolveNginxAccessFields(nil)
	if choice.Status != "status" || choice.Remote != "remote.keyword" ||
		choice.URI != "uri.keyword" || choice.Request != "request.keyword" {
		t.Fatalf("unexpected fallback choice: %#v", choice)
	}
	if choice.Latency != "request_time.keyword" || !choice.RuntimeLatency {
		t.Fatalf("fallback latency should use runtime mapping: %#v", choice)
	}
}

func TestShardFailureReason(t *testing.T) {
	t.Parallel()
	partial := map[string]any{
		"_shards": map[string]any{
			"failed": float64(2),
			"failures": []any{
				map[string]any{"reason": map[string]any{
					"type":   "illegal_argument_exception",
					"reason": "Text fields are not optimised for operations that require per-document field data",
				}},
			},
		},
	}
	if got := shardFailureReason(partial); !strings.Contains(got, "Text fields") {
		t.Fatalf("unexpected reason: %q", got)
	}
	if got := shardFailureReason(map[string]any{"_shards": map[string]any{"failed": float64(0)}}); got != "" {
		t.Fatalf("healthy response should return empty, got %q", got)
	}
	if got := shardFailureReason(map[string]any{}); got != "" {
		t.Fatalf("missing _shards should return empty, got %q", got)
	}
	if got := shardFailureReason(map[string]any{"_shards": map[string]any{"failed": float64(1)}}); got == "" {
		t.Fatal("count-only failure should be detected")
	}
}
