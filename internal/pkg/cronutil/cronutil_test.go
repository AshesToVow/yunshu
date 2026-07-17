package cronutil

import (
	"testing"
	"time"
)

func TestShouldRunWithDayAnchorFirstEnableBeforeDue(t *testing.T) {
	t.Parallel()
	spec := "0 45 22 * * *"
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 6, 30, 22, 44, 0, 0, loc)
	if ShouldRunWithDayAnchor(spec, nil, now) {
		t.Fatal("should not run before cron due time on first enable")
	}
}

func TestShouldRunWithDayAnchorFirstEnableAfterDueNoCatchUp(t *testing.T) {
	t.Parallel()
	spec := "0 0 2 * * *"
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 7, 4, 18, 4, 0, 0, loc)
	if ShouldRunWithDayAnchor(spec, nil, now) {
		t.Fatal("should not catch up when first enabled after today's cron time")
	}
}

func TestShouldRunWithDayAnchorFirstEnableAtDue(t *testing.T) {
	t.Parallel()
	spec := "0 45 22 * * *"
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 6, 30, 22, 45, 15, 0, loc)
	if !ShouldRunWithDayAnchor(spec, nil, now) {
		t.Fatal("should run after cron due time on first enable")
	}
}

func TestShouldRunWithDayAnchorAlreadyRanThisSlot(t *testing.T) {
	t.Parallel()
	spec := "0 45 22 * * *"
	loc := time.FixedZone("CST", 8*3600)
	last := time.Date(2026, 6, 30, 22, 45, 20, 0, loc)
	now := time.Date(2026, 6, 30, 22, 45, 30, 0, loc)
	if ShouldRunWithDayAnchor(spec, &last, now) {
		t.Fatal("should not run twice in same cron slot")
	}
}

func TestShouldRunWithDayAnchorNextDay(t *testing.T) {
	t.Parallel()
	spec := "0 45 22 * * *"
	loc := time.FixedZone("CST", 8*3600)
	last := time.Date(2026, 6, 30, 22, 45, 20, 0, loc)
	now := time.Date(2026, 7, 1, 22, 45, 10, 0, loc)
	if !ShouldRunWithDayAnchor(spec, &last, now) {
		t.Fatal("should run on next day's cron slot")
	}
}

func TestValidateSpec(t *testing.T) {
	t.Parallel()
	if err := ValidateSpec("", "cron"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSpec("0 0 2 * * *", "cron"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSpec("not a cron", "cron"); err == nil {
		t.Fatal("expected error")
	}
}
