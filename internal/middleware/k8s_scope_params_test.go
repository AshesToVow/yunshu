package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResolveK8sScopeParams(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		params    k8sScopeParams
		wantID    uint
		wantNS    string
		wantError string
	}{
		{
			name:   "consistent",
			params: k8sScopeParams{clusters: []uint{3, 3}, namespaces: []string{" app ", "app"}},
			wantID: 3,
			wantNS: "app",
		},
		{
			name:   "empty",
			params: k8sScopeParams{},
		},
		{
			name:      "cluster_mismatch",
			params:    k8sScopeParams{clusters: []uint{3, 4}},
			wantError: "cluster_id 在查询参数、路径与请求体中不一致",
		},
		{
			name:      "namespace_mismatch",
			params:    k8sScopeParams{namespaces: []string{"app", "prod"}},
			wantError: "namespace 在查询参数与请求体中不一致",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			id, ns, err := resolveK8sScopeParams(tc.params)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("err=%v want contains %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if id != tc.wantID || ns != tc.wantNS {
				t.Fatalf("id=%d ns=%q want %d %q", id, ns, tc.wantID, tc.wantNS)
			}
		})
	}
}

func newScopeTestContext(t *testing.T, method, target, body string, params gin.Params) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	c.Request = req
	c.Params = params
	return c
}

func TestExtractClusterNamespaceFromRequest_ClusterRoute(t *testing.T) {
	t.Parallel()
	c := newScopeTestContext(t, http.MethodPut, "/api/v1/clusters/3", "", gin.Params{{Key: "id", Value: "3"}})
	id, ns, err := extractClusterNamespaceFromRequest(c, "/api/v1/clusters/:id")
	if err != nil || id != 3 || ns != "" {
		t.Fatalf("id=%d ns=%q err=%v", id, ns, err)
	}

	c = newScopeTestContext(t, http.MethodPut, "/api/v1/clusters/3", "", gin.Params{{Key: "id", Value: "3"}})
	c.Request.URL.RawQuery = "cluster_id=4"
	_, _, err = extractClusterNamespaceFromRequest(c, "/api/v1/clusters/:id")
	if err == nil || !strings.Contains(err.Error(), "不一致") {
		t.Fatalf("path/query mismatch must be rejected, err=%v", err)
	}
}

func TestExtractClusterNamespaceFromRequest_ResourceIDNotCluster(t *testing.T) {
	t.Parallel()
	c := newScopeTestContext(t, http.MethodDelete, "/api/v1/pods/99?cluster_id=3&namespace=app", "",
		gin.Params{{Key: "id", Value: "99"}})
	id, ns, err := extractClusterNamespaceFromRequest(c, "/api/v1/pods/:id")
	if err != nil || id != 3 || ns != "app" {
		t.Fatalf("id=%d ns=%q err=%v", id, ns, err)
	}
}

func TestExtractClusterNamespaceFromRequest_QueryBodyMismatch(t *testing.T) {
	t.Parallel()
	c := newScopeTestContext(t, http.MethodPost, "/api/v1/pods/exec?cluster_id=3&namespace=app",
		`{"cluster_id":4,"namespace":"prod"}`, nil)
	_, _, err := extractClusterNamespaceFromRequest(c, "/api/v1/pods/exec")
	if err == nil || !strings.Contains(err.Error(), "不一致") {
		t.Fatalf("query/body mismatch must be rejected, err=%v", err)
	}
}

func TestCollectK8sScopeParams_RestoresBody(t *testing.T) {
	t.Parallel()
	body := `{"cluster_id":3,"namespace":"app"}`
	c := newScopeTestContext(t, http.MethodPost, "/api/v1/pods", body, nil)
	id, ns, err := extractClusterNamespaceFromRequest(c, "/api/v1/pods")
	if err != nil || id != 3 || ns != "app" {
		t.Fatalf("id=%d ns=%q err=%v", id, ns, err)
	}
	restored := make([]byte, len(body))
	n, _ := c.Request.Body.Read(restored)
	if string(restored[:n]) != body {
		t.Fatalf("request body not restored: %q", restored[:n])
	}
}
