package logger

import (
	"log/slog"
	"testing"

	"yunshu/internal/config"
)

func TestInitSetsDefault(t *testing.T) {
	l := New(config.LogConfig{Level: "debug", Output: "console"})
	Init(l)
	if slog.Default() == nil {
		t.Fatal("expected slog.Default to be set")
	}
}
