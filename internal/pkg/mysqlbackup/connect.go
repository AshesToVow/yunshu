package mysqlbackup

import (
	"fmt"
	"strings"
)

// FormatMysqlConnectShell 生成 mysql / xtrabackup 连接参数（优先 Unix Socket）。
func FormatMysqlConnectShell(socket, host string, port int, user string, quote func(string) string) (cliArgs, xbArgs, logLabel string) {
	user = strings.TrimSpace(user)
	socket = strings.TrimSpace(socket)
	if socket != "" {
		return fmt.Sprintf("-S %s -u%s", quote(socket), quote(user)),
			fmt.Sprintf("--socket=%s --user=%s", quote(socket), quote(user)),
			fmt.Sprintf("socket=%s user=%s", socket, user)
	}
	if port <= 0 {
		port = 3306
	}
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("-h%s -P%d -u%s", quote(host), port, quote(user)),
		fmt.Sprintf("--host=%s --port=%d --user=%s", quote(host), port, quote(user)),
		fmt.Sprintf("host=%s port=%d user=%s", host, port, user)
}

// FormatMysqldumpConnectArgs 生成 mysqldump 连接参数（优先 Unix Socket，避免 TCP 主机权限 1130）。
func FormatMysqldumpConnectArgs(socket, host string, port int, user string, quote func(string) string) (args string, logLabel string) {
	args, _, logLabel = FormatMysqlConnectShell(socket, host, port, user, quote)
	return args, logLabel
}

// NormalizeMysqlSocket 可选 Unix Socket 路径。
func NormalizeMysqlSocket(socket string) (string, error) {
	socket = strings.TrimSpace(socket)
	if socket == "" {
		return "", nil
	}
	if !strings.HasPrefix(socket, "/") {
		return "", fmt.Errorf("mysql_socket 须为绝对路径（以 / 开头）")
	}
	return socket, nil
}
