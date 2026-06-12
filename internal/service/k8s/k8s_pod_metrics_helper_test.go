package k8s

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsPodCountedOnNode(t *testing.T) {
	now := metav1.NewTime(time.Now())
	tests := []struct {
		name string
		pod  corev1.Pod
		want bool
	}{
		{"running", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning}}, true},
		{"pending", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}}, true},
		{"succeeded", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodSucceeded}}, false},
		{"failed", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}}, false},
		{"deleting", corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		}, false},
	}
	for _, tc := range tests {
		got := isPodCountedOnNode(tc.pod)
		if got != tc.want {
			t.Fatalf("%s: isPodCountedOnNode() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestPodMetricKey(t *testing.T) {
	if got := podMetricKey("kube-system", "coredns"); got != "kube-system/coredns" {
		t.Fatalf("podMetricKey() = %q", got)
	}
}
