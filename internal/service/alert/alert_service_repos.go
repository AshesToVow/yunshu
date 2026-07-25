package alert

import (
	"context"
	"strings"

	"yunshu/internal/model"
)

func (s *AlertService) loadEnabledChannels(ctx context.Context) ([]model.AlertChannel, error) {
	ptrs, err := s.channelRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.AlertChannel, len(ptrs))
	for i, ch := range ptrs {
		if ch != nil {
			out[i] = *ch
		}
	}
	return out, nil
}

func (s *AlertService) persistAlertEvent(ctx context.Context, event *model.AlertEvent) error {
	if event != nil && strings.TrimSpace(event.Fingerprint) == "" {
		fillAlertEventFingerprintFromPayload(event, nil)
		if strings.TrimSpace(event.Fingerprint) == "" {
			if fp := alertEventFingerprint(event); fp != "" {
				event.Fingerprint = truncateText(fp, 512)
			}
		}
	}
	if err := s.eventRepo.Create(ctx, event); err != nil {
		return err
	}
	if s.alertStateSvc != nil {
		fp := strings.TrimSpace(event.Fingerprint)
		if fp == "" {
			fp = alertEventFingerprint(event)
		}
		if fp != "" {
			_, _ = s.alertStateSvc.TouchFingerprint(ctx, fp, event.Status)
		}
	}
	return nil
}

func alertEventFingerprint(event *model.AlertEvent) string {
	if event == nil {
		return ""
	}
	if fp := strings.TrimSpace(event.Fingerprint); fp != "" {
		return fp
	}
	if fp := strings.TrimSpace(event.GroupKey); fp != "" {
		return fp
	}
	return strings.TrimSpace(event.LabelsDigest)
}
