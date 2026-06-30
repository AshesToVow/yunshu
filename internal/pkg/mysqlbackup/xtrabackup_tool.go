package mysqlbackup

import (
	"fmt"
	"strings"
)

const (
	XtrabackupToolAuto        = "auto"
	XtrabackupToolXtrabackup  = "xtrabackup"
	XtrabackupToolInnobackupex = "innobackupex"
)

// NormalizeXtrabackupTool 物理备份工具偏好：auto | xtrabackup | innobackupex。
func NormalizeXtrabackupTool(tool string) (string, error) {
	tool = strings.TrimSpace(strings.ToLower(tool))
	if tool == "" {
		return XtrabackupToolAuto, nil
	}
	switch tool {
	case XtrabackupToolAuto, XtrabackupToolXtrabackup, XtrabackupToolInnobackupex:
		return tool, nil
	default:
		return "", fmt.Errorf("xtrabackup_tool 须为 auto、xtrabackup 或 innobackupex")
	}
}

// NormalizeXtrabackupBin 可选 xtrabackup 可执行文件绝对路径。
func NormalizeXtrabackupBin(bin string) (string, error) {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		return "", nil
	}
	if !strings.HasPrefix(bin, "/") {
		return "", fmt.Errorf("xtrabackup_bin 须为绝对路径（以 / 开头）")
	}
	return bin, nil
}

// NormalizeInnobackupexBin 可选 innobackupex 可执行文件绝对路径。
func NormalizeInnobackupexBin(bin string) (string, error) {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		return "", nil
	}
	if !strings.HasPrefix(bin, "/") {
		return "", fmt.Errorf("innobackupex_bin 须为绝对路径（以 / 开头）")
	}
	return bin, nil
}
