package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsKubeControlPlanePod(t *testing.T) {
	tests := []struct {
		name string
		pod  corev1.Pod
		want bool
	}{
		{"apiserver", corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "kube-apiserver-master"}}, true},
		{"coredns", corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "coredns-abc"}}, false},
	}
	for _, tc := range tests {
		if got := isKubeControlPlanePod(tc.pod); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestNodeReadySummary(t *testing.T) {
	n := corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			}},
		},
	}
	ok, state, _, _ := nodeReadySummary(n)
	if !ok || state != "Ready" {
		t.Fatalf("nodeReadySummary() = %v %q", ok, state)
	}
}
