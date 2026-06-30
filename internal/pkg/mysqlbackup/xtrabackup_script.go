package mysqlbackup

import (
	"fmt"
	"strings"
)

// shellResolveXtrabackupTool 解析 xtrabackup / innobackupex（兼容非交互 SSH 与 MySQL 5.7 仅装 innobackupex 的场景）。
const shellResolveXtrabackupTool = `
resolve_physical_backup_bin() {
  local kind="$1"
  local preset=""
  if [ "$kind" = "xtrabackup" ]; then preset="${XTRABACKUP_BIN_PRESET:-}"; fi
  if [ "$kind" = "innobackupex" ]; then preset="${INNOBACKUPEX_BIN_PRESET:-}"; fi
  if [ -n "$preset" ] && [ -x "$preset" ]; then echo "$preset"; return 0; fi
  if command -v "$kind" >/dev/null 2>&1; then command -v "$kind"; return 0; fi
  for p in /usr/bin/$kind /usr/local/bin/$kind /usr/local/mysql/bin/$kind \
    /export/servers/app/*/bin/$kind /opt/percona-xtrabackup*/bin/$kind; do
    if [ -x "$p" ]; then echo "$p"; return 0; fi
  done
  return 1
}
pick_physical_backup_tool() {
  XB_KIND=""
  XB_BIN=""
  local pref="${XTRABACKUP_TOOL_PRESET:-auto}"
  case "$pref" in
    xtrabackup)
      XB_BIN=$(resolve_physical_backup_bin xtrabackup) || return 1
      XB_KIND=xtrabackup
      return 0
      ;;
    innobackupex)
      XB_BIN=$(resolve_physical_backup_bin innobackupex) || return 1
      XB_KIND=innobackupex
      return 0
      ;;
    auto|*)
      if XB_BIN=$(resolve_physical_backup_bin xtrabackup); then XB_KIND=xtrabackup; return 0; fi
      if XB_BIN=$(resolve_physical_backup_bin innobackupex); then XB_KIND=innobackupex; return 0; fi
      return 1
      ;;
  esac
}
`

// XtrabackupRemoteScriptParams 远端物理热备脚本（xtrabackup 或 innobackupex，须在能访问 MySQL datadir 的机器上执行）。
type XtrabackupRemoteScriptParams struct {
	DataDir         string
	LogDir          string
	Basename        string
	MySQLPass       string // 已 shell 转义
	MySQLDir        string // 可选：MySQL datadir，留空则 SELECT @@datadir
	Parallel        int
	ToolPref        string // auto | xtrabackup | innobackupex
	XtrabackupBin   string
	InnobackupexBin string
	ConnectLog      string
	CLIConnect      string
	XBConnect       string
	ShellQuote      func(string) string
}

