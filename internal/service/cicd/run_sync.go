package cicd

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/jenkins"
)

// cicdSyncTickTimeout 单轮 Jenkins 状态同步的耗时上界，保证优雅关闭可及时收敛。
const cicdSyncTickTimeout = 2 * time.Minute

// RunSyncWorker 后台同步 Jenkins 构建状态。
func (s *Service) RunSyncWorker(ctx context.Context) {
	interval := time.Duration(s.resolvedConfig(ctx).RunSyncIntervalSeconds) * time.Second
	if interval < 5*time.Second {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 单次迭代加上界：避免关闭时被 Jenkins 慢响应挂死，同时响应父 ctx 取消
			tickCtx, cancel := context.WithTimeout(ctx, cicdSyncTickTimeout)
			s.syncPendingRuns(tickCtx)
			cancel()
		}
	}
}

func (s *Service) syncPendingRuns(ctx context.Context) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	client, _, err := s.jenkinsClient(ctx)
	if err != nil {
		return
	}
	buildRuns, _ := s.repo.ListActiveBuildRuns(ctx, 50)
	for _, run := range buildRuns {
		s.syncOneBuildRun(ctx, client, run)
	}
	releaseRuns, _ := s.repo.ListActiveReleaseRuns(ctx, 50)
	for _, run := range releaseRuns {
		s.syncOneReleaseRun(ctx, client, run)
	}
	backfillBuilds, _ := s.repo.ListSuccessfulBuildRunsMissingArtifacts(ctx, 30)
	for _, run := range backfillBuilds {
		s.backfillBuildArtifacts(ctx, client, run)
	}
	s.syncApprovalReminders(ctx)
}

func (s *Service) backfillBuildArtifacts(ctx context.Context, client *jenkins.Client, run model.CicdBuildRun) {
	if run.BuildNumber <= 0 {
		return
	}
	needPackage := strings.TrimSpace(run.PackagePath) == ""
	needImage := strings.TrimSpace(run.ImageAddress) == ""
	if !needPackage && !needImage {
		return
	}
	svc, err := s.repo.GetServiceByID(ctx, run.ServiceID)
	if err != nil {
		return
	}
	ci, _ := s.repo.GetCIConfig(ctx, run.ServiceID)
	jobName := resolveJenkinsJobName(svc)
	if jobName == "" {
		return
	}
	logText, err := client.GetConsoleLog(ctx, jobName, run.BuildNumber)
	if err != nil {
		return
	}
	var ciRow model.CicdCiConfig
	if ci != nil {
		ciRow = *ci
	}
	artifacts := s.resolveBuildArtifactsFromLog(ctx, *svc, ciRow, logText)
	updates := map[string]any{}
	if needPackage && strings.TrimSpace(artifacts.PackagePath) != "" {
		updates["package_path"] = artifacts.PackagePath
	}
	if needImage && strings.TrimSpace(artifacts.ImageAddress) != "" {
		updates["image_address"] = artifacts.ImageAddress
	}
	if len(updates) == 0 {
		return
	}
	_ = s.repo.UpdateBuildRunFields(ctx, run.ID, updates)
}

func (s *Service) syncOneBuildRun(ctx context.Context, client *jenkins.Client, run model.CicdBuildRun) {
	if run.BuildNumber <= 0 {
		return
	}
	svc, err := s.repo.GetServiceByID(ctx, run.ServiceID)
	if err != nil {
		return
	}
	ci, _ := s.repo.GetCIConfig(ctx, run.ServiceID)
	jobName := resolveJenkinsJobName(svc)
	if jobName == "" {
		return
	}
	info, err := client.GetBuild(ctx, jobName, run.BuildNumber)
	if err != nil {
		return
	}
	status := jenkins.MapResultToStatus(info.Result, info.Building)
	updates := map[string]any{
		"jenkins_build_url": info.URL,
		"updated_at":        time.Now(),
	}
	var ciRow model.CicdCiConfig
	if ci != nil {
		ciRow = *ci
	}
	if strings.TrimSpace(run.PackagePath) == "" || strings.TrimSpace(run.ImageAddress) == "" {
		if logText, err := client.GetConsoleLog(ctx, jobName, run.BuildNumber); err == nil {
			artifacts := s.resolveBuildArtifactsFromLog(ctx, *svc, ciRow, logText)
			if strings.TrimSpace(run.PackagePath) == "" && strings.TrimSpace(artifacts.PackagePath) != "" {
				updates["package_path"] = artifacts.PackagePath
			}
			if strings.TrimSpace(run.ImageAddress) == "" && strings.TrimSpace(artifacts.ImageAddress) != "" {
				updates["image_address"] = artifacts.ImageAddress
			}
		}
	}
	if status == model.CicdRunStatusRunning {
		_ = s.repo.UpdateBuildRunFields(ctx, run.ID, updates)
		return
	}
	now := time.Now()
	updates["build_result"] = status
	if run.FinishedAt == nil {
		updates["finished_at"] = now
	}
	// 条件更新：仅当 DB 内仍是非终态时才写终态，避免与 Jenkins HMAC 回调并发时覆盖已落地结果。
	_, _ = s.repo.UpdateBuildRunFieldsIfStatus(ctx, run.ID, []string{model.CicdRunStatusRunning, model.CicdRunStatusPending}, updates)
}

