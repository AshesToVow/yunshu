package k8s

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestK8sDeleteOptionsToMetav1(t *testing.T) {
	grace := int64(15)
	bg := "Background"
	opts, err := (K8sDeleteOptions{
		GracePeriodSeconds: &grace,
		PropagationPolicy:  &bg,
	}).ToMetav1()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.GracePeriodSeconds == nil || *opts.GracePeriodSeconds != 15 {
		t.Fatalf("grace period = %v, want 15", opts.GracePeriodSeconds)
	}
	if opts.PropagationPolicy == nil || *opts.PropagationPolicy != metav1.DeletePropagationBackground {
		t.Fatalf("propagation = %v, want Background", opts.PropagationPolicy)
	}

	empty, err := (K8sDeleteOptions{}).ToMetav1()
	if err != nil {
		t.Fatalf("empty options error: %v", err)
	}
	if empty.GracePeriodSeconds != nil || empty.PropagationPolicy != nil {
		t.Fatalf("empty options should stay nil: %+v", empty)
	}

	bad := "Invalid"
	if _, err := (K8sDeleteOptions{PropagationPolicy: &bad}).ToMetav1(); err == nil {
		t.Fatal("expected invalid propagation error")
	}
}

func TestClusterNamespaceNameQueryDeleteOptionsBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(
		http.MethodDelete,
		"/deployments?cluster_id=1&namespace=default&name=demo&grace_period_seconds=0&propagation_policy=Background",
		nil,
	)

	var req ClusterNamespaceNameQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		t.Fatalf("bind query: %v", err)
	}
	if req.GracePeriodSeconds == nil || *req.GracePeriodSeconds != 0 {
		t.Fatalf("grace_period_seconds = %v, want 0", req.GracePeriodSeconds)
	}
	if req.PropagationPolicy == nil || *req.PropagationPolicy != "Background" {
		t.Fatalf("propagation_policy = %v, want Background", req.PropagationPolicy)
	}

	opts, err := req.K8sDeleteOptions.ToMetav1()
	if err != nil {
		t.Fatalf("to metav1: %v", err)
	}
	if opts.PropagationPolicy == nil || *opts.PropagationPolicy != metav1.DeletePropagationBackground {
		t.Fatalf("metav1 propagation = %v", opts.PropagationPolicy)
	}
}
