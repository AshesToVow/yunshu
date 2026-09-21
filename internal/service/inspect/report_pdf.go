package inspect

// renderBinaryPDF 服务端不再生成 PDF。
// 巡检报告 PDF 仅由浏览器 html2canvas + jsPDF 导出（见 report-pdf-inject.html），
// 避免 Chromium / wkhtmltopdf 依赖与版式不一致。
func renderBinaryPDF(_ ReportData, _ []byte) []byte {
	return nil
}
