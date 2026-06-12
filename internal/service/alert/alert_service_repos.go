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
	if err := s.eventRepo.Create(ctx, event); err != nil {
		return err
	}
	if s.alertStateSvc != nil {
		fp := alertEventFingerprint(event)
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
	if fp := strings.TrimSpace(event.GroupKey); fp != "" {
		return fp
	}
	return strings.TrimSpace(event.LabelsDigest)
}
