package cicd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
)

// releaseRequestSnapshot 审批通过后用于触发 Jenkins 的原始请求快照。
type releaseRequestSnapshot struct {
	DeployConfigID   uint   `json:"deploy_config_id"`
	Title            string `json:"title"`
	ReleaseOperation string `json:"release_operation"`
	PublishMode      string `json:"publish_mode"`
	ArtifactName     string `json:"artifact_name"`
	ImageAddress     string `json:"image_address"`
	BuildRunID       uint   `json:"build_run_id"`
	EmailUser        string `json:"email_user"`
}

type preparedRelease struct {
	svc         *model.CicdService
	ci          *model.CicdCiConfig
	dc          *model.CicdDeployConfig
	artifactName string
	imageAddress string
	publishMode  string
	releaseOp    string
	releaseType  string
	emailUser    string
	req          TriggerReleaseRequest
}

func (s *Service) prepareRelease(
	ctx context.Context,
	projectID, serviceID uint,
	req TriggerReleaseRequest,
	submitterUserID *uint,
) (*preparedRelease, error) {
	svc, err := s.loadService(ctx, projectID, serviceID)
	if err != nil {
		return nil, err
	}
	ci, err := s.requireCiConfig(ctx, projectID, serviceID)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("请先配置 CI 信息")
	}
	var dc model.CicdDeployConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND service_id = ?", req.DeployConfigID, serviceID).First(&dc).Error; err != nil {
		return nil, constants.ErrNotFound
	}

	artifactName := strings.TrimSpace(req.ArtifactName)
	imageAddress := strings.TrimSpace(req.ImageAddress)
	publishMode := strings.TrimSpace(req.PublishMode)
	releaseOp := strings.TrimSpace(req.ReleaseOperation)
	if releaseOp == "" {
		releaseOp = strings.TrimSpace(req.ReleaseType)
	}
	if strings.EqualFold(dc.DeployKind, model.CicdDeployKindContainer) {
		if req.BuildRunID > 0 {
			var br model.CicdBuildRun
			if err := s.db.WithContext(ctx).
				Where("id = ? AND service_id = ? AND project_id = ?", req.BuildRunID, serviceID, projectID).
				First(&br).Error; err != nil {
				return nil, constants.ErrNotFound
			}
			if strings.TrimSpace(br.ImageAddress) == "" {
				return nil, constants.ErrBadRequestWithMsg("所选构建记录尚无镜像地址，请等待 CI 完成或手动填写")
			}
			imageAddress = strings.TrimSpace(br.ImageAddress)
		}
		if imageAddress == "" {
			return nil, constants.ErrBadRequestWithMsg("容器化发布须选择 CI 构建镜像或填写镜像地址")
		}
		if releaseOp == "" {
			if strings.EqualFold(dc.DeployAction, "初始化部署") {
				releaseOp = model.CicdReleaseTypeServiceOnline
			} else {
				releaseOp = model.CicdReleaseTypePodUpdate
			}
		}
		if err := validateContainerReleaseOperation(releaseOp); err != nil {
			return nil, err
		}
		if releaseOp == model.CicdReleaseOpContainerRollback {
			publishMode = "回滚"
		} else {
			// Yunshu CD：已选 CI 镜像，Jenkins 传 FULL_IMAGE_NAME + 制品发布，跳过重复编译/推镜像。
			publishMode = model.CicdPublishModeArtifactDeploy
		}
	} else {
		if artifactName == "" {
			return nil, constants.ErrBadRequestWithMsg("请选择 MinIO 制品包")
		}
		if releaseOp == "" {
			releaseOp = defaultReleaseOperation(svc.ServiceType)
		}
		if err := validateReleaseOperation(svc.ServiceType, releaseOp); err != nil {
			return nil, err
		}
		publishMode = model.CicdPublishModeArtifactDeploy
	}

	return &preparedRelease{
		svc:          svc,
		ci:           ci,
		dc:           &dc,
		artifactName: artifactName,
		imageAddress: imageAddress,
		publishMode:  publishMode,
		releaseOp:    releaseOp,
		releaseType:  releaseOp,
		emailUser:    s.resolveNotifyEmail(ctx, req.EmailUser, svc, submitterUserID),
		req:          req,
	}, nil
}

func snapshotJSON(p *preparedRelease) string {
	snap := releaseRequestSnapshot{
		DeployConfigID:   p.req.DeployConfigID,
		Title:            strings.TrimSpace(p.req.Title),
		ReleaseOperation: p.releaseOp,
		PublishMode:      p.publishMode,
		ArtifactName:     p.artifactName,
		ImageAddress:     p.imageAddress,
		BuildRunID:       p.req.BuildRunID,
		EmailUser:        p.emailUser,
	}
	b, _ := json.Marshal(snap)
	return string(b)
}

