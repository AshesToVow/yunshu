package alert

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// ChannelRoute is the subscription routing outcome for one alert.
type ChannelRoute struct {
	ChannelIDs          map[uint]struct{}
	MatchedPolicyIDs    string
	MatchedPolicyNames  string
	SilenceSeconds      int
	ReceiverGroupIDs    []uint
	ReceiverGroupEmails []string
	EscalationLevel     int
}

// IngressHost implements delivery orchestration hooks (AlertService in parent package).
type IngressHost interface {
	LoadEnabledChannels(ctx context.Context) ([]model.AlertChannel, error)
	EnrichLabels(ctx context.Context, labels map[string]string, payloadReceiver, fingerprint string) map[string]string
	ComputeGroupKey(payloadReceiver, status, severity, eventName string, labels map[string]string) string
	LabelsDigestForGroupTiming(payloadReceiver, status, severity, eventName string, labels map[string]string) string
	ResolveEnvironmentLabel(labels map[string]string, payloadReceiver string, alertLabels map[string]string) string
	FirstMatchingSilenceID(ctx context.Context, labels map[string]string, now time.Time) (uint, bool, error)
	LogSilenceSuppressed(ctx context.Context, title string, severity string, status string, envLabel string, groupKey string, labelsDigest string, silenceID uint, payload map[string]interface{})
	TouchFingerprint(ctx context.Context, fingerprint, status string) (int64, error)
	CheckInhibition(ctx context.Context, labels map[string]string) (bool, *model.AlertInhibitionEvent)
	RecordSourceInhibition(ctx context.Context, labels map[string]string) error
	ClearSourceInhibition(ctx context.Context, labels map[string]string) error
	LogInhibitionEvent(ctx context.Context, title string, severity string, status string, envLabel string, groupKey string, labelsDigest string, inhEvent *model.AlertInhibitionEvent, outgoing map[string]interface{})
	ResolveMetricValues(ctx context.Context, source, status string, labels, annotations map[string]string, startsAt, endsAt time.Time, generatorURL string) (string, string)
	EnqueuePrometheusEnrich(fingerprint, generatorURL string)
	EnrichOutgoingProjectName(ctx context.Context, outgoing map[string]interface{})
	EnrichAssigneeAndDutyEmails(ctx context.Context, outgoing map[string]interface{}, labels map[string]string)
	ClearResolvedNotificationSent(ctx context.Context, fingerprint string) error
	IsAckActive(ctx context.Context, fingerprint string) bool
	PeekFiringGroupTiming(ctx context.Context, groupKey, labelsDigest string) (bool, string, int64, string, string)
	LogSuppressedFiringTiming(ctx context.Context, title, severity, status, groupKey, labelsDigest, reason string, outgoing map[string]interface{})
	ChannelRouteForAlert(ctx context.Context, status string, labels map[string]string, fingerprint string) ChannelRoute
	ExpandChannelSetForAssigneeNotification(ctx context.Context, channels map[uint]struct{}, receiverGroupIDs []uint, outgoing map[string]interface{})
	MaybeScheduleEscalation(ctx context.Context, env escalationPendingEnvelope, currentLevel int)
	ClearEscalationState(ctx context.Context, fingerprint string)
	LogNoMatchedChannel(ctx context.Context, title, severity, status, envLabel, groupKey, labelsDigest string, outgoing map[string]interface{}, reason string)
	ShouldSuppressByRouteSilence(ctx context.Context, status, groupKey, matchedNodeIDs string, silenceSeconds int, labels map[string]string) bool
	LogSuppressedRouteSilence(ctx context.Context, title, severity, status, envLabel, groupKey, labelsDigest string, silenceSeconds int, outgoing map[string]interface{})
	WasFiringDelivered(ctx context.Context, fingerprint string) bool
	LogResolvedSuppressedNoPriorFiringDelivery(ctx context.Context, title, severity, status, groupKey, labelsDigest string, outgoing map[string]interface{})
	ClearFingerprintState(ctx context.Context, fingerprint string) error
	ClearCurrentMetric(ctx context.Context, fingerprint string) error
	ClearGroupAggregateState(ctx context.Context, groupKey string) error
	MarkResolvedNotificationSent(ctx context.Context, fingerprint string) (bool, error)
	LogSuppressedResolvedAggregate(ctx context.Context, title, severity, status, groupKey string, outgoing map[string]interface{})
	ChannelMatchesAlert(channel model.AlertChannel, labels map[string]string) bool
	SendToChannel(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, outgoing map[string]interface{}) (int, error)
	CommitFiringGroupTimingSend(ctx context.Context, groupKey, labelsDigest string)
	MarkFiringDelivered(ctx context.Context, fingerprint string)
	TryLockGroupSend(ctx context.Context, groupKey string) bool
	UnlockGroupSend(ctx context.Context, groupKey string)
	SaveGroupWaitPending(ctx context.Context, env groupWaitPendingEnvelope, firstSeen string)
	ClearGroupWaitPending(ctx context.Context, groupKey string)
	ClearResolvedSentMark(ctx context.Context, fingerprint string) error
	LogAllChannelsDeliveryFailed(ctx context.Context, title, severity, status, envLabel, groupKey, labelsDigest string, outgoing map[string]interface{})
	OnResolvedComplete(ctx context.Context, fingerprint, groupKey string)
	// UpsertCurAlert / ResolveCurAlert：当前告警生命周期（屏蔽后不调用 Upsert）
	UpsertCurAlert(ctx context.Context, row *model.AlertCurEvent) error
	ResolveCurAlert(ctx context.Context, fingerprint string, resolvedAt time.Time) error
}
