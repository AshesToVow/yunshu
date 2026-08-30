package cicd

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"

	"github.com/gin-gonic/gin"
)

// JenkinsCallbackRequest Jenkins 共享库回调载荷。
type JenkinsCallbackRequest struct {
	Event       string                   `json:"event"` // stage|artifact|sonar|run
	RunKind     string                   `json:"run_kind"`
	RunID       uint                     `json:"run_id"`
	JenkinsJob  string                   `json:"jenkins_job"`
	BuildNumber int                      `json:"build_number"`
	GitCommit   string                   `json:"git_commit"`
	Stage       *JenkinsCallbackStage    `json:"stage,omitempty"`
	Sonar       *JenkinsCallbackSonar    `json:"sonar,omitempty"`
	Artifact    *JenkinsCallbackArtifact `json:"artifact,omitempty"`
	RunStatus   string                   `json:"run_status,omitempty"`
}

type JenkinsCallbackStage struct {
	Order        int    `json:"order"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	Logs         string `json:"logs"`
	DurationSec  int    `json:"duration_sec"`
}

type JenkinsCallbackSonar struct {
	ProjectKey      string  `json:"project_key"`
	QualityGate     string  `json:"quality_gate"`
	DashboardURL    string  `json:"dashboard_url"`
	Bugs            int     `json:"bugs"`
	Vulnerabilities int     `json:"vulnerabilities"`
	CodeSmells      int     `json:"code_smells"`
	Coverage        float64 `json:"coverage"`
	Duplications    float64 `json:"duplications"`
}

type JenkinsCallbackArtifact struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	StoragePath string `json:"storage_path"`
	Digest      string `json:"digest"`
	GitCommit   string `json:"git_commit"`
	SizeBytes   int64  `json:"size_bytes"`
}

// callbackTimestampSkew Jenkins 回调时间戳允许的最大偏差（前后各 5 分钟）。
// 超窗即拒：只有签名没有时间戳时，抓到一次请求即可无限重放。
const callbackTimestampSkew = 5 * time.Minute

// VerifyJenkinsCallbackHMAC 校验 X-Yunshu-Signature（sha256=<hex>）或 X-Hub-Signature-256。
// 签名内容为原始 body。仅在回调未携带 X-Yunshu-Timestamp 时使用（兼容存量共享库）。
func VerifyJenkinsCallbackHMAC(secret string, body []byte, signatureHeader string) bool {
	return verifyCallbackSignature(secret, body, signatureHeader)
}

// VerifyJenkinsCallbackSigned 校验带时间戳的回调签名，抵御重放。
// timestampHeader 为 X-Yunshu-Timestamp（Unix 秒）。签名内容为 "<timestamp>.<body>"。
// timestampHeader 为空时退化为 body-only 校验，保持与旧版 Jenkins 共享库兼容。
func VerifyJenkinsCallbackSigned(secret string, body []byte, signatureHeader, timestampHeader string, now time.Time) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return constants.ErrBadRequestWithMsg("未配置 cicd_jenkins_callback_hmac_secret，拒绝回调")
	}
	ts := strings.TrimSpace(timestampHeader)
	if ts == "" {
		// 存量共享库未升级：仅校验 body 签名（无重放保护）。
		if !verifyCallbackSignature(secret, body, signatureHeader) {
			return constants.ErrUnauthorizedWithMsg("Jenkins 回调签名校验失败")
		}
		return nil
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return constants.ErrBadRequestWithMsg("X-Yunshu-Timestamp 必须是 Unix 秒")
	}
	diff := now.Sub(time.Unix(sec, 0))
	if diff < 0 {
		diff = -diff
	}
	if diff > callbackTimestampSkew {
		return constants.ErrUnauthorizedWithMsg("Jenkins 回调时间戳超出允许窗口，疑似重放")
	}
	signed := make([]byte, 0, len(ts)+1+len(body))
	signed = append(signed, ts...)
	signed = append(signed, '.')
	signed = append(signed, body...)
	if !verifyCallbackSignature(secret, signed, signatureHeader) {
		return constants.ErrUnauthorizedWithMsg("Jenkins 回调签名校验失败")
	}
	return nil
}

func verifyCallbackSignature(secret string, payload []byte, signatureHeader string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	sig := strings.TrimSpace(signatureHeader)
	sig = strings.TrimPrefix(sig, "sha256=")
	sig = strings.TrimPrefix(sig, "SHA256=")
	if sig == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(sig)))
}

// HandleJenkinsCallbackRaw 供 handler 读取 body 后调用。
func (s *Service) HandleJenkinsCallbackRaw(ctx context.Context, body []byte, signatureHeader, timestampHeader string) error {
	cfg := s.resolvedConfig(ctx)
	secret := strings.TrimSpace(cfg.Callback.HMACSecret)
	if secret == "" {
		return constants.ErrBadRequestWithMsg("未配置 cicd_jenkins_callback_hmac_secret，拒绝回调")
	}
	if err := VerifyJenkinsCallbackSigned(secret, body, signatureHeader, timestampHeader, time.Now()); err != nil {
		return err
	}
	var req JenkinsCallbackRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return constants.ErrBadRequestWithMsg("回调 JSON 无效")
	}
	return s.applyJenkinsCallback(ctx, req)
}

func (s *Service) applyJenkinsCallback(ctx context.Context, req JenkinsCallbackRequest) error {
	runKind := strings.ToLower(strings.TrimSpace(req.RunKind))
	if runKind == "" {
		runKind = model.CicdRunKindBuild
	}
	event := strings.ToLower(strings.TrimSpace(req.Event))
	if event == "" {
		if req.Stage != nil {
			event = "stage"
		} else if req.Artifact != nil {
			event = "artifact"
		} else if req.Sonar != nil {
			event = "sonar"
		} else {
			event = "run"
		}
	}

	switch runKind {
	case model.CicdRunKindBuild:
		br, err := s.resolveBuildRunForCallback(ctx, req)
		if err != nil {
			return err
		}
		if err := s.applyBuildCallback(ctx, br, event, req); err != nil {
			return err
		}
	case model.CicdRunKindRelease:
		rr, err := s.resolveReleaseRunForCallback(ctx, req)
		if err != nil {
			return err
		}
		if err := s.applyReleaseCallback(ctx, rr, event, req); err != nil {
			return err
		}
	default:
		return constants.ErrBadRequestWithMsg("不支持的 run_kind：" + runKind)
	}
	return nil
}

func (s *Service) resolveBuildRunForCallback(ctx context.Context, req JenkinsCallbackRequest) (*model.CicdBuildRun, error) {
	if req.RunID > 0 {
		var br model.CicdBuildRun
		if err := s.db.WithContext(ctx).Where("id = ?", req.RunID).First(&br).Error; err != nil {
			return nil, constants.ErrNotFound
		}
		return &br, nil
	}
	job := strings.TrimSpace(req.JenkinsJob)
	if job == "" || req.BuildNumber <= 0 {
		return nil, constants.ErrBadRequestWithMsg("回调须提供 run_id，或 jenkins_job + build_number")
	}
	var svc model.CicdService
	if err := s.db.WithContext(ctx).
		Where("jenkins_job = ? OR identifier = ?", job, job).
		Order("id DESC").
		First(&svc).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	var br model.CicdBuildRun
	if err := s.db.WithContext(ctx).
		Where("service_id = ? AND build_number = ?", svc.ID, req.BuildNumber).
		Order("id DESC").
		First(&br).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	return &br, nil
}

func (s *Service) resolveReleaseRunForCallback(ctx context.Context, req JenkinsCallbackRequest) (*model.CicdReleaseRun, error) {
	if req.RunID > 0 {
		var rr model.CicdReleaseRun
		if err := s.db.WithContext(ctx).Where("id = ?", req.RunID).First(&rr).Error; err != nil {
			return nil, constants.ErrNotFound
		}
		return &rr, nil
	}
	job := strings.TrimSpace(req.JenkinsJob)
	if job == "" || req.BuildNumber <= 0 {
		return nil, constants.ErrBadRequestWithMsg("回调须提供 run_id，或 jenkins_job + build_number")
	}
	var svc model.CicdService
	if err := s.db.WithContext(ctx).
		Where("jenkins_job = ? OR identifier = ?", job, job).
		Order("id DESC").
		First(&svc).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	var rr model.CicdReleaseRun
	if err := s.db.WithContext(ctx).
		Where("service_id = ? AND jenkins_build_number = ?", svc.ID, req.BuildNumber).
		Order("id DESC").
		First(&rr).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	return &rr, nil
}

func (s *Service) applyBuildCallback(ctx context.Context, br *model.CicdBuildRun, event string, req JenkinsCallbackRequest) error {
	if req.Stage != nil {
		if err := s.upsertRunStage(ctx, br.ProjectID, br.ServiceID, model.CicdRunKindBuild, br.ID, req.Stage, req.Sonar); err != nil {
			return err
		}
	}
	updates := map[string]any{}
	if v := strings.TrimSpace(req.GitCommit); v != "" {
		updates["git_commit"] = v
	}
	sonar := req.Sonar
	if sonar != nil {
		qg := strings.ToUpper(strings.TrimSpace(sonar.QualityGate))
		if qg != "" {
			updates["quality_gate_status"] = qg
			pass := qg == model.CicdQualityGateOK || qg == model.CicdQualityGateWarn
			updates["security_scan_pass"] = pass
		}
		if v := strings.TrimSpace(sonar.ProjectKey); v != "" {
			updates["sonar_project_key"] = v
		}
		if v := strings.TrimSpace(sonar.DashboardURL); v != "" {
			updates["sonar_dashboard_url"] = v
		}
		if b, err := json.Marshal(sonar); err == nil {
			updates["sonar_summary_json"] = string(b)
		}
	} else if req.Stage != nil &&
		(strings.EqualFold(req.Stage.Type, model.CicdStageTypeQualityGate) ||
			strings.EqualFold(req.Stage.Type, model.CicdStageTypeSonar)) {
		// stage 事件可只带 status（无 sonar 明细）：门禁状态从 stage.status 映射。
		// sonar 与 quality_gate 两种 stage 类型都要覆盖，否则 sonar 类型阶段的门禁结果会丢失。
		qg := mapStageStatusToQualityGate(req.Stage.Status)
		updates["quality_gate_status"] = qg
		pass := qg == model.CicdQualityGateOK || qg == model.CicdQualityGateWarn
		updates["security_scan_pass"] = pass
	}
	if req.Artifact != nil {
		if err := s.upsertArtifactFromCallback(ctx, br, req.Artifact); err != nil {
			return err
		}
		at := strings.ToLower(strings.TrimSpace(req.Artifact.Type))
		path := strings.TrimSpace(req.Artifact.StoragePath)
		if path == "" {
			path = strings.TrimSpace(req.Artifact.Name)
		}
		switch at {
		case model.CicdArtifactTypeImage:
			if path != "" {
				updates["image_address"] = path
			}
		case model.CicdArtifactTypePackage, "":
			if path != "" {
				updates["package_path"] = path
			}
		}
	}
	if event == "run" {
		st := strings.ToLower(strings.TrimSpace(req.RunStatus))
		if st != "" {
			if !buildCallbackStatusAllowed(br.BuildResult, st) {
				return constants.ErrBadRequestWithMsg("Jenkins 回调状态不允许: " + br.BuildResult + " -> " + st)
			}
			updates["build_result"] = st
			if st != model.CicdRunStatusRunning && st != model.CicdRunStatusPending {
				now := time.Now()
				updates["finished_at"] = now
			}
		}
	}
	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		// 带 build_result 条件更新：与 run_sync 轮询并发时，避免用过期回调覆盖已落地的终态。
		q := s.db.WithContext(ctx).Model(&model.CicdBuildRun{}).Where("id = ?", br.ID)
		if _, ok := updates["build_result"]; ok {
			q = q.Where("build_result = ?", br.BuildResult)
		}
		res := q.Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			if _, ok := updates["build_result"]; ok {
				// 状态已被他人（轮询/并发回调）推进，本次回调按幂等处理，不报错。
				return nil
			}
		}
	}
	return nil
}

// buildCallbackStatusAllowed CI 回调仅允许 pending/running 向前推进，终态不可回退。
// 与 releaseCallbackStatusAllowed 对称：防止重复回调或乱序回调把已完成的构建改回 running。
func buildCallbackStatusAllowed(current, next string) bool {
	current = strings.ToLower(strings.TrimSpace(current))
	next = strings.ToLower(strings.TrimSpace(next))
	if next == "" {
		return false
	}
	switch current {
	case "", model.CicdRunStatusPending, model.CicdRunStatusRunning:
		switch next {
		case model.CicdRunStatusPending, model.CicdRunStatusRunning, model.CicdRunStatusSuccess,
			model.CicdRunStatusFailure, model.CicdRunStatusAborted, model.CicdRunStatusCancelled:
			return true
		default:
			return false
		}
	default:
		// 已是终态：只允许同值重复回调（幂等），不允许改成别的状态。
		return current == next
	}
}

func (s *Service) applyReleaseCallback(ctx context.Context, rr *model.CicdReleaseRun, event string, req JenkinsCallbackRequest) error {
	if req.Stage != nil {
		if err := s.upsertRunStage(ctx, rr.ProjectID, rr.ServiceID, model.CicdRunKindRelease, rr.ID, req.Stage, req.Sonar); err != nil {
			return err
		}
	}
	if event != "run" {
		return nil
	}
	st := strings.ToLower(strings.TrimSpace(req.RunStatus))
	if st == "" {
		return nil
	}
	if !releaseCallbackStatusAllowed(rr.Status, st) {
		return constants.ErrBadRequestWithMsg("Jenkins 回调状态不允许: " + rr.Status + " -> " + st)
	}
	updates := map[string]any{
		"status":     st,
		"updated_at": time.Now(),
	}
	if st != model.CicdRunStatusRunning && st != model.CicdRunStatusPending {
		now := time.Now()
		updates["finished_at"] = now
	}
	if req.BuildNumber > 0 && rr.JenkinsBuildNumber == 0 {
		updates["jenkins_build_number"] = req.BuildNumber
	}
	return s.db.WithContext(ctx).Model(&model.CicdReleaseRun{}).Where("id = ?", rr.ID).Updates(updates).Error
}

// releaseCallbackStatusAllowed Jenkins run 回调仅允许从执行中/待执行推进到终态或保持 running。
func releaseCallbackStatusAllowed(current, next string) bool {
	current = strings.ToLower(strings.TrimSpace(current))
	next = strings.ToLower(strings.TrimSpace(next))
	switch current {
	case model.CicdRunStatusRunning:
		switch next {
		case model.CicdRunStatusRunning, model.CicdRunStatusSuccess, model.CicdRunStatusFailure,
			model.CicdRunStatusAborted, model.CicdRunStatusCancelled:
			return true
		default:
			return false
		}
	case model.CicdRunStatusPendingExecution:
		return next == model.CicdRunStatusRunning
	default:
		return false
	}
}

func mapStageStatusToQualityGate(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case model.CicdStageStatusSuccess:
		return model.CicdQualityGateOK
	case model.CicdStageStatusFailed:
		return model.CicdQualityGateError
	case model.CicdStageStatusSkipped:
		return model.CicdQualityGateNone
	default:
		return model.CicdQualityGateNone
	}
}

func (s *Service) upsertRunStage(
	ctx context.Context,
	projectID, serviceID uint,
	runKind string,
	runID uint,
	stage *JenkinsCallbackStage,
	sonar *JenkinsCallbackSonar,
) error {
	if stage == nil {
		return nil
	}
	stageType := strings.TrimSpace(stage.Type)
	if stageType == "" {
		return constants.ErrBadRequestWithMsg("stage.type 不能为空")
	}
	status := strings.TrimSpace(stage.Status)
	if status == "" {
		status = model.CicdStageStatusRunning
	}
	name := strings.TrimSpace(stage.Name)
	if name == "" {
		name = stageType
	}
	extra := ""
	if sonar != nil {
		if b, err := json.Marshal(sonar); err == nil {
			extra = string(b)
		}
	}
	now := time.Now()
	var existing model.CicdRunStage
	err := s.db.WithContext(ctx).
		Where("run_kind = ? AND run_id = ? AND stage_type = ? AND stage_order = ?",
			runKind, runID, stageType, stage.Order).
		First(&existing).Error
	if err == nil {
		updates := map[string]any{
			"stage_name":    name,
			"status":        status,
			"error_message": strings.TrimSpace(stage.ErrorMessage),
			"updated_at":    now,
		}
		if stage.Logs != "" {
			updates["logs"] = stage.Logs
		}
		if stage.DurationSec > 0 {
			updates["duration_sec"] = stage.DurationSec
		}
		if extra != "" {
			updates["extra_json"] = extra
		}
		if status == model.CicdStageStatusRunning && existing.StartedAt == nil {
			updates["started_at"] = now
		}
		if status == model.CicdStageStatusSuccess || status == model.CicdStageStatusFailed || status == model.CicdStageStatusSkipped {
			updates["finished_at"] = now
			if existing.StartedAt == nil {
				updates["started_at"] = now
			}
		}
		return s.db.WithContext(ctx).Model(&existing).Updates(updates).Error
	}
	row := model.CicdRunStage{
		ProjectID:    projectID,
		ServiceID:    serviceID,
		RunKind:      runKind,
		RunID:        runID,
		StageOrder:   stage.Order,
		StageType:    stageType,
		StageName:    name,
		Status:       status,
		DurationSec:  stage.DurationSec,
		Logs:         stage.Logs,
		ErrorMessage: strings.TrimSpace(stage.ErrorMessage),
		ExtraJSON:    extra,
	}
	if status != model.CicdStageStatusPending {
		row.StartedAt = &now
	}
	if status == model.CicdStageStatusSuccess || status == model.CicdStageStatusFailed || status == model.CicdStageStatusSkipped {
		row.FinishedAt = &now
	}
	return s.db.WithContext(ctx).Create(&row).Error
}

func (s *Service) upsertArtifactFromCallback(ctx context.Context, br *model.CicdBuildRun, art *JenkinsCallbackArtifact) error {
	if art == nil {
		return nil
	}
	name := strings.TrimSpace(art.Name)
	path := strings.TrimSpace(art.StoragePath)
	if name == "" {
		name = path
	}
	if name == "" {
		return constants.ErrBadRequestWithMsg("artifact.name 或 storage_path 不能为空")
	}
	at := strings.ToLower(strings.TrimSpace(art.Type))
	if at == "" {
		at = model.CicdArtifactTypePackage
	}
	gitCommit := strings.TrimSpace(art.GitCommit)
	if gitCommit == "" {
		gitCommit = strings.TrimSpace(br.GitCommit)
	}
	var existing model.CicdArtifact
	q := s.db.WithContext(ctx).Where("build_run_id = ? AND artifact_type = ?", br.ID, at)
	if path != "" {
		q = q.Where("storage_path = ?", path)
	} else {
		q = q.Where("name = ?", name)
	}
	err := q.First(&existing).Error
	if err == nil {
		return s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
			"name":         name,
			"storage_path": path,
			"digest":       strings.TrimSpace(art.Digest),
			"git_commit":   gitCommit,
			"size_bytes":   art.SizeBytes,
			"updated_at":   time.Now(),
		}).Error
	}
	row := model.CicdArtifact{
		ProjectID:    br.ProjectID,
		ServiceID:    br.ServiceID,
		BuildRunID:   br.ID,
		ArtifactType: at,
		Name:         name,
		StoragePath:  path,
		Digest:       strings.TrimSpace(art.Digest),
		GitCommit:    gitCommit,
		SizeBytes:    art.SizeBytes,
	}
	return s.db.WithContext(ctx).Create(&row).Error
}

// ListBuildRunStages 返回构建阶段列表。
func (s *Service) ListBuildRunStages(ctx context.Context, projectID, runID uint, actor *auth.CurrentUser) ([]model.CicdRunStage, error) {
	var br model.CicdBuildRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&br).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	if err := s.AssertCicdAccess(ctx, projectID, br.ServiceID, actor, "view"); err != nil {
		return nil, err
	}
	var rows []model.CicdRunStage
	if err := s.db.WithContext(ctx).
		Where("run_kind = ? AND run_id = ?", model.CicdRunKindBuild, runID).
		Order("stage_order ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []model.CicdRunStage{}
	}
	return rows, nil
}

// ListBuildRunArtifactsMeta 返回构建关联的制品元数据。
func (s *Service) ListBuildRunArtifactsMeta(ctx context.Context, projectID, runID uint, actor *auth.CurrentUser) ([]model.CicdArtifact, error) {
	var br model.CicdBuildRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&br).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	if err := s.AssertCicdAccess(ctx, projectID, br.ServiceID, actor, "view"); err != nil {
		return nil, err
	}
	var rows []model.CicdArtifact
	if err := s.db.WithContext(ctx).
		Where("build_run_id = ?", runID).
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []model.CicdArtifact{}
	}
	return rows, nil
}

// ReadCallbackBody 读取 gin 请求体（供 handler 复用）。
func ReadCallbackBody(c *gin.Context) ([]byte, error) {
	defer c.Request.Body.Close()
	return io.ReadAll(io.LimitReader(c.Request.Body, 2<<20))
}
