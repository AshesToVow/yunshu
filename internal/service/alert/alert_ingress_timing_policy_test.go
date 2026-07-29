package alert

import (
	"testing"

	"yunshu/internal/config"
)

func TestApplyIngressGroupTimingPolicy(t *testing.T) {
	t.Parallel()

	s := &AlertService{cfg: config.AlertConfig{WebhookSkipGroupTiming: true}}

	amItems := []CanonicalIngressAlert{{
		Source:          IngressSourceAlertmanager,
		PayloadReceiver: "team-a",
		Alert:           IngressAlertDetail{Fingerprint: "fp-am"},
	}}
	s.applyIngressGroupTimingPolicy(amItems)
	if !amItems[0].Alert.SkipGroupTiming {
		t.Fatal("external AM webhook should skip yunshu group timing by default")
	}

	platformItems := []CanonicalIngressAlert{{
		Source:          IngressSourcePlatformMonitor,
		PayloadReceiver: "platform-monitor",
		Alert:           IngressAlertDetail{Fingerprint: "fp-pm"},
	}}
	s.applyIngressGroupTimingPolicy(platformItems)
	if platformItems[0].Alert.SkipGroupTiming {
		t.Fatal("platform-monitor must keep yunshu group timing")
	}

	cloudItems := []CanonicalIngressAlert{{
		Source:          IngressSourceCloudExpiry,
		PayloadReceiver: "cloud-expiry",
		Alert:           IngressAlertDetail{Fingerprint: "fp-ce", SkipGroupTiming: true},
	}}
	s.applyIngressGroupTimingPolicy(cloudItems)
	if !cloudItems[0].Alert.SkipGroupTiming {
		t.Fatal("cloud-expiry explicit skip should remain")
	}

	sLegacy := &AlertService{cfg: config.AlertConfig{WebhookSkipGroupTiming: false}}
	legacyItems := []CanonicalIngressAlert{{
		Source:          IngressSourceAlertmanager,
		PayloadReceiver: "team-a",
		Alert:           IngressAlertDetail{Fingerprint: "fp-legacy"},
	}}
	sLegacy.applyIngressGroupTimingPolicy(legacyItems)
	if legacyItems[0].Alert.SkipGroupTiming {
		t.Fatal("webhook_skip_group_timing=false should keep second layer for AM")
	}
}
