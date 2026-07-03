package alert

import "log/slog"

func alertLog() *slog.Logger {
	return slog.Default().With("component", "alert")
}
