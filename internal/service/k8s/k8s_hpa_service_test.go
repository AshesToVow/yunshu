package k8s

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestSummarizeHPAMetrics_V1(t *testing.T) {
	t.Parallel()
	obj := map[string]any{
		"spec": map[string]any{
			"targetCPUUtilizationPercentage": int64(80),
		},
	}
	if got := summarizeHPAMetrics(obj); got != "Resource/cpu(80%)" {
		t.Fatalf("got %q", got)
	}
}

func TestSummarizeHPAMetrics_V2(t *testing.T) {
	t.Parallel()
	obj := map[string]any{
		"spec": map[string]any{
			"metrics": []any{
				map[string]any{"type": "Resource", "resource": map[string]any{"name": "cpu"}},
				map[string]any{"type": "Pods", "pods": map[string]any{"metric": map[string]any{"name": "requests_per_second"}}},
			},
		},
	}
	if got := summarizeHPAMetrics(obj); got != "Resource/cpu,Pods/requests_per_second" {
		t.Fatalf("got %q", got)
	}
}

func TestHPAItemFromV1Unstructured(t *testing.T) {
	t.Parallel()
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "autoscaling/v1",
		"kind":       "HorizontalPodAutoscaler",
		"metadata": map[string]any{
			"name":      "web",
			"namespace": "default",
		},
		"spec": map[string]any{
			"minReplicas":                    int64(2),
			"maxReplicas":                    int64(6),
			"targetCPUUtilizationPercentage": int64(70),
			"scaleTargetRef": map[string]any{
				"kind": "Deployment",
				"name": "web",
			},
		},
	}}
	item := hpaToItem(u)
	if item.MinReplicas != "2" || item.MaxReplicas != "6" {
		t.Fatalf("replicas = %s/%s", item.MinReplicas, item.MaxReplicas)
	}
	if item.ScaleTargetRef != "Deployment/web" {
		t.Fatalf("ref = %q", item.ScaleTargetRef)
	}
	if item.MetricsSummary != "Resource/cpu(70%)" {
		t.Fatalf("metrics = %q", item.MetricsSummary)
	}
}
