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
	"yunshu/internal/service/k8s"

	"gorm.io/gorm"
)

const loggieHeartbeatTimeout = 3 * time.Minute
const loggieIngestFreshWindow = 5 * time.Minute

type LoggieAgentService struct {
	repo          interfaces.LoggieAgentRepository
	serverRepo    interfaces.ServerRepository
	logSourceRepo interfaces.LogSourceRepository
	esProvider    *ElasticsearchProvider
	aead          cipher.AEAD
	k8sRuntime    *k8s.K8sRuntimeService
	k8sWorkload   *k8s.K8sWorkloadService
}

func NewLoggieAgentService(
	repo interfaces.LoggieAgentRepository,
	serverRepo interfaces.ServerRepository,
	logSourceRepo interfaces.LogSourceRepository,
	esProvider *ElasticsearchProvider,
	encryptionKey string,
	k8sRuntime *k8s.K8sRuntimeService,
	k8sWorkload *k8s.K8sWorkloadService,
) (*LoggieAgentService, error) {
	aead, err := cryptox.NewAESGCMFromKeyString(encryptionKey)
	if err != nil {
		return nil, err
	}
	return &LoggieAgentService{
		repo:          repo,
		serverRepo:    serverRepo,
		logSourceRepo: logSourceRepo,
		esProvider:    esProvider,
		aead:          aead,
		k8sRuntime:    k8sRuntime,
		k8sWorkload:   k8sWorkload,
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
	MonitorDetail       string `json:"monitor_detail"`
}

type LoggieBootstrapRequest struct {
	ServerID             uint     `json:"server_id"`
	DeployMode           string   `json:"deploy_mode"` // binary | k8s
	ClusterID            uint     `json:"cluster_id"`
	K8sNamespace         string   `json:"k8s_namespace"`
	DaemonSetName        string   `json:"daemonset_name"`
	// K8sRequirePodLabel=true 时 ClusterLogConfig 仅匹配带 yunshu.project_id 的 Pod。
	// 默认 false：采集全部 Pod（便于联调；多项目共集群时请改为 true 并给业务打标）。
	K8sRequirePodLabel *bool `json:"k8s_require_pod_label"`
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
	ServerID       uint   `json:"server_id"`
	DeployMode     string `json:"deploy_mode"`
	ClusterID      uint   `json:"cluster_id"`
	RestartLoggie  bool   `json:"restart_loggie"`
	SyncFromDB     bool   `json:"sync_from_db"`
}

type LoggieDeployResult struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	Stdout         string `json:"stdout,omitempty"`
	Stderr         string `json:"stderr,omitempty"`
	PipelineCount  int    `json:"pipeline_count"`
	DeployedAt     string `json:"deployed_at,omitempty"`
	SourceCount    int    `json:"source_count"`
	DeployMode     string `json:"deploy_mode,omitempty"`
}

type LoggieBootstrapResult struct {
	Token              string   `json:"token"`
	ProjectID          uint     `json:"project_id"`
	ServerID           uint     `json:"server_id"`
	DeployMode         string   `json:"deploy_mode"`
	ClusterID          uint     `json:"cluster_id,omitempty"`
	K8sNamespace       string   `json:"k8s_namespace,omitempty"`
	DaemonSetName      string   `json:"daemonset_name,omitempty"`
	ESAddresses        []string `json:"es_addresses"`
	ESIndexPattern     string   `json:"es_index_pattern"`
	ReportURL          string   `json:"report_url"`
	PipelineHint       string   `json:"pipeline_hint"`
	PipelineYAML       string   `json:"pipeline_yaml"`
	PipelinesOnlyYAML  string   `json:"pipelines_only_yaml"`
	PipelineFilename   string   `json:"pipeline_filename"`
	PipelinesFilename  string   `json:"pipelines_filename"`
	EnvFile            string   `json:"env_file"`
	EnvFilename        string   `json:"env_filename"`
	HeartbeatScript    string   `json:"heartbeat_script"`
	HeartbeatFilename  string   `json:"heartbeat_filename"`
	MonitorPort        int      `json:"monitor_port"`
	PipelineCount      int      `json:"pipeline_count"`
	SourceCount        int      `json:"source_count"`
	Deployed           bool     `json:"deployed"`
	DeployMessage      string   `json:"deploy_message,omitempty"`
	K8sManifest        string   `json:"k8s_manifest,omitempty"`
}

