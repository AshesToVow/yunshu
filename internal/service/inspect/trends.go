package inspect

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// RunTrendItem 巡检执行趋势摘要（供趋势图使用）。
type RunTrendItem struct {
	ID            uint       `json:"id"`
	Score         float64    `json:"score"`
	Grade         string     `json:"grade"`
	CriticalCount int        `json:"critical_count"`
	WarningCount  int        `json:"warning_count"`
	FinishedAt    *time.Time `json:"finished_at"`
	Status        string     `json:"status"`
}

// ListRunTrends 返回项目最近巡检记录趋势（按 id 倒序）。
func (s *Service) ListRunTrends(ctx context.Context, projectID uint, limit int) ([]RunTrendItem, error) {
	if s == nil || s.db == nil || projectID == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	var runs []model.InspectRun
	err := s.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("id DESC").
		Limit(limit).
		Find(&runs).Error
	if err != nil {
		return nil, err
	}
	out := make([]RunTrendItem, 0, len(runs))
	for _, r := range runs {
		out = append(out, RunTrendItem{
			ID:            r.ID,
			Score:         r.Score,
			Grade:         r.Grade,
			CriticalCount: r.CriticalCount,
			WarningCount:  r.WarningCount,
			FinishedAt:    r.FinishedAt,
			Status:        r.Status,
		})
	}
	return out, nil
}
