package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestEphemeralContainerReady(t *testing.T) {
	t.Parallel()
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			EphemeralContainerStatuses: []corev1.ContainerStatus{
				{Name: "yunshu-debug-1", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
				{Name: "other", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{}}},
			},
		},
	}
	if !ephemeralContainerReady(pod, "yunshu-debug-1") {
		t.Fatal("expected running ephemeral ready")
	}
	if ephemeralContainerReady(pod, "other") {
		t.Fatal("waiting ephemeral should not be ready")
	}
	if ephemeralContainerReady(nil, "x") {
		t.Fatal("nil pod")
	}
}

func TestRequiredK8sCapabilityPodDebug(t *testing.T) {
	t.Parallel()
	if RequiredK8sCapability(nil, "/api/v1/pods/debug", "POST", "") != CapExec {
		t.Fatal("pods/debug should require CapExec")
	}
}
