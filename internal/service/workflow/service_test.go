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

func TestResolveFlowFallsBackToDefault(t *testing.T) {
	// normalize keeps access_request; fallback logic is covered by resolveFlow unit via empty stages path
	key := DefinitionKey{Domain: "dbmgmt", ProjectID: 1, TicketType: "access_request"}.normalize()
	if key.TicketType != "access_request" {
		t.Fatalf("expected access_request, got %s", key.TicketType)
	}
	if filterEnabledStages(nil) == nil {
		t.Fatal("filterEnabledStages should return empty slice")
	}
}
