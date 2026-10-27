package k8s

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestWatchStatusFatal(t *testing.T) {
	t.Parallel()
	if !watchStatusFatal(&metav1.Status{Code: 403}) {
		t.Fatal("403 must be terminal")
	}
	if !watchStatusFatal(&metav1.Status{Code: 401}) {
		t.Fatal("401 must be terminal")
	}
	if watchStatusFatal(&metav1.Status{Code: 410, Reason: metav1.StatusReasonExpired}) {
		t.Fatal("410 must be retried with a fresh resourceVersion")
	}
	if watchStatusFatal(nil) {
		t.Fatal("nil status must not be terminal")
	}
}

func TestObjectResourceVersion(t *testing.T) {
	t.Parallel()
	u := &unstructured.Unstructured{}
	u.SetResourceVersion("12345")
	if got := objectResourceVersion(u); got != "12345" {
		t.Fatalf("got %q", got)
	}
	if got := objectResourceVersion(nil); got != "" {
		t.Fatalf("nil object got %q", got)
	}
}

func TestWatchBackoffBounded(t *testing.T) {
	t.Parallel()
	if d := watchBackoff(0); d != time.Second {
		t.Fatalf("backoff(0)=%v", d)
	}
	if d := watchBackoff(1); d != time.Second {
		t.Fatalf("backoff(1)=%v", d)
	}
	if d := watchBackoff(10); d != 16*time.Second {
		t.Fatalf("backoff(10)=%v", d)
	}
}
