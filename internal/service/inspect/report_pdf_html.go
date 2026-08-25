package inspect

import (
	"bytes"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var wkhtmltopdfCandidates = []string{
	"/usr/local/bin/wkhtmltopdf",
	"/usr/bin/wkhtmltopdf",
}

// resolveWkhtmltopdfBin 解析 wkhtmltopdf 可执行文件路径（容器内常见安装在 /usr/local/bin）。
func resolveWkhtmltopdfBin() string {
	if bin := strings.TrimSpace(os.Getenv("INSPECT_WKHTMLTOPDF_BIN")); bin != "" {
		if st, err := os.Stat(bin); err == nil && !st.IsDir() {
			return bin
		}
	}
	if p, err := exec.LookPath("wkhtmltopdf"); err == nil {
		return p
	}
	for _, c := range wkhtmltopdfCandidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

// renderPDFFromHTML 优先用 wkhtmltopdf 将 HTML 转为 PDF，保证与 HTML 版式一致。
// 未安装 wkhtmltopdf 或转换失败时返回 nil，由调用方降级结构化 PDF。
func renderPDFFromHTML(html []byte) []byte {
	if len(html) == 0 {
		return nil
	}
	bin := resolveWkhtmltopdfBin()
	if bin == "" {
		slog.Default().With("component", "inspect.pdf").Info(
			"wkhtmltopdf not found, using structured PDF fallback (install wkhtmltopdf or set INSPECT_WKHTMLTOPDF_BIN)",
		)
		return nil
	}

	tmp, err := os.CreateTemp("", "yunshu-inspect-*.html")
	if err != nil {
		slog.Default().With("component", "inspect.pdf", "error", err).Warn("create temp html for wkhtmltopdf failed")
		return nil
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(html); err != nil {
		_ = tmp.Close()
		slog.Default().With("component", "inspect.pdf", "error", err).Warn("write temp html for wkhtmltopdf failed")
		return nil
	}
	if err := tmp.Close(); err != nil {
		slog.Default().With("component", "inspect.pdf", "error", err).Warn("close temp html for wkhtmltopdf failed")
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
		"--disable-smart-shrinking",
		"--dpi", "96",
		filepath.ToSlash(tmpPath),
		"-",
	}
	cmd := exec.Command(bin, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		slog.Default().With("component", "inspect.pdf").Warn("wkhtmltopdf failed, fallback structured pdf",
			"bin", bin, "error", err, "stderr", strings.TrimSpace(stderr.String()))
		return nil
	}
	b := out.Bytes()
	if len(b) < 4 || string(b[:4]) != "%PDF" {
		slog.Default().With("component", "inspect.pdf", "stderr", strings.TrimSpace(stderr.String())).
			Warn("wkhtmltopdf output is not a valid PDF, fallback structured pdf")
		return nil
	}
	slog.Default().With("component", "inspect.pdf", "bin", bin, "bytes", len(b)).Info("wkhtmltopdf rendered inspect report pdf")
	return b
}
