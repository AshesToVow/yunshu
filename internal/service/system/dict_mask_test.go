package system

import (
	"strings"
	"testing"

	"yunshu/internal/model"
	"yunshu/internal/pkg/dictmask"
)

func TestMaskDictEntryListItem(t *testing.T) {
	t.Parallel()
	secret := "super-secret-token-value"
	got := maskDictEntryListItem(model.DictEntry{DictType: "alert_webhook_token", Value: secret})
	if !got.Sensitive {
		t.Fatal("sensitive dict type should be flagged")
	}
	if got.Value == secret || strings.Contains(got.Value, secret) {
		t.Fatalf("value leaked: %q", got.Value)
	}
	if got.Value != dictmask.Preview(secret) {
		t.Fatalf("value=%q want preview", got.Value)
	}

	plain := maskDictEntryListItem(model.DictEntry{DictType: "alert_severity", Value: "critical"})
	if plain.Sensitive {
		t.Fatal("non-sensitive dict type must not be flagged")
	}
	if plain.Value != "critical" {
		t.Fatalf("non-sensitive value changed: %q", plain.Value)
	}
}

func TestResolveDictEntrySubmittedValue(t *testing.T) {
	t.Parallel()
	secret := "super-secret-token-value"
	preview := dictmask.Preview(secret)
	cases := []struct {
		name         string
		raw          string
		oldValue     string
		oldSensitive bool
		newSensitive bool
		want         string
	}{
		{
			name:         "preserve_plaintext_on_same_preview",
			raw:          "  " + preview + "  ",
			oldValue:     secret,
			oldSensitive: true,
			newSensitive: true,
			want:         secret,
		},
		{
			name:         "replace_with_new_value",
			raw:          "new-secret",
			oldValue:     secret,
			oldSensitive: true,
			newSensitive: true,
			want:         "new-secret",
		},
		{
			name:         "plain_type_submits_preview_literally",
			raw:          preview,
			oldValue:     secret,
			oldSensitive: false,
			newSensitive: false,
			want:         preview,
		},
		{
			name:         "type_becomes_sensitive_keeps_plaintext",
			raw:          preview,
			oldValue:     secret,
			oldSensitive: false,
			newSensitive: true,
			want:         secret,
		},
		{
			name:         "type_becomes_plain_keeps_old_plaintext_on_preview",
			raw:          preview,
			oldValue:     secret,
			oldSensitive: true,
			newSensitive: false,
			want:         secret,
		},
		{
			name:         "plain_text_trimmed",
			raw:          "  critical  ",
			oldValue:     "",
			oldSensitive: false,
			newSensitive: false,
			want:         "critical",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resolveDictEntrySubmittedValue(tc.raw, tc.oldValue, tc.oldSensitive, tc.newSensitive)
			if got != tc.want {
				t.Fatalf("value=%q want %q", got, tc.want)
			}
		})
	}
}
