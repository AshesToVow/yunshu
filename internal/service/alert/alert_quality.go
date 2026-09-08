package alert

import (
	"context"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/repository"
)

// AlertQualityReport 告警质量治理 MVP：噪音 Top、重复指纹、通知失败率、当前 firing。
type AlertQualityReport struct {
	WindowHours        int                 `json:"window_hours"`
	From               string              `json:"from"`
	To                 string              `json:"to"`
	ProjectID          uint                `json:"project_id,omitempty"`
	TotalEvents        int64               `json:"total_events"`
	CurFiringCount     int64               `json:"cur_firing_count"`
	NotifyFailRate     float64             `json:"notify_fail_rate"`
	NotifyFailed       int64               `json:"notify_failed"`
	QualityScore       int                 `json:"quality_score"`
	NoiseTop           []AlertNoiseItem    `json:"noise_top"`
	RepeatFingerprints []AlertRepeatItem   `json:"repeat_fingerprints"`
	RecentChangesHint  []model.ChangeEvent `json:"recent_changes_hint,omitempty"`
}

type AlertNoiseItem struct {
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Count       int64  `json:"count"`
	Fingerprint string `json:"fingerprint,omitempty"`
	Alertname   string `json:"alertname,omitempty"`
}

type AlertRepeatItem struct {
	Fingerprint string `json:"fingerprint"`
	Title       string `json:"title"`
	Count       int64  `json:"count"`
	Severity    string `json:"severity,omitempty"`
	Alertname   string `json:"alertname,omitempty"`
}

func (s *AlertService) QualityReport(ctx context.Context, windowHours int, projectID uint) (*AlertQualityReport, error) {
	if s == nil || s.eventRepo == nil {
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

	stats, err := s.eventRepo.QualityWindowStats(ctx, from, to, projectID)
	if err != nil {
		return nil, err
	}
	total := stats.Total
	failed := stats.Failed
	failRate := 0.0
	if total > 0 {
		failRate = float64(failed) / float64(total)
	}

	noiseTop := make([]AlertNoiseItem, 0, len(stats.Noise))
	for _, n := range stats.Noise {
		noiseTop = append(noiseTop, AlertNoiseItem{
			Title: n.Title, Severity: n.Severity, Count: n.Count,
			Fingerprint: n.Fingerprint, Alertname: n.Alertname,
		})
	}
	repeats := make([]AlertRepeatItem, 0, len(stats.Repeats))
	for _, f := range stats.Repeats {
		repeats = append(repeats, AlertRepeatItem{
			Fingerprint: f.Fingerprint, Title: f.Title, Count: f.Count, Severity: f.Severity,
		})
	}

	var curFiring int64
	if s.curHisRepo != nil {
		_, curFiring, _ = s.curHisRepo.ListCurEvents(ctx, repository.AlertCurEventListFilter{
			ProjectID: projectID,
		}, 0, 1)
	}

	// 质量分：100 - 失败率惩罚 - 噪音惩罚 - 当前堆积惩罚
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
	if curFiring > 50 {
		score -= 15
	} else if curFiring > 20 {
		score -= 8
	}
	if score < 0 {
		score = 0
	}

	out := &AlertQualityReport{
		WindowHours:        windowHours,
		From:               from.Format(time.RFC3339),
		To:                 to.Format(time.RFC3339),
		ProjectID:          projectID,
		TotalEvents:        total,
		CurFiringCount:     curFiring,
		NotifyFailRate:     failRate,
		NotifyFailed:       failed,
		QualityScore:       score,
		NoiseTop:           noiseTop,
		RepeatFingerprints: repeats,
	}
	if projectID > 0 && s.changeEventRepo != nil {
		changes, _ := s.changeEventRepo.ListByProjectInRange(ctx, projectID, from, to, 10)
		out.RecentChangesHint = changes
	}
	return out, nil
}