type loggieBootstrapSource struct {
	LogSourceID  uint     `json:"log_source_id"`
	ServiceID    uint     `json:"service_id"`
	LogType      string   `json:"log_type"`
	Path         string   `json:"path"`
	IncludeRegex string   `json:"include_regex,omitempty"`
	Paths        []string `json:"paths,omitempty"`
}

type loggieStoredBootstrapConfig struct {
	MonitorPort        int                     `json:"monitor_port"`
	YunshuURL          string                  `json:"yunshu_url"`
	DeployDir          string                  `json:"deploy_dir"`
	DeployMode         string                  `json:"deploy_mode"`
	ClusterID          uint                    `json:"cluster_id"`
	K8sNamespace       string                  `json:"k8s_namespace"`
	DaemonSetName      string                  `json:"daemonset_name"`
	K8sRequirePodLabel bool                    `json:"k8s_require_pod_label"`
	AutoFromLogSources bool                    `json:"auto_from_log_sources"`
	Sources            []loggieBootstrapSource `json:"sources"`
	// legacy single-pipeline fields
	LogPaths    []string `json:"log_paths,omitempty"`
	ServiceID   uint     `json:"service_id,omitempty"`
	LogSourceID uint     `json:"log_source_id,omitempty"`
}

type LoggieStatusItem struct {
	ServerID            uint    `json:"server_id"`
	ServerName          string  `json:"server_name"`
	ServerHost          string  `json:"server_host"`
	DeployMode          string  `json:"deploy_mode,omitempty"`
	ClusterID           uint    `json:"cluster_id,omitempty"`
	K8sNamespace        string  `json:"k8s_namespace,omitempty"`
	DaemonSetName       string  `json:"daemonset_name,omitempty"`
	Registered          bool    `json:"registered"`
	Online              bool    `json:"online"`
	RecentIngest        bool    `json:"recent_ingest"`
	Version             string  `json:"version,omitempty"`
	HealthStatus        string  `json:"health_status,omitempty"`
	PipelineStatus      string  `json:"pipeline_status,omitempty"`
	EsSinkOK            bool    `json:"es_sink_ok"`
	LinesPerMin         int     `json:"lines_per_min"`
	LastSeenAt          *string `json:"last_seen_at,omitempty"`
	LastIngestAt        *string `json:"last_ingest_at,omitempty"`
	LastError           string  `json:"last_error,omitempty"`
	RecentDocCount      int64   `json:"recent_doc_count"`
	MonitorPort         int     `json:"monitor_port"`
	MonitorReachable    bool    `json:"monitor_reachable"`
	ActivePipelineCount int     `json:"active_pipeline_count"`
	ActiveFdCount       int     `json:"active_fd_count"`
	MonitorDetail       string  `json:"monitor_detail,omitempty"`
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
	if v := strings.TrimSpace(req.MonitorDetail); v != "" {
		agent.MonitorDetail = truncateString(v, 4096)
	}
	if req.LinesPerMin > 0 {
		agent.LastIngestAt = &now
	}
	return s.repo.Save(ctx, agent)
}

func (s *LoggieAgentService) Bootstrap(ctx context.Context, projectID uint, req LoggieBootstrapRequest) (*LoggieBootstrapResult, error) {
	if projectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 必填")
	}
	mode := normalizeDeployMode(req.DeployMode)
	if mode == deployModeK8s {
		return s.bootstrapK8s(ctx, projectID, req)
	}
	return s.bootstrapBinary(ctx, projectID, req)
}

