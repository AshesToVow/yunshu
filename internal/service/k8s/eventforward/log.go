package eventforward

import "log/slog"

func forwardLog() *slog.Logger {
	return slog.Default().With("component", "k8s.event_forward")
}
