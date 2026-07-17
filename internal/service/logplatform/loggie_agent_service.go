package logplatform

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/repository"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

const loggieHeartbeatTimeout = 3 * time.Minute
const loggieIngestFreshWindow = 5 * time.Minute

type LoggieAgentService struct {
	repo          interfaces.LoggieAgentRepository
	serverRepo    interfaces.ServerRepository
	logSourceRepo interfaces.LogSourceRepository
	projectRepo   interfaces.ProjectRepository
	serviceRepo   interfaces.ServiceRepository
	esProvider    *ElasticsearchProvider
	kafkaProvider *KafkaProvider
	aead          cipher.AEAD
	loggieCfg     config.LoggieConfig
}

func NewLoggieAgentService(
	repo interfaces.LoggieAgentRepository,
	serverRepo interfaces.ServerRepository,
	logSourceRepo interfaces.LogSourceRepository,
	projectRepo interfaces.ProjectRepository,
	serviceRepo interfaces.ServiceRepository,
	esProvider *ElasticsearchProvider,
	kafkaProvider *KafkaProvider,
	encryptionKey string,
	loggieCfg config.LoggieConfig,
) (*LoggieAgentService, error) {
	aead, err := cryptox.NewAESGCMFromKeyString(encryptionKey)
	if err != nil {
		return nil, err
	}
	return &LoggieAgentService{
		repo:          repo,
		serverRepo:    serverRepo,
		logSourceRepo: logSourceRepo,
		projectRepo:   projectRepo,
		serviceRepo:   serviceRepo,
		esProvider:    esProvider,
		kafkaProvider: kafkaProvider,
		aead:          aead,
		loggieCfg:     loggieCfg.Normalized(),
	}, nil
}

type LoggieHeartbeatRequest struct {
	Token               string `json:"token" binding:"required"`
	Version             string `json:"version"`
	HealthStatus        string `json:"health_status"`
	PipelineStatus      string `json:"pipeline_status"`
	EsSinkOK            bool   `json:"es_sink_ok"`
	LinesPerMin         int    `json:"lines_per_min"`
	LastError           string `json:"last_error"`
	MonitorReachable    bool   `json:"monitor_reachable"`
	MonitorPort         int    `json:"monitor_port"`
	ActivePipelineCount int    `json:"active_pipeline_count"`
	ActiveFdCount       int    `json:"active_fd_count"`
	InactiveFdCount     int    `json:"inactive_fd_count"`
	MonitorDetail       string `json:"monitor_detail"`
}

type LoggieBootstrapRequest struct {
	ServerID             uint     `json:"server_id"`
	LogPaths             []string `json:"log_paths"`
	ServiceID            uint     `json:"service_id"`
	LogSourceID          uint     `json:"log_source_id"`
	MonitorPort          int      `json:"monitor_port"`
	YunshuURL            string   `json:"yunshu_url"`
	DeployDir            string   `json:"deploy_dir"`
	AutoFromLogSources   *bool    `json:"auto_from_log_sources"`
	DeployAfterBootstrap bool     `json:"deploy_after_bootstrap"`
}

type LoggieDeployRequest struct {
	ServerID      uint `json:"server_id"`
	RestartLoggie bool `json:"restart_loggie"`
	SyncFromDB    bool `json:"sync_from_db"`
}

type LoggieInstallRequest struct {
	ServerID    uint   `json:"server_id"`
	BinaryURL   string `json:"binary_url"`
	DeployDir   string `json:"deploy_dir"`
	YunshuURL   string `json:"yunshu_url"`
	MonitorPort int    `json:"monitor_port"`
}

// LoggieUninstallRequest 删除 Agent：远端卸载 + 清除平台登记。
type LoggieUninstallRequest struct {
	ServerID    uint `json:"server_id"`
	SkipRemote  bool `json:"skip_remote"`  // 仅清登记，不 SSH
	KeepFiles   bool `json:"keep_files"`   // 远端保留部署目录文件
	ForceLocal  bool `json:"force_local"`  // SSH 失败时仍清登记
}

type LoggieDeployResult struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	Stdout        string `json:"stdout,omitempty"`
	Stderr        string `json:"stderr,omitempty"`
	PipelineCount int    `json:"pipeline_count"`
	DeployedAt    string `json:"deployed_at,omitempty"`
	SourceCount   int    `json:"source_count"`
}

