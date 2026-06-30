package mysqlbackup

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var backupNameUnsafe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// BackupNameLocation 备份文件名时间戳时区（CST / Asia/Shanghai，UTC+8）。
func BackupNameLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

func formatBackupTimestamp(at time.Time) string {
	return at.In(BackupNameLocation()).Format("20060102_150405")
}

// BuildArtifactNamePrefix 备份文件名前缀（项目名_IP_端口_），用于按实例匹配已有备份。
func BuildArtifactNamePrefix(projectName, mysqlHost string, mysqlPort int) string {
	projectName = sanitizeBackupNameSegment(projectName)
	host := sanitizeBackupHost(mysqlHost)
	if mysqlPort <= 0 {
		mysqlPort = 3306
	}
	return fmt.Sprintf("%s_%s_%d_", projectName, host, mysqlPort)
}

// BuildMinioObjectKey 生成 MinIO 对象键（不含字典前缀 minio_backup_prefix）。
// 基名已含项目名、IP、端口与时间戳，直接平铺在前缀下即可。
func BuildMinioObjectKey(basename, ext string) string {
	basename = strings.TrimSpace(basename)
	ext = strings.TrimPrefix(strings.TrimSpace(ext), ".")
	if basename == "" {
		return ""
	}
	if ext == "" {
		ext = "sql.gz"
	}
	return basename + "." + ext
}

// BuildArtifactBasename 生成备份文件基名：项目名_IP_端口_年月日_时分秒（CST）。
func BuildArtifactBasename(projectName, mysqlHost string, mysqlPort int, at time.Time) string {
	projectName = sanitizeBackupNameSegment(projectName)
	host := sanitizeBackupHost(mysqlHost)
	if mysqlPort <= 0 {
		mysqlPort = 3306
	}
	ts := formatBackupTimestamp(at)
	return fmt.Sprintf("%s_%s_%d_%s", projectName, host, mysqlPort, ts)
}

func sanitizeBackupNameSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "project"
	}
	s = backupNameUnsafe.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	if s == "" {
		return "project"
	}
	return s
}

func sanitizeBackupHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "127.0.0.1"
	}
	host = strings.ReplaceAll(host, ":", "_")
	return backupNameUnsafe.ReplaceAllString(host, "_")
}
