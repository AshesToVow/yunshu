package alert

import (
	"testing"
	"time"
)

func TestCanonicalAlertsFromAlertmanagerPayload(t *testing.T) {
	t.Parallel()
	p := AlertManagerPayload{
		Receiver: "platform-monitor",
		Status:   "firing",
		Alerts: []AlertManagerAlert{
			{
				Status:      "firing",
				Labels:      map[string]string{"alertname": "A"},
				Fingerprint: "fp1",
				StartsAt:    time.Now(),
			},
			{
				Status:      "firing",
				Labels:      map[string]string{"alertname": "B"},
				Fingerprint: "fp2",
			},
		},
	}
	items := CanonicalAlertsFromAlertmanagerPayload(p)
	if len(items) != 2 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0].Source != IngressSourcePlatformMonitor {
		t.Fatalf("source=%q", items[0].Source)
	}
	if items[0].Alert.Fingerprint != "fp1" {
		t.Fatalf("fp=%q", items[0].Alert.Fingerprint)
	}
}

func TestCanonicalAlertsCloudExpirySource(t *testing.T) {
	t.Parallel()
	p := AlertManagerPayload{Receiver: "cloud-expiry", Alerts: []AlertManagerAlert{{}}}
	items := CanonicalAlertsFromAlertmanagerPayload(p)
	if len(items) != 1 || items[0].Source != IngressSourceCloudExpiry {
		t.Fatalf("got %+v", items)
	}
}

func TestCanonicalAlertsDefaultSource(t *testing.T) {
	t.Parallel()
	p := AlertManagerPayload{Receiver: "team-a", Alerts: []AlertManagerAlert{{}}}
	items := CanonicalAlertsFromAlertmanagerPayload(p)
	if items[0].Source != IngressSourceAlertmanager {
		t.Fatalf("got %q", items[0].Source)
	}
}

func TestNewCanonicalAlertPlatformBuilder(t *testing.T) {
	t.Parallel()
	now := time.Now()
	item := NewCanonicalAlert(
		IngressSourcePlatformMonitor,
		"platform-monitor",
		"firing",
		map[string]string{"alertname": "CPU"},
		map[string]string{"alertname": "CPU", "severity": "warning"},
		IngressAlertDetail{
			Status:      "firing",
			Fingerprint: "fp-pm",
			StartsAt:    now,
			Labels:      map[string]string{"alertname": "CPU"},
		},
	)
	if item.Source != IngressSourcePlatformMonitor {
		t.Fatalf("source=%q", item.Source)
	}
	if item.Alert.Fingerprint != "fp-pm" {
		t.Fatalf("fp=%q", item.Alert.Fingerprint)
	}
}

func TestBuildOutgoingPayloadIncludesCloudExtension(t *testing.T) {
	t.Parallel()
	now := time.Now()
	ca := CanonicalIngressAlert{
		Source:          IngressSourceCloudExpiry,
		PayloadReceiver: "cloud-expiry",
		Cloud: &CloudExpiryExtension{
			Provider:   "aliyun",
			AccountID:  9,
			InstanceID: "i-1",
			Region:     "cn-hangzhou",
			ExpiresAt:  now.Add(48 * time.Hour),
			DaysLeft:   2,
			ProjectID:  3,
		},
		Alert: IngressAlertDetail{Fingerprint: "fp-cloud", StartsAt: now, EndsAt: now},
	}
	out := buildOutgoingPayload(ca, ca.Alert, "t", "s", "warning", "firing",
		map[string]string{}, map[string]string{}, "prod", "cloud_expiry",
		0, "", "", "gk", "digest", 1)
	cloud, ok := out["cloud"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing cloud extension: %#v", out)
	}
	if cloud["provider"] != "aliyun" || cloud["instance_id"] != "i-1" || cloud["days_left"] != 2 {
		t.Fatalf("cloud=%#v", cloud)
	}
}
