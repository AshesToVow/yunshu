package inspect

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// MigrateReportsToMinIO 将项目下仍存于本地的巡检报告迁移到 MinIO（需 MinIO 已配置）。
func (s *Service) MigrateReportsToMinIO(ctx context.Context, projectID uint) (int, error) {
	if s == nil || s.repo == nil || projectID == 0 {
		return 0, nil
	}
	info := s.ReportStorageInfo(ctx)
	if !info.MinioReady {
		return 0, fmt.Errorf("MinIO 未就绪：%s", strings.TrimSpace(info.MinioReason))
	}
	minioStore := s.store(ctx)
	if minioStore.Backend() != StorageMinio {
		return 0, fmt.Errorf("当前存储后端不是 MinIO")
	}
	local := newLocalReportStore(s.reportDir)

	runs, err := s.repo.ListLocalStorageRuns(ctx, projectID)
	if err != nil {
		return 0, err
	}
	migrated := 0
	for _, run := range runs {
		keys := []struct {
			field *string
			key   string
			ctype string
		}{
			{&run.ReportHTMLPath, reportObjectKey(projectID, run.ID, "html"), "text/html; charset=utf-8"},
			{&run.ReportPDFPath, reportObjectKey(projectID, run.ID, "pdf"), "application/pdf"},
			{&run.ReportExcelPath, reportObjectKey(projectID, run.ID, "xlsx"), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		}
		changed := false
		for _, item := range keys {
			oldKey := strings.TrimSpace(*item.field)
			if oldKey == "" {
				continue
			}
			body, err := readReportBytesFromStore(ctx, local, s.reportDir, oldKey)
			if err != nil || len(body) == 0 {
				continue
			}
			newKey := item.key
			if err := minioStore.Put(ctx, newKey, body, item.ctype); err != nil {
				return migrated, err
			}
			*item.field = newKey
			changed = true
		}
		if changed {
			run.Storage = StorageMinio
			if err := s.repo.SaveRun(ctx, &run); err != nil {
				return migrated, err
			}
			migrated++
		}
	}
	return migrated, nil
}

func readReportBytesFromStore(ctx context.Context, local ReportStore, reportDir, key string) ([]byte, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("empty key")
	}
	if body, err := local.Get(ctx, key); err == nil {
		return body, nil
	}
	if path, ok := resolveLegacyReportPath(reportDir, key); ok {
		return os.ReadFile(path)
	}
	return nil, fmt.Errorf("report not found: %s", key)
}
