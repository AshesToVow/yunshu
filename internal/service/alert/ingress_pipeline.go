package alert

import (
	"context"
	"maps"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	bizerrors "yunshu/internal/pkg/errors"
)

// IngressAlertDetail is one alert instance inside a canonical ingress item.
type IngressAlertDetail struct {
	Status          string
	Labels          map[string]string
	Annotations     map[string]string
	StartsAt        time.Time
	EndsAt          time.Time
	GeneratorURL    string
	Fingerprint     string
	SkipGroupTiming bool
}

// RunIngressPipeline processes canonical ingress items (ingest → routing → delivery → state).
func RunIngressPipeline(ctx context.Context, host IngressHost, items []CanonicalIngressAlert) error {
	channels, err := host.LoadEnabledChannels(ctx)
	if err != nil {
		return bizerrors.Pass(ctx, "alert.ingest", "RunIngressPipeline", err)
	}
	for _, ca := range items {
		alert := ca.Alert
		labels := mergeStringMaps(ca.CommonLabels, alert.Labels)
		fp := strings.TrimSpace(alert.Fingerprint)
		if fp == "" {
			fp = stableFingerprint(labels)
		}
		labels = host.EnrichLabels(ctx, labels, ca.PayloadReceiver, fp)
		if fp != "" {
			alert.Fingerprint = fp
		}
		var dsID uint
		if v := strings.TrimSpace(labels["datasource_id"]); v != "" {
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				dsID = uint(n)
			}
		}
		dsName := strings.TrimSpace(labels["datasource_name"])
		dsType := strings.TrimSpace(labels["datasource_type"])
		monitorPipeline := strings.TrimSpace(labels["monitor_pipeline"])
		annotations := mergeStringMaps(ca.CommonAnnotations, alert.Annotations)
		status := normalizeIngressStatus(alert.Status, ca.PayloadStatus)
		eventName := pickAlertName(labels, ca.CommonLabels)
		summary := pickSummary(annotations, ca.CommonAnnotations)
		severity := pickSeverity(labels, ca.CommonLabels)
		title := eventName

		groupKey := host.ComputeGroupKey(ca.PayloadReceiver, status, severity, eventName, labels)
		labelsDigest := host.LabelsDigestForGroupTiming(ca.PayloadReceiver, status, severity, eventName, labels)
		envLabel := host.ResolveEnvironmentLabel(labels, ca.PayloadReceiver, alert.Labels)

		if sid, muted, err := host.FirstMatchingSilenceID(ctx, labels, time.Now()); err == nil && muted {
			minPayload := map[string]any{
				"labels": labels, "annotations": annotations, "severity": severity, "status": status,
				"receiver": ca.PayloadReceiver, "fingerprint": alert.Fingerprint,
				"groupKey": groupKey, "cluster": envLabel, "labelsDigest": labelsDigest,
				"monitorPipeline": monitorPipeline,
				"datasourceId":    dsID, "datasourceName": dsName, "datasourceType": dsType,
				"source": ca.Source,
			}
			host.LogSilenceSuppressed(ctx, title, severity, status, envLabel, groupKey, labelsDigest, sid, minPayload)
			continue
		}

		// 当前告警：屏蔽之后落库；resolved 迁入历史。与投递流水 alert_events 分离。
		// K8s Event 为瞬时事件流，无 resolved，不进入 alert_cur_events。
		if status == "firing" && ca.Source != IngressSourceK8sEvent {
			value := strings.TrimSpace(annotations["value"])
			_ = host.UpsertCurAlert(ctx, buildCurEventFromIngress(
				ca.Source, ca.PayloadReceiver, title, severity, status, envLabel, groupKey,
				dsID, labels, annotations, alert.Fingerprint, alert.StartsAt, value,
			))
		} else if status == "resolved" {
			_ = host.ResolveCurAlert(ctx, alert.Fingerprint, alert.EndsAt)
		}

		count, err := host.TouchFingerprint(ctx, alert.Fingerprint, status)
		if err != nil {
			return err
		}

		outgoing := buildOutgoingPayload(ca, alert, title, summary, severity, status, labels, annotations, envLabel,
			monitorPipeline, dsID, dsName, dsType, groupKey, labelsDigest, count)

		if status == "firing" {
			if inhibited, inhEvent := host.CheckInhibition(ctx, labels); inhibited {
				host.LogInhibitionEvent(ctx, title, severity, status, envLabel, groupKey, labelsDigest, inhEvent, outgoing)
				_ = host.RecordSourceInhibition(ctx, labels)
				continue
			}
			_ = host.RecordSourceInhibition(ctx, labels)
		}
		if status == "resolved" {
			_ = host.ClearSourceInhibition(ctx, labels)
		}

		currentValue, resolvedValue := host.ResolveMetricValues(ctx, ca.Source, status, labels, annotations, alert.StartsAt, alert.EndsAt, alert.GeneratorURL)
		if currentValue != "" {
			outgoing["current"] = currentValue
		}
		if resolvedValue != "" {
			outgoing["current_resolved"] = resolvedValue
		}
		if status == "firing" {
			host.EnqueuePrometheusEnrich(alert.Fingerprint, alert.GeneratorURL)
		}
		host.EnrichOutgoingProjectName(ctx, outgoing)
		host.EnrichAssigneeAndDutyEmails(ctx, outgoing, labels)

		if status == "firing" && host.IsAckActive(ctx, alert.Fingerprint) {
			outgoing["suppressed_reason"] = "ack_active"
			host.LogSuppressedFiringTiming(ctx, title, severity, status, groupKey, labelsDigest, "ack_active", outgoing)
			continue
		}

		if status == "firing" {
			_ = host.ClearResolvedNotificationSent(ctx, alert.Fingerprint)
			var shouldSend bool
			var reason string
			var aggCount int64
			var firstSeen, lastSeen string
			if alert.SkipGroupTiming {
				shouldSend, reason, aggCount, firstSeen, lastSeen = true, "skip_group_timing_immediate", 1, "", ""
			} else {
				shouldSend, reason, aggCount, firstSeen, lastSeen = host.PeekFiringGroupTiming(ctx, groupKey, labelsDigest)
			}
			outgoing["agg_count"] = aggCount
			outgoing["agg_first_seen"] = firstSeen
			outgoing["agg_last_seen"] = lastSeen
			if !shouldSend {
				outgoing["suppressed_reason"] = reason
				host.LogSuppressedFiringTiming(ctx, title, severity, status, groupKey, labelsDigest, reason, outgoing)
				if reason == "group_wait_suppressed" {
					host.SaveGroupWaitPending(ctx, groupWaitPendingEnvelope{
						GroupKey:     groupKey,
						Fingerprint:  alert.Fingerprint,
						LabelsDigest: labelsDigest,
						Source:       ca.Source,
						Title:        title,
						Severity:     severity,
						Status:       status,
						EnvLabel:     envLabel,
						FirstSeen:    firstSeen,
						Labels:       labels,
						Outgoing:     outgoing,
					}, firstSeen)
				}
				continue
			}
		}

		route := host.ChannelRouteForAlert(ctx, status, labels, alert.Fingerprint)
		host.ExpandChannelSetForAssigneeNotification(ctx, route.ChannelIDs, route.ReceiverGroupIDs, outgoing)
		outgoing["matchedPolicyIds"] = route.MatchedPolicyIDs
		outgoing["matchedPolicyNames"] = route.MatchedPolicyNames
		outgoing["subscription_silence_seconds"] = route.SilenceSeconds
		outgoing["escalation_level"] = route.EscalationLevel
		if len(route.ReceiverGroupEmails) > 0 {
			outgoing["receiver_group_emails"] = route.ReceiverGroupEmails
		}

		if len(route.ChannelIDs) == 0 {
			host.LogNoMatchedChannel(ctx, title, severity, status, envLabel, groupKey, labelsDigest, outgoing, "no_policy_matched")
			if status == "firing" {
				host.MaybeScheduleEscalation(ctx, escalationPendingEnvelope{
					Fingerprint:  alert.Fingerprint,
					GroupKey:     groupKey,
					LabelsDigest: labelsDigest,
					Source:       ca.Source,
					Title:        title,
					Severity:     severity,
					Status:       status,
					EnvLabel:     envLabel,
					Labels:       labels,
					Outgoing:     outgoing,
				}, route.EscalationLevel)
			}
			continue
		}
		if host.ShouldSuppressByRouteSilence(ctx, status, groupKey, route.MatchedPolicyIDs, route.SilenceSeconds, labels) {
			host.LogSuppressedRouteSilence(ctx, title, severity, status, envLabel, groupKey, labelsDigest, route.SilenceSeconds, outgoing)
			continue
		}
		if status == "resolved" && !host.WasFiringDelivered(ctx, alert.Fingerprint) {
			host.LogResolvedSuppressedNoPriorFiringDelivery(ctx, title, severity, status, groupKey, labelsDigest, outgoing)
			_ = host.ClearFingerprintState(ctx, alert.Fingerprint)
			_ = host.ClearCurrentMetric(ctx, alert.Fingerprint)
			_ = host.ClearGroupAggregateState(ctx, groupKey)
			host.ClearGroupWaitPending(ctx, groupKey)
			continue
		}
		if status == "resolved" {
			firstResolved, _ := host.MarkResolvedNotificationSent(ctx, alert.Fingerprint)
			if !firstResolved {
				outgoing["resolved_sent"] = false
				outgoing["summary"] = "重复恢复事件已抑制（同一告警实例仅发送一次恢复通知）。"
				host.LogSuppressedResolvedAggregate(ctx, title, severity, status, groupKey, outgoing)
				continue
			}
			outgoing["resolved_sent"] = true
		}

		if status == "firing" {
			if !host.TryLockGroupSend(ctx, groupKey) {
				continue
			}
		}
		sentCount, okDeliveries := deliverToChannels(ctx, host, channels, route.ChannelIDs, ca.Source, title, severity, status, labels, outgoing)
		if status == "firing" {
			host.UnlockGroupSend(ctx, groupKey)
		}
		if status == "firing" && okDeliveries > 0 {
			host.CommitFiringGroupTimingSend(ctx, groupKey, labelsDigest)
			host.MarkFiringDelivered(ctx, alert.Fingerprint)
			host.ClearGroupWaitPending(ctx, groupKey)
			host.MaybeScheduleEscalation(ctx, escalationPendingEnvelope{
				Fingerprint:  alert.Fingerprint,
				GroupKey:     groupKey,
				LabelsDigest: labelsDigest,
				Source:       ca.Source,
				Title:        title,
				Severity:     severity,
				Status:       status,
				EnvLabel:     envLabel,
				Labels:       labels,
				Outgoing:     outgoing,
			}, route.EscalationLevel)
		}
		if status == "resolved" && okDeliveries == 0 {
			_ = host.ClearResolvedSentMark(ctx, alert.Fingerprint)
		}
		if sentCount == 0 {
			reason := "no_channel_matched"
			if len(channels) == 0 {
				reason = "no_enabled_channels"
			} else if len(route.ChannelIDs) > 0 {
				reason = "no_channel_matched_subscription"
			}
			host.LogNoMatchedChannel(ctx, title, severity, status, envLabel, groupKey, labelsDigest, outgoing, reason)
		}
		if status == "firing" && sentCount > 0 && okDeliveries == 0 {
			host.LogAllChannelsDeliveryFailed(ctx, title, severity, status, envLabel, groupKey, labelsDigest, outgoing)
		}
		if status == "resolved" {
			host.OnResolvedComplete(ctx, alert.Fingerprint, groupKey)
		}
	}
	return nil
}