type LoggieBootstrapResult struct {
	Token             string   `json:"token"`
	ProjectID         uint     `json:"project_id"`
	ServerID          uint     `json:"server_id"`
	ESAddresses       []string `json:"es_addresses"`
	ESIndexPattern    string   `json:"es_index_pattern"`
	ReportURL         string   `json:"report_url"`
	PipelineHint      string   `json:"pipeline_hint"`
	PipelineYAML      string   `json:"pipeline_yaml"`
	PipelinesOnlyYAML string   `json:"pipelines_only_yaml"`
	PipelineFilename  string   `json:"pipeline_filename"`
	PipelinesFilename string   `json:"pipelines_filename"`
	EnvFile           string   `json:"env_file"`
	EnvFilename       string   `json:"env_filename"`
	HeartbeatScript   string   `json:"heartbeat_script"`
	HeartbeatFilename string   `json:"heartbeat_filename"`
	StartScript       string   `json:"start_script"`
	StartFilename     string   `json:"start_filename"`
	MonitorPort       int      `json:"monitor_port"`
	PipelineCount     int      `json:"pipeline_count"`
	SourceCount       int      `json:"source_count"`
	Deployed          bool     `json:"deployed"`
	DeployMessage     string   `json:"deploy_message,omitempty"`
}

type loggieBootstrapSource struct {
	LogSourceID  uint     `json:"log_source_id"`
	ServiceID    uint     `json:"service_id"`
	LogType      string   `json:"log_type"`
	Path         string   `json:"path"`
	IncludeRegex string   `json:"include_regex,omitempty"`
	ExcludeRegex string   `json:"exclude_regex,omitempty"`
	Encoding     string   `json:"encoding,omitempty"`
	Paths        []string `json:"paths,omitempty"`
}

type loggieStoredBootstrapConfig struct {
	MonitorPort        int                     `json:"monitor_port"`
	YunshuURL          string                  `json:"yunshu_url"`
	DeployDir          string                  `json:"deploy_dir"`
	AutoFromLogSources bool                    `json:"auto_from_log_sources"`
	Sources            []loggieBootstrapSource `json:"sources"`
	// legacy single-pipeline fields
	LogPaths    []string `json:"log_paths,omitempty"`
	ServiceID   uint     `json:"service_id,omitempty"`
	LogSourceID uint     `json:"log_source_id,omitempty"`
}

type LoggieStatusItem struct {
	ServerID            uint                      `json:"server_id"`
	ServerName          string                    `json:"server_name"`
	ServerHost          string                    `json:"server_host"`
	Registered          bool                      `json:"registered"`
	Online              bool                      `json:"online"`
	RecentIngest        bool                      `json:"recent_ingest"`
	Version             string                    `json:"version,omitempty"`
	HealthStatus        string                    `json:"health_status,omitempty"`
	PipelineStatus      string                    `json:"pipeline_status,omitempty"`
	EsSinkOK            bool                      `json:"es_sink_ok"`
	LinesPerMin         int                       `json:"lines_per_min"`
	LastSeenAt          *string                   `json:"last_seen_at,omitempty"`
	LastIngestAt        *string                   `json:"last_ingest_at,omitempty"`
	LastError           string                    `json:"last_error,omitempty"`
	RecentDocCount      int64                     `json:"recent_doc_count"`
	MonitorPort         int                       `json:"monitor_port"`
	MonitorReachable    bool                      `json:"monitor_reachable"`
	ActivePipelineCount int                       `json:"active_pipeline_count"`
	ActiveFdCount       int                       `json:"active_fd_count"`
	InactiveFdCount     int                       `json:"inactive_fd_count"`
	MonitorDetail       string                    `json:"monitor_detail,omitempty"`
	LiveProbe           *LoggieMonitorProbeResult `json:"live_probe,omitempty"`
}

