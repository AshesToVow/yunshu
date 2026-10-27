package alert

import (
	"bytes"
	"context"
	"encoding/csv"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/repository"
)

const alertExportMaxRows = 5000

// ExportHisEventsCSV 导出历史告警为 CSV（最多 5000 行）。
func (s *AlertService) ExportHisEventsCSV(ctx context.Context, q AlertHisEventListQuery) ([]byte, error) {
	if s == nil || s.curHisRepo == nil {
		return nil, constants.ErrInternal
	}
	projectID, projectIDs, denyAll, err := resolveAlertEventScope(ctx, s.memberRepo, q.ProjectID)
	if err != nil {
		return nil, err
	}
	var list []model.AlertHisEvent
	if !denyAll {
		list, _, err = s.curHisRepo.ListHisEvents(ctx, repository.AlertHisEventListFilter{
			ProjectID:    projectID,
			ProjectIDs:   projectIDs,
			DatasourceID: q.DatasourceID,
			Severity:     q.Severity,
			Keyword:      q.Keyword,
		}, 0, alertExportMaxRows)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "alert.export", "ExportHisEventsCSV", err)
		}
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "alertname", "severity", "status", "cluster", "fingerprint", "summary", "resolved_at"})
	for _, row := range list {
		resolved := ""
		if !row.ResolvedAt.IsZero() {
			resolved = row.ResolvedAt.Format(time.RFC3339)
		}
		_ = w.Write([]string{
			strconv.FormatUint(uint64(row.ID), 10),
			sanitizeCSVCell(row.Alertname),
			sanitizeCSVCell(row.Severity),
			sanitizeCSVCell(row.Status),
			sanitizeCSVCell(row.Cluster),
			sanitizeCSVCell(row.Fingerprint),
			sanitizeCSVCell(strings.TrimSpace(row.Summary)),
			resolved,
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.export", "ExportHisEventsCSV", err)
	}
	return buf.Bytes(), nil
}

// sanitizeCSVCell 阻断 Excel/WPS 公式注入（= + - @ 与制表回车开头）。
func sanitizeCSVCell(v string) string {
	if v == "" {
		return v
	}
	switch v[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + v
	}
	return v
}