func parseReleaseSnapshot(raw string) (*releaseRequestSnapshot, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, constants.ErrBadRequestWithMsg("发布请求快照缺失")
	}
	var snap releaseRequestSnapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return nil, constants.ErrBadRequestWithMsg("发布请求快照无效")
	}
	return &snap, nil
}

func (s *Service) createPendingRelease(
	ctx context.Context,
	projectID, serviceID uint,
	p *preparedRelease,
	submitterUserID *uint,
	submitterName string,
) (*model.CicdReleaseRun, error) {
	now := time.Now()
	dcID := p.dc.ID
	release := model.CicdReleaseRun{
		ProjectID:       projectID,
		ServiceID:       serviceID,
		DeployConfigID:  &dcID,
		Title:           strings.TrimSpace(p.req.Title),
		ReleaseKind:     p.dc.DeployKind,
		ReleaseType:     p.releaseType,
		Tenv:            p.dc.Tenv,
		Status:          model.CicdRunStatusPendingApproval,
		SubmitterUserID: submitterUserID,
		SubmitterName:   submitterName,
		ImageAddress:    p.imageAddress,
		ArtifactName:    p.artifactName,
		AuditEnabled:    true,
		RequestJSON:     snapshotJSON(p),
		StartedAt:       &now,
	}
	if err := s.db.WithContext(ctx).Create(&release).Error; err != nil {
		return nil, err
	}
	if err := s.initReleaseApprovalSteps(ctx, &release); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", release.ID).First(&release).Error; err != nil {
		return nil, err
	}
	return &release, nil
}

func (s *Service) executeReleaseRun(ctx context.Context, release *model.CicdReleaseRun, executorUserID *uint) error {
	if release == nil {
		return constants.ErrBadRequestWithMsg("工单不存在")
	}
	snap, err := parseReleaseSnapshot(release.RequestJSON)
	if err != nil {
		return err
	}
	p, err := s.prepareRelease(ctx, release.ProjectID, release.ServiceID, TriggerReleaseRequest{
		DeployConfigID:   snap.DeployConfigID,
		Title:            snap.Title,
		ReleaseOperation: snap.ReleaseOperation,
		PublishMode:      snap.PublishMode,
		ArtifactName:     snap.ArtifactName,
		ImageAddress:     snap.ImageAddress,
		BuildRunID:       snap.BuildRunID,
		EmailUser:        snap.EmailUser,
	}, release.SubmitterUserID)
	if err != nil {
		return err
	}

	client, cfg, err := s.jenkinsClient(ctx)
	if err != nil {
		return err
	}
	if _, err := s.syncJenkinsJob(ctx, p.svc, p.ci); err != nil {
		return err
	}
	if err := s.ensureReleaseNamespace(ctx, p.dc); err != nil {
		return fmt.Errorf("ensure k8s namespace: %w", err)
	}
	destIPs, err := s.resolveDestIPs(ctx, release.ProjectID, p.dc)
	if err != nil {
		return err
	}
	params := BuildJenkinsParams(BuildParamsInput{
		Service:          p.svc,
		CiConfig:         p.ci,
		DeployConfig:     p.dc,
		Cfg:              cfg,
		PublishMode:      p.publishMode,
		Tenv:             p.dc.Tenv,
		DestIPs:          destIPs,
		EmailUser:        p.emailUser,
		ImageAddress:     p.imageAddress,
		SelectedVersion:  p.artifactName,
		ReleaseOperation: p.releaseOp,
		ForceCleanDeploy: releaseForceCleanDeploy(p.releaseOp),
		UsesK8sPipeline:  s.serviceUsesK8sPipeline(ctx, p.svc),
	})
	lastNum, _ := client.GetLastBuildNumber(ctx, p.svc.JenkinsJob)
	queuePath, err := client.BuildWithParameters(ctx, p.svc.JenkinsJob, params)
	if err != nil {
		return fmt.Errorf("trigger jenkins release: %w", err)
	}

	now := time.Now()
	updates := map[string]any{
		"status":       model.CicdRunStatusRunning,
		"params_json":  ParamsJSON(params),
		"started_at":   now,
		"image_address": p.imageAddress,
		"artifact_name": p.artifactName,
	}
	if buildNum, err := client.ResolveQueueBuildNumber(ctx, queuePath, lastNum, 90*time.Second); err == nil && buildNum > 0 {
		updates["jenkins_build_number"] = buildNum
		updates["jenkins_build_url"] = client.BuildURL(p.svc.JenkinsJob, buildNum)
	}
	_ = executorUserID
	return s.db.WithContext(ctx).Model(release).Updates(updates).Error
}

