package logger

import (
	"testing"

	"yunshu/internal/config"
)

func TestOpenRotatingLogFile(t *testing.T) {
	dir := t.TempDir()
	cfg := config.LogConfig{
		MaxSizeMB:  1,
		MaxAgeDays: 1,
		MaxBackups: 2,
		Compress:   false,
	}
	w := openRotatingLogFile(cfg, dir, channelInfo)
	if w == nil {
		t.Fatal("expected rotating writer")
	}
}
