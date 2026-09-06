package project

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
)

// PortraitHealth 解释型健康分（告警 / 发布 / Ready 三因子）。
type PortraitHealth struct {
	Score       int                  `json:"score"`
	Grade       string               `json:"grade"`
	Factors     []PortraitHealthFactor `json:"factors"`
	CheckedAt   string               `json:"checked_at"`
}

type PortraitHealthFactor struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Score   int    `json:"score"`
	Max     int    `json:"max"`
	Detail  string `json:"detail"`
	Deduct  int    `json:"deduct"`
}

func (s *ServiceCatalogService) buildHealth(ctx context.Context, item *ServiceCatalogItem) *PortraitHealth {
	if item == nil || s.db == nil {
		return nil
	}
	now := time.Now()
	since := now.Add(-24 * time.Hour)
	factors := make([]PortraitHealthFactor, 0, 4)

	// 告警因子：满分 40；firing critical -15/条(上限30)，warning -5/条(上限10)
	alertScore, alertMax := 40, 40
	var firingCritical, firingWarning int64
	_ = s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("status = ? AND severity = ? AND created_at >= ?", "firing", "critical", since).
		Count(&firingCritical).Error
	_ = s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("status = ? AND severity = ? AND created_at >= ?", "firing", "warning", since).
		Count(&firingWarning).Error
	// 若有 monitor 绑定，优先按 fingerprint 粗过滤不可行时仍用项目级，并在 detail 标明
	deductAlert := int(firingCritical)*15 + int(firingWarning)*5
	if deductAlert > 40 {
		deductAlert = 40
	}
	alertScore -= deductAlert
	factors = append(factors, PortraitHealthFactor{
		Key: "alert", Label: "告警", Score: alertScore, Max: alertMax, Deduct: deductAlert,
		Detail: fmt.Sprintf("近24h firing critical=%d warning=%d", firingCritical, firingWarning),
	})

	// 发布因子：满分 35；近 7 天成功率
	relScore, relMax := 35, 35
	cicdID := uint(0)
	for _, l := range item.Links {
		if l.LinkType == model.ServiceLinkCicdService && l.RefID != nil {
			cicdID = *l.RefID
			break
		}
	}
	relDetail := "无 CI/CD 绑定，按满分计"
	if cicdID > 0 {
		week := now.Add(-7 * 24 * time.Hour)
		var total, failed int64
		_ = s.db.WithContext(ctx).Model(&model.CicdReleaseRun{}).
			Where("project_id = ? AND service_id = ? AND created_at >= ?", item.ProjectID, cicdID, week).
			Count(&total).Error
		_ = s.db.WithContext(ctx).Model(&model.CicdReleaseRun{}).
			Where("project_id = ? AND service_id = ? AND created_at >= ? AND status IN ?",
				item.ProjectID, cicdID, week, []string{"failed", "cancelled"}).
			Count(&failed).Error
		if total == 0 {
			relDetail = "近7天无发布"
		} else {
			rate := float64(total-failed) / float64(total)
			relScore = int(float64(relMax) * rate)
			relDetail = fmt.Sprintf("近7天发布 %d 次，失败/取消 %d，成功率 %.0f%%", total, failed, rate*100)
		}
	}
	factors = append(factors, PortraitHealthFactor{
		Key: "release", Label: "发布", Score: relScore, Max: relMax, Deduct: relMax - relScore, Detail: relDetail,
	})

	// Ready 因子：满分 25；有 k8s 绑定且近期无失败变更则满分
	readyScore, readyMax := 25, 25
	readyDetail := "未绑定工作负载，按满分计"
	hasWL := false
	for _, l := range item.Links {
		if l.LinkType == model.ServiceLinkK8sWorkload && strings.TrimSpace(l.RefKey) != "" {
			hasWL = true
			readyDetail = "已绑定 " + l.RefKey
			break
		}
	}
	if hasWL {
		var failN int64
		_ = s.db.WithContext(ctx).Model(&model.ChangeEvent{}).
			Where("project_id = ? AND service_id = ? AND source = ? AND status = ? AND started_at >= ?",
				item.ProjectID, item.ID, model.ChangeSourceK8s, model.ChangeStatusFailed, since).
			Count(&failN).Error
		if failN > 0 {
			readyScore = 10
			readyDetail += fmt.Sprintf("；近24h K8s 失败变更 %d", failN)
		} else {
			readyDetail += "；近24h 无失败变更"
		}
	}
	factors = append(factors, PortraitHealthFactor{
		Key: "ready", Label: "Ready", Score: readyScore, Max: readyMax, Deduct: readyMax - readyScore, Detail: readyDetail,
	})

	// 日志异常因子：满分 10；open critical -4/条(上限8)，warning -2/条(上限4)
	logScore, logMax := 10, 10
	var openCritical, openWarning int64
	_ = s.db.WithContext(ctx).Model(&model.LogAnomaly{}).
		Where("project_id = ? AND status = ? AND severity = ?", item.ProjectID, model.LogAnomalyStatusOpen, model.LogAnomalySeverityCritical).
		Count(&openCritical).Error
	_ = s.db.WithContext(ctx).Model(&model.LogAnomaly{}).
		Where("project_id = ? AND status = ? AND severity = ?", item.ProjectID, model.LogAnomalyStatusOpen, model.LogAnomalySeverityWarning).
		Count(&openWarning).Error
	deductLog := int(openCritical)*4 + int(openWarning)*2
	if deductLog > 10 {
		deductLog = 10
	}
	logScore -= deductLog
	logDetail := fmt.Sprintf("open 日志异常 critical=%d warning=%d", openCritical, openWarning)
	factors = append(factors, PortraitHealthFactor{
		Key: "log", Label: "日志", Score: logScore, Max: logMax, Deduct: deductLog, Detail: logDetail,
	})

	score := alertScore + relScore + readyScore + logScore
	if score > 100 {
		score = 100
	}
	grade := "A"
	switch {
	case score >= 85:
		grade = "A"
	case score >= 70:
		grade = "B"
	case score >= 50:
		grade = "C"
	default:
		grade = "D"
	}
	return &PortraitHealth{
		Score:     score,
		Grade:     grade,
		Factors:   factors,
		CheckedAt: now.Format(time.RFC3339),
	}
}
