package inspect

import (
	"bytes"
	"embed"
	"strings"
)

//go:embed templates/report-pdf-inject.html
var reportPDFInjectRaw string

//go:embed static/*
var pdfLibFS embed.FS

const pdfLibMarker = "id=\"yunshuPdfBtn\""

// EnhanceReportHTML 为 HTML 报告注入 PDF 导出按钮与脚本（旧报告、自定义模板均可用）。
func EnhanceReportHTML(html []byte, pdfLibBase, uploadURL string) []byte {
	if len(html) == 0 || bytes.Contains(html, []byte(pdfLibMarker)) {
		return html
	}
	if pdfLibBase == "" {
		pdfLibBase = "/api/v1/inspect/pdf-libs"
	}
	inject := strings.ReplaceAll(reportPDFInjectRaw, "__PDF_LIB_BASE__", pdfLibBase)
	inject = strings.ReplaceAll(inject, "__UPLOAD_URL__", uploadURL)
	lower := bytes.ToLower(html)
	idx := bytes.LastIndex(lower, []byte("</body>"))
	if idx < 0 {
		return append(html, []byte(inject)...)
	}
	out := make([]byte, 0, len(html)+len(inject))
	out = append(out, html[:idx]...)
	out = append(out, inject...)
	out = append(out, html[idx:]...)
	return out
}

func allowedPDFLibName(name string) bool {
	switch strings.TrimSpace(name) {
	case "html2canvas.min.js", "jspdf.umd.min.js":
		return true
	default:
		return false
	}
}

// ReadPDFLib 返回内嵌的 html2canvas / jsPDF 静态库。
func ReadPDFLib(name string) ([]byte, error) {
	if !allowedPDFLibName(name) {
		return nil, errPDFLibNotFound
	}
	return pdfLibFS.ReadFile("static/" + name)
}

func readPDFLib(name string) ([]byte, error) {
	return ReadPDFLib(name)
}

var errPDFLibNotFound = errInspectPDFLibNotFound{}

type errInspectPDFLibNotFound struct{}

func (errInspectPDFLibNotFound) Error() string { return "pdf lib not found" }
