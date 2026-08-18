package alert

import (
	"context"
	"sync"
	"testing"
	"time"

	"yunshu/internal/model"
)

// pipelineTestHost is a minimal IngressHost for end-to-end pipeline tests (no DB).
type pipelineTestHost struct {
	mu sync.Mutex

	channels []model.AlertChannel
	route    ChannelRoute

	touchCount      int64
	firingDelivered bool
	resolvedSent    bool
	sends           []string
	deliverCode     int

	skipGroupTiming bool
	peekShouldSend  bool
}

func (h *pipelineTestHost) LoadEnabledChannels(context.Context) ([]model.AlertChannel, error) {
	return h.channels, nil
}

func (h *pipelineTestHost) EnrichLabels(_ context.Context, labels map[string]string, _, _ string) map[string]string {
	return labels
}

func (h *pipelineTestHost) ComputeGroupKey(_, _, _, _ string, _ map[string]string) string {
	return "gk-test"
}

func (h *pipelineTestHost) LabelsDigestForGroupTiming(_, _, _, _ string, _ map[string]string) string {
	return "digest-test"
}

func (h *pipelineTestHost) ResolveEnvironmentLabel(_ map[string]string, _ string, _ map[string]string) string {
	return "test"
}

func (h *pipelineTestHost) FirstMatchingSilenceID(context.Context, map[string]string, time.Time) (uint, bool, error) {
	return 0, false, nil
}

func (h *pipelineTestHost) LogSilenceSuppressed(context.Context, string, string, string, string, string, string, uint, map[string]interface{}) {
}

func (h *pipelineTestHost) TouchFingerprint(_ context.Context, _, _ string) (int64, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.touchCount++
	return h.touchCount, nil
}

func (h *pipelineTestHost) CheckInhibition(context.Context, map[string]string) (bool, *model.AlertInhibitionEvent) {
	return false, nil
}

func (h *pipelineTestHost) RecordSourceInhibition(context.Context, map[string]string) error { return nil }

func (h *pipelineTestHost) ClearSourceInhibition(context.Context, map[string]string) error { return nil }

func (h *pipelineTestHost) LogInhibitionEvent(context.Context, string, string, string, string, string, string, *model.AlertInhibitionEvent, map[string]interface{}) {
}

func (h *pipelineTestHost) ResolveMetricValues(context.Context, string, string, map[string]string, map[string]string, time.Time, time.Time, string) (string, string) {
	return "", ""
}

func (h *pipelineTestHost) EnqueuePrometheusEnrich(string, string) {}

func (h *pipelineTestHost) EnrichOutgoingProjectName(context.Context, map[string]interface{}) {}

func (h *pipelineTestHost) EnrichAssigneeAndDutyEmails(context.Context, map[string]interface{}, map[string]string) {
}

func (h *pipelineTestHost) IsAckActive(context.Context, string) bool { return false }

func (h *pipelineTestHost) ClearResolvedNotificationSent(context.Context, string) error { return nil }

func (h *pipelineTestHost) PeekFiringGroupTiming(context.Context, string, string) (bool, string, int64, string, string) {
	if h.skipGroupTiming {
		return true, "skip", 1, "", ""
	}
	if h.peekShouldSend {
		return true, "ok", 1, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339)
	}
	return false, "suppressed_timing", 1, "", ""
}

func (h *pipelineTestHost) LogSuppressedFiringTiming(context.Context, string, string, string, string, string, string, map[string]interface{}) {
}

func (h *pipelineTestHost) ChannelRouteForAlert(context.Context, string, map[string]string) ChannelRoute {
	return h.route
}

func (h *pipelineTestHost) ExpandChannelSetForAssigneeNotification(context.Context, map[uint]struct{}, []uint, map[string]interface{}) {
}

func (h *pipelineTestHost) LogNoMatchedChannel(context.Context, string, string, string, string, string, string, map[string]interface{}, string) {
}

func (h *pipelineTestHost) ShouldSuppressByRouteSilence(context.Context, string, string, string, int, map[string]string) bool {
	return false
}

func (h *pipelineTestHost) LogSuppressedRouteSilence(context.Context, string, string, string, string, string, string, int, map[string]interface{}) {
}

func (h *pipelineTestHost) WasFiringDelivered(_ context.Context, _ string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.firingDelivered
}

func (h *pipelineTestHost) LogResolvedSuppressedNoPriorFiringDelivery(context.Context, string, string, string, string, string, map[string]interface{}) {
}

func (h *pipelineTestHost) ClearFingerprintState(context.Context, string) error { return nil }

func (h *pipelineTestHost) ClearCurrentMetric(context.Context, string) error { return nil }

func (h *pipelineTestHost) ClearGroupAggregateState(context.Context, string) error { return nil }

func (h *pipelineTestHost) MarkResolvedNotificationSent(_ context.Context, _ string) (bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.resolvedSent {
		return false, nil
	}
	h.resolvedSent = true
	return true, nil
}

func (h *pipelineTestHost) LogSuppressedResolvedAggregate(context.Context, string, string, string, string, map[string]interface{}) {
}