// releaseStuckTimeout 发布工单在 running 且构建号仍未落库时，允许的最长补偿窗口。
// 超过后仍拿不到构建号，则判定为触发/回填异常并置为 failure，避免永久卡死。
const releaseStuckTimeout = 30 * time.Minute

func (s *Service) syncOneReleaseRun(ctx context.Context, client *jenkins.Client, run model.CicdReleaseRun) {
	svc, err := s.repo.GetServiceByID(ctx, run.ServiceID)
	if err != nil {
		return
	}
	// 构建号未落库（如触发后请求上下文被取消）：先尝试用 queue_url 补偿解析。
	if run.JenkinsBuildNumber <= 0 {
		if !s.recoverReleaseBuildNumber(ctx, client, svc, &run) {
			return
		}
	}
	info, err := client.GetBuild(ctx, svc.JenkinsJob, run.JenkinsBuildNumber)
	if err != nil {
		return
	}
	status := jenkins.MapResultToStatus(info.Result, info.Building)
	if status == model.CicdRunStatusRunning {
		return
	}
	now := time.Now()
	updates := map[string]any{
		"status":            status,
		"jenkins_build_url": info.URL,
		"updated_at":        now,
	}
	if run.FinishedAt == nil {
		updates["finished_at"] = now
	}
	// 条件更新：仅当 DB 内仍是非终态时才写终态，避免覆盖回调或人工终止已落地的状态。
	_, _ = s.repo.UpdateReleaseRunFieldsIfStatus(ctx, run.ID, []string{model.CicdRunStatusRunning, model.CicdRunStatusPending}, updates)
}

// recoverReleaseBuildNumber 针对构建号未落库的 running 工单做补偿：
// 依据存下的 queue_url 解析 Jenkins 已分配的构建号并回填 run（含内存副本）。
// 解析成功返回 true；仍未分配则返回 false（下轮再试）；
// 若已超过 releaseStuckTimeout 仍无构建号，则置为 failure 以避免永久卡死。
func (s *Service) recoverReleaseBuildNumber(ctx context.Context, client *jenkins.Client, svc *model.CicdService, run *model.CicdReleaseRun) bool {
	queueURL := strings.TrimSpace(run.JenkinsQueueURL)
	if queueURL != "" {
		if buildNum, err := client.QueueBuildNumber(ctx, queueURL); err == nil && buildNum > 0 {
			updates := map[string]any{
				"jenkins_build_number": buildNum,
				"jenkins_build_url":    client.BuildURL(svc.JenkinsJob, buildNum),
			}
			if affected, err := s.repo.UpdateReleaseRunFieldsIfBuildNumberZero(ctx, run.ID, updates); err == nil && affected > 0 {
				run.JenkinsBuildNumber = buildNum
				return true
			}
		}
	}
	// 触发时机参考 started_at（无则回退 created_at）；超窗仍无构建号判定为失败。
	ref := run.StartedAt
	if ref == nil {
		ref = &run.CreatedAt
	}
	if ref != nil && time.Since(*ref) > releaseStuckTimeout {
		now := time.Now()
		_, _ = s.repo.ClaimReleaseRunNoBuildNumber(ctx, run.ID, model.CicdRunStatusRunning, map[string]any{
			"status":      model.CicdRunStatusFailure,
			"finished_at": now,
			"updated_at":  now,
		})
	}
	return false
}
