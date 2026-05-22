package alert

import (
	"context"
)

func (s *AlertService) ingestCanonicalAlerts(ctx context.Context, items []CanonicalIngressAlert) error {
	return RunIngressPipeline(ctx, s.ingressHost(), canonicalItemsToIngress(items))
}

func (s *AlertService) touchFingerprintState(ctx context.Context, fingerprint, status string) (int64, error) {
	if s.alertStateSvc != nil {
		return s.alertStateSvc.TouchFingerprint(ctx, fingerprint, status)
	}
	count, _, err := s.updateFingerprintState(ctx, fingerprint, status)
	return count, err
}