func (s *LoggieAgentService) bootstrapBinary(ctx context.Context, projectID uint, req LoggieBootstrapRequest) (*LoggieBootstrapResult, error) {
	if req.ServerID == 0 {
		return nil, constants.ErrBadRequestWithMsg("二进制模式需填写 server_id")
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
	stored.DeployMode = deployModeBinary
	sources, err := s.resolveBootstrapSources(ctx, projectID, req.ServerID, req, stored.AutoFromLogSources)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "Bootstrap", err)
	}
	stored = s.buildStoredConfigFromRequest(req, sources)
	stored.DeployMode = deployModeBinary
	if raw, err := json.Marshal(stored); err == nil {
		agent.BootstrapConfig = string(raw)
	}
	agent.MonitorPort = stored.MonitorPort
	if err := s.repo.Save(ctx, agent); err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "Bootstrap", err)
	}
	var esCfg config.ElasticsearchConfig
	if s.esProvider != nil {
		esCfg, _ = s.esProvider.Resolve(ctx)
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
	res := bootstrapResultFromBundle(agent, projectID, req.ServerID, bundle, esCfg, len(finalSources), deployed, deployMsg)
	res.DeployMode = deployModeBinary
	return res, nil
}

func (s *LoggieAgentService) bootstrapK8s(ctx context.Context, projectID uint, req LoggieBootstrapRequest) (*LoggieBootstrapResult, error) {
	if req.ClusterID == 0 {
		return nil, constants.ErrBadRequestWithMsg("K8s 模式需填写 cluster_id")
	}
	if err := s.ensureClusterForProject(ctx, projectID, req.ClusterID); err != nil {
		return nil, err
	}
	// K8s 模式用 server_id=0 作为项目级 Agent 槽位
	agent, err := s.ensureAgent(ctx, projectID, 0, req.MonitorPort)
	if err != nil {
		return nil, err
	}
	stored := s.buildStoredConfigFromRequest(req, nil)
	stored.DeployMode = deployModeK8s
	stored.ClusterID = req.ClusterID
	stored.K8sNamespace = defaultK8sNamespace(req.K8sNamespace)
	stored.DaemonSetName = defaultK8sDaemonSet(req.DaemonSetName)
	stored.AutoFromLogSources = false
	if raw, err := json.Marshal(stored); err == nil {
		agent.BootstrapConfig = string(raw)
	}
	agent.MonitorPort = stored.MonitorPort
	if err := s.repo.Save(ctx, agent); err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "Bootstrap", err)
	}
	var esCfg config.ElasticsearchConfig
	if s.esProvider != nil {
		esCfg, _ = s.esProvider.Resolve(ctx)
	}
	k8sBundle := BuildK8sLoggieBundle(projectID, stored.ClusterID, stored.K8sNamespace, stored.DaemonSetName, esCfg, stored.K8sRequirePodLabel)
	deployed := false
	deployMsg := ""
	if req.DeployAfterBootstrap {
		if err := s.applyK8sLoggieManifest(ctx, stored.ClusterID, k8sBundle.CombinedManifest); err != nil {
			deployMsg = truncateDeployOutput(err.Error(), 512)
		} else {
			_ = s.restartLoggieDaemonSet(ctx, stored.ClusterID, stored.K8sNamespace, stored.DaemonSetName)
			deployed = true
			deployMsg = "已 apply Sink/ClusterLogConfig 并滚动重启 DaemonSet"
		}
	}
	return &LoggieBootstrapResult{
		Token:             agent.Token,
		ProjectID:         projectID,
		ServerID:          0,
		DeployMode:        deployModeK8s,
		ClusterID:         stored.ClusterID,
		K8sNamespace:      stored.K8sNamespace,
		DaemonSetName:     stored.DaemonSetName,
		ESAddresses:       esCfg.Addresses,
		ESIndexPattern:    esCfg.IndexPattern,
		ReportURL:         "/api/v1/loggie/heartbeat/report",
		PipelineHint:      fmt.Sprintf("k8s ClusterLogConfig=%s sink=%s label=yunshu.project_id=%d", k8sBundle.ClusterLogConfigName, k8sBundle.SinkName, projectID),
		PipelineYAML:      k8sBundle.CombinedManifest,
		PipelinesOnlyYAML: k8sBundle.ClusterLogConfigYAML,
		PipelineFilename:  "loggie-k8s-manifest.yaml",
		PipelinesFilename: "clusterlogconfig.yaml",
		MonitorPort:       stored.MonitorPort,
		PipelineCount:     1,
		SourceCount:       1,
		Deployed:          deployed,
		DeployMessage:     deployMsg,
		K8sManifest:       k8sBundle.CombinedManifest,
	}, nil
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