func (s *LoggieAgentService) ReportHeartbeat(ctx context.Context, req LoggieHeartbeatRequest) error {
	agent, err := s.repo.GetByToken(ctx, strings.TrimSpace(req.Token))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrAgentTokenInvalid
		}
		return bizerrors.Pass(ctx, "loggie", "ReportHeartbeat", err)
	}
	now := time.Now()
	agent.LastSeenAt = &now
	if v := strings.TrimSpace(req.Version); v != "" {
		agent.Version = v
	}
	if v := strings.TrimSpace(req.HealthStatus); v != "" {
		agent.HealthStatus = v
	}
	if v := strings.TrimSpace(req.PipelineStatus); v != "" {
		agent.PipelineStatus = v
	}
	agent.EsSinkOK = req.EsSinkOK
	if req.LinesPerMin > 0 {
		agent.LinesPerMin = req.LinesPerMin
	}
	agent.LastError = strings.TrimSpace(req.LastError)
	agent.MonitorReachable = req.MonitorReachable
	if req.MonitorPort > 0 {
		agent.MonitorPort = req.MonitorPort
	} else if agent.MonitorPort == 0 {
		agent.MonitorPort = 9196
	}
	agent.ActivePipelineCount = req.ActivePipelineCount
	agent.ActiveFdCount = req.ActiveFdCount
	agent.InactiveFdCount = req.InactiveFdCount
	if v := strings.TrimSpace(req.MonitorDetail); v != "" {
		agent.MonitorDetail = truncateString(v, 4096)
		// 兼容旧心跳脚本：从 monitor_detail 回填 inactive
		if agent.InactiveFdCount == 0 {
			var snap struct {
				InactiveFD int `json:"inactive_fd"`
			}
			if json.Unmarshal([]byte(v), &snap) == nil && snap.InactiveFD > 0 {
				agent.InactiveFdCount = snap.InactiveFD
			}
		}
	}
	if req.LinesPerMin > 0 {
		agent.LastIngestAt = &now
	}
	return s.repo.Save(ctx, agent)
}

func (s *LoggieAgentService) Bootstrap(ctx context.Context, projectID uint, req LoggieBootstrapRequest) (*LoggieBootstrapResult, error) {
	if projectID == 0 || req.ServerID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 与 server_id 必填")
	}
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrLogSourceServerNotFound
		}
		return nil, bizerrors.Pass(ctx, "loggie", "Bootstrap", err)
	}
	if sv.ProjectID != projectID {
		return nil, constants.ErrServerNotInCurrentProject
	}
	agent, err := s.ensureAgent(ctx, projectID, req.ServerID, req.MonitorPort)
	if err != nil {
		return nil, err
	}
	stored := s.buildStoredConfigFromRequest(req, nil)
	sources, err := s.resolveBootstrapSources(ctx, projectID, req.ServerID, req, stored.AutoFromLogSources)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "Bootstrap", err)
	}
	stored = s.buildStoredConfigFromRequest(req, sources)
	if raw, err := json.Marshal(stored); err == nil {
		agent.BootstrapConfig = string(raw)
	}
	agent.MonitorPort = stored.MonitorPort
	if err := s.repo.Save(ctx, agent); err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "Bootstrap", err)
	}
	bundle, finalSources, err := s.bundleFromStored(ctx, projectID, req.ServerID, agent, stored, false)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "Bootstrap", err)
	}
	if len(finalSources) > 0 && stored.AutoFromLogSources {
		stored.Sources = finalSources
		if raw, err := json.Marshal(stored); err == nil {
			agent.BootstrapConfig = string(raw)
			_ = s.repo.Save(ctx, agent)
		}
	}
	deployed := false
	deployMsg := ""
	if req.DeployAfterBootstrap {
		out, errOut, err := s.deployBundleOverSSH(ctx, req.ServerID, bundle)
		if err != nil {
			deployMsg = truncateDeployOutput(err.Error()+": "+out+errOut, 512)
		} else {
			deployed = true
			deployMsg = truncateDeployOutput(out, 256)
		}
	}
	return bootstrapResultFromBundle(agent, projectID, req.ServerID, bundle, esCfg, len(finalSources), deployed, deployMsg), nil
}

