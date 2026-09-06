package inspect

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"yunshu/internal/pkg/exportutil"
)

// reportDownloadFilename 生成下载友好的报告文件名：优先项目名，附带 run id 避免重名覆盖。
func reportDownloadFilename(projectName string, runID uint, ext string) string {
	ext = strings.TrimPrefix(strings.TrimSpace(ext), ".")
	if ext == "" {
		ext = "bin"
	}
	base := strings.TrimSpace(projectName)
	base = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		default:
			if unicode.IsControl(r) {
				return -1
			}
			return r
		}
	}, base)
	base = strings.Trim(base, " .")
	if base == "" {
		base = fmt.Sprintf("inspect-run-%d", runID)
	} else if runID > 0 {
		base = fmt.Sprintf("%s-巡检-%d", base, runID)
	}
	name := base + "." + ext
	return exportutil.SanitizeFilename(filepath.Base(name))
}