func (s *LoggieAgentService) ensureClusterForProject(ctx context.Context, projectID, clusterID uint) error {
	if s.k8sRuntime == nil {
		return constants.ErrBadRequestWithMsg("K8s 运行时未配置")
	}
	cluster, _, err := s.k8sRuntime.GetClusterKubectl(ctx, clusterID)
	if err != nil {
		return err
	}
	if cluster.OwningProjectID != nil && *cluster.OwningProjectID != 0 && *cluster.OwningProjectID != projectID {
		return constants.ErrBadRequestWithMsg("集群不属于当前项目")
	}
	return nil
}

func (s *LoggieAgentService) GeneratePipelineBundle(ctx context.Context, projectID, serverID uint) (*LoggiePipelineBundle, error) {
	if projectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 必填")
	}
	agent, err := s.repo.GetByProjectAndServer(ctx, projectID, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrBadRequestWithMsg("请先执行引导登记 Agent")
		}
		return nil, bizerrors.Pass(ctx, "loggie", "GeneratePipelineBundle", err)
	}
	stored := parseStoredBootstrapConfig(agent.BootstrapConfig)
	if normalizeDeployMode(stored.DeployMode) == deployModeK8s {
		var esCfg config.ElasticsearchConfig
		if s.esProvider != nil {
			esCfg, _ = s.esProvider.Resolve(ctx)
		}
		k8sBundle := BuildK8sLoggieBundle(projectID, stored.ClusterID, stored.K8sNamespace, stored.DaemonSetName, esCfg, stored.K8sRequirePodLabel)
		return &LoggiePipelineBundle{
			PipelineYAML:      k8sBundle.CombinedManifest,
			PipelinesOnlyYAML: k8sBundle.ClusterLogConfigYAML,
			PipelineFilename:  "loggie-k8s-manifest.yaml",
			PipelinesFilename: "clusterlogconfig.yaml",
			PipelineCount:     1,
		}, nil
	}
	bundle, _, err := s.bundleFromStored(ctx, projectID, serverID, agent, stored, stored.AutoFromLogSources)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "loggie", "GeneratePipelineBundle", err)
	}
	return &bundle, nil
}

func (s *LoggieAgentService) DeployConfig(ctx context.Context, projectID uint, req LoggieDeployRequest) (*LoggieDeployResult, error) {
	if projectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 必填")
	}
	agent, stored, err := s.resolveAgentForDeploy(ctx, projectID, req)
	if err != nil {
		return nil, err
	}
	mode := normalizeDeployMode(firstNonEmpty(req.DeployMode, stored.DeployMode))
	if mode == deployModeK8s {
		return s.deployConfigK8s(ctx, projectID, agent, stored)
	}
	if req.ServerID == 0 {
		req.ServerID = agent.ServerID
	}
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
		DeployMode:    deployModeBinary,
	}
	if err != nil {
		result.Message = truncateDeployOutput(err.Error(), 512)
		return result, nil
	}
	result.Message = "配置已下发并重启 Loggie（二进制）"
	return result, nil
}

