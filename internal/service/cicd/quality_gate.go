package cicd

import (
	"context"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
)

// assertBuildQualityGate 在 Sonar 门禁开启且要求拦截时，校验所选构建记录是否通过质量门禁。
// 未关联 build_run、或字典关闭门禁时放行。
func (s *Service) assertBuildQualityGate(ctx context.Context, projectID, serviceID, buildRunID uint) error {
	if buildRunID == 0 {
		return nil
	}
	cfg := s.resolvedConfig(ctx)
	if !cfg.Sonar.Enabled || !cfg.Sonar.GateBlock {
		return nil
	}
	var br model.CicdBuildRun
	if err := s.db.WithContext(ctx).
		Where("id = ? AND service_id = ? AND project_id = ?", buildRunID, serviceID, projectID).
		First(&br).Error; err != nil {
		return constants.ErrNotFound
	}
	return qualityGateBlockReason(cfg, &br)
}

func qualityGateBlockReason(cfg config.CicdConfig, br *model.CicdBuildRun) error {
	if br == nil {
		return nil
	}
	if !cfg.Sonar.Enabled || !cfg.Sonar.GateBlock {
		return nil
	}
	status := strings.ToUpper(strings.TrimSpace(br.QualityGateStatus))
	if status == "" {
		if br.SecurityScanPass != nil && !*br.SecurityScanPass {
			return constants.ErrBadRequestWithMsg("SonarQube 质量门禁未通过，禁止进入审批/发布")
		}
		// 已开启门禁但尚未收到回调：构建未完成或流水线未上报。
		if br.BuildResult == model.CicdRunStatusRunning || br.BuildResult == model.CicdRunStatusPending {
			return constants.ErrBadRequestWithMsg("构建尚未完成质量门禁检查，请稍后再发布")
		}
		if br.BuildResult == model.CicdRunStatusSuccess {
			return constants.ErrBadRequestWithMsg("未收到 SonarQube 质量门禁结果，请确认 Jenkins 已回调 Yunshu")
		}
		return constants.ErrBadRequestWithMsg("构建未成功，无法用于发布")
	}
	switch status {
	case model.CicdQualityGateOK, model.CicdQualityGateWarn:
		return nil
	case model.CicdQualityGateError:
		msg := "SonarQube 质量门禁未通过（ERROR），禁止进入审批/发布"
		if u := strings.TrimSpace(br.SonarDashboardURL); u != "" {
			msg += "：" + u
		}
		return constants.ErrBadRequestWithMsg(msg)
	case model.CicdQualityGateNone:
		return constants.ErrBadRequestWithMsg("SonarQube 未扫描（NONE），禁止进入审批/发布")
	default:
		return constants.ErrBadRequestWithMsg("未知质量门禁状态：" + status)
	}
}
