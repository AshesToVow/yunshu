package inspect

import (
	"fmt"
	"os"
	"strings"
)

// renderBinaryPDF 生成服务端 PDF。默认不生成简化 PDF，由前端 html2canvas 上传高质量版。
// INSPECT_USE_WKHTMLTOPDF=true  启用 wkhtmltopdf
// INSPECT_SERVER_PDF=structured  降级结构化 PDF；html_only/off（默认）不生成服务端 PDF
func renderBinaryPDF(data ReportData, html []byte) []byte {
	if inspectUseWkhtmltopdf() {
		if pdf := renderPDFFromHTML(html); len(pdf) > 0 {
			return pdf
		}
	}
	mode := inspectServerPDFMode()
	if mode == "off" || mode == "none" || mode == "client" || mode == "html_only" {
		return nil
	}
	return renderStructuredPDF(data)
}

func inspectUseWkhtmltopdf() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("INSPECT_USE_WKHTMLTOPDF")))
	return v == "1" || v == "true" || v == "yes"
}

func inspectServerPDFMode() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("INSPECT_SERVER_PDF")))
	if v == "" {
		return "html_only"
	}
	return v
}

// renderPDFFromHTMLBytes 保留签名兼容；HTML→PDF 已取消，请改用 renderBinaryPDF(data)。
func renderPDFFromHTMLBytes(_ any, _ []byte) []byte {
	return nil
}

type pdfPage struct {
	b strings.Builder
}