func (s *LoggieAgentService) deployConfigK8s(ctx context.Context, projectID uint, agent *model.LoggieAgent, stored loggieStoredBootstrapConfig) (*LoggieDeployResult, error) {
	if stored.ClusterID == 0 {
		return nil, constants.ErrBadRequestWithMsg("请先引导并填写 cluster_id")
	}
	if err := s.ensureClusterForProject(ctx, projectID, stored.ClusterID); err != nil {
		return nil, err
	}
	var esCfg config.ElasticsearchConfig
	if s.esProvider != nil {
		esCfg, _ = s.esProvider.Resolve(ctx)
	}
	bundle := BuildK8sLoggieBundle(projectID, stored.ClusterID, stored.K8sNamespace, stored.DaemonSetName, esCfg, stored.K8sRequirePodLabel)
	err := s.applyK8sLoggieManifest(ctx, stored.ClusterID, bundle.CombinedManifest)
	result := &LoggieDeployResult{
		Success:       err == nil,
		PipelineCount: 1,
		SourceCount:   1,
		DeployedAt:    formatDeployTime(time.Now()),
		DeployMode:    deployModeK8s,
		Stdout:        truncateDeployOutput(bundle.CombinedManifest, 2048),
	}
	if err != nil {
		result.Message = truncateDeployOutput(err.Error(), 512)
		return result, nil
	}
	if rerr := s.restartLoggieDaemonSet(ctx, stored.ClusterID, stored.K8sNamespace, stored.DaemonSetName); rerr != nil {
		result.Message = "CR 已 apply，但 DaemonSet 重启失败: " + truncateDeployOutput(rerr.Error(), 256)
		result.Success = false
		return result, nil
	}
	_ = agent
	result.Message = "已 apply ClusterLogConfig/Sink 并滚动重启 Loggie DaemonSet"
	return result, nil
}

func (s *LoggieAgentService) RestartLoggie(ctx context.Context, projectID uint, req LoggieDeployRequest) (*LoggieDeployResult, error) {
	if projectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 必填")
	}
	agent, stored, err := s.resolveAgentForDeploy(ctx, projectID, req)
	if err != nil {
		return nil, err
	}
	mode := normalizeDeployMode(firstNonEmpty(req.DeployMode, stored.DeployMode))
	if mode == deployModeK8s {
		clusterID := stored.ClusterID
		if req.ClusterID > 0 {
			clusterID = req.ClusterID
		}
		if err := s.restartLoggieDaemonSet(ctx, clusterID, stored.K8sNamespace, stored.DaemonSetName); err != nil {
			return &LoggieDeployResult{Success: false, Message: truncateDeployOutput(err.Error(), 512), DeployMode: deployModeK8s}, nil
		}
		return &LoggieDeployResult{Success: true, Message: "Loggie DaemonSet 已滚动重启", DeployMode: deployModeK8s}, nil
	}
	stdout, stderr, err := s.restartLoggieOverSSH(ctx, agent.ServerID)
	result := &LoggieDeployResult{
		Success:    err == nil,
		Stdout:     truncateDeployOutput(stdout, 2048),
		Stderr:     truncateDeployOutput(stderr, 2048),
		DeployMode: deployModeBinary,
	}
	if err != nil {
		result.Message = truncateDeployOutput(err.Error(), 512)
		return result, nil
	}
	result.Message = "Loggie 已重启（systemctl）"
	return result, nil
}

func (s *LoggieAgentService) SyncFromLogSources(ctx context.Context, projectID uint, req LoggieDeployRequest) (*LoggieDeployResult, error) {
	req.SyncFromDB = true
	req.RestartLoggie = true
	return s.DeployConfig(ctx, projectID, req)
}

