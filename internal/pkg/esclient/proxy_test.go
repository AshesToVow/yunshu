package esclient

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestProxyPathAllowed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		method string
		path   string
		ok     bool
	}{
		{http.MethodGet, "/_cluster/health", true},
		{http.MethodGet, "/_cat/indices?v", true},
		{http.MethodHead, "/", true},
		{http.MethodPost, "/my-index/_search", true},
		{http.MethodPut, "/my-index", true},
		{http.MethodPut, "/my-index/_settings", true},
		{http.MethodPut, "/my-index/_doc/1", true},
		{http.MethodDelete, "/my-index", true},
		{http.MethodDelete, "/my-index/_doc/1", true},
		{http.MethodPost, "/my-index/_refresh", true},
		{http.MethodPut, "/_cluster/settings", false},
		{http.MethodDelete, "/_alias/foo", false},
		{http.MethodPost, "/my-index/_delete_by_query", false},
		{http.MethodPost, "/my-index/_reindex", false},
		{http.MethodGet, "/_scripts/foo", false},
		{http.MethodPost, "/my-index/_search?q=painless", false},
		{http.MethodGet, "/_nodes/shutdown", false},
		{"PATCH", "/my-index", false},
		{http.MethodGet, "/_unknown_system", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()
			got := proxyPathAllowed(tc.method, tc.path)
			if got != tc.ok {
				t.Fatalf("proxyPathAllowed(%q, %q)=%v want %v", tc.method, tc.path, got, tc.ok)
			}
		})
	}
}

func TestEncodeProxyBody(t *testing.T) {
	t.Parallel()
	plain := encodeProxyBody([]byte("health status index\nyellow open foo 1"))
	if !json.Valid(plain) {
		t.Fatalf("plain text body must become valid JSON, got %s", plain)
	}
	var s string
	if err := json.Unmarshal(plain, &s); err != nil {
		t.Fatalf("expected JSON string wrap: %v", err)
	}
	obj := encodeProxyBody([]byte(`{"ok":true}`))
	var m map[string]any
	if err := json.Unmarshal(obj, &m); err != nil || m["ok"] != true {
		t.Fatalf("json body should pass through: %s err=%v", obj, err)
	}
	empty := encodeProxyBody(nil)
	if string(empty) != "null" {
		t.Fatalf("empty -> null, got %s", empty)
	}
	type wrap struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	}
	b, err := json.Marshal(wrap{Status: 200, Body: plain})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if !json.Valid(b) {
		t.Fatalf("envelope invalid: %s", b)
	}
}
