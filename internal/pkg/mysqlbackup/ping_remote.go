package mysqlbackup

import (
	"fmt"
	"strings"
)

// DeriveMysqlBinFromMysqldump 由 mysqldump 路径推导同目录 mysql 客户端路径。
func DeriveMysqlBinFromMysqldump(mysqldumpBin string) string {
	mysqldumpBin = strings.TrimSpace(mysqldumpBin)
	if mysqldumpBin == "" {
		return ""
	}
	return strings.Replace(mysqldumpBin, "mysqldump", "mysql", 1)
}

// BuildMysqlPingRemoteScript 在目标服务器上检测 MySQL 连通（与备份脚本一致走 SSH 远端执行）。
func BuildMysqlPingRemoteScript(socket, host string, port int, user, password, mysqldumpBin string, quote func(string) string) string {
	connectArgs, connectLog := FormatMysqldumpConnectArgs(socket, host, port, user, quote)
	binPreset := ""
	if bin := strings.TrimSpace(DeriveMysqlBinFromMysqldump(mysqldumpBin)); bin != "" {
		binPreset = fmt.Sprintf("MYSQLDUMP_BIN_PRESET=%s\nMYSQL_BIN_PRESET=%s\n", quote(strings.TrimSpace(mysqldumpBin)), quote(bin))
	}
	return fmt.Sprintf(`set -euo pipefail
export PATH="/usr/bin:/bin:${PATH:-}"
export MYSQL_PWD=%s
%s`+shellResolveMysql+`
ERR=$(mktemp)
MYSQL_BIN=$(resolve_mysql) || {
  echo "mysqlping,%s status=0i error=mysql_client_not_found"
  rm -f "$ERR"
  exit 1
}
if "$MYSQL_BIN" --no-defaults %s -e "SELECT 1" 2>"$ERR"; then
  echo "mysqlping,%s status=1i bin=$MYSQL_BIN"
  rm -f "$ERR"
  exit 0
fi
MSG=$(tr '\n' ' ' <"$ERR" | sed 's/[[:space:]]\+/ /g' | cut -c1-240)
rm -f "$ERR"
echo "mysqlping,%s status=0i bin=$MYSQL_BIN error=$MSG"
exit 1
`, quote(password), binPreset, connectLog, connectArgs, connectLog, connectLog)
}
