package esclient

import (
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
		{http.MethodPut, "/_cluster/settings", true},
		{http.MethodDelete, "/_alias/foo", true},
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
