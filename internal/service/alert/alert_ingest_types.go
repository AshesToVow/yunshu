package alert

import (
	"context"
	"fmt"
	"strings"
)

// CanonicalIngressAlert 统一入站模型：由 Alertmanager Webhook 载荷或内置评估路径构造。
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
	Alert             AlertManagerAlert
}

// CanonicalAlertsFromAlertmanagerPayload 将 Alertmanager Webhook 形态转为统一入站切片。
func CanonicalAlertsFromAlertmanagerPayload(p AlertManagerPayload) []CanonicalIngressAlert {
	rcv := strings.TrimSpace(p.Receiver)
	src := "alertmanager"
	switch rcv {
	case "platform-monitor":
		src = "platform_monitor"
	case "cloud-expiry":
		src = "cloud_expiry"
	case "k8s-events":
		src = "k8s_event"
	}
	out := make([]CanonicalIngressAlert, 0, len(p.Alerts))
	for i := range p.Alerts {
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
			Alert:             p.Alerts[i],
		})
	}
	return out
}

func (s *AlertService) receiveAlertmanagerPayloadSync(ctx context.Context, payload AlertManagerPayload) error {
	items := CanonicalAlertsFromAlertmanagerPayload(payload)
	s.applyIngressGroupTimingPolicy(items)
	return s.ingestCanonicalAlerts(ctx, items)
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
