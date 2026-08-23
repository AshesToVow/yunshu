package alert

import (
	"bytes"
	"context"
	"encoding/csv"
	"strconv"
	"strings"
	"time"

	bizerrors "yunshu/internal/pkg/errors"
)

// ExportHisEventsCSV 导出历史告警为 CSV（最多 5000 行）。
func (s *AlertService) ExportHisEventsCSV(ctx context.Context, q AlertHisEventListQuery) ([]byte, error) {
	q.Page = 1
	q.PageSize = 5000
	list, _, _, _, err := s.ListHisEvents(ctx, q)
	if err != nil {
		return nil, err
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
			row.Alertname,
			row.Severity,
			row.Status,
			row.Cluster,
			row.Fingerprint,
			strings.TrimSpace(row.Summary),
			resolved,
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.export", "ExportHisEventsCSV", err)
	}
	return buf.Bytes(), nil
}
