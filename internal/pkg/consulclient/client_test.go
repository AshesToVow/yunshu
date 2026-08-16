package consulclient

import "testing"

func TestHasTag(t *testing.T) {
	t.Parallel()
	if !HasTag([]string{"a", "yunshu-metrics"}, "yunshu-metrics") {
		t.Fatal("expected match")
	}
	if HasTag([]string{"a"}, "yunshu-metrics") {
		t.Fatal("expected miss")
	}
	if !HasTag([]string{"x"}, "") {
		t.Fatal("empty want matches all")
	}
}

func TestAggregateHealth(t *testing.T) {
	t.Parallel()
	if AggregateHealth([]CatalogCheck{{Status: "passing"}, {Status: "warning"}}) != "warning" {
		t.Fatal("warning")
	}
	if AggregateHealth([]CatalogCheck{{Status: "warning"}, {Status: "critical"}}) != "critical" {
		t.Fatal("critical")
	}
}
