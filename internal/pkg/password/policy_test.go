package password

import (
	"testing"
	"time"

	"yunshu/internal/dictconfig"
)

func TestValidateComplexity(t *testing.T) {
	cfg := dictconfig.DefaultPasswordPolicy()
	if err := ValidateComplexity("Ab1!", "alice", cfg); err == nil {
		t.Fatal("expected too short")
	}
	if err := ValidateComplexity("abcdefgh", "alice", cfg); err == nil {
		t.Fatal("expected missing complexity")
	}
	if err := ValidateComplexity("Alice123!", "alice", cfg); err == nil {
		t.Fatal("expected forbid username")
	}
	if err := ValidateComplexity("GoodPass1!", "alice", cfg); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestIsExpired(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	changed := now.AddDate(0, 0, -91)
	if !IsExpired(&changed, now, 90, now) {
		t.Fatal("expected expired")
	}
	fresh := now.AddDate(0, 0, -10)
	if IsExpired(&fresh, now, 90, now) {
		t.Fatal("expected not expired")
	}
	if IsExpired(&changed, now, 0, now) {
		t.Fatal("expiry disabled should never expire")
	}
}
