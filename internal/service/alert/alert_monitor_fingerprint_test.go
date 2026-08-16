package alert

import "testing"

func TestMonitorSeriesFingerprintStableAndDistinct(t *testing.T) {
	t.Parallel()
	a := map[string]string{"alertname": "cpu", "instance": "a:9100", "severity": "warning"}
	b := map[string]string{"alertname": "cpu", "instance": "b:9100", "severity": "warning"}
	fp1 := monitorSeriesFingerprint(7, a)
	fp2 := monitorSeriesFingerprint(7, a)
	fp3 := monitorSeriesFingerprint(7, b)
	fp4 := monitorSeriesFingerprint(8, a)
	if fp1 == "" || fp1 != fp2 {
		t.Fatalf("unstable: %q vs %q", fp1, fp2)
	}
	if fp1 == fp3 {
		t.Fatal("different series must differ")
	}
	if fp1 == fp4 {
		t.Fatal("different rules must differ")
	}
	if monitorSeriesFingerprint(7, map[string]string{"instance": "a:9100", "fingerprint": "ignore"}) == "" {
		t.Fatal("empty")
	}
}
