package alert

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizeDutyRotationDefaults(t *testing.T) {
	req := AlertDutyRotationUpsertRequest{
		MonitorRuleID:     9,
		Name:              " 数据库轮班 ",
		ParticipantsJSON:  "[3,1,3,2]",
		DepartmentIDsJSON: "[4,4]",
		ExtraEmailsJSON:   `["DBA@Example.com","dba@example.com"]`,
		AnchorAt:          time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
		PeriodMinutes:     1440,
	}
	row, err := normalizeDutyRotation(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row.Name != "数据库轮班" {
		t.Fatalf("name not trimmed: %q", row.Name)
	}
	if !row.Enabled {
		t.Fatal("rotation should default to enabled")
	}
	var participants []uint
	if err := json.Unmarshal([]byte(row.ParticipantsJSON), &participants); err != nil {
		t.Fatalf("participants json invalid: %v", err)
	}
	if len(participants) != 3 || participants[0] != 3 || participants[2] != 2 {
		t.Fatalf("participants should keep order and dedupe, got %v", participants)
	}
	var depts []uint
	if err := json.Unmarshal([]byte(row.DepartmentIDsJSON), &depts); err != nil {
		t.Fatalf("departments json invalid: %v", err)
	}
	if len(depts) != 1 || depts[0] != 4 {
		t.Fatalf("departments should dedupe, got %v", depts)
	}
	var emails []string
	if err := json.Unmarshal([]byte(row.ExtraEmailsJSON), &emails); err != nil {
		t.Fatalf("emails json invalid: %v", err)
	}
	if len(emails) != 1 || emails[0] != "dba@example.com" {
		t.Fatalf("emails should be lowercased and deduped, got %v", emails)
	}
}

func TestNormalizeDutyRotationValidation(t *testing.T) {
	base := AlertDutyRotationUpsertRequest{
		MonitorRuleID:    1,
		Name:             "轮班",
		ParticipantsJSON: "[1]",
		AnchorAt:         time.Now(),
		PeriodMinutes:    1440,
	}
	cases := []struct {
		name   string
		mutate func(*AlertDutyRotationUpsertRequest)
	}{
		{"empty name", func(r *AlertDutyRotationUpsertRequest) { r.Name = " " }},
		{"no participants", func(r *AlertDutyRotationUpsertRequest) { r.ParticipantsJSON = "[]" }},
		{"zero anchor", func(r *AlertDutyRotationUpsertRequest) { r.AnchorAt = time.Time{} }},
		{"period too small", func(r *AlertDutyRotationUpsertRequest) { r.PeriodMinutes = 5 }},
		{"period too large", func(r *AlertDutyRotationUpsertRequest) { r.PeriodMinutes = 60 * 24 * 60 }},
		{"bad departments", func(r *AlertDutyRotationUpsertRequest) { r.DepartmentIDsJSON = `{"a":1}` }},
		{"bad emails", func(r *AlertDutyRotationUpsertRequest) { r.ExtraEmailsJSON = `"x"` }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := base
			tc.mutate(&req)
			if _, err := normalizeDutyRotation(req); err == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestDedupeUintIDs(t *testing.T) {
	got := dedupeUintIDs([]uint{0, 2, 1, 2, 3, 1})
	want := []uint{2, 1, 3}
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %v want %v", i, got, want)
		}
	}
}
