package ai

import "testing"

func TestExtractInvestigationWriteCandidates_AlertSilence(t *testing.T) {
	t.Parallel()
	report := &InvestigationReport{
		Actions: []map[string]any{
			{"action": "check_changes"},
		},
	}
	req := StartInvestigationRequest{Kind: "alert", ProjectID: 3, Fingerprint: "fp-1"}
	cands := extractInvestigationWriteCandidates("alert", req, report)
	if len(cands) != 1 {
		t.Fatalf("want 1 silence candidate, got %d", len(cands))
	}
	if cands[0].Tool != "create_alert_silence" {
		t.Fatalf("tool=%s", cands[0].Tool)
	}
}

func TestExtractInvestigationWriteCandidates_ScaleNeedsReplicas(t *testing.T) {
	t.Parallel()
	report := &InvestigationReport{
		Actions: []map[string]any{
			{"action": "scale_deployment", "cluster_id": 1, "namespace": "ns", "name": "web"},
		},
	}
	cands := extractInvestigationWriteCandidates("pod", StartInvestigationRequest{}, report)
	if len(cands) != 0 {
		t.Fatalf("scale without replicas should skip, got %d", len(cands))
	}
	report.Actions[0]["replicas"] = 3
	cands = extractInvestigationWriteCandidates("pod", StartInvestigationRequest{}, report)
	if len(cands) != 1 || cands[0].Tool != "scale_deployment" {
		t.Fatalf("got %#v", cands)
	}
}

func TestNormalizeWriteToolName(t *testing.T) {
	t.Parallel()
	if got := normalizeWriteToolName("silence"); got != "create_alert_silence" {
		t.Fatalf("got %s", got)
	}
	if got := normalizeWriteToolName("restart"); got != "restart_deployment" {
		t.Fatalf("got %s", got)
	}
}