func deliverToChannels(
	ctx context.Context,
	host IngressHost,
	channels []model.AlertChannel,
	subscriptionChannels map[uint]struct{},
	source, title, severity, status string,
	labels map[string]string,
	outgoing map[string]any,
) (sentCount, okDeliveries int) {
	_ = labels
	for i := range channels {
		if _, ok := subscriptionChannels[channels[i].ID]; !ok {
			continue
		}
		if !host.ChannelMatchesAlert(channels[i], labels) {
			continue
		}
		sentCount++
		code, err := host.SendToChannel(ctx, &channels[i], source, title, severity, status, outgoing)
		if err == nil && code >= 200 && code < 300 {
			okDeliveries++
		}
	}
	return sentCount, okDeliveries
}

func buildOutgoingPayload(
	ca CanonicalIngressAlert,
	alert IngressAlertDetail,
	title, summary, severity, status string,
	labels, annotations map[string]string,
	envLabel, monitorPipeline string,
	dsID uint, dsName, dsType, groupKey, labelsDigest string,
	count int64,
) map[string]any {
	out := map[string]any{
		"source": ca.Source, "title": title, "summary": summary, "severity": severity, "status": status,
		"receiver": ca.PayloadReceiver, "fingerprint": alert.Fingerprint, "count": count,
		"labels": labels, "annotations": annotations, "group_labels": ca.GroupLabels,
		"am_version": ca.Version, "startsAt": alert.StartsAt, "endsAt": alert.EndsAt,
		"generatorURL": alert.GeneratorURL, "truncated": ca.TruncatedAlerts,
		"occurredAt": time.Now().Format(time.RFC3339), "cluster": envLabel,
		"monitorPipeline": monitorPipeline, "datasourceId": dsID, "datasourceName": dsName,
		"datasourceType": dsType, "groupKey": groupKey, "labelsDigest": labelsDigest,
	}
	if pid := parseLabelUintOrZero(labels["project_id"]); pid > 0 {
		out["project_id"] = pid
	} else if ca.Cloud != nil && ca.Cloud.ProjectID > 0 {
		out["project_id"] = ca.Cloud.ProjectID
	}
	if ca.Cloud != nil {
		cloud := map[string]any{
			"provider":      ca.Cloud.Provider,
			"account_id":    ca.Cloud.AccountID,
			"instance_id":   ca.Cloud.InstanceID,
			"instance_name": ca.Cloud.InstanceName,
			"region":        ca.Cloud.Region,
			"days_left":     ca.Cloud.DaysLeft,
			"project_id":    ca.Cloud.ProjectID,
		}
		if !ca.Cloud.ExpiresAt.IsZero() {
			cloud["expires_at"] = ca.Cloud.ExpiresAt.Format(time.RFC3339)
		}
		out["cloud"] = cloud
	}
	return out
}