func (s *LoggieAgentService) resolveAgentForDeploy(ctx context.Context, projectID uint, req LoggieDeployRequest) (*model.LoggieAgent, loggieStoredBootstrapConfig, error) {
	serverID := req.ServerID
	mode := normalizeDeployMode(req.DeployMode)
	if mode == deployModeK8s || (serverID == 0 && req.ClusterID > 0) {
		serverID = 0
	}
	agent, err := s.repo.GetByProjectAndServer(ctx, projectID, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) && serverID != 0 {
			// 兼容：尝试项目级 K8s Agent
			agent, err = s.repo.GetByProjectAndServer(ctx, projectID, 0)
		}
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, loggieStoredBootstrapConfig{}, constants.ErrBadRequestWithMsg("请先执行引导登记 Agent")
			}
			return nil, loggieStoredBootstrapConfig{}, bizerrors.Pass(ctx, "loggie", "Deploy", err)
		}
	}
	stored := parseStoredBootstrapConfig(agent.BootstrapConfig)
	if req.ClusterID > 0 {
		stored.ClusterID = req.ClusterID
	}
	return agent, stored, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
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
		agentByServer[a.ServerID] = a
	}
	ingestMap := s.recentIngestByServer(ctx, projectID)
	now := time.Now()
	out := make([]LoggieStatusItem, 0, len(servers)+1)

	// 项目级 K8s Loggie（server_id=0）
	if ag, ok := agentByServer[0]; ok {
		stored := parseStoredBootstrapConfig(ag.BootstrapConfig)
		item := LoggieStatusItem{
			ServerID:      0,
			ServerName:    "K8s DaemonSet",
			ServerHost:    fmt.Sprintf("cluster/%d", stored.ClusterID),
			DeployMode:    deployModeK8s,
			ClusterID:     stored.ClusterID,
			K8sNamespace:  defaultK8sNamespace(stored.K8sNamespace),
			DaemonSetName: defaultK8sDaemonSet(stored.DaemonSetName),
			Registered:    true,
			Version:       ag.Version,
			HealthStatus:  ag.HealthStatus,
			PipelineStatus: ag.PipelineStatus,
			EsSinkOK:      ag.EsSinkOK,
			LinesPerMin:   ag.LinesPerMin,
			LastError:     ag.LastError,
			LastSeenAt:    formatRFC3339Ptr(ag.LastSeenAt),
			LastIngestAt:  formatRFC3339Ptr(ag.LastIngestAt),
			MonitorPort:   ag.MonitorPort,
			MonitorReachable: ag.MonitorReachable,
			ActivePipelineCount: ag.ActivePipelineCount,
			ActiveFdCount: ag.ActiveFdCount,
			MonitorDetail: ag.MonitorDetail,
		}
		if item.MonitorPort == 0 {
			item.MonitorPort = 9196
		}
		if stored.ClusterID > 0 {
			ready, desired, err := s.probeK8sLoggieDaemonSet(ctx, stored.ClusterID, stored.K8sNamespace, stored.DaemonSetName)
			if err != nil {
				item.LastError = truncateString(err.Error(), 512)
				item.MonitorDetail = fmt.Sprintf("DaemonSet 探测失败: %v", err)
			} else {
				item.Online = desired > 0 && ready >= desired
				item.ActivePipelineCount = int(ready)
				item.ActiveFdCount = int(desired)
				item.MonitorDetail = fmt.Sprintf("DaemonSet %s/%s ready=%d desired=%d", item.K8sNamespace, item.DaemonSetName, ready, desired)
				if item.Online {
					item.HealthStatus = "running"
					item.PipelineStatus = "running"
				}
			}
		}
		if cnt, ok := ingestMap[0]; ok {
			item.RecentDocCount = cnt
			item.RecentIngest = cnt > 0
		}
		out = append(out, item)
	}

	for _, sv := range servers {
		item := LoggieStatusItem{
			ServerID:   sv.ID,
			ServerName: sv.Name,
			ServerHost: fmt.Sprintf("%s:%d", sv.Host, sv.Port),
			DeployMode: deployModeBinary,
		}
		if ag, ok := agentByServer[sv.ID]; ok {
			stored := parseStoredBootstrapConfig(ag.BootstrapConfig)
			if normalizeDeployMode(stored.DeployMode) == deployModeK8s {
				item.DeployMode = deployModeK8s
				item.ClusterID = stored.ClusterID
				item.K8sNamespace = stored.K8sNamespace
				item.DaemonSetName = stored.DaemonSetName
			}
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
			item.MonitorDetail = ag.MonitorDetail
			if ag.LastSeenAt != nil && now.Sub(*ag.LastSeenAt) <= loggieHeartbeatTimeout {
				item.Online = strings.EqualFold(ag.HealthStatus, "running") || ag.HealthStatus == "" || ag.HealthStatus == "unknown"
			}
			probe := ProbeLoggieMonitor(ctx, sv.Host, item.MonitorPort)
			if probe.Reachable || probe.Error != "" {
				item.LiveProbe = &probe
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
	raw, err := cli.Search(ctx, cfg.IndexPattern, body)
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
