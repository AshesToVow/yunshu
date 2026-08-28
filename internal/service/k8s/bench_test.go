package k8s

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func BenchmarkWorkloadContainerIndex(b *testing.B) {
	containers := []corev1.Container{
		{Name: "c1"}, {Name: "c2"}, {Name: "c3"},
	}
	b.ResetTimer()
	for b.Loop() {
		workloadContainerIndex(containers, "c2")
	}
}

func BenchmarkDeploymentResourceSummary(b *testing.B) {
	cpu := resource.MustParse("250m")
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceCPU: cpu},
						},
					}},
				},
			},
		},
	}
	b.ResetTimer()
	for b.Loop() {
		deploymentResourceSummary(d)
	}
}
