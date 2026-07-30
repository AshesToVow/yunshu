package cicd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/changegate"
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
	recordReleaseChange(ctx, s.db, &release, "release_create", model.ChangeStatusStarted,
		fmt.Sprintf("创建发布工单 #%d：%s", release.ID, release.Title))
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
	harborURL, harborProject := s.loadProjectHarbor(ctx, release.ProjectID)
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
		HarborURL:        harborURL,
		HarborProject:    harborProject,
	})
	lastNum, _ := client.GetLastBuildNumber(ctx, p.svc.JenkinsJob)
	queuePath, err := client.BuildWithParameters(ctx, p.svc.JenkinsJob, params)
	if err != nil {
		return fmt.Errorf("trigger jenkins release: %w", err)
	}

	// Jenkins 构建已触发。以下状态持久化必须与请求上下文解耦：
	// 若沿用 ctx，客户端/网关超时取消请求会导致构建号乃至 running 状态写不进库，
	// 而 Jenkins 侧任务照常执行，工单将永久卡在旧状态。用后台 ctx 先落库，
	// 并存下 queue_url 供同步/补偿流程后续解析构建号。
	persistCtx := context.WithoutCancel(ctx)
	now := time.Now()
	updates := map[string]any{
		"status":        model.CicdRunStatusRunning,
		"params_json":   ParamsJSON(params),
		"started_at":    now,
		"image_address": p.imageAddress,
		"artifact_name": p.artifactName,
		"jenkins_queue_url": strings.TrimSpace(queuePath),
	}
	if err := s.db.WithContext(persistCtx).Model(release).Updates(updates).Error; err != nil {
		return err
	}
	// 尽力而为地立即解析构建号；失败/超时不影响工单状态，
	// 交由 RunSyncWorker 通过 queue_url 补偿（见 recoverReleaseBuildNumber）。
	if buildNum, err := client.ResolveQueueBuildNumber(ctx, queuePath, lastNum, 90*time.Second); err == nil && buildNum > 0 {
		_ = s.db.WithContext(persistCtx).Model(&model.CicdReleaseRun{}).
			Where("id = ? AND jenkins_build_number = 0", release.ID).
			Updates(map[string]any{
				"jenkins_build_number": buildNum,
				"jenkins_build_url":    client.BuildURL(p.svc.JenkinsJob, buildNum),
			}).Error
	}
	_ = executorUserID
	return nil
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

// transitionReleaseStatus 以期望状态为条件原子推进工单状态，返回本次是否抢到转换。
// 工单表无乐观锁列，靠 WHERE status 条件 + RowsAffected 防止并发审批/执行/终止重复推进。
func (s *Service) transitionReleaseStatus(ctx context.Context, runID uint, fromStatus string, updates map[string]any) (bool, error) {
	res := s.db.WithContext(ctx).Model(&model.CicdReleaseRun{}).
		Where("id = ? AND status = ?", runID, fromStatus).
		Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// claimApprovalStep 以 pending 为条件原子占用审批节点，返回本次是否抢到。
// 同一节点的 approve / reject 并发时，只有一方成功，另一方视为已被处理。
func (s *Service) claimApprovalStep(ctx context.Context, stepID uint, updates map[string]any) (bool, error) {
	res := s.db.WithContext(ctx).Model(&model.CicdReleaseApprovalStep{}).
		Where("id = ? AND status = ?", stepID, model.CicdApprovalStepPending).
		Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// errReleaseConflict 并发状态流转冲突：本次调用未抢到转换（已被他人处理）。
var errReleaseConflict = constants.ErrBadRequestWithMsg("工单状态已变更，请刷新后重试")

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
	ok, err := s.transitionReleaseStatus(ctx, runID, model.CicdRunStatusPendingApproval, updates)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errReleaseConflict
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
	claimed, err := s.claimApprovalStep(ctx, step.ID, map[string]any{
		"status":           model.CicdApprovalStepApproved,
		"reviewer_user_id": reviewerUserID,
		"reviewer_name":    strings.TrimSpace(reviewerName),
		"review_comment":   strings.TrimSpace(comment),
		"reviewed_at":      now,
	})
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, errReleaseConflict
	}
	step.Status = model.CicdApprovalStepApproved
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
		ok, err := s.transitionReleaseStatus(ctx, runID, model.CicdRunStatusPendingApproval, updates)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errReleaseConflict
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
	claimed, err := s.claimApprovalStep(ctx, step.ID, map[string]any{
		"status":           model.CicdApprovalStepRejected,
		"reviewer_user_id": reviewerUserID,
		"reviewer_name":    strings.TrimSpace(reviewerName),
		"review_comment":   strings.TrimSpace(comment),
		"reviewed_at":      now,
	})
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, errReleaseConflict
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
	var catalogID *uint
	if release.ServiceID > 0 {
		var link model.ServiceLink
		if err := s.db.WithContext(ctx).
			Joins("JOIN service_catalog sc ON sc.id = service_links.service_id AND sc.project_id = ? AND sc.deleted_at IS NULL", projectID).
			Where("service_links.link_type = ? AND service_links.ref_id = ? AND service_links.deleted_at IS NULL",
				model.ServiceLinkCicdService, release.ServiceID).
			Order("service_links.id DESC").First(&link).Error; err == nil {
			id := link.ServiceID
			catalogID = &id
		}
	}
	if err := changegate.AssertWritable(ctx, changegate.CheckInput{
		ProjectID: projectID,
		Source:    model.ChangeSourceCicd,
		Env:       strings.ToLower(strings.TrimSpace(release.Tenv)),
		ServiceID: catalogID,
		Action:    "release_execute",
	}); err != nil {
		return nil, err
	}
	// 触发 Jenkins 前先原子占用（pending_execution -> running），防止并发/重复点击重复触发构建。
	claimed, err := s.transitionReleaseStatus(ctx, runID, model.CicdRunStatusPendingExecution, map[string]any{
		"status": model.CicdRunStatusRunning,
	})
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, errReleaseConflict
	}
	if err := s.executeReleaseRun(ctx, &release, executorUserID); err != nil {
		// 触发失败：回退到待执行，允许提交人重试（executeReleaseRun 尚未成功触发构建）。
		_, _ = s.transitionReleaseStatus(context.WithoutCancel(ctx), runID, model.CicdRunStatusRunning, map[string]any{
			"status": model.CicdRunStatusPendingExecution,
		})
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", runID).First(&release).Error; err != nil {
		return nil, err
	}
	recordReleaseChange(ctx, s.db, &release, "release_execute", model.ChangeStatusStarted,
		fmt.Sprintf("执行发布 #%d：%s", release.ID, release.Title))
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
	ok, err := s.transitionReleaseStatus(ctx, runID, model.CicdRunStatusPendingExecution, updates)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errReleaseConflict
	}
	if err := s.db.WithContext(ctx).Where("id = ?", runID).First(&release).Error; err != nil {
		return nil, err
	}
	recordReleaseChange(ctx, s.db, &release, "release_terminate", model.ChangeStatusAborted,
		fmt.Sprintf("终止发布 #%d：%s", release.ID, release.Title))
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
