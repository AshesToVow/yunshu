package esmgmt

import "testing"

func TestValidateIndexRef(t *testing.T) {
	if _, err := validateIndexRef(".kibana", false); err == nil {
		t.Fatal("system index should be rejected")
	}
	if _, err := validateIndexRef("logs-*", false); err == nil {
		t.Fatal("wildcard should be rejected for exact index")
	}
	if _, err := validateIndexRef("logs-*", true); err != nil {
		t.Fatal(err)
	}
	if _, err := validateIndexRef("app_logs-2026.09.21", false); err != nil {
		t.Fatal(err)
	}
}

func TestSafeTaskID(t *testing.T) {
	if !safeTaskID("abc:123") {
		t.Fatal("expected valid task id")
	}
	if safeTaskID("../_cluster") {
		t.Fatal("path traversal should be rejected")
	}
}
