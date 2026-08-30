package inspect

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// chromiumLookupNames Chromium/Chrome/Edge 在 PATH 中的常见可执行名。
var chromiumLookupNames = []string{
	"chromium",
	"chromium-browser",
	"google-chrome",
	"google-chrome-stable",
	"headless-shell",
	"headless_shell",
}

// chromiumCandidates 容器与桌面环境下的常见安装路径。
var chromiumCandidates = map[string][]string{
	"linux": {
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/usr/local/bin/chromium",
		"/usr/local/bin/headless-shell",
		"/opt/google/chrome/chrome",
	},
	"windows": {
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	},
	"darwin": {
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	},
}

// resolveChromiumBin 解析 headless Chromium 可执行文件路径。
// 优先 INSPECT_CHROMIUM_BIN，其次 PATH，最后按平台探测常见安装位置。
func resolveChromiumBin() string {
	if bin := strings.TrimSpace(os.Getenv("INSPECT_CHROMIUM_BIN")); bin != "" {
		if st, err := os.Stat(bin); err == nil && !st.IsDir() {
			return bin
		}
		slog.Default().With("component", "inspect.pdf", "bin", bin).
			Warn("INSPECT_CHROMIUM_BIN not found, fallback to auto detect")
	}
	for _, name := range chromiumLookupNames {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	for _, c := range chromiumCandidates[runtime.GOOS] {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

// chromiumPDFTimeout 单次渲染超时，默认 90s，可用 INSPECT_PDF_TIMEOUT_SECONDS 调整。
func chromiumPDFTimeout() time.Duration {
	const def = 90 * time.Second
	v := strings.TrimSpace(os.Getenv("INSPECT_PDF_TIMEOUT_SECONDS"))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v + "s")
	if err != nil || d <= 0 {
		return def
	}
	return d
}

// renderPDFWithChromium 用 headless Chromium 把报告 HTML 渲染为矢量 PDF。
//
// 相比 wkhtmltopdf（Qt WebKit，不支持 CSS Grid / flex gap，会把报告排版打散），
// Chromium 完整支持模板使用的 grid 布局与 @page 规则，且输出带文字层：
// 可搜索、可复制、体积小；分页由模板的 break-inside/@page 控制，不会把表格行切两半。
//
// 渲染失败或未安装时返回 nil，由调用方按顺序降级。
func renderPDFWithChromium(html []byte) []byte {
	if len(html) == 0 {
		return nil
	}
	log := slog.Default().With("component", "inspect.pdf", "renderer", "chromium")
	bin := resolveChromiumBin()
	if bin == "" {
		log.Info("chromium not found, skip vector pdf (install chromium or set INSPECT_CHROMIUM_BIN)")
		return nil
	}

	workDir, err := os.MkdirTemp("", "yunshu-inspect-pdf-")
	if err != nil {
		log.Warn("create temp dir failed", "error", err)
		return nil
	}
	defer os.RemoveAll(workDir)

	htmlPath := filepath.Join(workDir, "report.html")
	if err := os.WriteFile(htmlPath, html, 0o600); err != nil {
		log.Warn("write temp html failed", "error", err)
		return nil
	}
	outPath := filepath.Join(workDir, "report.pdf")

	ctx, cancel := context.WithTimeout(context.Background(), chromiumPDFTimeout())
	defer cancel()

	args := []string{
		"--headless=new",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--disable-extensions",
		"--disable-background-networking",
		"--hide-scrollbars",
		"--no-first-run",
		"--no-default-browser-check",
		"--user-data-dir=" + filepath.Join(workDir, "profile"),
		"--run-all-compositor-stages-before-draw",
		"--virtual-time-budget=10000",
		// 新旧版本的「不打印页眉页脚」开关名不同，一起传；Chromium 会忽略未知开关。
		"--no-pdf-header-footer",
		"--print-to-pdf-no-header",
		"--print-to-pdf=" + outPath,
		fileURL(htmlPath),
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	combined, runErr := cmd.CombinedOutput()
	stderr := strings.TrimSpace(string(combined))
	if runErr != nil {
		log.Warn("chromium render failed, fallback next renderer", "bin", bin, "error", runErr, "output", truncateLog(stderr))
		return nil
	}

	pdf, err := os.ReadFile(outPath)
	if err != nil {
		log.Warn("read chromium pdf failed", "error", err, "output", truncateLog(stderr))
		return nil
	}
	if len(pdf) < 4 || string(pdf[:4]) != "%PDF" {
		log.Warn("chromium output is not a valid pdf", "bytes", len(pdf), "output", truncateLog(stderr))
		return nil
	}
	log.Info("chromium rendered inspect report pdf", "bin", bin, "bytes", len(pdf))
	return pdf
}

// fileURL 把本地路径转成 file:// URL（Windows 下需要额外的 / 前缀）。
func fileURL(path string) string {
	p := filepath.ToSlash(path)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return "file://" + p
}

func truncateLog(s string) string {
	const max = 600
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
