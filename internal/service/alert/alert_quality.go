package alert

import (
	"context"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
)

// AlertQualityReport 告警质量治理 MVP：噪音 Top、重复指纹、通知失败率。
type AlertQualityReport struct {
	WindowHours      int                    `json:"window_hours"`
	From             string                 `json:"from"`
	To               string                 `json:"to"`
	TotalEvents      int64                  `json:"total_events"`
	NotifyFailRate   float64                `json:"notify_fail_rate"`
	NotifyFailed     int64                  `json:"notify_failed"`
	QualityScore     int                    `json:"quality_score"`
	NoiseTop         []AlertNoiseItem       `json:"noise_top"`
	RepeatFingerprints []AlertRepeatItem    `json:"repeat_fingerprints"`
	RecentChangesHint []model.ChangeEvent   `json:"recent_changes_hint,omitempty"`
}

type AlertNoiseItem struct {
	Title    string `json:"title"`
	Severity string `json:"severity"`
	Count    int64  `json:"count"`
}

type AlertRepeatItem struct {
	Fingerprint string `json:"fingerprint"`
	Title       string `json:"title"`
	Count       int64  `json:"count"`
}

func (s *AlertService) QualityReport(ctx context.Context, windowHours int, projectID uint) (*AlertQualityReport, error) {
	if s.eventRepo == nil && s.db == nil {
		return nil, constants.ErrBadRequestWithMsg("alert store unavailable")
	}
	if windowHours <= 0 {
		windowHours = 24
	}
	if windowHours > 24*14 {
		windowHours = 24 * 14
	}
	to := time.Now()
	from := to.Add(-time.Duration(windowHours) * time.Hour)
	db := s.db
	if db == nil {
		return nil, constants.ErrBadRequestWithMsg("db unavailable")
	}

	var total, failed int64
	_ = db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("created_at >= ? AND created_at <= ?", from, to).Count(&total).Error
	_ = db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("created_at >= ? AND created_at <= ? AND success = ?", from, to, false).Count(&failed).Error
	failRate := 0.0
	if total > 0 {
		failRate = float64(failed) / float64(total)
	}

	type rowCount struct {
		Title    string
		Severity string
		Count    int64
	}
	var noise []rowCount
	_ = db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("title, severity, COUNT(*) as count").
		Where("created_at >= ? AND created_at <= ?", from, to).
		Group("title, severity").Order("count DESC").Limit(10).Scan(&noise).Error
	noiseTop := make([]AlertNoiseItem, 0, len(noise))
	for _, n := range noise {
		noiseTop = append(noiseTop, AlertNoiseItem{Title: n.Title, Severity: n.Severity, Count: n.Count})
	}

	type fpRow struct {
		Fingerprint string
		Title       string
		Count       int64
	}
	var fps []fpRow
	_ = db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("fingerprint, MAX(title) as title, COUNT(*) as count").
		Where("created_at >= ? AND created_at <= ? AND fingerprint <> ''", from, to).
		Group("fingerprint").Having("COUNT(*) >= ?", 3).
		Order("count DESC").Limit(10).Scan(&fps).Error
	repeats := make([]AlertRepeatItem, 0, len(fps))
	for _, f := range fps {
		repeats = append(repeats, AlertRepeatItem{Fingerprint: f.Fingerprint, Title: f.Title, Count: f.Count})
	}

	// 质量分：100 - 失败率惩罚 - 噪音惩罚
	score := 100
	score -= int(failRate * 40)
	if len(noiseTop) > 0 && noiseTop[0].Count > 20 {
		score -= 20
	} else if len(noiseTop) > 0 && noiseTop[0].Count > 10 {
		score -= 10
	}
	if len(repeats) > 5 {
		score -= 15
	} else if len(repeats) > 0 {
		score -= 5
	}
	if score < 0 {
		score = 0
	}

	out := &AlertQualityReport{
		WindowHours:        windowHours,
		From:               from.Format(time.RFC3339),
		To:                 to.Format(time.RFC3339),
		TotalEvents:        total,
		NotifyFailRate:     failRate,
		NotifyFailed:       failed,
		QualityScore:       score,
		NoiseTop:           noiseTop,
		RepeatFingerprints: repeats,
	}
	if projectID > 0 {
		var changes []model.ChangeEvent
		_ = db.WithContext(ctx).Where("project_id = ? AND started_at >= ?", projectID, from).
			Order("id DESC").Limit(10).Find(&changes).Error
		out.RecentChangesHint = changes
	}
	_ = projectID
	return out, nil
}