func (s *LoggieAgentService) ensureAgent(ctx context.Context, projectID, serverID uint, monitorPort int) (*model.LoggieAgent, error) {
	agent, err := s.repo.GetByProjectAndServer(ctx, projectID, serverID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizerrors.Pass(ctx, "loggie", "Bootstrap", err)
		}
		token, err := newLoggieToken()
		if err != nil {
			return nil, err
		}
		return &model.LoggieAgent{
			ProjectID:      projectID,
			ServerID:       serverID,
			Token:          token,
			HealthStatus:   "unknown",
			PipelineStatus: "unknown",
			MonitorPort:    defaultMonitorPort(monitorPort),
		}, nil
	}
	if strings.TrimSpace(agent.Token) == "" {
		token, err := newLoggieToken()
		if err != nil {
			return nil, err
		}
		agent.Token = token
	}
	return agent, nil
}

func (s *LoggieAgentService) GeneratePipelineBundle(ctx context.Context, projectID, serverID uint) (*LoggiePipelineBundle, error) {
	if projectID == 0 || serverID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 与 server_id 必填")
	}
	agent, err := s.repo.GetByProjectAndServer(ctx, projectID, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrBadRequestWithMsg("请先执行引导登记 Agent")
		}
		return nil, bizerrors.Pass(ctx, "loggie", "GeneratePipelineBundle", err)
	}
	stored := parseStoredBootstrapConfig(agent.BootstrapConfig)
	bundle, _, err := s.bundleFromStored(ctx, projectID, serverID, agent, stored, stored.AutoFromLogSources)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "GeneratePipelineBundle", err)
	}
	return &bundle, nil
}

func (s *LoggieAgentService) DeployConfig(ctx context.Context, projectID uint, req LoggieDeployRequest) (*LoggieDeployResult, error) {
	if projectID == 0 || req.ServerID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 与 server_id 必填")
	}
	agent, err := s.repo.GetByProjectAndServer(ctx, projectID, req.ServerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrBadRequestWithMsg("请先执行引导登记 Agent")
		}
		return nil, bizerrors.Pass(ctx, "loggie", "DeployConfig", err)
	}
	stored := parseStoredBootstrapConfig(agent.BootstrapConfig)
	bundle, sources, err := s.bundleFromStored(ctx, projectID, req.ServerID, agent, stored, req.SyncFromDB || stored.AutoFromLogSources)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "DeployConfig", err)
	}
	if req.SyncFromDB || stored.AutoFromLogSources {
		stored.Sources = sources
		if raw, err := json.Marshal(stored); err == nil {
			agent.BootstrapConfig = string(raw)
			_ = s.repo.Save(ctx, agent)
		}
	}
	stdout, stderr, err := s.deployBundleOverSSH(ctx, req.ServerID, bundle)
	result := &LoggieDeployResult{
		Success:       err == nil,
		PipelineCount: bundle.PipelineCount,
		SourceCount:   len(sources),
		Stdout:        truncateDeployOutput(stdout, 2048),
		Stderr:        truncateDeployOutput(stderr, 2048),
		DeployedAt:    formatDeployTime(time.Now()),
	}
	if err != nil {
		result.Message = truncateDeployOutput(err.Error(), 512)
		return result, nil
	}
	result.Message = "配置已下发"
	out := strings.TrimSpace(stdout)
	if strings.Contains(out, "CONFIG_UPLOADED") {
		result.Message = "配置已下发（尚未安装 systemd 单元，请执行「安装」）"
	} else {
		result.Message = "配置已下发并热更/重启 Loggie"
	}
	return result, nil
}

func (s *LoggieAgentService) StartLoggie(ctx context.Context, projectID uint, req LoggieDeployRequest) (*LoggieDeployResult, error) {
	if projectID == 0 || req.ServerID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 与 server_id 必填")
	}
	if _, err := s.repo.GetByProjectAndServer(ctx, projectID, req.ServerID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrBadRequestWithMsg("请先执行引导登记 Agent")
		}
		return nil, bizerrors.Pass(ctx, "loggie", "StartLoggie", err)
	}
	stdout, stderr, err := s.startLoggieOverSSH(ctx, req.ServerID)
	result := &LoggieDeployResult{
		Success: err == nil,
		Stdout:  truncateDeployOutput(stdout, 2048),
		Stderr:  truncateDeployOutput(stderr, 2048),
	}
	if err != nil {
		result.Message = truncateDeployOutput(err.Error(), 512)
		return result, nil
	}
	result.Message = "Loggie 已启动"
	return result, nil
}