type ReviewReleaseRequest struct {
	Comment string `json:"comment" binding:"omitempty,max=512"`
}

type BatchReleaseIDsRequest struct {
	IDs     []uint `json:"ids" binding:"required,min=1"`
	Comment string `json:"comment" binding:"omitempty,max=512"`
}

func (s *Service) hasApprovalSteps(ctx context.Context, releaseRunID uint) (bool, error) {
	var n int64
	err := s.db.WithContext(ctx).Model(&model.CicdReleaseApprovalStep{}).Where("release_run_id = ?", releaseRunID).Count(&n).Error
	return n > 0, err
}

func (s *Service) approveLegacySingleStep(ctx context.Context, release *model.CicdReleaseRun, runID uint, reviewerUserID *uint, reviewerName, comment string) (*model.CicdReleaseRun, error) {
	now := time.Now()
	updates := map[string]any{
		"status":           model.CicdRunStatusPendingExecution,
		"reviewer_user_id": reviewerUserID,
		"reviewer_name":    strings.TrimSpace(reviewerName),
		"review_comment":   strings.TrimSpace(comment),
		"reviewed_at":      now,
		"current_stage_key": "",
	}
	if err := s.db.WithContext(ctx).Model(release).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", runID).First(release).Error; err != nil {
		return nil, err
	}
	return release, nil
}

func (s *Service) ApproveReleaseRun(ctx context.Context, projectID, runID uint, reviewerUserID *uint, reviewerName, comment string) (*model.CicdReleaseRun, error) {
	var release model.CicdReleaseRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&release).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	if release.Status != model.CicdRunStatusPendingApproval {
		return nil, constants.ErrBadRequestWithMsg("仅待审核工单可审批通过")
	}
	hasSteps, err := s.hasApprovalSteps(ctx, runID)
	if err != nil {
		return nil, err
	}
	if !hasSteps {
		return s.approveLegacySingleStep(ctx, &release, runID, reviewerUserID, reviewerName, comment)
	}
	if reviewerUserID == nil || *reviewerUserID == 0 {
		return nil, constants.ErrBadRequestWithMsg("未登录无法审批")
	}
	step, err := s.getCurrentPendingStep(ctx, runID)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("当前无待审批节点")
	}
	can, err := s.userCanApproveStep(ctx, *reviewerUserID, step)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, constants.ErrBadRequestWithMsg("您不是当前审批节点的审批人")
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(step).Updates(map[string]any{
		"status":           model.CicdApprovalStepApproved,
		"reviewer_user_id": reviewerUserID,
		"reviewer_name":    strings.TrimSpace(reviewerName),
		"review_comment":   strings.TrimSpace(comment),
		"reviewed_at":      now,
	}).Error; err != nil {
		return nil, err
	}
	if err := s.advanceReleaseAfterApproval(ctx, &release, step); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", runID).First(&release).Error; err != nil {
		return nil, err
	}
	return &release, nil
}

func (s *Service) RejectReleaseRun(ctx context.Context, projectID, runID uint, reviewerUserID *uint, reviewerName, comment string) (*model.CicdReleaseRun, error) {
	var release model.CicdReleaseRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&release).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	if release.Status != model.CicdRunStatusPendingApproval {
		return nil, constants.ErrBadRequestWithMsg("仅待审核工单可驳回")
	}
	hasSteps, err := s.hasApprovalSteps(ctx, runID)
	if err != nil {
		return nil, err
	}
	if !hasSteps {
		now := time.Now()
		updates := map[string]any{
			"status":           model.CicdRunStatusRejected,
			"reviewer_user_id": reviewerUserID,
			"reviewer_name":    strings.TrimSpace(reviewerName),
			"review_comment":   strings.TrimSpace(comment),
			"reviewed_at":      now,
			"finished_at":      now,
		}
		if err := s.db.WithContext(ctx).Model(&release).Updates(updates).Error; err != nil {
			return nil, err
		}
		if err := s.db.WithContext(ctx).Where("id = ?", runID).First(&release).Error; err != nil {
			return nil, err
		}
		return &release, nil
	}
	if reviewerUserID == nil || *reviewerUserID == 0 {
		return nil, constants.ErrBadRequestWithMsg("未登录无法审批")
	}
	step, err := s.getCurrentPendingStep(ctx, runID)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("当前无待审批节点")
	}
	can, err := s.userCanApproveStep(ctx, *reviewerUserID, step)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, constants.ErrBadRequestWithMsg("您不是当前审批节点的审批人")
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(step).Updates(map[string]any{
		"status":           model.CicdApprovalStepRejected,
		"reviewer_user_id": reviewerUserID,
		"reviewer_name":    strings.TrimSpace(reviewerName),
		"review_comment":   strings.TrimSpace(comment),
		"reviewed_at":      now,
	}).Error; err != nil {
		return nil, err
	}
	updates := map[string]any{
		"status":            model.CicdRunStatusRejected,
		"reviewer_user_id":  reviewerUserID,
		"reviewer_name":     strings.TrimSpace(reviewerName),
		"review_comment":    strings.TrimSpace(comment),
		"reviewed_at":       now,
		"finished_at":       now,
		"current_stage_key": "",
	}
	if err := s.db.WithContext(ctx).Model(&release).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", runID).First(&release).Error; err != nil {
		return nil, err
	}
	return &release, nil
}

