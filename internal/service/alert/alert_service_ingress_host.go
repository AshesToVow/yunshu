package alert

import (
	"context"
	"time"

	"yunshu/internal/alertdispatch"
	"yunshu/internal/model"
	"yunshu/internal/pkg/alertnotify"
)

type alertIngressHost struct{ s *AlertService }

func (s *AlertService) ingressHost() IngressHost {
	return alertIngressHost{s: s}
}

func (h alertIngressHost) LoadEnabledChannels(ctx context.Context) ([]model.AlertChannel, error) {
	return h.s.loadEnabledChannels(ctx)
}

func (h alertIngressHost) EnrichLabels(ctx context.Context, labels map[string]string, payloadReceiver, fingerprint string) map[string]string {
	return h.s.enrichCanonicalIngressLabels(ctx, labels, payloadReceiver, fingerprint)
}

func (h alertIngressHost) ComputeGroupKey(payloadReceiver, status, severity, eventName string, labels map[string]string) string {
	return h.s.computeGroupKey(payloadReceiver, status, severity, eventName, labels, alertnotify.ExtractDims(labels))
}

func (h alertIngressHost) LabelsDigestForGroupTiming(payloadReceiver, status, severity, eventName string, labels map[string]string) string {
	return h.s.labelsDigestForGroupTiming(payloadReceiver, status, severity, eventName, labels)
}

func (h alertIngressHost) ResolveEnvironmentLabel(labels map[string]string, payloadReceiver string, alertLabels map[string]string) string {
	return h.s.resolveAlertEnvironmentLabel(labels, payloadReceiver, alertnotify.ExtractDims(labels), alertLabels)
}

func (h alertIngressHost) FirstMatchingSilenceID(ctx context.Context, labels map[string]string, now time.Time) (uint, bool, error) {
	if h.s.maintenanceSvc != nil {
		if id, ok, err := h.s.maintenanceSvc.FirstMatchingID(ctx, labels, now); err != nil {
			return 0, false, err
		} else if ok {
			return id, true, nil
		}
	}
	if h.s.silenceSvc == nil {
		return 0, false, nil
	}
	return h.s.silenceSvc.FirstMatchingSilenceID(ctx, labels, now)
}

func (h alertIngressHost) LogSilenceSuppressed(ctx context.Context, title, severity, status, envLabel, groupKey, labelsDigest string, silenceID uint, payload map[string]interface{}) {
	h.s.logSilenceSuppressed(ctx, title, severity, status, envLabel, groupKey, labelsDigest, silenceID, payload)
}

func (h alertIngressHost) TouchFingerprint(ctx context.Context, fingerprint, status string) (int64, error) {
	return h.s.touchFingerprintState(ctx, fingerprint, status)
}

func (h alertIngressHost) CheckInhibition(ctx context.Context, labels map[string]string) (bool, *model.AlertInhibitionEvent) {
	return h.s.CheckInhibition(ctx, labels)
}

func (h alertIngressHost) RecordSourceInhibition(ctx context.Context, labels map[string]string) error {
	return h.s.RecordSourceInhibition(ctx, labels)
}

func (h alertIngressHost) ClearSourceInhibition(ctx context.Context, labels map[string]string) error {
	return h.s.ClearSourceInhibition(ctx, labels)
}

func (h alertIngressHost) LogInhibitionEvent(ctx context.Context, title string, severity string, status string, envLabel string, groupKey string, labelsDigest string, inhEvent *model.AlertInhibitionEvent, outgoing map[string]interface{}) {
	h.s.logInhibitionEvent(ctx, title, severity, status, envLabel, groupKey, labelsDigest, inhEvent, outgoing)
}

func (h alertIngressHost) ResolveMetricValues(ctx context.Context, source, status string, labels, annotations map[string]string, startsAt, endsAt time.Time, generatorURL string) (string, string) {
	alert := AlertManagerAlert{StartsAt: startsAt, EndsAt: endsAt, GeneratorURL: generatorURL}
	return h.s.resolveIngressMetricValues(ctx, source, status, labels, annotations, alert)
}

func (h alertIngressHost) EnqueuePrometheusEnrich(fingerprint, generatorURL string) {
	h.s.enqueuePrometheusEnrich(promEnrichTask{Fingerprint: fingerprint, GeneratorURL: generatorURL})
}

func (h alertIngressHost) EnrichOutgoingProjectName(ctx context.Context, outgoing map[string]interface{}) {
	h.s.enrichOutgoingProjectName(ctx, outgoing)
}

func (h alertIngressHost) EnrichAssigneeAndDutyEmails(ctx context.Context, outgoing map[string]interface{}, labels map[string]string) {
	h.s.enrichAssigneeAndDutyEmails(ctx, outgoing, labels)
}

func (h alertIngressHost) ClearResolvedNotificationSent(ctx context.Context, fingerprint string) error {
	return h.s.clearResolvedNotificationSent(ctx, fingerprint)
}

func (h alertIngressHost) PeekFiringGroupTiming(ctx context.Context, groupKey, labelsDigest string) (bool, string, int64, string, string) {
	return h.s.peekFiringGroupTiming(ctx, groupKey, labelsDigest)
}

func (h alertIngressHost) LogSuppressedFiringTiming(ctx context.Context, title, severity, status, groupKey, labelsDigest, reason string, outgoing map[string]interface{}) {
	h.s.logSuppressedFiringTiming(ctx, title, severity, status, groupKey, labelsDigest, reason, outgoing)
}

