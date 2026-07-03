package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestWithRequestIDInLogs(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	info := slog.New(wrapHandler(h, channelInfo))
	errH := slog.New(wrapHandler(h, channelError))

	l := &Logger{Info: info, Error: errH, SQL: info, Default: slog.New(&routeHandler{info: wrapHandler(h, channelInfo), err: wrapHandler(h, channelError)})}
	Init(l)

	ctx := WithRequestID(context.Background(), "req-abc")
	With(ctx, "component", "test").Info("hello", "k", "v")

	if !strings.Contains(buf.String(), "req-abc") {
		t.Fatalf("expected request_id in log, got: %s", buf.String())
	}
}

func TestWithUserInLogs(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	info := slog.New(wrapHandler(h, channelInfo))
	errH := slog.New(wrapHandler(h, channelError))

	l := &Logger{Info: info, Error: errH, SQL: info, Default: slog.New(&routeHandler{info: wrapHandler(h, channelInfo), err: wrapHandler(h, channelError)})}
	Init(l)

	ctx := WithUser(context.Background(), 42, "alice")
	With(ctx, "component", "test").Info("user action")

	out := buf.String()
	if !strings.Contains(out, "alice") || !strings.Contains(out, "42") {
		t.Fatalf("expected user fields in log, got: %s", out)
	}
}
