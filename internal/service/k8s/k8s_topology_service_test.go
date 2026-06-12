package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServiceMatchesWorkloadSelector(t *testing.T) {
	svc := corev1.Service{
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "nginx"},
		},
	}
	workload := &metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "nginx", "version": "v1"},
	}
	if !serviceMatchesWorkloadSelector(svc, workload) {
		t.Fatal("service selector should be subset of workload labels")
	}

	strictSvc := corev1.Service{
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "nginx", "version": "v2"},
		},
	}
	if serviceMatchesWorkloadSelector(strictSvc, workload) {
		t.Fatal("service selector must not exceed workload labels")
	}
}