func (s *LoggieAgentService) StopLoggie(ctx context.Context, projectID uint, req LoggieDeployRequest) (*LoggieDeployResult, error) {
	if projectID == 0 || req.ServerID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 与 server_id 必填")
	}
	if _, err := s.repo.GetByProjectAndServer(ctx, projectID, req.ServerID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrBadRequestWithMsg("请先执行引导登记 Agent")
		}
		return nil, bizerrors.Pass(ctx, "loggie", "StopLoggie", err)
	}
	stdout, stderr, err := s.stopLoggieOverSSH(ctx, req.ServerID)
	result := &LoggieDeployResult{
		Success: err == nil,
		Stdout:  truncateDeployOutput(stdout, 2048),
		Stderr:  truncateDeployOutput(stderr, 2048),
	}
	if err != nil {
		result.Message = truncateDeployOutput(err.Error(), 512)
		return result, nil
	}
	result.Message = "Loggie 已停止"
	return result, nil
}

func (s *LoggieAgentService) InstallLoggie(ctx context.Context, projectID uint, req LoggieInstallRequest) (*LoggieDeployResult, error) {
	if projectID == 0 || req.ServerID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 与 server_id 必填")
	}
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrLogSourceServerNotFound
		}
		return nil, bizerrors.Pass(ctx, "loggie", "InstallLoggie", err)
	}
	if sv.ProjectID != projectID {
		return nil, constants.ErrServerNotInCurrentProject
	}
	agent, err := s.ensureAgent(ctx, projectID, req.ServerID, req.MonitorPort)
	if err != nil {
		return nil, err
	}
	stored := parseStoredBootstrapConfig(agent.BootstrapConfig)
	if mp := defaultMonitorPort(req.MonitorPort); mp > 0 {
		stored.MonitorPort = mp
		agent.MonitorPort = mp
	}
	if d := strings.TrimSpace(req.DeployDir); d != "" {
		stored.DeployDir = normalizeDeployDir(d)
	}
	if u := strings.TrimSpace(req.YunshuURL); u != "" {
		stored.YunshuURL = u
	}
	stored.AutoFromLogSources = true
	sources, err := s.loadBootstrapSourcesFromDB(ctx, projectID, req.ServerID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "InstallLoggie", err)
	}
	stored.Sources = sources
	if raw, err := json.Marshal(stored); err == nil {
		agent.BootstrapConfig = string(raw)
	}
	if err := s.repo.Save(ctx, agent); err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "InstallLoggie", err)
	}
	bundle, _, err := s.bundleFromStored(ctx, projectID, req.ServerID, agent, stored, true)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "InstallLoggie", err)
	}
	stdout, stderr, err := s.installLoggieOverSSH(ctx, req.ServerID, bundle, req.BinaryURL)
	result := &LoggieDeployResult{
		Success:       err == nil,
		PipelineCount: bundle.PipelineCount,
		SourceCount:   len(sources),
		Stdout:        truncateDeployOutput(stdout, 4096),
		Stderr:        truncateDeployOutput(stderr, 2048),
		DeployedAt:    formatDeployTime(time.Now()),
	}
	if err != nil {
		result.Message = truncateDeployOutput(err.Error(), 512)
		return result, nil
	}
	result.Message = "Agent 安装完成并已启动"
	return result, nil
}

