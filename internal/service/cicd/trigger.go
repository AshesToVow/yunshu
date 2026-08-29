package cicd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
)

// --- Trigger Build / Release ---

type TriggerBuildRequest struct {
	BranchName  string `json:"branch_name" binding:"omitempty,max=128"`
	PublishMode string `json:"publish_mode" binding:"omitempty,max=32"`
	Tenv        string `json:"tenv" binding:"omitempty,max=16"`
	EmailUser   string `json:"email_user" binding:"omitempty,max=128"`
}

type TriggerReleaseRequest struct {
	DeployConfigID   uint   `json:"deploy_config_id" binding:"required"`
	Title            string `json:"title" binding:"required,max=256"`
	ReleaseOperation string `json:"release_operation" binding:"omitempty,max=32"`
	PublishMode      string `json:"publish_mode" binding:"omitempty,max=32"`
	ArtifactName     string `json:"artifact_name" binding:"omitempty,max=256"`
	ImageAddress     string `json:"image_address" binding:"omitempty,max=512"`
	BuildRunID       uint   `json:"build_run_id" binding:"omitempty"`
	ReleaseType      string `json:"release_type" binding:"omitempty,max=32"`
	EmailUser        string `json:"email_user" binding:"omitempty,max=128"`
}

func (s *Service) TriggerBuild(ctx context.Context, projectID, serviceID uint, req TriggerBuildRequest, builderUserID *uint, builderName string) (*model.CicdBuildRun, error) {
	svc, err := s.loadService(ctx, projectID, serviceID)
	if err != nil {
		return nil, err
	}
	ci, err := s.requireCiConfig(ctx, projectID, serviceID)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("请先配置 CI 信息")
	}
	client, cfg, err := s.jenkinsClient(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.syncJenkinsJob(ctx, svc, ci); err != nil {
		return nil, err
	}
	tenv := strings.TrimSpace(req.Tenv)
	if tenv == "" {
		tenv = s.defaultBuildTenv(ctx, serviceID)
	}
	dc := s.primaryContainerDeployConfig(ctx, serviceID)
	if dc == nil {
		dc = s.firstDeployConfig(ctx, serviceID)
	}
	ov := s.loadProjectCicdOverrides(ctx, projectID)

	// 先落库再触发，便于 Jenkins 回调携带 YUNSHU_BUILD_RUN_ID。
	now := time.Now()
	run := model.CicdBuildRun{
		ProjectID:     projectID,
		ServiceID:     serviceID,
		BuildNumber:   0,
		BranchName:    strings.TrimSpace(req.BranchName),
		PublishMode:   model.CicdPublishModeBuildOnly,
		Tenv:          tenv,
		BuildResult:   model.CicdRunStatusPending,
		BuilderUserID: builderUserID,
		BuilderName:   builderName,
		Version:       ci.Version,
		StartedAt:     &now,
	}
	if run.BranchName == "" && ci != nil {
		run.BranchName = strings.TrimSpace(ci.RefName)
	}
	if err := s.db.WithContext(ctx).Create(&run).Error; err != nil {
		return nil, err
	}

	params := BuildJenkinsParams(BuildParamsInput{
		Service:          svc,
		CiConfig:         ci,
		DeployConfig:     dc,
		Cfg:              cfg,
		BranchName:       run.BranchName,
		PublishMode:      model.CicdPublishModeBuildOnly,
		Tenv:             tenv,
		EmailUser:        s.resolveNotifyEmail(ctx, req.EmailUser, svc, builderUserID),
		UsesK8sPipeline:  s.serviceUsesK8sPipeline(ctx, svc),
		HarborURL:        ov.HarborURL,
		HarborProject:    ov.HarborProject,
		ApolloMeta:       ov.ApolloMeta,
		ApolloEnv:        ov.ApolloEnv,
		ApolloNamespaces: ov.ApolloNamespaces,
		YunshuBuildRunID: run.ID,
	})
	lastNum, _ := client.GetLastBuildNumber(ctx, svc.JenkinsJob)
	queuePath, err := client.BuildWithParameters(ctx, svc.JenkinsJob, params)
	if err != nil {
		_ = s.db.WithContext(ctx).Model(&run).Updates(map[string]any{
			"build_result": model.CicdRunStatusFailure,
			"params_json":  ParamsJSON(params),
			"finished_at":  time.Now(),
		}).Error
		return nil, fmt.Errorf("trigger jenkins build: %w", err)
	}
	updates := map[string]any{
		"build_result": model.CicdRunStatusRunning,
		"params_json":  ParamsJSON(params),
		"branch_name":  params["branchName"],
		"publish_mode": params["publishMode"],
		"tenv":         params["Tenv"],
		"build_number": lastNum + 1,
	}
	if buildNum, err := client.ResolveQueueBuildNumber(ctx, queuePath, lastNum, 90*time.Second); err == nil && buildNum > 0 {
		updates["build_number"] = buildNum
		updates["jenkins_build_url"] = client.BuildURL(svc.JenkinsJob, buildNum)
	}
	if err := s.db.WithContext(ctx).Model(&run).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", run.ID).First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *Service) TriggerRelease(ctx context.Context, projectID, serviceID uint, req TriggerReleaseRequest, submitterUserID *uint, submitterName string) (*model.CicdReleaseRun, error) {
	p, err := s.prepareRelease(ctx, projectID, serviceID, req, submitterUserID)
	if err != nil {
		return nil, err
	}
	if p.dc.AuditEnabled {
		return s.createPendingRelease(ctx, projectID, serviceID, p, submitterUserID, submitterName)
	}
	auditRequired := s.enforceProdDeployAudit(ctx, p.dc.Tenv, boolPtr(p.dc.AuditEnabled))
	if auditRequired {
		return s.createPendingRelease(ctx, projectID, serviceID, p, submitterUserID, submitterName)
	}

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
		Status:          model.CicdRunStatusRunning,
		SubmitterUserID: submitterUserID,
		SubmitterName:   submitterName,
		ImageAddress:    p.imageAddress,
		ArtifactName:    p.artifactName,
		AuditEnabled:    false,
		RequestJSON:     snapshotJSON(p),
		StartedAt:       &now,
	}
	if err := s.db.WithContext(ctx).Create(&release).Error; err != nil {
		return nil, err
	}
	if err := s.executeReleaseRun(ctx, &release, submitterUserID); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", release.ID).First(&release).Error; err != nil {
		return nil, err
	}
	recordReleaseChange(ctx, s.db, &release, "release_create", model.ChangeStatusStarted,
		fmt.Sprintf("创建并执行发布 #%d：%s", release.ID, release.Title))
	return &release, nil
}

func (s *Service) resolveDestIPs(ctx context.Context, projectID uint, dc *model.CicdDeployConfig) (string, error) {
	if dc == nil || strings.EqualFold(dc.DeployKind, model.CicdDeployKindContainer) {
		return "", nil
	}
	ids, err := ParseServerIDs(dc.ServerIDsJSON)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", nil
	}
	if s.serverRepo == nil {
		return "", nil
	}
	var hosts []string
	for _, id := range ids {
		srv, err := s.serverRepo.GetByID(ctx, id)
		if err != nil {
			continue
		}
		if srv.ProjectID != projectID {
			continue
		}
		host := strings.TrimSpace(srv.Host)
		if host == "" {
			continue
		}
		if srv.Port > 0 && srv.Port != 22 {
			host = host + ":" + strconv.Itoa(srv.Port)
		}
		hosts = append(hosts, host)
	}
	return strings.Join(hosts, ","), nil
}

func boolPtr(v bool) *bool {
	return &v
}
