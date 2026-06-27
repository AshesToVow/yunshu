package alert

import (
	"testing"

	"yunshu/internal/config"
)

func TestApplyIngressGroupTimingPolicy(t *testing.T) {
	t.Parallel()

	s := &AlertService{cfg: config.AlertConfig{WebhookSkipGroupTiming: true}}

	amItems := []CanonicalIngressAlert{{
		Source:          "alertmanager",
		PayloadReceiver: "team-a",
		Alert:           AlertManagerAlert{Fingerprint: "fp-am"},
	}}
	s.applyIngressGroupTimingPolicy(amItems)
	if !amItems[0].Alert.SkipGroupTiming {
		t.Fatal("external AM webhook should skip yunshu group timing by default")
	}

	platformItems := []CanonicalIngressAlert{{
		Source:          "platform_monitor",
		PayloadReceiver: "platform-monitor",
		Alert:           AlertManagerAlert{Fingerprint: "fp-pm"},
	}}
	s.applyIngressGroupTimingPolicy(platformItems)
	if platformItems[0].Alert.SkipGroupTiming {
		t.Fatal("platform-monitor must keep yunshu group timing")
	}

	cloudItems := []CanonicalIngressAlert{{
		Source:          "cloud_expiry",
		PayloadReceiver: "cloud-expiry",
		Alert:           AlertManagerAlert{Fingerprint: "fp-ce", SkipGroupTiming: true},
	}}
	s.applyIngressGroupTimingPolicy(cloudItems)
	if !cloudItems[0].Alert.SkipGroupTiming {
		t.Fatal("cloud-expiry explicit skip should remain")
	}

	sLegacy := &AlertService{cfg: config.AlertConfig{WebhookSkipGroupTiming: false}}
	legacyItems := []CanonicalIngressAlert{{
		Source:          "alertmanager",
		PayloadReceiver: "team-a",
		Alert:           AlertManagerAlert{Fingerprint: "fp-legacy"},
	}}
	sLegacy.applyIngressGroupTimingPolicy(legacyItems)
	if legacyItems[0].Alert.SkipGroupTiming {
		t.Fatal("webhook_skip_group_timing=false should keep second layer for AM")
	}
}
