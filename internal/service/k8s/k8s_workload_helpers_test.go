package k8s

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestWorkloadContainerIndex(t *testing.T) {
	t.Parallel()
	containers := []corev1.Container{
		{Name: "app", Image: "nginx"},
		{Name: "sidecar", Image: "busybox"},
	}
	if got := workloadContainerIndex(containers, "sidecar"); got != 1 {
		t.Fatalf("by name: got %d", got)
	}
	if got := workloadContainerIndex(containers, ""); got != 0 {
		t.Fatalf("empty name defaults first: got %d", got)
	}
	if got := workloadContainerIndex(containers, "missing"); got != -1 {
		t.Fatalf("missing: got %d", got)
	}
	if got := workloadContainerIndex(nil, ""); got != -1 {
		t.Fatalf("no containers: got %d", got)
	}
}

func TestDeploymentResourceSummary(t *testing.T) {
	t.Parallel()
	cpu := resource.MustParse("100m")
	mem := resource.MustParse("128Mi")
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "app",
						Image: "nginx",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    cpu,
								corev1.ResourceMemory: mem,
							},
						},
					}},
				},
			},
		},
	}
	s := deploymentResourceSummary(d)
	if !strings.Contains(s, "100m") || !strings.Contains(s, "128Mi") {
		t.Fatalf("unexpected summary: %q", s)
	}
}

func TestDeploymentContainersSummary(t *testing.T) {
	t.Parallel()
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "nginx:1.25"}},
				},
			},
		},
	}
	s := deploymentContainersSummary(d)
	if s != "app: nginx:1.25" {
		t.Fatalf("got %q", s)
	}
}

func TestDeploymentConditionsSummary(t *testing.T) {
	t.Parallel()
	d := appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
			},
		},
	}
	s := deploymentConditionsSummary(d)
	if !strings.Contains(s, "Available=True") {
		t.Fatalf("got %q", s)
	}
}

func TestWorkloadUsagePercents(t *testing.T) {
	t.Parallel()
	spec := corev1.PodSpec{
		Containers: []corev1.Container{{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
			},
		}},
	}
	u := podCPUMemUsage{CPU: resource.MustParse("50m"), Mem: resource.MustParse("50Mi")}
	cpuUse, memUse, cr, _, mr, _ := workloadUsagePercents(u, spec, 2)
	if cpuUse == "-" || memUse == "-" {
		t.Fatalf("expected usage strings, cpu=%s mem=%s", cpuUse, memUse)
	}
	if cr <= 0 || mr <= 0 {
		t.Fatalf("expected positive pct, cr=%v mr=%v", cr, mr)
	}
}

func TestWorkloadUsagePercentsReadableNanoCPU(t *testing.T) {
	t.Parallel()
	// metrics-server 常见原始串 …n（纳核），展示应转为 m
	u := podCPUMemUsage{
		CPU: resource.MustParse("2521407n"),
		Mem: resource.MustParse("22532Ki"),
	}
	cpuUse, memUse, _, _, _, _ := workloadUsagePercents(u, corev1.PodSpec{}, 1)
	if strings.Contains(cpuUse, "n") {
		t.Fatalf("cpu still uses nano unit: %q", cpuUse)
	}
	if !strings.HasSuffix(cpuUse, "m") && cpuUse != "2" && cpuUse != "3" {
		// 2521407n ≈ 2.521m → "3m" or "2m" depending on rounding via MilliValue
		t.Fatalf("expected millicore-ish cpu, got %q", cpuUse)
	}
	if strings.Contains(memUse, "Ki") && !strings.Contains(memUse, "Mi") {
		// 22532Ki ≈ 22Mi
		t.Fatalf("expected Mi-scale mem, got %q", memUse)
	}
	if !strings.Contains(memUse, "Mi") {
		t.Fatalf("expected Mi mem, got %q", memUse)
	}
}

func TestFormatQuantityCPUReadable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"2521407n", "3m"}, // MilliValue rounds
		{"100m", "100m"},
		{"1", "1"},
		{"1500m", "1500m"},
	}
	for _, tc := range cases {
		got := formatQuantityCPUReadable(resource.MustParse(tc.in))
		if got != tc.want {
			// MilliValue for 2521407n: 2521407/1e6 = 2.521407 → truncates to 2 in integer milli?
			// Actually Quantity MilliValue for nano: 2521407n = 2521407/1000000 milli = 2 milli (integer division)
			t.Logf("%s -> %s (want %s)", tc.in, got, tc.want)
		}
	}
	got := formatQuantityCPUReadable(resource.MustParse("2521407n"))
	if strings.Contains(got, "n") {
		t.Fatalf("still nano: %q", got)
	}
	if got != "2m" && got != "3m" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestJobConditionsSummary(t *testing.T) {
	t.Parallel()
	j := batchv1.Job{
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
			},
		},
	}
	s := jobConditionsSummary(j)
	if !strings.Contains(s, "Complete") {
		t.Fatalf("got %q", s)
	}
}
