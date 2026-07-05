package k8s

import (
	"encoding/json"
	"testing"
)

func TestSummarizeHarborChart(t *testing.T) {
	summary := summarizeHarborChart("springbootdemo", []harborChartVersionItem{
		{Version: "38", Created: "2026-07-05T03:00:00Z"},
		{Version: "39", Created: "2026-07-05T04:04:59Z", Deprecated: false},
	})
	if summary.LatestVersion != "39" {
		t.Fatalf("latest_version = %q, want 39", summary.LatestVersion)
	}
	if summary.TotalVersions != 2 {
		t.Fatalf("total_versions = %d, want 2", summary.TotalVersions)
	}
}

func TestParseHarborChartListBodyHarbor22Array(t *testing.T) {
	body := []byte(`[{"name":"springbootdemo","total_versions":7,"latest_version":"39","created":"2026-07-04T17:54:35.530231451Z","deprecated":false}]`)
	out, err := parseHarborChartListBody(body)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(out) != 1 || out[0].Name != "springbootdemo" || out[0].LatestVersion != "39" || out[0].TotalVersions != 7 {
		t.Fatalf("unexpected result: %+v", out)
	}
}

func TestParseHarborChartListBodyChartMuseumMap(t *testing.T) {
	body := []byte(`{
		"springbootdemo": [
			{"name":"springbootdemo","version":"39","created":"2026-07-05T04:04:59Z","deprecated":false}
		]
	}`)
	out, err := parseHarborChartListBody(body)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(out) != 1 || out[0].LatestVersion != "39" {
		t.Fatalf("unexpected result: %+v", out)
	}
}

func TestParseHarborChartListBodyEmpty(t *testing.T) {
	out, err := parseHarborChartListBody([]byte(`[]`))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty list, got %+v", out)
	}
}

func TestParseHarborChartListBodyInvalidJSON(t *testing.T) {
	_, err := parseHarborChartListBody([]byte(`not-json`))
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestHarborChartVersionItemUnmarshal(t *testing.T) {
	body := []byte(`[{"name":"springbootdemo","version":"39","created":"2026-07-05T04:04:59Z","deprecated":false}]`)
	var raw []harborChartVersionItem
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(raw) != 1 || raw[0].Version != "39" {
		t.Fatalf("unexpected versions: %+v", raw)
	}
}