func (s *Service) ExecuteReleaseRun(ctx context.Context, projectID, runID uint, executorUserID *uint) (*model.CicdReleaseRun, error) {
	var release model.CicdReleaseRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&release).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	if release.Status != model.CicdRunStatusPendingExecution {
		return nil, constants.ErrBadRequestWithMsg("仅待执行工单可发布")
	}
	if executorUserID == nil || release.SubmitterUserID == nil || *executorUserID != *release.SubmitterUserID {
		return nil, constants.ErrBadRequestWithMsg("仅提交人可执行发布")
	}
	if err := s.executeReleaseRun(ctx, &release, executorUserID); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", runID).First(&release).Error; err != nil {
		return nil, err
	}
	return &release, nil
}

func (s *Service) TerminateReleaseRun(ctx context.Context, projectID, runID uint, reviewerUserID *uint, reviewerName, comment string) (*model.CicdReleaseRun, error) {
	var release model.CicdReleaseRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&release).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	if release.Status != model.CicdRunStatusPendingExecution {
		return nil, constants.ErrBadRequestWithMsg("仅待执行工单可终止")
	}
	if reviewerUserID == nil || release.SubmitterUserID == nil || *reviewerUserID != *release.SubmitterUserID {
		return nil, constants.ErrBadRequestWithMsg("仅提交人可终止发布")
	}
	now := time.Now()
	updates := map[string]any{
		"status":           model.CicdRunStatusCancelled,
		"reviewer_user_id": reviewerUserID,
		"reviewer_name":    strings.TrimSpace(reviewerName),
		"review_comment":   strings.TrimSpace(comment),
		"reviewed_at":      now,
		"finished_at":      now,
	}
	if err := s.db.WithContext(ctx).Model(&release).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", runID).First(&release).Error; err != nil {
		return nil, err
	}
	return &release, nil
}

func (s *Service) BatchApproveReleaseRuns(ctx context.Context, projectID uint, ids []uint, reviewerUserID *uint, reviewerName, comment string) (int, error) {
	ok := 0
	for _, id := range ids {
		if _, err := s.ApproveReleaseRun(ctx, projectID, id, reviewerUserID, reviewerName, comment); err == nil {
			ok++
		}
	}
	if ok == 0 {
		return 0, constants.ErrBadRequestWithMsg("没有工单审批成功")
	}
	return ok, nil
}

func (s *Service) BatchRejectReleaseRuns(ctx context.Context, projectID uint, ids []uint, reviewerUserID *uint, reviewerName, comment string) (int, error) {
	ok := 0
	for _, id := range ids {
		if _, err := s.RejectReleaseRun(ctx, projectID, id, reviewerUserID, reviewerName, comment); err == nil {
			ok++
		}
	}
	if ok == 0 {
		return 0, constants.ErrBadRequestWithMsg("没有工单驳回成功")
	}
	return ok, nil
}

func (s *Service) BatchExecuteReleaseRuns(ctx context.Context, projectID uint, ids []uint, executorUserID *uint) (int, error) {
	ok := 0
	for _, id := range ids {
		if _, err := s.ExecuteReleaseRun(ctx, projectID, id, executorUserID); err == nil {
			ok++
		}
	}
	if ok == 0 {
		return 0, constants.ErrBadRequestWithMsg("没有工单执行成功")
	}
	return ok, nil
}

func (s *Service) BatchTerminateReleaseRuns(ctx context.Context, projectID uint, ids []uint, reviewerUserID *uint, reviewerName, comment string) (int, error) {
	ok := 0
	for _, id := range ids {
		if _, err := s.TerminateReleaseRun(ctx, projectID, id, reviewerUserID, reviewerName, comment); err == nil {
			ok++
		}
	}
	if ok == 0 {
		return 0, constants.ErrBadRequestWithMsg("没有工单终止成功")
	}
	return ok, nil
}
