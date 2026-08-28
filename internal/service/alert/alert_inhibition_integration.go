package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	bizerrors "yunshu/internal/pkg/errors"

	"yunshu/internal/model"
)

func (s *AlertService) RecordSourceInhibition(ctx context.Context, labels map[string]string) error {
	if s.inhibitionSvc == nil {
		return nil
	}
	ruleIDs, err := s.inhibitionSvc.CheckSourceMatch(ctx, labels)
	if err != nil || len(ruleIDs) == 0 {
		return bizerrors.Pass(ctx, "alert.inhibition", "RecordSourceInhibition", err)
	}
	fp := stableLabelsFingerprint(labels)
	if fp != "" && labels["fingerprint"] == "" {
		labels["fingerprint"] = fp
	}
	for _, ruleID := range ruleIDs {
		if err := s.inhibitionSvc.RecordSourceAlert(ctx, ruleID, fp, labels); err != nil {
			continue
		}
	}
	return nil
}

func (s *AlertService) CheckInhibition(ctx context.Context, labels map[string]string) (bool, *model.AlertInhibitionEvent) {
	if s.inhibitionSvc == nil {
		return false, nil
	}
	inhibited, event, err := s.inhibitionSvc.CheckInhibition(ctx, labels)
	if err != nil {
		return false, nil
	}
	return inhibited, event
}

func (s *AlertService) ClearSourceInhibition(ctx context.Context, labels map[string]string) error {
	if s.inhibitionSvc == nil {
		return nil
	}
	ruleIDs, err := s.inhibitionSvc.CheckSourceMatch(ctx, labels)
	if err != nil {
		return bizerrors.Pass(ctx, "alert.inhibition", "ClearSourceInhibition", err)
	}
	fp := stableLabelsFingerprint(labels)
	if fp != "" && labels["fingerprint"] == "" {
		labels["fingerprint"] = fp
	}
	for _, ruleID := range ruleIDs {
		if err := s.inhibitionSvc.ClearSourceAlert(ctx, ruleID, fp); err != nil {
			continue
		}
	}
	return nil
}

func (s *AlertService) logInhibitionEvent(ctx context.Context, title, severity, status, cluster, groupKey, labelsDigest string, event *model.AlertInhibitionEvent, payload map[string]any) {
	reqBytes, _ := json.Marshal(payload)
	e := model.AlertEvent{
		Source:          alertEventSourceFromPayload(payload),
		Title:           title + " (inhibition suppressed)",
		Severity:        severity,
		Status:          status,
		Cluster:         cluster,
		MonitorPipeline: monitorPipelineFromPayload(payload),
		GroupKey:        groupKey,
		LabelsDigest:    labelsDigest,
		ChannelID:       0,
		ChannelName:     fmt.Sprintf("(inhibition source=%s)", event.SourceAlertName),
		Success:         true,
		HTTPStatusCode:  200,
		ErrorMessage:    fmt.Sprintf("inhibition_suppressed: rule=%s source=%s", event.RuleName, event.SourceFingerprint),
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText(fmt.Sprintf("suppressed by source fingerprint: %s", event.SourceFingerprint), s.cfg.MaxPayloadChars),
	}
	fillAlertEventDatasourceFromPayload(&e, payload)
	_ = s.persistAlertEvent(ctx, &e)
}

func (s *AlertService) startInhibitionPruner(ctx context.Context) {
	if s.inhibitionSvc == nil {
		return
	}
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = s.inhibitionSvc.RefreshCache(ctx)
			}
		}
	}()
}
