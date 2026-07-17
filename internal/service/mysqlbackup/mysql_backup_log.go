package mysqlbackup

import "log/slog"

func mysqlBackupLog() *slog.Logger {
	return slog.Default().With("component", "mysql.backup")
}