func (h *pipelineTestHost) ChannelMatchesAlert(_ model.AlertChannel, _ map[string]string) bool {
	return true
}

func (h *pipelineTestHost) SendToChannel(_ context.Context, ch *model.AlertChannel, _, _, _, _ string, _ map[string]interface{}) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sends = append(h.sends, ch.Name+"|"+ch.Type)
	code := h.deliverCode
	if code == 0 {
		code = 200
	}
	return code, nil
}

func (h *pipelineTestHost) CommitFiringGroupTimingSend(context.Context, string, string) {}

func (h *pipelineTestHost) TryLockGroupSend(context.Context, string) bool { return true }

func (h *pipelineTestHost) UnlockGroupSend(context.Context, string) {}

func (h *pipelineTestHost) SaveGroupWaitPending(context.Context, groupWaitPendingEnvelope, string) {}

func (h *pipelineTestHost) ClearGroupWaitPending(context.Context, string) {}

func (h *pipelineTestHost) MarkFiringDelivered(_ context.Context, _ string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.firingDelivered = true
}

func (h *pipelineTestHost) ClearResolvedSentMark(context.Context, string) error { return nil }

func (h *pipelineTestHost) LogAllChannelsDeliveryFailed(context.Context, string, string, string, string, string, string, map[string]interface{}) {
}

func (h *pipelineTestHost) OnResolvedComplete(context.Context, string, string) {}

func (h *pipelineTestHost) UpsertCurAlert(context.Context, *model.AlertCurEvent) error { return nil }

func (h *pipelineTestHost) ResolveCurAlert(context.Context, string, time.Time) error { return nil }

func testChannel() model.AlertChannel {
	return model.AlertChannel{
		ID:      1,
		Name:    "test-ch",
		Type:    "webhook",
		Enabled: true,
	}
}

func testFiringItem(fp string) CanonicalIngressAlert {
	now := time.Now()
	return CanonicalIngressAlert{
		Source:          IngressSourceAlertmanager,
		PayloadReceiver: "yunshu",
		PayloadStatus:   "firing",
		Alert: IngressAlertDetail{
			Status:          "firing",
			Fingerprint:     fp,
			StartsAt:        now,
			EndsAt:          now.Add(time.Hour),
			Labels:          map[string]string{"alertname": "HighCPU", "severity": "warning"},
			Annotations:     map[string]string{"summary": "cpu high"},
			SkipGroupTiming: true,
		},
	}
}

func testResolvedItem(fp string) CanonicalIngressAlert {
	now := time.Now()
	return CanonicalIngressAlert{
		Source:          IngressSourceAlertmanager,
		PayloadReceiver: "yunshu",
		PayloadStatus:   "resolved",
		Alert: IngressAlertDetail{
			Status:      "resolved",
			Fingerprint: fp,
			StartsAt:    now.Add(-time.Minute),
			EndsAt:      now,
			Labels:      map[string]string{"alertname": "HighCPU", "severity": "warning"},
			Annotations: map[string]string{"summary": "cpu high"},
		},
	}
}

func TestRunIngressPipeline_FiringDeliverThenResolved(t *testing.T) {
	fp := "fp-e2e-pipeline"
	host := &pipelineTestHost{
		channels:        []model.AlertChannel{testChannel()},
		route:           ChannelRoute{ChannelIDs: map[uint]struct{}{1: {}}},
		skipGroupTiming: true,
		peekShouldSend:  true,
		deliverCode:     200,
	}
	ctx := context.Background()

	if err := RunIngressPipeline(ctx, host, []CanonicalIngressAlert{testFiringItem(fp)}); err != nil {
		t.Fatalf("firing pipeline: %v", err)
	}
	host.mu.Lock()
	if !host.firingDelivered {
		t.Fatal("expected MarkFiringDelivered after successful firing delivery")
	}
	if len(host.sends) != 1 {
		t.Fatalf("expected 1 firing send, got %d: %v", len(host.sends), host.sends)
	}
	host.mu.Unlock()

	if err := RunIngressPipeline(ctx, host, []CanonicalIngressAlert{testResolvedItem(fp)}); err != nil {
		t.Fatalf("resolved pipeline: %v", err)
	}
	host.mu.Lock()
	defer host.mu.Unlock()
	if len(host.sends) != 2 {
		t.Fatalf("expected 2 sends (firing+resolved), got %d: %v", len(host.sends), host.sends)
	}
}

func TestRunIngressPipeline_ResolvedWithoutPriorFiringSkipped(t *testing.T) {
	fp := "fp-no-prior-firing"
	host := &pipelineTestHost{
		channels:       []model.AlertChannel{testChannel()},
		route:          ChannelRoute{ChannelIDs: map[uint]struct{}{1: {}}},
		peekShouldSend: true,
		deliverCode:    200,
	}
	if err := RunIngressPipeline(context.Background(), host, []CanonicalIngressAlert{testResolvedItem(fp)}); err != nil {
		t.Fatal(err)
	}
	host.mu.Lock()
	defer host.mu.Unlock()
	if len(host.sends) != 0 {
		t.Fatalf("resolved without prior firing should not deliver, sends=%v", host.sends)
	}
}
