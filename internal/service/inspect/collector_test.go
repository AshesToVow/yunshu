package inspect

import "testing"

func TestGetStatusGreater(t *testing.T) {
	if getStatus(90, 80, "greater") != "critical" {
		t.Fatal("expected critical")
	}
	if getStatus(75, 80, "greater") != "warning" {
		t.Fatal("expected warning near threshold")
	}
	if getStatus(50, 80, "greater") != "normal" {
		t.Fatal("expected normal")
	}
}

func TestGetStatusEqual(t *testing.T) {
	if getStatus(1, 1, "equal") != "normal" {
		t.Fatal("equal match should be normal")
	}
	if getStatus(0, 1, "equal") != "critical" {
		t.Fatal("equal mismatch should be critical")
	}
}