func normalizeIngressStatus(alertStatus, payloadStatus string) string {
	status := strings.TrimSpace(strings.ToLower(alertStatus))
	if status == "" {
		status = strings.TrimSpace(strings.ToLower(payloadStatus))
	}
	if status == "" {
		status = "firing"
	}
	return status
}

func pickAlertName(labels, common map[string]string) string {
	if v := strings.TrimSpace(labels["alertname"]); v != "" {
		return v
	}
	if v := strings.TrimSpace(common["alertname"]); v != "" {
		return v
	}
	return "平台告警"
}

func pickSummary(annotations, common map[string]string) string {
	for _, k := range []string{"summary", "description"} {
		if v := strings.TrimSpace(annotations[k]); v != "" {
			return v
		}
	}
	if v := strings.TrimSpace(common["summary"]); v != "" {
		return v
	}
	return "告警通知"
}

func pickSeverity(labels, common map[string]string) string {
	if v := strings.TrimSpace(labels["severity"]); v != "" {
		return v
	}
	if v := strings.TrimSpace(common["severity"]); v != "" {
		return v
	}
	return "warning"
}

func mergeStringMaps(base, overlay map[string]string) map[string]string {
	out := make(map[string]string)
	maps.Copy(out, base)
	maps.Copy(out, overlay)
	return out
}