func (h alertIngressHost) ChannelRouteForAlert(ctx context.Context, status string, labels map[string]string) ChannelRoute {
	r := h.s.channelRouteForAlert(ctx, status, labels)
	return ChannelRoute{
		ChannelIDs:         r.ChannelIDs,
		MatchedPolicyIDs:   r.MatchedPolicyIDs,
		MatchedPolicyNames: r.MatchedPolicyNames,
		SilenceSeconds:     r.SilenceSeconds,
		ReceiverGroupIDs:   r.ReceiverGroupIDs,
	}
}

func (h alertIngressHost) ExpandChannelSetForAssigneeNotification(ctx context.Context, channels map[uint]struct{}, receiverGroupIDs []uint, outgoing map[string]interface{}) {
	h.s.expandChannelSetForAssigneeNotification(ctx, channels, receiverGroupIDs, outgoing)
}

func (h alertIngressHost) LogNoMatchedChannel(ctx context.Context, title, severity, status, envLabel, groupKey, labelsDigest string, outgoing map[string]interface{}, reason string) {
	h.s.logNoMatchedChannel(ctx, title, severity, status, envLabel, groupKey, labelsDigest, outgoing, reason)
}

func (h alertIngressHost) ShouldSuppressByRouteSilence(ctx context.Context, status, groupKey, matchedNodeIDs string, silenceSeconds int, labels map[string]string) bool {
	return h.s.shouldSuppressByRouteSilence(ctx, status, groupKey, matchedNodeIDs, silenceSeconds, labels)
}

func (h alertIngressHost) LogSuppressedRouteSilence(ctx context.Context, title, severity, status, envLabel, groupKey, labelsDigest string, silenceSeconds int, outgoing map[string]interface{}) {
	h.s.logSuppressedRouteSilence(ctx, title, severity, status, envLabel, groupKey, labelsDigest, silenceSeconds, outgoing)
}

func (h alertIngressHost) WasFiringDelivered(ctx context.Context, fingerprint string) bool {
	return h.s.alertFiringWasDelivered(ctx, fingerprint)
}

func (h alertIngressHost) LogResolvedSuppressedNoPriorFiringDelivery(ctx context.Context, title, severity, status, groupKey, labelsDigest string, outgoing map[string]interface{}) {
	h.s.logResolvedSuppressedNoPriorFiringDelivery(ctx, title, severity, status, groupKey, labelsDigest, outgoing)
}

func (h alertIngressHost) ClearFingerprintState(ctx context.Context, fingerprint string) error {
	return h.s.clearFingerprintState(ctx, fingerprint)
}

func (h alertIngressHost) ClearCurrentMetric(ctx context.Context, fingerprint string) error {
	if h.s.alertStateSvc != nil {
		return h.s.alertStateSvc.ClearCurrentMetric(ctx, fingerprint)
	}
	if h.s.redis != nil {
		return h.s.redis.Del(ctx, "alert:current:"+fingerprint).Err()
	}
	return nil
}

func (h alertIngressHost) ClearGroupAggregateState(ctx context.Context, groupKey string) error {
	return h.s.clearGroupAggregateState(ctx, groupKey)
}

func (h alertIngressHost) MarkResolvedNotificationSent(ctx context.Context, fingerprint string) (bool, error) {
	return h.s.markResolvedNotificationSent(ctx, fingerprint)
}

func (h alertIngressHost) LogSuppressedResolvedAggregate(ctx context.Context, title, severity, status, groupKey string, outgoing map[string]interface{}) {
	h.s.logSuppressedResolvedAggregate(ctx, title, severity, status, groupKey, outgoing)
}

func (h alertIngressHost) ChannelMatchesAlert(channel model.AlertChannel, labels map[string]string) bool {
	settings, _ := parseChannelSettings(channel.HeadersJSON)
	return channelMatchesAlert(settings, labels, alertnotify.ExtractDims(labels))
}

func (h alertIngressHost) SendToChannel(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, outgoing map[string]interface{}) (int, error) {
	code, _, err := h.s.sendToChannel(ctx, channel, alertdispatch.NewEnvelope(source, title, severity, status, outgoing))
	return code, err
}

func (h alertIngressHost) CommitFiringGroupTimingSend(ctx context.Context, groupKey, labelsDigest string) {
	h.s.commitFiringGroupTimingSend(ctx, groupKey, labelsDigest)
}

func (h alertIngressHost) MarkFiringDelivered(ctx context.Context, fingerprint string) {
	h.s.markAlertFiringDelivered(ctx, fingerprint)
}

func (h alertIngressHost) ClearResolvedSentMark(ctx context.Context, fingerprint string) error {
	return h.s.clearResolvedNotificationSent(ctx, fingerprint)
}

func (h alertIngressHost) LogAllChannelsDeliveryFailed(ctx context.Context, title, severity, status, envLabel, groupKey, labelsDigest string, outgoing map[string]interface{}) {
	h.s.logAllChannelsDeliveryFailed(ctx, title, severity, status, envLabel, groupKey, labelsDigest, outgoing)
}

func (h alertIngressHost) OnResolvedComplete(ctx context.Context, fingerprint, groupKey string) {
	_ = h.s.clearFingerprintState(ctx, fingerprint)
	_ = h.ClearCurrentMetric(ctx, fingerprint)
	_ = h.s.clearGroupAggregateState(ctx, groupKey)
	h.s.clearAlertFiringDelivered(ctx, fingerprint)
}

func (h alertIngressHost) UpsertCurAlert(ctx context.Context, row *model.AlertCurEvent) error {
	return h.s.UpsertCurEvent(ctx, row)
}

func (h alertIngressHost) ResolveCurAlert(ctx context.Context, fingerprint string, resolvedAt time.Time) error {
	return h.s.ResolveCurEvent(ctx, fingerprint, resolvedAt)
}
