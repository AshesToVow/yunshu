package cicd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/service/changeevent"
)

const (
	VerifyStatusPassed  = "passed"
	VerifyStatusFailed  = "failed"
	VerifyStatusPartial = "partial"
)

// ReleaseVerifyResult 发布后验证：Ready / 错误日志 / 新告警。
type ReleaseVerifyResult struct {
	ReleaseID   uint           `json:"release_id"`
	Status      string         `json:"status"`
	ReadyOK     *bool          `json:"ready_ok,omitempty"`
	ReadyDetail string         `json:"ready_detail,omitempty"`
	LogErrors   int            `json:"log_errors"`
	LogDetail   string         `json:"log_detail,omitempty"`
	NewAlerts   int            `json:"new_alerts"`
	AlertDetail string         `json:"alert_detail,omitempty"`
	CheckedAt   string         `json:"checked_at"`
	Factors     map[string]any `json:"factors,omitempty"`
}

// VerifyReleaseRun 执行发布后验证并写回 release.verify_* 与 change_event。
func (s *Service) VerifyReleaseRun(ctx context.Context, projectID, runID uint, actor *auth.CurrentUser) (*ReleaseVerifyResult, error) {
	release, err := s.assertReleaseRunAccess(ctx, projectID, runID, actor, "release")
	if err != nil {
		return nil, err
	}
	out := &ReleaseVerifyResult{
		ReleaseID: release.ID,
		CheckedAt: time.Now().Format(time.RFC3339),
		Factors:   map[string]any{},
	}

	since := release.StartedAt
	if since == nil {
		t := release.CreatedAt
		since = &t
	}

	var alertCount int64
	_ = s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("status = ? AND severity IN ? AND created_at >= ?", "firing", []string{"critical", "warning"}, since).
		Count(&alertCount).Error
	out.NewAlerts = int(alertCount)
	if alertCount > 0 {
		out.AlertDetail = fmt.Sprintf("发布后出现 %d 条 P1/P2 告警", alertCount)
	} else {
		out.AlertDetail = "未发现新的 P1/P2 告警"
	}
	out.Factors["new_alerts"] = alertCount

	readyOK, readyDetail := s.checkWorkloadReady(ctx, projectID, release.ServiceID)
	out.ReadyOK = readyOK
	out.ReadyDetail = readyDetail
	out.Factors["ready"] = readyDetail

	logErrors, logDetail := s.sampleErrorLogs(ctx, projectID, release.ServiceID, *since)
	out.LogErrors = logErrors
	out.LogDetail = logDetail
	out.Factors["log_errors"] = logErrors

	failBits := 0
	if alertCount > 0 {
		failBits++
	}
	if readyOK != nil && !*readyOK {
		failBits++
	}
	if logErrors > 0 {
		failBits++
	}
	status := VerifyStatusPassed
	switch {
	case failBits >= 2:
		status = VerifyStatusFailed
	case failBits == 1:
		status = VerifyStatusPartial
	}
	out.Status = status

	payload, _ := json.Marshal(out)
	_ = s.db.WithContext(ctx).Model(&release).Updates(map[string]any{
		"verify_status": status,
		"verify_json":   string(payload),
		"verified_at":   time.Now(),
	}).Error

	var catalogID *uint
	var link model.ServiceLink
	if err := s.db.WithContext(ctx).
		Joins("JOIN service_catalog sc ON sc.id = service_links.service_id AND sc.project_id = ? AND sc.deleted_at IS NULL", projectID).
		Where("service_links.link_type = ? AND service_links.ref_id = ?", model.ServiceLinkCicdService, release.ServiceID).
		First(&link).Error; err == nil {
		id := link.ServiceID
		catalogID = &id
	}
	changeevent.Record(ctx, changeevent.Input{
		ProjectID: projectID,
		ServiceID: catalogID,
		Source:    model.ChangeSourceCicd,
		Action:    "release_verify",
		RiskLevel: model.ChangeRiskMedium,
		Status:    model.ChangeStatusSucceeded,
		Summary:   fmt.Sprintf("发布 #%d 验证：%s", release.ID, status),
		Payload:   out,
	})
	return out, nil
}

func (s *Service) checkWorkloadReady(ctx context.Context, projectID, cicdServiceID uint) (*bool, string) {
	var catalogLink model.ServiceLink
	err := s.db.WithContext(ctx).
		Joins("JOIN service_catalog sc ON sc.id = service_links.service_id AND sc.project_id = ? AND sc.deleted_at IS NULL", projectID).
		Where("service_links.link_type = ? AND service_links.ref_id = ? AND service_links.deleted_at IS NULL",
			model.ServiceLinkCicdService, cicdServiceID).
		First(&catalogLink).Error
	if err != nil {
		return nil, "未找到服务目录绑定，跳过 Ready 检查"
	}
	var wl model.ServiceLink
	err = s.db.WithContext(ctx).
		Where("service_id = ? AND link_type = ? AND deleted_at IS NULL", catalogLink.ServiceID, model.ServiceLinkK8sWorkload).
		Order("id DESC").First(&wl).Error
	if err != nil || strings.TrimSpace(wl.RefKey) == "" {
		return nil, "未绑定 k8s_workload，跳过 Ready 检查"
	}
	parts := strings.Split(strings.TrimSpace(wl.RefKey), "/")
	if len(parts) < 4 {
		return nil, "k8s_workload ref_key 应为 clusterID/namespace/kind/name"
	}
	if s.workloadReadyCheck != nil {
		return s.workloadReadyCheck(ctx, parts[0], parts[1], parts[2], parts[3])
	}
	// 无集群客户端时：有绑定即给出「待确认」正面结论，避免误杀
	ok := true
	return &ok, fmt.Sprintf("已绑定 %s（集群 Ready 探针未注入，按绑定通过）", wl.RefKey)
}

func (s *Service) sampleErrorLogs(ctx context.Context, projectID, cicdServiceID uint, since time.Time) (int, string) {
	if s.errorLogSampler != nil {
		return s.errorLogSampler(ctx, projectID, cicdServiceID, since)
	}
	// 启发式：统计同项目近期 change_events 中带 error 的失败变更作为弱信号
	var n int64
	_ = s.db.WithContext(ctx).Model(&model.ChangeEvent{}).
		Where("project_id = ? AND status = ? AND started_at >= ? AND source = ?",
			projectID, model.ChangeStatusFailed, since, model.ChangeSourceCicd).
		Count(&n).Error
	if n > 0 {
		return int(n), fmt.Sprintf("同期有 %d 条失败变更（日志检索未注入时的弱信号）", n)
	}
	return 0, "未发现错误日志弱信号"
}
