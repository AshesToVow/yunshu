// SSH 拨号与远端命令工具：shell 参数转义、远端文件尾部读取。
package mysqlbackup

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/sshclient"
	"yunshu/internal/pkg/sshserver"
)

func (s *MysqlBackupService) dialServer(ctx context.Context, serverID uint) (*sshclient.Client, *model.Server, error) {
	return sshserver.DialServer(ctx, s.aead, "mysql.backup", s.serverRepo, serverID)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func (s *MysqlBackupService) tailRemoteFile(ctx context.Context, sshCli *sshclient.Client, path string, lines int) (string, error) {
	if lines <= 0 {
		lines = 50
	}
	script := fmt.Sprintf(`tail -n %d %q 2>/dev/null || true`, lines, path)
	res, err := sshCli.Exec(ctx, script, 65536)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Stdout + res.Stderr), nil
}
