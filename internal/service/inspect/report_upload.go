package inspect

import (
	"context"
	"strings"

	"yunshu/internal/pkg/constants"
)

type ReportPDFStatus struct {
	Exists   bool   `json:"exists"`
	Size     int    `json:"size,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// CheckReportPDF 检查是否已有高质量 PDF（非结构化降级版）。
func (s *Service) CheckReportPDF(ctx context.Context, projectID, runID uint) (*ReportPDFStatus, error) {
	run, err := s.GetRun(ctx, projectID, runID)
	if err != nil {
		return nil, err
	}
	filename := reportDownloadFilename(s.projectName(ctx, projectID), runID, "pdf")
	key := strings.TrimSpace(run.ReportPDFPath)
	if key == "" {
		return &ReportPDFStatus{Exists: false, Filename: filename}, nil
	}
	body, err := s.readReportBytes(ctx, run, key)
	if err != nil || len(body) < 4 || string(body[:4]) != "%PDF" {
		return &ReportPDFStatus{Exists: false, Filename: filename}, nil
	}
	return &ReportPDFStatus{
		Exists:   true,
		Size:     len(body),
		Filename: filename,
	}, nil
}

func (s *Service) projectName(ctx context.Context, projectID uint) string {
	if s == nil || s.projects == nil || projectID == 0 {
		return ""
	}
	p, err := s.projects.GetByID(ctx, projectID)
	if err != nil || p == nil {
		return ""
	}
	return strings.TrimSpace(p.Name)
}

// ReportDownloadFilename 对外提供报告下载文件名（含项目名）。
func (s *Service) ReportDownloadFilename(ctx context.Context, projectID, runID uint, kind string) string {
	ext := "bin"
	switch kind {
	case "pdf":
		ext = "pdf"
	case "excel":
		ext = "xlsx"
	case "html", "print":
		ext = "html"
	}
	return reportDownloadFilename(s.projectName(ctx, projectID), runID, ext)
}

// SaveReportPDF 保存前端 html2canvas 生成的高质量 PDF。
func (s *Service) SaveReportPDF(ctx context.Context, projectID, runID uint, pdf []byte) error {
	if len(pdf) < 4 || string(pdf[:4]) != "%PDF" {
		return constants.ErrBadRequestWithMsg("无效的 PDF 文件")
	}
	run, err := s.GetRun(ctx, projectID, runID)
	if err != nil {
		return err
	}
	if run.Status != "success" {
		return constants.ErrBadRequestWithMsg("仅成功的巡检可保存 PDF")
	}
	key := reportObjectKey(projectID, runID, "pdf")
	store := s.store(ctx)
	if err := store.Put(ctx, key, pdf, "application/pdf"); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(run).Updates(map[string]any{
		"report_pdf_path": key,
	}).Error
}
