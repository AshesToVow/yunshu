package alert

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// 平台内部入站来源标识（流水线与投递 payload 的 source 字段）。
const (
	IngressSourceAlertmanager    = "alertmanager"
	IngressSourcePlatformMonitor = "platform_monitor"
	IngressSourceCloudExpiry     = "cloud_expiry"
	IngressSourceK8sEvent        = "k8s_event"
)

// CloudExpiryExtension 云到期告警一等扩展字段（不再只能塞进 labels/annotations）。
type CloudExpiryExtension struct {
	Provider     string
	AccountID    uint
	InstanceID   string
	InstanceName string
	Region       string
	ExpiresAt    time.Time
	DaysLeft     int
	ProjectID    uint
}

// CanonicalIngressAlert 平台统一入站模型。流水线只消费本结构；
// Alertmanager Webhook 经 CanonicalAlertsFromAlertmanagerPayload 适配进来，
// 平台自研入口（监控规则 / 云到期）直接构造，不再绕 AM Payload。
type CanonicalIngressAlert struct {
	Source            string
	PayloadReceiver   string
	PayloadStatus     string
	GroupLabels       map[string]string
	CommonLabels      map[string]string
	CommonAnnotations map[string]string
	Version           string
	ExternalURL       string
	TruncatedAlerts   int
	Alert             IngressAlertDetail
	Cloud             *CloudExpiryExtension
}

// NewCanonicalAlert 构造单条平台入站告警。
func NewCanonicalAlert(
	source, receiver, payloadStatus string,
	groupLabels, commonLabels map[string]string,
	detail IngressAlertDetail,
) CanonicalIngressAlert {
	return CanonicalIngressAlert{
		Source:          strings.TrimSpace(source),
		PayloadReceiver: strings.TrimSpace(receiver),
		PayloadStatus:   strings.TrimSpace(payloadStatus),
		GroupLabels:     groupLabels,
		CommonLabels:    commonLabels,
		Alert:           detail,
	}
}

// CanonicalAlertsFromAlertmanagerPayload 将外部 Alertmanager Webhook 形态适配为统一入站切片。
func CanonicalAlertsFromAlertmanagerPayload(p AlertManagerPayload) []CanonicalIngressAlert {
	rcv := strings.TrimSpace(p.Receiver)
	src := IngressSourceAlertmanager
	switch rcv {
	case "platform-monitor":
		src = IngressSourcePlatformMonitor
	case "cloud-expiry":
		src = IngressSourceCloudExpiry
	case "k8s-events":
		src = IngressSourceK8sEvent
	}
	out := make([]CanonicalIngressAlert, 0, len(p.Alerts))
	for i := range p.Alerts {
		a := p.Alerts[i]
		out = append(out, CanonicalIngressAlert{
			Source:            src,
			PayloadReceiver:   p.Receiver,
			PayloadStatus:     p.Status,
			GroupLabels:       p.GroupLabels,
			CommonLabels:      p.CommonLabels,
			CommonAnnotations: p.CommonAnnotations,
			Version:           p.Version,
			ExternalURL:       p.ExternalURL,
			TruncatedAlerts:   p.TruncatedAlerts,
			Alert: IngressAlertDetail{
				Status:          a.Status,
				Labels:          a.Labels,
				Annotations:     a.Annotations,
				StartsAt:        a.StartsAt,
				EndsAt:          a.EndsAt,
				GeneratorURL:    a.GeneratorURL,
				Fingerprint:     a.Fingerprint,
				SkipGroupTiming: a.SkipGroupTiming,
			},
		})
	}
	return out
}

// receiveCanonicalSync 对平台自研入口：应用分组节流策略后入站。
func (s *AlertService) receiveCanonicalSync(ctx context.Context, items ...CanonicalIngressAlert) error {
	if len(items) == 0 {
		return nil
	}
	s.applyIngressGroupTimingPolicy(items)
	return s.ingestCanonicalAlerts(ctx, items)
}

func (s *AlertService) receiveAlertmanagerPayloadSync(ctx context.Context, payload AlertManagerPayload) error {
	items := CanonicalAlertsFromAlertmanagerPayload(payload)
	filtered := make([]CanonicalIngressAlert, 0, len(items))
	skipped := 0
	for _, it := range items {
		if s.shouldSkipAlertmanagerAsPlatformDuplicate(it) {
			skipped++
			continue
		}
		filtered = append(filtered, it)
	}
	if skipped > 0 {
		s.logWebhookWarn("skipped alertmanager alerts already owned by platform monitor",
			"skipped", skipped, "kept", len(filtered), "receiver", payload.Receiver)
	}
	if len(filtered) == 0 {
		return nil
	}
	return s.receiveCanonicalSync(ctx, filtered...)
}

// shouldSkipAlertmanagerAsPlatformDuplicate 避免平台规则与 AM 双路径对同一规则二次投递。
// 判定：labels 带 source=prometheus_monitor / monitor_rule_id，或 receiver=platform-monitor。
func (s *AlertService) shouldSkipAlertmanagerAsPlatformDuplicate(ca CanonicalIngressAlert) bool {
	labels := mergeStringMaps(ca.CommonLabels, ca.Alert.Labels)
	if s.isPlatformMonitor(labels, ca.PayloadReceiver) {
		return true
	}
	if labels != nil && strings.TrimSpace(labels["monitor_rule_id"]) != "" {
		return true
	}
	return false
}

// applyIngressGroupTimingPolicy 按告警入口决定是否走 Yunshu 第二层 group timing：
// - 入口 A（外部 Alertmanager Webhook）：默认 SkipGroupTiming，信任 AM group_wait/interval/repeat
// - 入口 B（platform-monitor 等平台自研路径）：保留 Redis group_wait/interval/repeat
func (s *AlertService) applyIngressGroupTimingPolicy(items []CanonicalIngressAlert) {
	for i := range items {
		if items[i].Alert.SkipGroupTiming {
			continue
		}
		if s.ingressUsesGroupTiming(items[i]) {
			continue
		}
		if s.cfg.WebhookSkipGroupTiming {
			items[i].Alert.SkipGroupTiming = true
		}
	}
}

func (s *AlertService) ingressUsesGroupTiming(ca CanonicalIngressAlert) bool {
	return s.isPlatformMonitor(ca.CommonLabels, ca.PayloadReceiver)
}

// enrichCanonicalIngressLabels 与入站 ingest 使用同一套标签补全，保证 groupKey / labelsDigest 与分组节流 Redis 状态一致。
func (s *AlertService) enrichCanonicalIngressLabels(ctx context.Context, labels map[string]string, payloadReceiver, fingerprint string) map[string]string {
	out := mergeStringMap(nil, labels)
	dsID, dsName, dsType, pipelineSlug := s.resolveAlertDatasourceMeta(ctx, out, payloadReceiver)
	out["monitor_pipeline"] = pipelineSlug
	if dsID > 0 {
		out["datasource_id"] = fmt.Sprintf("%d", dsID)
	}
	if strings.TrimSpace(dsName) != "" {
		out["datasource_name"] = strings.TrimSpace(dsName)
	}
	if strings.TrimSpace(dsType) != "" {
		out["datasource_type"] = strings.TrimSpace(dsType)
	}
	if fp := strings.TrimSpace(fingerprint); fp != "" {
		out["fingerprint"] = fp
	}
	return out
}
