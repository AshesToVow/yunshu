package k8s

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeploymentTracks(t *testing.T) {
	t.Parallel()
	if deploymentTracks(nil, "canary") {
		t.Fatal("nil deployment must not track")
	}
	byLabel := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Labels: map[string]string{progressiveTrackLabel: "stable"},
	}}
	if !deploymentTracks(byLabel, "stable") || deploymentTracks(byLabel, "canary") {
		t.Fatal("label-based track detection failed")
	}
	bySelector := &appsv1.Deployment{Spec: appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{progressiveTrackLabel: "canary"}},
	}}
	if !deploymentTracks(bySelector, "canary") || deploymentTracks(bySelector, "stable") {
		t.Fatal("selector-based track detection failed")
	}
	if deploymentLabeled(byLabel, progressiveTrackLabel, "") {
		t.Fatal("empty value must not match")
	}
}
