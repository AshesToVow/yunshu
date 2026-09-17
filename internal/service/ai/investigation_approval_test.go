package ai

import (
	"testing"

	"yunshu/internal/model"
)

func TestExtractInvestigationWriteCandidates_AlertNoDefaultSilence(t *testing.T) {
	t.Parallel()
	report := &InvestigationReport{
		Actions: []map[string]any{
			{"action": "suggest_silence", "hint": "optional"},
			{"action": "check_changes"},
		},
	}
	req := StartInvestigationRequest{Kind: "alert", ProjectID: 3, Fingerprint: "fp-1"}
	cands := extractInvestigationWriteCandidates("alert", req, report)
	if len(cands) != 0 {
		t.Fatalf("suggest_silence must not create approval, got %d", len(cands))
	}
}

func TestExtractInvestigationWriteCandidates_ExplicitSilence(t *testing.T) {
	t.Parallel()
	report := &InvestigationReport{
		Actions: []map[string]any{
			{"action": "create_alert_silence", "fingerprint": "fp-1", "project_id": 3},
		},
	}
	cands := extractInvestigationWriteCandidates("alert", StartInvestigationRequest{ProjectID: 3, Fingerprint: "fp-1"}, report)
	if len(cands) != 1 || cands[0].Tool != "create_alert_silence" {
		t.Fatalf("got %#v", cands)
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

func TestLinkedApprovalIDsFromInvestigation(t *testing.T) {
	t.Parallel()
	aid := uint(9)
	inv := &model.AiInvestigation{
		ApprovalID:   &aid,
		AnalysisJSON: `{"approvals":[{"approval_id":9},{"approval_id":10}]}`,
	}
	ids := linkedApprovalIDsFromInvestigation(inv)
	if len(ids) != 2 {
		t.Fatalf("want 2 ids, got %v", ids)
	}
}
