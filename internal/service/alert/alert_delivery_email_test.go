package alert

import "testing"

func TestMergeAssigneeEmailsStrictPriority(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"assignee_emails": []string{"rule@x.com"},
		"recipient_mode":  RecipientModeAssigneeOnly,
	}
	out := mergeAssigneeEmails([]string{"channel@x.com"}, payload)
	if len(out) != 1 || out[0] != "rule@x.com" {
		t.Fatalf("assignee only: %v", out)
	}
}

func TestMergeAssigneeEmailsDefaultAndCC(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"assignee_emails": []string{"rule@x.com"},
	}
	out := mergeAssigneeEmails([]string{"channel@x.com"}, payload)
	if len(out) != 2 {
		t.Fatalf("default assignee_and_cc: %v", out)
	}
}

func TestMergeAssigneeEmailsChannelOnly(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"assignee_emails": []string{"rule@x.com"},
		"recipient_mode":  RecipientModeChannelOnly,
	}
	out := mergeAssigneeEmails([]string{"channel@x.com"}, payload)
	if len(out) != 1 || out[0] != "channel@x.com" {
		t.Fatalf("channel only: %v", out)
	}
}

func TestMergeAssigneeEmailsChannelFallback(t *testing.T) {
	t.Parallel()
	out := mergeAssigneeEmails([]string{"A@x.com", "a@x.com"}, map[string]any{})
	if len(out) != 1 || out[0] != "a@x.com" {
		t.Fatalf("got %v", out)
	}
}

func TestPayloadHasAssigneeEmails(t *testing.T) {
	t.Parallel()
	if payloadHasAssigneeEmails(nil) {
		t.Fatal("nil payload")
	}
	if !payloadHasAssigneeEmails(map[string]any{
		"assignee_emails": []string{"a@x.com"},
	}) {
		t.Fatal("expected true")
	}
}

func TestMergeAssigneeEmailsWithReceiverGroup(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"receiver_group_emails": []string{"cc@x.com"},
	}
	out := mergeAssigneeEmailsWithReceiverGroup([]string{"to@x.com"}, payload)
	if len(out) != 2 {
		t.Fatalf("got %v", out)
	}
	payload2 := map[string]any{
		"assignee_emails":       []string{"rule@x.com"},
		"receiver_group_emails": []string{"cc@x.com"},
		"recipient_mode":        RecipientModeAssigneeOnly,
	}
	out2 := mergeAssigneeEmailsWithReceiverGroup([]string{"to@x.com"}, payload2)
	if len(out2) != 1 || out2[0] != "to@x.com" {
		t.Fatalf("assignee_only skips group merge: %v", out2)
	}
	payload3 := map[string]any{
		"assignee_emails":       []string{"rule@x.com"},
		"receiver_group_emails": []string{"cc@x.com"},
		"recipient_mode":        RecipientModeAssigneeAndCC,
	}
	out3 := mergeAssigneeEmailsWithReceiverGroup([]string{"rule@x.com", "to@x.com"}, payload3)
	if len(out3) != 3 {
		t.Fatalf("assignee_and_cc merges group: %v", out3)
	}
}
