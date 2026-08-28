package alert

import (
	"testing"

	"yunshu/internal/model"
)

func TestParseAckTTLDictValues(t *testing.T) {
	t.Parallel()
	got := parseAckTTLDictValues([]model.DictEntry{
		{Value: "15"},
		{Value: " 30 "},
		{Value: "15"},
		{Value: "abc"},
		{Value: "0"},
		{Value: "99999"},
	})
	if len(got) != 3 || got[0] != 15 || got[1] != 30 || got[2] != maxAckTTLMinutes {
		t.Fatalf("parseAckTTLDictValues: %v", got)
	}
}

func TestParseAckTTLMinutes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		allowed   []int
		requested int
		want      int
		wantErr   bool
	}{
		{name: "empty dict default", allowed: nil, requested: 0, want: fallbackAckTTLMinutes},
		{name: "empty dict explicit", allowed: nil, requested: 45, want: 45},
		{name: "dict default first", allowed: []int{30, 60}, requested: 0, want: 30},
		{name: "dict match", allowed: []int{30, 60}, requested: 60, want: 60},
		{name: "dict mismatch", allowed: []int{30, 60}, requested: 15, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseAckTTLMinutes(tc.allowed, tc.requested)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}
