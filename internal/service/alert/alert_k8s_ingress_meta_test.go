package alert

import (
	"context"
	"testing"

	"yunshu/internal/config"
)

func TestResolveAlertDatasourceMeta_K8sEvents(t *testing.T) {
	t.Parallel()
	s := &AlertService{}
	id, name, typ, slug := s.resolveAlertDatasourceMeta(context.Background(), map[string]string{
		"source": "k8s_event",
	}, "k8s-events")
	if id != 0 || name != "K8s Event 转发" || typ != "k8s_event" || slug != "k8s_event" {
		t.Fatalf("got id=%d name=%q typ=%q slug=%q", id, name, typ, slug)
	}
}

func TestValidateK8sEventIngressToken(t *testing.T) {
	t.Parallel()
	s := &AlertService{cfg: config.AlertConfig{WebhookToken: ""}}
	if s.ValidateK8sEventIngressToken("", "127.0.0.1") != true {
		t.Fatal("empty token from loopback should pass when webhook_token unset")
	}
	if s.ValidateK8sEventIngressToken("", "10.0.0.1") {
		t.Fatal("empty token from non-loopback should fail")
	}
	s.cfg.WebhookToken = "secret"
	if !s.ValidateK8sEventIngressToken("secret", "10.0.0.1") {
		t.Fatal("matching token should pass")
	}
	if s.ValidateK8sEventIngressToken("", "127.0.0.1") {
		t.Fatal("empty token should fail when webhook_token configured")
	}
}