func renderStructuredPDF(data ReportData) []byte {
	const (
		pageW   = 595.0
		pageH   = 842.0
		marginL = 40.0
		marginR = 40.0
		marginT = 42.0
		marginB = 46.0
	)

	pages := make([]*pdfPage, 0, 4)
	var cur *pdfPage
	y := 0.0

	newPage := func() {
		cur = &pdfPage{}
		pages = append(pages, cur)
		y = pageH - marginT
	}
	ensureSpace := func(need float64) {
		if cur == nil || y-need < marginB {
			newPage()
		}
	}
	drawRect := func(x, bottom, w, h, r, g, b float64, fill bool) {
		cur.b.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg\n", r, g, b))
		cur.b.WriteString(fmt.Sprintf("%.1f %.1f %.1f %.1f re\n", x, bottom, w, h))
		if fill {
			cur.b.WriteString("f\n")
		} else {
			cur.b.WriteString("S\n")
		}
	}
	drawLine := func(x1, y1, x2, y2, r, g, b float64) {
		cur.b.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG\n1 w\n", r, g, b))
		cur.b.WriteString(fmt.Sprintf("%.1f %.1f m %.1f %.1f l S\n", x1, y1, x2, y2))
	}
	textAt := func(x, baseline, size float64, s string) {
		cur.b.WriteString("BT\n")
		cur.b.WriteString(fmt.Sprintf("/F1 %.1f Tf\n%.1f %.1f Td\n%s Tj\n", size, x, baseline, pdfUTF16Hex(s)))
		cur.b.WriteString("ET\n")
	}
	writeWrapped := func(x, size, leading float64, maxChars int, s string) {
		for _, line := range wrapPDFLine(s, maxChars) {
			ensureSpace(leading + 2)
			textAt(x, y, size, line)
			y -= leading
		}
	}
	statusCN := func(st string) string {
		switch strings.ToLower(strings.TrimSpace(st)) {
		case "critical":
			return "严重"
		case "warning":
			return "警告"
		default:
			return "正常"
		}
	}
	pdfActionHint := func(status, errMsg string) string {
		if note := sampleNoteText(errMsg); note != "" {
			return note
		}
		switch status {
		case "critical":
			return "建议立即排查并处理，必要时扩容或修复故障。"
		case "warning":
			return "请核查相关实例指标与阈值配置。"
		default:
			return "状态正常，保持例行观察。"
		}
	}

	newPage()

	// 顶栏
	drawRect(0, pageH-28, pageW, 28, 0.07, 0.24, 0.36, true)
	textAt(marginL, pageH-18, 10, "Yunshu · 运维巡检报告")

	y -= 8
	writeWrapped(marginL, 18, 22, 28, data.Project+" 巡检报告")
	meta := data.Timestamp.Format("2006-01-02 15:04:05")
	if data.InspectionUser != "" {
		meta += "  ·  执行人 " + data.InspectionUser
	}
	if data.Datasource != "" {
		meta += "  ·  数据源 " + data.Datasource
	}
	writeWrapped(marginL, 9, 13, 70, meta)
	y -= 6
	drawLine(marginL, y, pageW-marginR, y, 0.75, 0.80, 0.84)
	y -= 16

	// 健康分
	ensureSpace(78)
	scoreBottom := y - 64
	drawRect(marginL, scoreBottom, 110, 64, 0.95, 0.97, 0.98, true)
	drawRect(marginL, scoreBottom, 110, 64, 0.82, 0.86, 0.90, false)
	textAt(marginL+18, scoreBottom+30, 26, fmt.Sprintf("%.0f", data.Score))
	textAt(marginL+14, scoreBottom+10, 9, "健康等级 "+data.Grade)

	infoX := marginL + 124
	textAt(infoX, y-4, 10, truncateRunes(data.Summary, 46))
	y -= 18
	textAt(infoX, y, 10, fmt.Sprintf("总计 %d    严重 %d    警告 %d    正常 %d",
		data.Total, data.Critical, data.Warning, data.Normal))
	if data.ReportListHint != "" {
		y -= 14
		textAt(infoX, y, 9, data.ReportListHint)
	}
	y = scoreBottom - 18

	// 一、风险与处置建议
	ensureSpace(40)
	textAt(marginL, y, 13, "一、风险与处置建议")
	y -= 6
	drawLine(marginL, y, pageW-marginR, y, 0.85, 0.88, 0.90)
	y -= 16
	if len(data.Findings) == 0 {
		writeWrapped(marginL, 10, 13, 62, "本期未发现需要特别处理的异常项。")
	} else {
		for i, f := range data.Findings {
			ensureSpace(42)
			title := fmt.Sprintf("[%s] %s · %s（影响 %d 处）", statusCN(f.Severity), f.Type, f.Name, f.Count)
			writeWrapped(marginL, 10, 13, 58, title)
			writeWrapped(marginL+8, 9, 12, 60, "建议处置："+f.Hint)
			y -= 6
			_ = i
		}
	}
	y -= 10

	// 二、巡检明细
	ensureSpace(40)
	textAt(marginL, y, 13, "二、巡检明细")
	y -= 6
	drawLine(marginL, y, pageW-marginR, y, 0.85, 0.88, 0.90)
	y -= 14

	if len(data.Groups) == 0 {
		writeWrapped(marginL, 10, 13, 62, "当前过滤模式下无异常明细。")
	} else {
		for _, g := range data.Groups {
			ensureSpace(36)
			writeWrapped(marginL, 11, 14, 50, fmt.Sprintf("%s（%d 条 · 严重 %d · 警告 %d · 正常 %d）",
				g.Type, g.Stats.Total, g.Stats.Critical, g.Stats.Warning, g.Stats.Normal))
			ensureSpace(16)
			textAt(marginL, y, 8, "检查项")
			textAt(marginL+120, y, 8, "实例")
			textAt(marginL+250, y, 8, "当前值")
			textAt(marginL+310, y, 8, "阈值")
			textAt(marginL+360, y, 8, "状态")
			y -= 4
			drawLine(marginL, y, pageW-marginR, y, 0.88, 0.90, 0.92)
			y -= 12

			for _, m := range g.Metrics {
				inst := m.Instance
				if inst == "" {
					inst = "—"
				}
				ensureSpace(28)
				textAt(marginL, y, 8, truncateRunes(m.Name, 14))
				textAt(marginL+120, y, 8, truncateRunes(inst, 12))
				textAt(marginL+250, y, 8, fmt.Sprintf("%.2f%s", m.Value, m.Unit))
				textAt(marginL+310, y, 8, fmt.Sprintf("%.2f%s", m.Threshold, m.Unit))
				textAt(marginL+360, y, 8, statusCN(m.Status))
				y -= 11
				writeWrapped(marginL+8, 8, 11, 64, "处置："+pdfActionHint(m.Status, m.Error))
				y -= 4
			}
			y -= 8
		}
	}

	// 三、覆盖范围
	if len(data.ContentGroups) > 0 {
		ensureSpace(40)
		textAt(marginL, y, 13, "三、巡检覆盖范围")
		y -= 6
		drawLine(marginL, y, pageW-marginR, y, 0.85, 0.88, 0.90)
		y -= 14
		for _, cg := range data.ContentGroups {
			names := make([]string, 0, len(cg.Items))
			for _, it := range cg.Items {
				names = append(names, it.Name)
			}
			writeWrapped(marginL, 9, 12, 58, truncateRunes(fmt.Sprintf("%s：%s", cg.Type, strings.Join(names, "、")), 120))
		}
	}

	total := len(pages)
	for i, p := range pages {
		footer := fmt.Sprintf("Yunshu 运维巡检报告 · 第 %d / %d 页 · 简化版（请查看 HTML 报告获取完整样式）", i+1, total)
		p.b.WriteString("BT\n")
		p.b.WriteString(fmt.Sprintf("/F1 8 Tf\n%.1f %.1f Td\n%s Tj\n", marginL, 24.0, pdfUTF16Hex(footer)))
		p.b.WriteString("ET\n")
	}

	streams := make([]string, len(pages))
	for i, p := range pages {
		streams[i] = p.b.String()
	}
	return assembleCJKPDFPages(streams, pageW, pageH)
}
