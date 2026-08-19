package inspect

import (
	"bytes"
	"fmt"
	"strings"
)

// renderSimplePDF 生成可打印的中文巡检摘要 PDF。
// 使用 Adobe 标准 CJK 字体 STSong-Light + UniGB-UCS2-H，由阅读器映射系统中文字体，
// 避免 Helvetica 将非 ASCII 替换为「?」。
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
		fmt.Sprintf("样本：总计 %d · 严重 %d · 警告 %d · 正常 %d", data.Total, data.Critical, data.Warning, data.Normal),
		data.Summary,
		"",
		"【主要发现】",
	)
	if len(data.Findings) == 0 {
		lines = append(lines, "本期无异常关注项。")
	} else {
		for i, f := range data.Findings {
			if i >= 25 {
				lines = append(lines, fmt.Sprintf("…其余 %d 项见 HTML 报告", len(data.Findings)-25))
				break
			}
			lines = append(lines, fmt.Sprintf("· [%s] %s / %s ×%d", statusCN(f.Severity), f.Type, f.Name, f.Count))
			if h := strings.TrimSpace(f.Hint); h != "" {
				lines = append(lines, "  "+h)
			}
		}
	}
	lines = append(lines, "", "【分类汇总】")
	if len(data.Groups) == 0 {
		lines = append(lines, "无分类明细。")
	} else {
		for _, g := range data.Groups {
			lines = append(lines, fmt.Sprintf("· %s：共 %d · 严重 %d · 警告 %d · 正常 %d",
				g.Type, g.Stats.Total, g.Stats.Critical, g.Stats.Warning, g.Stats.Normal))
			shown := 0
			for _, m := range g.Metrics {
				if m.Status != "critical" && m.Status != "warning" {
					continue
				}
				if shown >= 8 {
					lines = append(lines, "  …更多明细见 HTML / Excel")
					break
				}
				inst := strings.TrimSpace(m.Instance)
				if inst == "" {
					inst = "—"
				}
				line := fmt.Sprintf("  - %s | %s | 值 %s%s / 阈值 %s%s | %s",
					m.Name, inst,
					fmtFloatPDF(m.Value), m.Unit,
					fmtFloatPDF(m.Threshold), m.Unit,
					statusCN(m.Status))
				lines = append(lines, truncateRunes(line, 96))
				if note := sampleNoteText(m.Error); note != "" {
					lines = append(lines, "    "+note)
				}
				shown++
			}
		}
	}
	lines = append(lines, "", "— Yunshu 巡检报告 —")
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

func fmtFloatPDF(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func sampleNoteText(err string) string {
	err = strings.TrimSpace(err)
	if err == "" {
		return ""
	}
	switch {
	case strings.Contains(err, "无返回样本"):
		return "采集无数据：请核对指标名 / job 是否与 Telegraf、Blackbox 一致"
	case strings.Contains(err, "NaN") || strings.Contains(err, "Inf"):
		return "样本无效（NaN/Inf）"
	case strings.Contains(err, "empty query"):
		return "巡检项查询为空"
	default:
		return truncateRunes(err, 80)
	}
}

func buildCJKTextPDF(lines []string) []byte {
	const (
		pageW    = 595.0
		pageH    = 842.0
		marginL  = 48.0
		marginB  = 48.0
		fontSize = 10.0
		leading  = 14.0
		maxChars = 62
	)
	marginT := 56.0

	wrapped := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		wrapped = append(wrapped, wrapPDFLine(line, maxChars)...)
	}

	pageStreams := make([]string, 0, 4)
	var cur strings.Builder
	y := pageH - marginT
	lineIdx := 0

	beginPage := func() {
		cur.Reset()
		y = pageH - marginT
		lineIdx = 0
		cur.WriteString(fmt.Sprintf("BT\n/F1 %.0f Tf\n%.0f %.0f Td\n", fontSize, marginL, y))
	}
	flushPage := func() {
		cur.WriteString("ET\n")
		pageStreams = append(pageStreams, cur.String())
	}

	beginPage()
	for _, line := range wrapped {
		if y-leading < marginB {
			flushPage()
			beginPage()
		}
		hex := pdfUTF16Hex(line)
		if lineIdx == 0 {
			cur.WriteString(hex)
			cur.WriteString(" Tj\n")
		} else {
			cur.WriteString(fmt.Sprintf("0 -%.0f Td %s Tj\n", leading, hex))
		}
		y -= leading
		lineIdx++
	}
	flushPage()

	return assembleCJKPDFPages(pageStreams, pageW, pageH)
}

