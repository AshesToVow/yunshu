package inspect

import (
	"bytes"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

// renderPDFFromHTML 优先用 wkhtmltopdf 将 HTML 转为 PDF，保证与 HTML 版式一致。
// 未安装 wkhtmltopdf 或转换失败时返回 nil，由调用方降级结构化 PDF。
func renderPDFFromHTML(html []byte) []byte {
	if len(html) == 0 {
		return nil
	}
	bin := strings.TrimSpace(os.Getenv("INSPECT_WKHTMLTOPDF_BIN"))
	if bin == "" {
		bin = "wkhtmltopdf"
	}
	if _, err := exec.LookPath(bin); err != nil {
		return nil
	}

	args := []string{
		"--quiet",
		"--encoding", "utf-8",
		"--page-size", "A4",
		"--margin-top", "10mm",
		"--margin-bottom", "10mm",
		"--margin-left", "10mm",
		"--margin-right", "10mm",
		"--enable-local-file-access",
		"--print-media-type",
		"-", "-",
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdin = bytes.NewReader(html)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		slog.Default().With("component", "inspect.pdf").Warn("wkhtmltopdf failed, fallback structured pdf",
			"error", err, "stderr", strings.TrimSpace(stderr.String()))
		return nil
	}
	b := out.Bytes()
	if len(b) < 4 || string(b[:4]) != "%PDF" {
		return nil
	}
	return b
}
