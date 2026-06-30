package mysqlbackup

import (
	"fmt"
	"strings"
)

// MysqldumpRemoteScriptParams 远端 mysqldump 脚本参数（Linux bash）。
type MysqldumpRemoteScriptParams struct {
	WorkDir     string
	Basename    string
	MySQLHost   string
	MySQLPort   int
	MySQLUser   string
	MySQLPass   string // 已 shell 转义
	DumpFlags   string
	DumpTarget      string // 已格式化目标库表参数
	DumpTargetLabel string // 可读目标描述（写入日志）
	MysqldumpBin string // 可选，远端 mysqldump 绝对路径
	ConnectArgs  string // 已格式化 -S/-h 连接参数
	ConnectLog   string // 日志可读连接描述
	ShellQuote   func(string) string
}

// BuildMysqldumpRemoteScript 执行 mysqldump，stderr/进度写入 .log，数据写入 .sql.gz。
func BuildMysqldumpRemoteScript(p MysqldumpRemoteScriptParams) string {
	q := p.ShellQuote
	workDir := q(p.WorkDir)
	logPath := q(p.WorkDir + "/" + p.Basename + ".log")
	sqlPath := q(p.WorkDir + "/" + p.Basename + ".sql.gz")
	binPreset := ""
	if strings.TrimSpace(p.MysqldumpBin) != "" {
		binPreset = fmt.Sprintf("MYSQLDUMP_BIN_PRESET=%s\n", q(strings.TrimSpace(p.MysqldumpBin)))
	}
	return fmt.Sprintf(`set -euo pipefail
mkdir -p %s
LOG=%s
SQL=%s
export MYSQL_PWD=%s
	echo "[$(date '+%%F %%T')] mysqldump start %s -> $SQL" > "$LOG"
( while sleep 30; do echo "[$(date '+%%F %%T')] progress sql.gz $(stat -c%%s "$SQL" 2>/dev/null || echo 0) bytes"; done >>"$LOG" ) &
MON=$!
%s` + shellMysqldumpToGz + `
kill "$MON" 2>/dev/null || true
wait "$MON" 2>/dev/null || true
SZ=$(stat -c%%s "$SQL" 2>/dev/null || echo 0)
echo "[$(date '+%%F %%T')] mysqldump exit=$EC sql.gz $SZ bytes" >>"$LOG"
if [ "$EC" -eq 0 ] && [ "$SZ" -lt 1 ]; then
  echo "ERROR: sql.gz empty" >>"$LOG"
  exit 1
fi
if [ "$EC" -eq 0 ]; then
  echo "` + BackupCompletedMarker + ` file=$SQL size=$SZ" >>"$LOG"
fi
tail -n 120 "$LOG" 2>/dev/null || true
exit $EC
`,
		workDir, logPath, sqlPath, p.MySQLPass,
		p.ConnectLog,
		binPreset,
		p.ConnectArgs, p.DumpFlags, p.DumpTarget, p.ConnectLog, q(p.DumpTargetLabel), p.DumpFlags,
	)
}
