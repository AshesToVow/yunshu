package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClassifyDrainPodSkipsDaemonSet(t *testing.T) {
	ctrl := true
	p := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ds-pod",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "DaemonSet", Controller: &ctrl},
			},
		},
	}
	item, skip := classifyDrainPod(p, true, true, false)
	if !skip || item.Reason != "DaemonSet" {
		t.Fatalf("expected DaemonSet skip, got skip=%v reason=%q", skip, item.Reason)
	}
}

func TestClassifyDrainPodBareNeedsForce(t *testing.T) {
	p := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "bare", Namespace: "default"},
	}
	_, skip := classifyDrainPod(p, true, true, false)
	if !skip {
		t.Fatal("bare pod should skip without force")
	}
	_, skip = classifyDrainPod(p, true, true, true)
	if skip {
		t.Fatal("bare pod should not skip with force")
	}
}

func TestClassifyDrainPodSkipsCompletedAndMirrorAnyNamespace(t *testing.T) {
	mirror := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "etcd-mirror",
			Namespace:   "custom-system",
			Annotations: map[string]string{"kubernetes.io/config.mirror": "abc"},
		},
	}
	if _, skip := classifyDrainPod(mirror, true, true, true); !skip {
		t.Fatal("mirror pod must be skipped regardless of namespace")
	}
	completed := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "job-done", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodSucceeded},
	}
	item, skip := classifyDrainPod(completed, true, true, true)
	if !skip || item.Reason != "Pod 已结束" {
		t.Fatalf("completed pod should be skipped, got skip=%v reason=%q", skip, item.Reason)
	}
	if drainPodOutstanding(completed, true, true, true) {
		t.Fatal("completed pod must not count as outstanding")
	}
}
