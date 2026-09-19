package ai

import (
	"encoding/json"
	"testing"
)

func TestParseUintSlice(t *testing.T) {
	t.Parallel()
	got := parseUintSlice([]any{float64(1), float64(2), float64(0), "x"})
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("got %#v", got)
	}
	var raw any
	_ = json.Unmarshal([]byte(`[3,4]`), &raw)
	got2 := parseUintSlice(raw)
	if len(got2) != 2 || got2[0] != 3 || got2[1] != 4 {
		t.Fatalf("json got %#v", got2)
	}
}

func TestMonitorToolDefinitionsPresent(t *testing.T) {
	t.Parallel()
	s := &Service{}
	defs := s.monitorToolDefinitions()
	names := map[string]bool{}
	for _, d := range defs {
		names[d.Function.Name] = true
	}
	for _, want := range []string{
		"list_alert_datasources", "query_prometheus", "query_prometheus_range",
		"list_prometheus_active_alerts", "get_alert_detail",
	} {
		if !names[want] {
			t.Fatalf("missing tool %s", want)
		}
	}
}
