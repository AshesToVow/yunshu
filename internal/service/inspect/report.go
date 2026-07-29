package inspect

import (
	"bytes"
	"embed"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

//go:embed templates/*.html
var reportFS embed.FS

// ReportData HTML 报告数据。
type ReportData struct {
	Timestamp      time.Time
	Project        string
	Datasource     string
	InspectionUser string
	Score          float64
	Grade          string
	Summary        string
	ReportListMode string
	ReportListHint string
	Total          int
	Critical       int
	Warning        int
	Normal         int
	Groups         []ReportGroup
	Findings       []Finding
}

type ReportGroup struct {
	Type    string
	Metrics []MetricSample
	Stats   GroupStats
}

type GroupStats struct {
	Total    int
	Critical int
	Warning  int
	Normal   int
}

type Finding struct {
	Type     string
	Name     string
	Severity string
	Count    int
	Hint     string
}

func buildReportData(projectName, dsName, user, listMode string, collected CollectResult) ReportData {
	mode := strings.TrimSpace(listMode)
	if mode == "" {
		mode = "abnormal_only"
	}
	filtered := filterSamples(collected.Samples, mode)
	byType := map[string][]MetricSample{}
	for _, s := range filtered {
		t := s.Type
		if t == "" {
			t = "未分类"
		}
		byType[t] = append(byType[t], s)
	}
	types := make([]string, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)
	groups := make([]ReportGroup, 0, len(types))
	for _, t := range types {
		ms := byType[t]
		st := GroupStats{}
		for _, m := range ms {
			st.Total++
			switch m.Status {
			case "critical":
				st.Critical++
			case "warning":
				st.Warning++
			default:
				st.Normal++
			}
		}
		groups = append(groups, ReportGroup{Type: t, Metrics: ms, Stats: st})
	}
	score, grade := scorecard(collected)
	findings := buildFindings(collected.Samples)
	hint := "展示全部样本"
	switch mode {
	case "abnormal_only":
		hint = "仅展示异常（critical/warning）样本"
	case "summary":
		hint = "摘要模式：每类仅展示异常样本"
	}
	return ReportData{
		Timestamp:      time.Now(),
		Project:        projectName,
		Datasource:     dsName,
		InspectionUser: user,
		Score:          score,
		Grade:          grade,
		Summary:        fmt.Sprintf("共 %d 条样本：严重 %d、警告 %d、正常 %d。健康分 %.0f（%s）。", collected.Total, collected.Critical, collected.Warning, collected.Normal, score, grade),
		ReportListMode: mode,
		ReportListHint: hint,
		Total:          collected.Total,
		Critical:       collected.Critical,
		Warning:        collected.Warning,
		Normal:         collected.Normal,
		Groups:         groups,
		Findings:       findings,
	}
}

func filterSamples(samples []MetricSample, mode string) []MetricSample {
	switch mode {
	case "all":
		return samples
	case "summary", "abnormal_only":
		out := make([]MetricSample, 0, len(samples))
		for _, s := range samples {
			if s.Status == "critical" || s.Status == "warning" || s.Error != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return samples
	}
}

func scorecard(c CollectResult) (float64, string) {
	if c.Total == 0 {
		return 100, "A"
	}
	// 严重扣 8 分、警告扣 3 分，下限 0
	score := 100.0 - float64(c.Critical)*8 - float64(c.Warning)*3
	if score < 0 {
		score = 0
	}
	score = math.Round(score*10) / 10
	grade := "A"
	switch {
	case score < 60:
		grade = "D"
	case score < 75:
		grade = "C"
	case score < 90:
		grade = "B"
	}
	return score, grade
}

func buildFindings(samples []MetricSample) []Finding {
	type key struct{ t, n, sev string }
	counts := map[key]int{}
	for _, s := range samples {
		if s.Status != "critical" && s.Status != "warning" {
			continue
		}
		k := key{t: s.Type, n: s.Name, sev: s.Status}
		counts[k]++
	}
	out := make([]Finding, 0, len(counts))
	for k, n := range counts {
		hint := "请核查相关实例指标与阈值配置。"
		if k.sev == "critical" {
			hint = "建议立即排查并处理，必要时扩容或修复故障。"
		}
		out = append(out, Finding{Type: k.t, Name: k.n, Severity: k.sev, Count: n, Hint: hint})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity < out[j].Severity // critical before warning alphabetically? critical < warning
		}
		return out[i].Count > out[j].Count
	})
	if len(out) > 30 {
		out = out[:30]
	}
	return out
}

func renderHTML(data ReportData) ([]byte, error) {
	return renderHTMLWithTemplate("default", "", data)
}

func renderBinaryPDF(data ReportData) []byte {
	return renderSimplePDF(data)
}

// renderSimplePDF 生成纯文本摘要 PDF（无第三方依赖）。
func renderSimplePDF(data ReportData) []byte {
	lines := []string{
		fmt.Sprintf("Yunshu Inspect Report — %s", data.Project),
		fmt.Sprintf("Time: %s", data.Timestamp.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("Datasource: %s", data.Datasource),
		fmt.Sprintf("Score: %.1f (%s)", data.Score, data.Grade),
		data.Summary,
		"",
		"Top findings:",
	}
	for i, f := range data.Findings {
		if i >= 20 {
			break
		}
		lines = append(lines, fmt.Sprintf("- [%s] %s/%s x%d", f.Severity, f.Type, f.Name, f.Count))
	}
	return buildTextPDF(lines)
}

func buildTextPDF(lines []string) []byte {
	var content strings.Builder
	y := 800
	content.WriteString("BT /F1 11 Tf 50 " + fmt.Sprintf("%d", y) + " Td\n")
	for i, line := range lines {
		esc := pdfEscape(line)
		if i == 0 {
			content.WriteString(fmt.Sprintf("(%s) Tj\n", esc))
		} else {
			content.WriteString(fmt.Sprintf("0 -16 Td (%s) Tj\n", esc))
		}
	}
	content.WriteString("ET\n")
	stream := content.String()
	objs := []string{
		"1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj\n",
		"2 0 obj<< /Type /Pages /Kids [3 0 R] /Count 1 >>endobj\n",
		"3 0 obj<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>endobj\n",
		fmt.Sprintf("4 0 obj<< /Length %d >>stream\n%s\nendstream\nendobj\n", len(stream), stream),
		"5 0 obj<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>endobj\n",
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
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

func pdfEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	// PDF Helvetica 对非 ASCII 不友好，替换为 ?
	var b strings.Builder
	for _, r := range s {
		if r < 128 {
			b.WriteRune(r)
		} else {
			b.WriteByte('?')
		}
	}
	return b.String()
}
