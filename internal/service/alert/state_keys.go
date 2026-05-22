package alert

import "strings"

func fingerprintRedisKey(fingerprint string) string {
	return "alert:fingerprint:" + strings.TrimSpace(fingerprint)
}

func resolvedSentRedisKey(fingerprint string) string {
	return "alert:resolved:sent:" + strings.TrimSpace(fingerprint)
}

func firingDeliveredRedisKey(fingerprint string) string {
	return "alert:firing_delivered:" + strings.TrimSpace(fingerprint)
}

func currentMetricRedisKey(fingerprint string) string {
	return "alert:current:" + strings.TrimSpace(fingerprint)
}
