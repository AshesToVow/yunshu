package exportutil

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"yunshu/internal/pkg/constants"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// MaxExcelExportRows 单次 Excel 导出最大行数（不含表头）。
const MaxExcelExportRows = 50000

// MaxExcelImportBytes Excel 导入文件大小上限。
const MaxExcelImportBytes = 10 << 20 // 10MB

// MaxExcelImportRows Excel 导入最大行数（不含表头）。
const MaxExcelImportRows = 10000

// LimitedImportReader 限制 Excel 导入读取大小。
func LimitedImportReader(r io.Reader) io.Reader {
	return io.LimitReader(r, MaxExcelImportBytes+1)
}

// CheckImportReadSize 校验已读取字节数是否超过导入上限。
func CheckImportReadSize(n int64) error {
	if n > MaxExcelImportBytes {
		return constants.ErrBadRequestWithMsg(fmt.Sprintf("上传文件超过 %d MB 限制", MaxExcelImportBytes>>20))
	}
	return nil
}

// ContentDispositionAttachment 生成安全的 attachment Content-Disposition（RFC 5987）。
func ContentDispositionAttachment(filename string) string {
	safe := SanitizeFilename(filename)
	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", safe, url.PathEscape(safe))
}

// SanitizeFilename 去除路径分量与危险字符。
func SanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == '"' || r == '\\' {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." {
		return "download.bin"
	}
	return name
}

// ServeBytes 先缓冲再响应，避免先写 200 后失败导致损坏文件。
func ServeBytes(c *gin.Context, filename, contentType string, body []byte) {
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", ContentDispositionAttachment(filename))
	c.Header("Content-Transfer-Encoding", "binary")
	c.Data(http.StatusOK, contentType, body)
}

// ServeExcel 将 excelize 文件写入缓冲后响应。
func ServeExcel(c *gin.Context, filename string, f *excelize.File) error {
	buf, err := f.WriteToBuffer()
	if err != nil {
		return err
	}
	ServeBytes(c, filename, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
	return nil
}

// ExportRowLimitError 导出结果超过上限时返回。
func ExportRowLimitError(count int64) error {
	return constants.ErrBadRequestWithMsg(fmt.Sprintf(
		"匹配记录共 %d 条，超过单次导出上限 %d 条，请缩小筛选条件后重试",
		count, MaxExcelExportRows,
	))
}