func wrapPDFLine(s string, maxChars int) []string {
	if maxChars < 8 {
		maxChars = 8
	}
	r := []rune(s)
	if len(r) == 0 {
		return []string{""}
	}
	out := make([]string, 0, (len(r)/maxChars)+1)
	for len(r) > maxChars {
		out = append(out, string(r[:maxChars]))
		r = r[maxChars:]
	}
	out = append(out, string(r))
	return out
}

// pdfUTF16Hex 生成 UniGB-UCS2-H 所需的 UTF-16BE 十六进制字符串。
func pdfUTF16Hex(s string) string {
	var b strings.Builder
	b.Grow(len(s)*4 + 2)
	b.WriteByte('<')
	for _, r := range s {
		if r > 0xFFFF {
			r -= 0x10000
			hi := 0xD800 + (r >> 10)
			lo := 0xDC00 + (r & 0x3FF)
			b.WriteString(fmt.Sprintf("%04X%04X", hi, lo))
			continue
		}
		b.WriteString(fmt.Sprintf("%04X", r))
	}
	b.WriteByte('>')
	return b.String()
}

func assembleCJKPDFPages(pageStreams []string, pageW, pageH float64) []byte {
	if len(pageStreams) == 0 {
		pageStreams = []string{"BT /F1 10 Tf 48 800 Td <> Tj ET\n"}
	}

	const fixedObjs = 5
	nPages := len(pageStreams)
	kids := make([]string, 0, nPages)
	for i := 0; i < nPages; i++ {
		pageObj := fixedObjs + 1 + i*2
		kids = append(kids, fmt.Sprintf("%d 0 R", pageObj))
	}

	objs := make([]string, 0, fixedObjs+nPages*2)
	objs = append(objs,
		"1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj\n",
		fmt.Sprintf("2 0 obj<< /Type /Pages /Kids [%s] /Count %d >>endobj\n", strings.Join(kids, " "), nPages),
		"3 0 obj<< /Type /Font /Subtype /Type0 /BaseFont /STSong-Light /Encoding /UniGB-UCS2-H /DescendantFonts [4 0 R] >>endobj\n",
		"4 0 obj<< /Type /Font /Subtype /CIDFontType0 /BaseFont /STSong-Light /CIDSystemInfo << /Registry (Adobe) /Ordering (GB1) /Supplement 2 >> /FontDescriptor 5 0 R /DW 1000 >>endobj\n",
		"5 0 obj<< /Type /FontDescriptor /FontName /STSong-Light /Flags 6 /FontBBox [-250 -183 463 866] /ItalicAngle 0 /Ascent 752 /Descent -271 /CapHeight 737 /StemV 58 >>endobj\n",
	)

	for i, stream := range pageStreams {
		pageObjNum := fixedObjs + 1 + i*2
		contentObjNum := pageObjNum + 1
		objs = append(objs, fmt.Sprintf(
			"%d 0 obj<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.0f %.0f] /Contents %d 0 R /Resources << /Font << /F1 3 0 R >> >> >>endobj\n",
			pageObjNum, pageW, pageH, contentObjNum,
		))
		objs = append(objs, fmt.Sprintf(
			"%d 0 obj<< /Length %d >>stream\n%s\nendstream\nendobj\n",
			contentObjNum, len(stream), stream,
		))
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objs)+1)
	for i, o := range objs {
		offsets[i+1] = buf.Len()
		buf.WriteString(o)
	}
	xref := buf.Len()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n", len(objs)+1))
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objs); i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	buf.WriteString(fmt.Sprintf("trailer<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref))
	return buf.Bytes()
}
