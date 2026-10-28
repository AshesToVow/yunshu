package alert

import (
	"testing"
	"time"
)

func TestComputeRotationSlotsDaily(t *testing.T) {
	anchor := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	period := 24 * time.Hour
	participants := []uint{1, 2, 3}
	from := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 4, 12, 0, 0, 0, time.UTC)

	slots := ComputeRotationSlots(anchor, period, participants, from, to)
	if len(slots) != 4 {
		t.Fatalf("expected 4 slots, got %d: %+v", len(slots), slots)
	}
	wantUsers := []uint{1, 2, 3, 1}
	for i, slot := range slots {
		if len(slot.UserIDs) != 1 || slot.UserIDs[0] != wantUsers[i] {
			t.Fatalf("slot %d expected user %d, got %v", i, wantUsers[i], slot.UserIDs)
		}
	}
	if !slots[0].StartsAt.Equal(from) {
		t.Fatalf("first slot should start at query from, got %v", slots[0].StartsAt)
	}
	if !slots[3].EndsAt.Equal(to) {
		t.Fatalf("last slot should end at query to, got %v", slots[3].EndsAt)
	}
	if !slots[1].StartsAt.Equal(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("second slot unexpected start %v", slots[1].StartsAt)
	}
}

func TestComputeRotationSlotsBeforeAnchor(t *testing.T) {
	anchor := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	participants := []uint{10, 20, 30}
	from := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	slots := ComputeRotationSlots(anchor, 24*time.Hour, participants, from, to)
	if len(slots) != 2 {
		t.Fatalf("expected 2 slots, got %d: %+v", len(slots), slots)
	}
	if slots[0].UserIDs[0] != 30 {
		t.Fatalf("slot before anchor should wrap to last participant, got %v", slots[0].UserIDs)
	}
	if slots[1].UserIDs[0] != 10 || !slots[1].EndsAt.Equal(to) {
		t.Fatalf("anchor slot unexpected: %+v", slots[1])
	}
}

func TestComputeRotationSlotsRejectsInvalidInput(t *testing.T) {
	anchor := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := ComputeRotationSlots(anchor, 0, []uint{1}, anchor, anchor.Add(time.Hour)); got != nil {
		t.Fatalf("zero period should yield nil, got %+v", got)
	}
	if got := ComputeRotationSlots(anchor, time.Hour, nil, anchor, anchor.Add(time.Hour)); got != nil {
		t.Fatalf("empty participants should yield nil, got %+v", got)
	}
	if got := ComputeRotationSlots(anchor, time.Hour, []uint{1}, anchor, anchor); got != nil {
		t.Fatalf("empty window should yield nil, got %+v", got)
	}
}

func TestCurrentRotationSlot(t *testing.T) {
	anchor := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	participants := []uint{7, 8}

	slot, ok := CurrentRotationSlot(anchor, 8*time.Hour, participants, anchor.Add(9*time.Hour))
	if !ok || slot.UserIDs[0] != 8 {
		t.Fatalf("expected user 8 at +9h, got ok=%v slot=%+v", ok, slot)
	}
	if !slot.StartsAt.Equal(anchor.Add(8 * time.Hour)) || !slot.EndsAt.Equal(anchor.Add(16*time.Hour)) {
		t.Fatalf("unexpected slot window: %v ~ %v", slot.StartsAt, slot.EndsAt)
	}

	before, ok := CurrentRotationSlot(anchor, 8*time.Hour, participants, anchor.Add(-time.Hour))
	if !ok || before.UserIDs[0] != 8 {
		t.Fatalf("expected last participant before anchor, got ok=%v slot=%+v", ok, before)
	}
}
