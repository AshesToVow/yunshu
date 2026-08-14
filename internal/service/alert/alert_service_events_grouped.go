package alert

import (
	"context"
	"strings"

	"yunshu/internal/pkg/pagination"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/repository"
)

type AlertEventGroupItem struct {
	GroupKey string `json:"group_key"`
	Title    string `json:"title"`
	Count    int64  `json:"count"`
	LastAt   string `json:"last_at"`
	Status   string `json:"status"`
	Severity string `json:"severity"`
	Cluster  string `json:"cluster"`
}

func (s *AlertService) ListEventsGrouped(ctx context.Context, q AlertEventListQuery) (list []AlertEventGroupItem, total int64, page int, pageSize int, err error) {
	s.ensureAlertEventProjectBackfill(ctx)
	page, pageSize = pagination.Normalize(q.Page, q.PageSize)
	f := repository.AlertEventListFilter{
		Keyword: q.Keyword, Cluster: q.Cluster, AlertIP: q.AlertIP, Status: q.Status,
		Severity: q.Severity, MonitorPipeline: q.MonitorPipeline, DatasourceID: q.DatasourceID, ProjectID: q.ProjectID,
		GroupKey: q.GroupKey, Fingerprint: q.Fingerprint,
	}
	if v := strings.TrimSpace(q.Category); v != "" && ValidAlertEventCategory(v) {
		f.Category = v
	}
	rows, total, err := s.eventRepo.ListGroupedByGroupKey(ctx, f, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert", "ListEventsGrouped", err)
	}
	list = make([]AlertEventGroupItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, AlertEventGroupItem{
			GroupKey: row.GroupKey,
			Title:    row.Title,
			Count:    row.Count,
			LastAt:   row.LastAt.Format("2006-01-02 15:04:05"),
			Status:   row.Status,
			Severity: row.Severity,
			Cluster:  row.Cluster,
		})
	}
	return list, total, page, pageSize, nil
}
