package cicd

import "testing"

func TestValidateReleaseOperation(t *testing.T) {
	if err := validateReleaseOperation("frontend", "frontend_online"); err != nil {
		t.Fatal(err)
	}
	if err := validateReleaseOperation("frontend", "backend_update"); err == nil {
		t.Fatal("expected error for wrong op on frontend")
	}
	if err := validateReleaseOperation("backend", "backend_initial"); err != nil {
		t.Fatal(err)
	}
	if err := validateReleaseOperation("backend", "frontend_rollback"); err == nil {
		t.Fatal("expected error for wrong op on backend")
	}
}

func TestReleaseForceCleanDeploy(t *testing.T) {
	if !releaseForceCleanDeploy("backend_initial") {
		t.Fatal("backend_initial should force clean deploy")
	}
	if releaseForceCleanDeploy("backend_update") {
		t.Fatal("backend_update should not force clean deploy")
	}
}
