package inspect

// 巡检报告读取与生命周期：按类型取回报告字节、模板渲染入口、过期清理。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/platformtpl"
)

// ReadReport 按 kind（html/print/pdf/excel）取回报告内容与 Content-Type。
func (s *Service) ReadReport(ctx context.Context, projectID, runID uint, kind string) ([]byte, string, error) {
	run, err := s.GetRun(ctx, projectID, runID)
	if err != nil {
		return nil, "", err
	}
	key := strings.TrimSpace(run.ReportHTMLPath)
	ctype := "text/html; charset=utf-8"
	switch kind {
	case "pdf":
		key = strings.TrimSpace(run.ReportPDFPath)
		ctype = "application/pdf"
	case "excel", "xlsx":
		key = strings.TrimSpace(run.ReportExcelPath)
		ctype = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "print":
		// 兼容旧数据：print.html 键或回退 HTML
		if strings.Contains(run.ReportPDFPath, "print.html") {
			key = run.ReportPDFPath
		} else {
			key = reportObjectKey(projectID, runID, "print.html")
		}
		ctype = "text/html; charset=utf-8"
	}
	if key == "" {
		if kind == "pdf" {
			return nil, "", constants.ErrNotFoundWithMsg("PDF 尚未生成，请在平台点击 PDF 按钮生成")
		}
		return nil, "", constants.ErrNotFoundWithMsg("报告文件不存在")
	}

	body, err := s.readReportBytes(ctx, run, key)
	if err != nil {
		return nil, "", constants.ErrNotFoundWithMsg("报告文件不存在")
	}
	if kind == "pdf" {
		if len(body) < 4 || string(body[:4]) != "%PDF" {
			return nil, "", constants.ErrNotFoundWithMsg("PDF 尚未生成，请在平台点击 PDF 按钮生成")
		}
	}
	return body, ctype, nil
}

// readReportBytes 依次尝试当前后端、run 记录的本地后端、旧版本地路径。
func (s *Service) readReportBytes(ctx context.Context, run *model.InspectRun, key string) ([]byte, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("empty key")
	}
	store := s.store(ctx)
	if b, err := store.Get(ctx, key); err == nil {
		return b, nil
	}
	if run != nil && run.Storage == StorageLocal {
		local := newLocalReportStore(s.reportDir)
		if b, err := local.Get(ctx, key); err == nil {
			return b, nil
		}
	}
	if path, ok := resolveLegacyReportPath(s.reportDir, key); ok {
		if b, err := os.ReadFile(path); err == nil {
			return b, nil
		}
	}
	return newLocalReportStore(s.reportDir).Get(ctx, key)
}

// CleanupExpiredReports 按计划 retain_days 清理成功记录与其报告文件。
func (s *Service) CleanupExpiredReports(ctx context.Context) (int, error) {
	var plans []model.InspectPlan
	if err := s.db.WithContext(ctx).Where("retain_days > 0").Find(&plans).Error; err != nil {
		return 0, err
	}
	store := s.store(ctx)
	deleted := 0
	for _, plan := range plans {
		cutoff := time.Now().AddDate(0, 0, -plan.RetainDays)
		var runs []model.InspectRun
		if err := s.db.WithContext(ctx).
			Where("project_id = ? AND created_at < ? AND status = ?", plan.ProjectID, cutoff, "success").
			Find(&runs).Error; err != nil {
			continue
		}
		for i := range runs {
			run := &runs[i]
			for _, key := range []string{run.ReportHTMLPath, run.ReportPDFPath, run.ReportExcelPath} {
				key = strings.TrimSpace(key)
				if key == "" {
					continue
				}
				_ = store.Delete(ctx, key)
				if strings.Contains(key, string(filepath.Separator)) {
					_ = os.Remove(key)
				}
			}
			if err := s.db.WithContext(ctx).Delete(run).Error; err == nil {
				deleted++
			}
		}
	}
	return deleted, nil
}

// renderReportHTML 优先用传入正文，其次平台模板中心已发布正文，最后 embed 内置模板。
func (s *Service) renderReportHTML(ctx context.Context, code, body string, data ReportData) ([]byte, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		if overlay := s.platformReportOverlay(ctx, code); overlay != "" {
			body = overlay
		}
	}
	return renderHTMLWithTemplate(code, body, data)
}

// platformReportOverlay 平台模板中心已发布正文；未发布则空串，继续用 embed。
func (s *Service) platformReportOverlay(ctx context.Context, code string) string {
	if s == nil || s.db == nil {
		return ""
	}
	key := "inspect.report.default"
	switch strings.TrimSpace(code) {
	case "executive":
		key = "inspect.report.executive"
	case "compact":
		return ""
	case "print", "default", "":
		key = "inspect.report.default"
	default:
		key = "inspect.report." + strings.TrimSpace(code)
	}
	res, err := platformtpl.NewService(s.db).ResolvePublished(ctx, key)
	if err != nil || res == nil || res.Source != "published" {
		return ""
	}
	return strings.TrimSpace(res.Content)
}
