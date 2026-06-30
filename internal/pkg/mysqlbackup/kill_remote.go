package mysqlbackup

import (
	"fmt"
	"path/filepath"
	"strings"
)

// BuildKillBackupByArtifactScript 按备份产物路径前缀终止远端 mysqldump/bash 管道（best-effort）。
func BuildKillBackupByArtifactScript(remotePath string) string {
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return ""
	}
	base := filepath.Base(remotePath)
	base = strings.TrimSuffix(base, ".sql.gz")
	base = strings.TrimSuffix(base, ".tar.gz")
	if base == "" {
		return ""
	}
	// 匹配 bash -c 脚本中的 LOG=/path/base.log 或 mysqldump 命令行
	return fmt.Sprintf(`pkill -f '%s' 2>/dev/null || true`, base)
}
