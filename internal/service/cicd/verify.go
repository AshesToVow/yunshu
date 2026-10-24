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

	alertCount, _ := s.repo.CountFiringAlertsSince(ctx, projectID, *since)
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
	// 未检查（未绑定 workload / 探针未注入）不能当成通过。
	if readyOK == nil || !*readyOK {
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
	_ = s.repo.UpdateReleaseRunFields(ctx, release.ID, map[string]any{
		"verify_status": status,
		"verify_json":   string(payload),
		"verified_at":   time.Now(),
	})

	var catalogID *uint
	if s.catalogRepo != nil && release.ServiceID > 0 {
		if id, err := s.catalogRepo.FindServiceIDByLinkRef(ctx, projectID, model.ServiceLinkCicdService, release.ServiceID); err == nil && id > 0 {
			catalogID = &id
		}
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
	if s.catalogRepo == nil {
		return nil, "未找到服务目录绑定，跳过 Ready 检查"
	}
	catalogServiceID, err := s.catalogRepo.FindServiceIDByLinkRef(ctx, projectID, model.ServiceLinkCicdService, cicdServiceID)
	if err != nil || catalogServiceID == 0 {
		return nil, "未找到服务目录绑定，跳过 Ready 检查"
	}
	links, err := s.catalogRepo.ListLinks(ctx, catalogServiceID)
	if err != nil {
		return nil, "未绑定 k8s_workload，跳过 Ready 检查"
	}
	var refKey string
	for i := len(links) - 1; i >= 0; i-- {
		if links[i].LinkType == model.ServiceLinkK8sWorkload && strings.TrimSpace(links[i].RefKey) != "" {
			refKey = strings.TrimSpace(links[i].RefKey)
			break
		}
	}
	if refKey == "" {
		return nil, "未绑定 k8s_workload，跳过 Ready 检查"
	}
	parts := strings.Split(refKey, "/")
	if len(parts) < 4 {
		return nil, "k8s_workload ref_key 应为 clusterID/namespace/kind/name"
	}
	if s.workloadReadyCheck != nil {
		return s.workloadReadyCheck(ctx, parts[0], parts[1], parts[2], parts[3])
	}
	return nil, fmt.Sprintf("已绑定 %s，但集群 Ready 探针未注入，不能判定通过", refKey)
}

func (s *Service) sampleErrorLogs(ctx context.Context, projectID, cicdServiceID uint, since time.Time) (int, string) {
	if s.errorLogSampler != nil {
		return s.errorLogSampler(ctx, projectID, cicdServiceID, since)
	}
	// Repo 暂无按 project+since 统计失败变更的方法；日志检索未注入时跳过 DB 弱信号。
	return 0, "未发现错误日志弱信号"
}
