package mysqlbackup

import "yunshu/internal/pkg/logutil"

func mysqlBackupLog() *logutil.Component {
	return logutil.Worker("mysql.backup")
}
