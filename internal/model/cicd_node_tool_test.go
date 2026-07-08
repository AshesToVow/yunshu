package model

import "testing"

func TestNodeToolNameFromConfig(t *testing.T) {
	if got := NodeToolNameFromConfig(nil); got != DefaultNodeToolName {
		t.Fatalf("nil config: got %q want %q", got, DefaultNodeToolName)
	}
	if got := NodeToolNameFromConfig(&CicdCiConfig{}); got != DefaultNodeToolName {
		t.Fatalf("empty node_version: got %q", got)
	}
	if got := NodeToolNameFromConfig(&CicdCiConfig{NodeVersion: "node18"}); got != "node18" {
		t.Fatalf("got %q want node18", got)
	}
}
