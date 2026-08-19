package inspect

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// renderBinaryPDF 优先将 HTML 报告（含样式）打印为 PDF；失败时降级为文本 PDF。
func renderBinaryPDF(data ReportData) []byte {
	html, err := renderHTML(data)
	if err == nil && len(html) > 0 {
		if pdf, err := renderPDFFromHTML(context.Background(), html); err == nil && len(pdf) > 4 && string(pdf[:4]) == "%PDF" {
			return pdf
		} else if err != nil {
			slog.Warn("inspect html-to-pdf failed, fallback to text pdf", "err", err)
		}
	}
	return renderSimplePDF(data)
}

// renderPDFFromHTMLBytes 使用与归档 HTML 相同的正文生成 PDF（含 CSS）。
func renderPDFFromHTMLBytes(ctx context.Context, html []byte) []byte {
	if len(html) == 0 {
		return nil
	}
	pdf, err := renderPDFFromHTML(ctx, html)
	if err != nil || len(pdf) < 4 || string(pdf[:4]) != "%PDF" {
		if err != nil {
			slog.Warn("inspect html-to-pdf failed", "err", err)
		}
		return nil
	}
	return pdf
}

func renderPDFFromHTML(parent context.Context, html []byte) ([]byte, error) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("hide-scrollbars", true),
	)
	if bin := resolveChromeBinary(); bin != "" {
		opts = append(opts, chromedp.ExecPath(bin))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	var pdfBuf []byte
	err := chromedp.Run(taskCtx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, string(html)).Do(ctx)
		}),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(600*time.Millisecond),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				WithPaperWidth(8.27).  // A4 inches
				WithPaperHeight(11.69).
				WithMarginTop(0.4).
				WithMarginBottom(0.4).
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuf = buf
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}
	if len(pdfBuf) < 4 || string(pdfBuf[:4]) != "%PDF" {
		return nil, fmt.Errorf("chromedp returned non-pdf payload")
	}
	return pdfBuf, nil
}

func resolveChromeBinary() string {
	candidates := make([]string, 0, 16)
	for _, env := range []string{"YUNSHU_CHROME_PATH", "CHROME_PATH"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			candidates = append(candidates, v)
		}
	}
	candidates = append(candidates,
		"chromium",
		"chromium-browser",
		"google-chrome",
		"google-chrome-stable",
		"chrome",
		`/usr/bin/chromium`,
		`/usr/bin/chromium-browser`,
		`/usr/bin/google-chrome`,
		`/usr/bin/google-chrome-stable`,
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
	)
	seen := map[string]struct{}{}
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		if strings.Contains(c, `\`) || strings.HasPrefix(c, "/") {
			if st, err := os.Stat(c); err == nil && !st.IsDir() {
				return c
			}
			continue
		}
		if p, err := exec.LookPath(c); err == nil && p != "" {
			return p
		}
	}
	return ""
}

// --- 文本 PDF 降级（无浏览器时） ---

func renderSimplePDF(data ReportData) []byte {
	lines := make([]string, 0, 80)
	lines = append(lines,
		fmt.Sprintf("%s 巡检报告", data.Project),
		fmt.Sprintf("时间：%s", data.Timestamp.Format("2006-01-02 15:04:05")),
	)
	if u := strings.TrimSpace(data.InspectionUser); u != "" {
		lines = append(lines, fmt.Sprintf("执行人：%s", u))
	}
	if ds := strings.TrimSpace(data.Datasource); ds != "" {
		lines = append(lines, fmt.Sprintf("数据源：%s", ds))
	}
	lines = append(lines,
		fmt.Sprintf("健康分：%.1f（等级 %s）", data.Score, data.Grade),
		data.Summary,
		"",
		"【异常与建议】",
	)
	if len(data.Findings) == 0 {
		lines = append(lines, "本期无异常关注项。")
	} else {
		for i, f := range data.Findings {
			if i >= 25 {
				break
			}
			lines = append(lines, fmt.Sprintf("· [%s] %s / %s ×%d — %s", statusCN(f.Severity), f.Type, f.Name, f.Count, f.Hint))
		}
	}
	lines = append(lines, "", "— Yunshu 巡检报告（文本降级；请安装 Chromium 以输出带样式 PDF）—")
	return buildCJKTextPDF(lines)
}

func statusCN(s string) string {
	switch s {
	case "critical":
		return "严重"
	case "warning":
		return "警告"
	default:
		return "正常"
	}
}