func (s *LoggieAgentService) UninstallLoggie(ctx context.Context, projectID uint, req LoggieUninstallRequest) (*LoggieDeployResult, error) {
	if projectID == 0 || req.ServerID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 与 server_id 必填")
	}
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrLogSourceServerNotFound
		}
		return nil, bizerrors.Pass(ctx, "loggie", "UninstallLoggie", err)
	}
	if sv.ProjectID != projectID {
		return nil, constants.ErrServerNotInCurrentProject
	}

	agent, err := s.repo.GetByProjectAndServer(ctx, projectID, req.ServerID)
	registered := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizerrors.Pass(ctx, "loggie", "UninstallLoggie", err)
	}
	if !registered && req.SkipRemote {
		return &LoggieDeployResult{Success: true, Message: "未登记，无需删除"}, nil
	}

	deployDir := s.defaultDeployDir()
	if registered {
		stored := parseStoredBootstrapConfig(agent.BootstrapConfig)
		if d := strings.TrimSpace(stored.DeployDir); d != "" {
			deployDir = d
		}
	}

	result := &LoggieDeployResult{}
	if !req.SkipRemote {
		stdout, stderr, remoteErr := s.uninstallLoggieOverSSH(ctx, req.ServerID, deployDir, !req.KeepFiles)
		result.Stdout = truncateDeployOutput(stdout, 4096)
		result.Stderr = truncateDeployOutput(stderr, 2048)
		if remoteErr != nil {
			if !req.ForceLocal && registered {
				result.Success = false
				result.Message = truncateDeployOutput("远端卸载失败，登记未清除："+remoteErr.Error(), 512)
				return result, nil
			}
			result.Message = truncateDeployOutput("远端卸载失败："+remoteErr.Error(), 400)
		}
	}

	if registered {
		if err := s.repo.DeleteByProjectAndServer(ctx, projectID, req.ServerID); err != nil {
			return nil, bizerrors.Pass(ctx, "loggie", "UninstallLoggie", err)
		}
	}

	result.Success = true
	switch {
	case req.SkipRemote:
		result.Message = "已清除平台登记（未操作远端）"
	case strings.TrimSpace(result.Message) != "":
		result.Message = "已清除平台登记；" + result.Message
	case !registered:
		result.Message = "未登记；远端已尝试卸载"
	default:
		result.Message = "已卸载远端并清除平台登记"
	}
	return result, nil
}

func (s *LoggieAgentService) RestartLoggie(ctx context.Context, projectID uint, req LoggieDeployRequest) (*LoggieDeployResult, error) {
	if projectID == 0 || req.ServerID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 与 server_id 必填")
	}
	if _, err := s.repo.GetByProjectAndServer(ctx, projectID, req.ServerID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrBadRequestWithMsg("请先执行引导登记 Agent")
		}
		return nil, bizerrors.Pass(ctx, "loggie", "RestartLoggie", err)
	}
	stdout, stderr, err := s.restartLoggieOverSSH(ctx, req.ServerID)
	result := &LoggieDeployResult{
		Success: err == nil,
		Stdout:  truncateDeployOutput(stdout, 2048),
		Stderr:  truncateDeployOutput(stderr, 2048),
	}
	if err != nil {
		result.Message = truncateDeployOutput(err.Error(), 512)
		return result, nil
	}
	result.Message = "Loggie 已重启"
	return result, nil
}

func (s *LoggieAgentService) SyncFromLogSources(ctx context.Context, projectID uint, req LoggieDeployRequest) (*LoggieDeployResult, error) {
	req.SyncFromDB = true
	req.RestartLoggie = true
	return s.DeployConfig(ctx, projectID, req)
}

