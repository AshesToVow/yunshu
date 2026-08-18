package eventforward

import (
	"context"
	"strings"
	"testing"

	"yunshu/internal/model"
)

func TestEventSeverity(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "Warning", want: "warning"},
		{in: "Normal", want: "info"},
		{in: "Error", want: "critical"},
		{in: "Critical", want: "critical"},
		{in: "SomethingElse", want: "warning"},
	}
	for _, tc := range cases {
		if got := eventSeverity(tc.in); got != tc.want {
			t.Fatalf("eventSeverity(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestBuildAlertManagerPayload_ProjectID(t *testing.T) {
	p := buildAlertManagerPayload("r1", "1", "local", 42, []model.K8sForwardedEvent{
		{EvtKey: "uid-1", Type: "Warning", Reason: "Unhealthy", Namespace: "kube-system", Name: "Pod/x", Message: "probe failed"},
	})
	if len(p.Alerts) != 1 {
		t.Fatalf("alerts len %d", len(p.Alerts))
	}
	if p.Alerts[0].Labels["project_id"] != "42" {
		t.Fatalf("project_id=%q", p.Alerts[0].Labels["project_id"])
	}
	if p.Alerts[0].Labels["severity"] != "warning" {
		t.Fatalf("severity=%q", p.Alerts[0].Labels["severity"])
	}
	if p.Alerts[0].Labels["source"] != "k8s_event" {
		t.Fatalf("source=%q", p.Alerts[0].Labels["source"])
	}
}

func TestBuildAlertManagerPayload_FingerprintIgnoresMessage(t *testing.T) {
	base := model.K8sForwardedEvent{
		EvtKey: "uid-stable", Type: "Warning", Reason: "Unhealthy",
		Namespace: "ns", Name: "pod-a", Message: "first",
	}
	p1 := buildAlertManagerPayload("r1", "1", "c", 0, []model.K8sForwardedEvent{base})
	base.Message = "second message changed"
	p2 := buildAlertManagerPayload("r1", "1", "c", 0, []model.K8sForwardedEvent{base})
	if p1.Alerts[0].Fingerprint == "" || p1.Alerts[0].Fingerprint != p2.Alerts[0].Fingerprint {
		t.Fatalf("fingerprint should be stable across message updates: %q vs %q",
			p1.Alerts[0].Fingerprint, p2.Alerts[0].Fingerprint)
	}
}

func TestWatcherEnqueueTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := &Watcher{
		ctx:     ctx,
		eventCh: make(chan *model.K8sForwardedEvent), // unbuffered + no consumer => timeout
	}
	err := w.enqueue(&model.K8sForwardedEvent{EvtKey: "x"})
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

