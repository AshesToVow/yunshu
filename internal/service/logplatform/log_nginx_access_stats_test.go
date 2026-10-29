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
		[]string{"remote.keyword"},
		[]string{"status.keyword"},
		[]string{"uri.keyword"},
		[]string{"request.keyword"},
		[]string{"request_time"},
	)
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
		[]string{"remote.keyword"},
		[]string{"status.keyword"},
		[]string{"uri.keyword"},
		[]string{"request.keyword"},
		[]string{"request_time_num"},
	)
	body["runtime_mappings"] = nginxAccessLatencyRuntimeMapping()
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