// BuildXtrabackupRemoteScript 执行备份 → prepare → 打包为 ${basename}.tar.gz，日志 ${basename}.log。
func BuildXtrabackupRemoteScript(p XtrabackupRemoteScriptParams) string {
	q := p.ShellQuote
	if p.Parallel <= 0 {
		p.Parallel = 4
	}
	toolPref := strings.TrimSpace(p.ToolPref)
	if toolPref == "" {
		toolPref = XtrabackupToolAuto
	}
	outDir := q(p.DataDir)
	logDir := q(p.LogDir)
	mysqlDirOverride := `""`
	if d := strings.TrimSpace(p.MySQLDir); d != "" {
		mysqlDirOverride = q(d)
	}
	tmpDir := q(p.DataDir + "/." + p.Basename + ".tmp")
	archive := q(p.DataDir + "/" + p.Basename + ".tar.gz")
	logPath := q(p.LogDir + "/" + p.Basename + ".log")

	xbPreset := ""
	if b := strings.TrimSpace(p.XtrabackupBin); b != "" {
		xbPreset = fmt.Sprintf("XTRABACKUP_BIN_PRESET=%s\n", q(b))
	}
	ibPreset := ""
	if b := strings.TrimSpace(p.InnobackupexBin); b != "" {
		ibPreset = fmt.Sprintf("INNOBACKUPEX_BIN_PRESET=%s\n", q(b))
	}

	return fmt.Sprintf(`set -euo pipefail
export PATH="/usr/bin:/bin:${PATH:-}"
XTRABACKUP_TOOL_PRESET=%s
%s%s`+shellResolveXtrabackupTool+`
pick_physical_backup_tool || {
  echo "ERROR: 未找到 xtrabackup/innobackupex。请在实例中配置 xtrabackup_bin / innobackupex_bin，或安装 Percona XtraBackup" >&2
  exit 127
}
mkdir -p %s %s
export MYSQL_PWD=%s
LOG=%s
ARCHIVE=%s
TMP=%s
MYSQL_DIR_OVERRIDE=%s
: >"$LOG"
echo "[$(date '+%%F %%T')] physical backup start tool=$XB_KIND bin=$XB_BIN %s basename=%s" >>"$LOG"
if [ -n "$MYSQL_DIR_OVERRIDE" ]; then
  MYSQL_DATADIR="$MYSQL_DIR_OVERRIDE"
else
  MYSQL_DATADIR=$(mysql --no-defaults %s -Nse "SELECT @@datadir" 2>>"$LOG" | tr -d '\r\n')
fi
MYSQL_DATADIR="${MYSQL_DATADIR%%/}"
if [ -z "$MYSQL_DATADIR" ] || [ ! -d "$MYSQL_DATADIR" ]; then
  echo "ERROR: MySQL datadir 无效或不可访问: ${MYSQL_DATADIR:-<empty>}" >>"$LOG"
  echo "提示: 物理备份须在 datadir 所在主机执行；Docker 请在实例中填写宿主机 datadir" >>"$LOG"
  exit 1
fi
echo "[$(date '+%%F %%T')] using datadir=$MYSQL_DATADIR" >>"$LOG"
rm -rf "$TMP"
if [ "$XB_KIND" = "xtrabackup" ]; then
  "$XB_BIN" --no-defaults --backup \
    --datadir="$MYSQL_DATADIR" \
    %s --password=%s \
    --target-dir="$TMP" --parallel=%d >>"$LOG" 2>&1
  "$XB_BIN" --prepare --target-dir="$TMP" >>"$LOG" 2>&1
else
  "$XB_BIN" --no-defaults --no-timestamp \
    %s --password=%s \
    --parallel=%d "$TMP" >>"$LOG" 2>&1
  _prep_ec=0
  "$XB_BIN" --prepare --target-dir="$TMP" >>"$LOG" 2>&1 || _prep_ec=$?
  if [ "$_prep_ec" -ne 0 ]; then
    echo "[$(date '+%%F %%T')] innobackupex --prepare failed ($_prep_ec), try --apply-log" >>"$LOG"
    "$XB_BIN" --apply-log "$TMP" >>"$LOG" 2>&1
  fi
fi
`+shellTarGzFromDir+`
rm -rf "$TMP"
SZ=$(stat -c%%s "$ARCHIVE" 2>/dev/null || echo 0)
echo "[$(date '+%%F %%T')] archive $ARCHIVE size=$SZ bytes" >>"$LOG"
echo "`+BackupCompletedMarker+` tool=$XB_KIND archive=$ARCHIVE size=$SZ" >>"$LOG"
tail -n 80 "$LOG" 2>/dev/null || true
`,
		q(toolPref), xbPreset, ibPreset,
		outDir, logDir, p.MySQLPass, logPath, archive, tmpDir, mysqlDirOverride,
		p.ConnectLog, p.Basename,
		p.CLIConnect,
		p.XBConnect, p.MySQLPass, p.Parallel,
		p.XBConnect, p.MySQLPass, p.Parallel,
	)
}

// DetectPhysicalBackupExecTool 从远端日志识别实际使用的物理备份工具。
func DetectPhysicalBackupExecTool(log string) string {
	if strings.Contains(log, "using tool=innobackupex") || strings.Contains(log, "tool=innobackupex ") {
		return XtrabackupToolInnobackupex
	}
	return XtrabackupToolXtrabackup
}
