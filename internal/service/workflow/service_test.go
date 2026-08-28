package workflow

import (
	"testing"

	"yunshu/internal/model"
)

func TestDefaultStagesNonEmpty(t *testing.T) {
	if len(DefaultDbmgmtStages()) == 0 || len(DefaultCicdStages()) == 0 || len(DefaultIncidentStages()) == 0 {
		t.Fatal("default workflow stages should not be empty")
	}
	if DefaultIncidentStages()[0].AssigneeRuleType != model.WorkflowAssigneeDuty {
		t.Fatalf("incident first stage should use duty assignee")
	}
}

func TestNormalizeStageKey(t *testing.T) {
	key, err := normalizeStageKey("dba_lead")
	if err != nil || key != "dba_lead" {
		t.Fatalf("unexpected: %q %v", key, err)
	}
	_, err = normalizeStageKey("1bad")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}
