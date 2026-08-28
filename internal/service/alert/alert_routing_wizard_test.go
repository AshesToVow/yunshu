package alert

import (
	"testing"
)

func TestWizardNodeName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		projectID uint
		severity  string
		want      string
	}{
		{name: "global all", projectID: 0, severity: "", want: "向导 · 全部级别"},
		{name: "global critical", projectID: 0, severity: "critical", want: "向导 · critical"},
		{name: "project warning", projectID: 12, severity: "warning", want: "向导 · 项目 12 · warning"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := wizardNodeName(tc.projectID, tc.severity); got != tc.want {
				t.Fatalf("wizardNodeName(%d,%q)=%q want %q", tc.projectID, tc.severity, got, tc.want)
			}
		})
	}
}

func TestNormalizeWizardSeverity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{in: " CRITICAL ", want: "critical"},
		{in: "warning", want: "warning"},
		{in: "critical,warning", want: "critical,warning"},
		{in: "warning,critical", want: "critical,warning"},
		{in: "  ", want: ""},
		{in: "p1", want: "p1"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := normalizeWizardSeverity(tc.in); got != tc.want {
				t.Fatalf("normalizeWizardSeverity(%q)=%q want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestUniquePositiveUints(t *testing.T) {
	t.Parallel()
	got := uniquePositiveUints([]uint{0, 2, 2, 1, 0, 1})
	if len(got) != 2 || got[0] != 2 || got[1] != 1 {
		t.Fatalf("uniquePositiveUints: %v", got)
	}
}

func TestUniqueEmails(t *testing.T) {
	t.Parallel()
	got := uniqueEmails([]string{" A@B.com ", "a@b.com", "", "c@d.com"})
	if len(got) != 2 || got[0] != "a@b.com" || got[1] != "c@d.com" {
		t.Fatalf("uniqueEmails: %v", got)
	}
}
