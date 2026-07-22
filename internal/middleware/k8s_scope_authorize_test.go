package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"yunshu/internal/middleware"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

func TestK8sScopeAuthorize_SetsRequestScopeAndNSGate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(auth.ContextUserKey, &auth.CurrentUser{
			ID:        1,
			RoleCodes: []string{"super-admin"},
		})
		c.Next()
	})
	var gotScope k8sauth.RequestScope
	var gotOK bool
	r.GET("/api/v1/pods", middleware.K8sScopeAuthorize(nil, nil, nil, nil, nil), func(c *gin.Context) {
		gotScope, gotOK = k8sauth.RequestScopeFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pods?cluster_id=7&namespace=production", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if !gotOK || gotScope.ClusterID != 7 || gotScope.Namespace != "production" {
		t.Fatalf("scope=%+v ok=%v", gotScope, gotOK)
	}
}

func TestIsK8sReadAPIPath_Pods(t *testing.T) {
	if !service.IsK8sReadAPIPath("/api/v1/pods") {
		t.Fatal("expected pods to be read path")
	}
}