func (s *LoggieAgentService) ListStatus(ctx context.Context, projectID uint) ([]LoggieStatusItem, error) {
	if projectID == 0 {
		return nil, constants.ErrProjectIDRequired
	}
	servers, _, err := s.serverRepo.List(ctx, repository.ServerListParams{
		ProjectID: projectID,
		Page:      1,
		PageSize:  5000,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "ListStatus", err)
	}
	agents, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "ListStatus", err)
	}
	agentByServer := map[uint]model.LoggieAgent{}
	for _, a := range agents {
		if a.ServerID == 0 {
			continue // 忽略历史 K8s 项目级槽位
		}
		agentByServer[a.ServerID] = a
	}
	ingestMap := s.recentIngestByServer(ctx, projectID)
	now := time.Now()
	out := make([]LoggieStatusItem, 0, len(servers))
	for _, sv := range servers {
		item := LoggieStatusItem{
			ServerID:   sv.ID,
			ServerName: sv.Name,
			ServerHost: fmt.Sprintf("%s:%d", sv.Host, sv.Port),
		}
		if ag, ok := agentByServer[sv.ID]; ok {
			item.Registered = true
			item.Version = ag.Version
			item.HealthStatus = ag.HealthStatus
			item.PipelineStatus = ag.PipelineStatus
			item.EsSinkOK = ag.EsSinkOK
			item.LinesPerMin = ag.LinesPerMin
			item.LastError = ag.LastError
			item.LastSeenAt = formatRFC3339Ptr(ag.LastSeenAt)
			item.LastIngestAt = formatRFC3339Ptr(ag.LastIngestAt)
			item.MonitorPort = ag.MonitorPort
			if item.MonitorPort == 0 {
				item.MonitorPort = 9196
			}
			item.MonitorReachable = ag.MonitorReachable
			item.ActivePipelineCount = ag.ActivePipelineCount
			item.ActiveFdCount = ag.ActiveFdCount
			item.InactiveFdCount = ag.InactiveFdCount
			item.MonitorDetail = ag.MonitorDetail
			if ag.LastSeenAt != nil && now.Sub(*ag.LastSeenAt) <= loggieHeartbeatTimeout {
				item.Online = strings.EqualFold(ag.HealthStatus, "running") || ag.HealthStatus == "" || ag.HealthStatus == "unknown"
			}
			probe := ProbeLoggieMonitor(ctx, sv.Host, item.MonitorPort)
			if probe.Reachable || probe.Error != "" {
				item.LiveProbe = &probe
				// 远程探测成功时优先用实时 FD；心跳字段作兜底
				if probe.Reachable {
					item.ActiveFdCount = probe.ActiveFdCount
					item.InactiveFdCount = probe.InActiveFdCount
				}
			}
		}
		if cnt, ok := ingestMap[sv.ID]; ok {
			item.RecentDocCount = cnt
			if cnt > 0 {
				item.RecentIngest = true
			}
		}
		if !item.RecentIngest && item.LinesPerMin > 0 && item.Online {
			item.RecentIngest = true
		}
		out = append(out, item)
	}
	return out, nil
}

// ESConfigPreviewItem 供控制台展示 ES 连接信息（不含密码）。
type ESConfigPreviewItem struct {
	Enabled       bool     `json:"enabled"`
	Addresses     []string `json:"addresses"`
	Username      string   `json:"username"`
	IndexPattern  string   `json:"index_pattern"`
	HasPassword   bool     `json:"has_password"`
}

func (s *LoggieAgentService) ESConfigForUI(ctx context.Context) (*ESConfigPreviewItem, error) {
	if s.esProvider == nil {
		return &ESConfigPreviewItem{}, nil
	}
	cfg, err := s.esProvider.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	return &ESConfigPreviewItem{
		Enabled:      cfg.Enabled,
		Addresses:    cfg.Addresses,
		Username:     cfg.Username,
		IndexPattern: cfg.IndexPattern,
		HasPassword:  strings.TrimSpace(cfg.Password) != "",
	}, nil
}

func (s *LoggieAgentService) recentIngestByServer(ctx context.Context, projectID uint) map[uint]int64 {
	out := map[uint]int64{}
	if s.esProvider == nil {
		return out
	}
	cli, cfg, err := s.esProvider.Client(ctx)
	if err != nil {
		return out
	}
	since := time.Now().UTC().Add(-loggieIngestFreshWindow).Format(time.RFC3339)
	body := map[string]any{
		"size": 0,
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []map[string]any{
					termIDFilter("project_id", fmt.Sprintf("%d", projectID)),
					timeRangeFilter(since, "", cfg.TimestampField),
				},
			},
		},
		"aggs": map[string]any{
			"by_server": map[string]any{
				"terms": map[string]any{"field": "fields.server_id.keyword", "size": 500},
			},
		},
	}
	raw, err := cli.Search(ctx, GlobalAgentIndexPattern(), body)
	if err != nil {
		return out
	}
	aggs, _ := raw["aggregations"].(map[string]any)
	byServer, _ := aggs["by_server"].(map[string]any)
	buckets, _ := byServer["buckets"].([]any)
	for _, b := range buckets {
		bm, ok := b.(map[string]any)
		if !ok {
			continue
		}
		keyStr, _ := bm["key"].(string)
		var sid uint
		fmt.Sscanf(keyStr, "%d", &sid)
		if v, ok := bm["doc_count"].(float64); ok {
			out[sid] = int64(v)
		}
	}
	return out
}

func newLoggieToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func formatRFC3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
