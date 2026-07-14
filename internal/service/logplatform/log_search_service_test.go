package logplatform

import "testing"

func TestTermIDFilterUsesKeyword(t *testing.T) {
	q := termIDFilter("project_id", "1")
	boolQ, ok := q["bool"].(map[string]any)
	if !ok {
		t.Fatalf("expected bool query, got %#v", q)
	}
	should, ok := boolQ["should"].([]map[string]any)
	if !ok || len(should) < 2 {
		t.Fatalf("expected should clauses, got %#v", boolQ)
	}
	foundKeyword := false
	foundNested := false
	for _, clause := range should {
		term, ok := clause["term"].(map[string]any)
		if !ok {
			continue
		}
		if _, ok := term["project_id.keyword"]; ok {
			foundKeyword = true
		}
		if _, ok := term["fields.project_id.keyword"]; ok {
			foundNested = true
		}
	}
	if !foundKeyword {
		t.Fatal("expected project_id.keyword term clause")
	}
	if !foundNested {
		t.Fatal("expected fields.project_id.keyword term clause")
	}
}

func TestTimeRangeFilterIncludesLegacyMissingTimestamp(t *testing.T) {
	q := timeRangeFilter("2026-07-13T00:00:00Z", "2026-07-13T23:59:59Z", "@timestamp")
	boolQ, ok := q["bool"].(map[string]any)
	if !ok {
		t.Fatalf("expected bool query, got %#v", q)
	}
	should, ok := boolQ["should"].([]map[string]any)
	if !ok || len(should) < 2 {
		t.Fatalf("expected multiple should clauses, got %#v", boolQ)
	}
}
