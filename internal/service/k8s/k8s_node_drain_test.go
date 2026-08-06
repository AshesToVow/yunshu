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
