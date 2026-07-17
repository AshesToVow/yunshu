package logplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/model"
)

type LoggieBootstrapSourcePreview struct {
	LogSourceID uint   `json:"log_source_id"`
	ServiceID   uint   `json:"service_id"`
	LogType     string `json:"log_type"`
	Path        string `json:"path"`
	GlobPath    string `json:"glob_path"`
}

func autoFromLogSourcesEnabled(v *bool) bool {
	if v == nil {
		return true
	}
	return *v
}

func logSourcesToBootstrapSources(list []model.ServiceLogSource) []loggieBootstrapSource {
	out := make([]loggieBootstrapSource, 0, len(list))
	for _, src := range list {
		if strings.EqualFold(strings.TrimSpace(src.LogType), "journal") {
			continue
		}
		glob := logSourceToGlobPath(src)
		if glob == "" {
			continue
		}
		inc := ""
		if src.IncludeRegex != nil {
			inc = strings.TrimSpace(*src.IncludeRegex)
		}
		out = append(out, loggieBootstrapSource{
			LogSourceID:  src.ID,
			ServiceID:    src.ServiceID,
			LogType:      src.LogType,
			Path:         strings.TrimSpace(src.Path),
			IncludeRegex: inc,
			ExcludeRegex: derefStr(src.ExcludeRegex),
			Encoding:     derefStr(src.Encoding),
			Paths:        []string{glob},
		})
	}
	return out
}

func bootstrapSourcesToEntries(projectID, serverID uint, sources []loggieBootstrapSource) []LoggiePipelineSourceEntry {
	out := make([]LoggiePipelineSourceEntry, 0, len(sources))
	for _, src := range sources {
		paths := src.Paths
		if len(paths) == 0 && strings.TrimSpace(src.Path) != "" {
			paths = []string{strings.TrimSpace(src.Path)}
		}
		paths = defaultLogPaths(paths)
		if len(paths) == 0 {
			continue
		}
		out = append(out, LoggiePipelineSourceEntry{
			ServiceID:    src.ServiceID,
			LogSourceID:  src.LogSourceID,
			LogType:      src.LogType,
			Paths:        paths,
			PipelineName: pipelineNameForSource(projectID, serverID, src.LogSourceID, src.ServiceID),
			ParseProfile: detectParseProfile(paths[0], nil),
			ExcludeRegex: src.ExcludeRegex,
			Encoding:     src.Encoding,
		})
	}
	return out
}

func legacyBootstrapSources(stored loggieStoredBootstrapConfig) []loggieBootstrapSource {
	if len(stored.Sources) > 0 {
		return stored.Sources
	}
	paths := defaultLogPaths(stored.LogPaths)
	if len(paths) == 0 {
		return nil
	}
	return []loggieBootstrapSource{{
		ServiceID:   stored.ServiceID,
		LogSourceID: stored.LogSourceID,
		LogType:     "file",
		Paths:       paths,
	}}
}

func normalizeDeployDir(dir string) string {
	dir = strings.TrimRight(strings.TrimSpace(dir), "/")
	if dir == "" {
		return defaultLoggieDeployDir
	}
	return dir
}

func parseStoredBootstrapConfig(raw string) loggieStoredBootstrapConfig {
	var stored loggieStoredBootstrapConfig
	if strings.TrimSpace(raw) == "" {
		stored.MonitorPort = 9196
		stored.DeployDir = defaultLoggieDeployDir
		return stored
	}
	_ = json.Unmarshal([]byte(raw), &stored)
	if stored.MonitorPort == 0 {
		stored.MonitorPort = 9196
	}
	stored.DeployDir = normalizeDeployDir(stored.DeployDir)
	return stored
}

func (s *LoggieAgentService) loadBootstrapSourcesFromDB(ctx context.Context, projectID, serverID uint) ([]loggieBootstrapSource, error) {
	if s.logSourceRepo == nil {
		return nil, nil
	}
	list, err := s.logSourceRepo.ListByProjectAndServer(ctx, projectID, serverID)
	if err != nil {
		return nil, err
	}
	return logSourcesToBootstrapSources(list), nil
}

func (s *LoggieAgentService) resolveBootstrapSources(
	ctx context.Context,
	projectID, serverID uint,
	req LoggieBootstrapRequest,
	autoFromDB bool,
) ([]loggieBootstrapSource, error) {
	if autoFromDB {
		sources, err := s.loadBootstrapSourcesFromDB(ctx, projectID, serverID)
		if err != nil {
			return nil, err
		}
		if len(sources) > 0 {
			return sources, nil
		}
	}
	paths := defaultLogPaths(req.LogPaths)
	if len(paths) == 0 && len(req.LogPaths) == 0 {
		return nil, nil
	}
	return []loggieBootstrapSource{{
		ServiceID:   req.ServiceID,
		LogSourceID: req.LogSourceID,
		LogType:     "file",
		Paths:       paths,
	}}, nil
}

func (s *LoggieAgentService) buildStoredConfigFromRequest(req LoggieBootstrapRequest, sources []loggieBootstrapSource) loggieStoredBootstrapConfig {
	return loggieStoredBootstrapConfig{
		MonitorPort:        defaultMonitorPort(req.MonitorPort),
		YunshuURL:          strings.TrimSpace(req.YunshuURL),
		DeployDir:          normalizeDeployDir(req.DeployDir),
		AutoFromLogSources: autoFromLogSourcesEnabled(req.AutoFromLogSources),
		Sources:            sources,
		LogPaths:           req.LogPaths,
		ServiceID:          req.ServiceID,
		LogSourceID:        req.LogSourceID,
	}
}

func (s *LoggieAgentService) bundleFromStored(
	ctx context.Context,
	projectID, serverID uint,
	agent *model.LoggieAgent,
	stored loggieStoredBootstrapConfig,
	refreshFromDB bool,
) (LoggiePipelineBundle, []loggieBootstrapSource, error) {
	sources := stored.Sources
	if refreshFromDB || (stored.AutoFromLogSources && len(sources) == 0) {
		dbSources, err := s.loadBootstrapSourcesFromDB(ctx, projectID, serverID)
		if err != nil {
			return LoggiePipelineBundle{}, nil, err
		}
		if len(dbSources) > 0 {
			sources = dbSources
			stored.Sources = sources
		}
	}
	if len(sources) == 0 {
		sources = legacyBootstrapSources(stored)
	}
	entries := bootstrapSourcesToEntries(projectID, serverID, sources)
	s.enrichPipelineEntries(ctx, projectID, serverID, entries)
	var esCfg config.ElasticsearchConfig
	if s.esProvider != nil {
		esCfg, _ = s.esProvider.Resolve(ctx)
	}
	var kafkaCfg config.KafkaConfig
	if s.kafkaProvider != nil {
		kafkaCfg, _ = s.kafkaProvider.Resolve(ctx)
	}
	if kafkaCfg.SinkViaKafka() {
		if topic, err := EnsureAgentKafkaTopic(ctx, kafkaCfg, serverID); err != nil {
			slog.Default().With("component", "loggie").Warn("ensure kafka topic failed",
				"server_id", serverID, "topic", topic, "err", err)
		}
	}
	bundle := BuildMultiPipelineBundle(projectID, serverID, entries, stored.MonitorPort, esCfg, kafkaCfg, agent.Token, stored.YunshuURL, stored.DeployDir)
	return bundle, sources, nil
}

func (s *LoggieAgentService) enrichPipelineEntries(ctx context.Context, projectID, serverID uint, entries []LoggiePipelineSourceEntry) {
	if len(entries) == 0 {
		return
	}
	var projectCode, projectName, serverHost, serverName string
	if s.projectRepo != nil {
		if p, err := s.projectRepo.GetByID(ctx, projectID); err == nil && p != nil {
			projectCode = strings.TrimSpace(p.Code)
			projectName = strings.TrimSpace(p.Name)
		}
	}
	if s.serverRepo != nil {
		if sv, err := s.serverRepo.GetByID(ctx, serverID); err == nil && sv != nil {
			serverHost = strings.TrimSpace(sv.Host)
			serverName = strings.TrimSpace(sv.Name)
		}
	}
	svcNames := map[uint]string{}
	for i := range entries {
		entries[i].ProjectCode = projectCode
		entries[i].ProjectName = projectName
		entries[i].ServerHost = serverHost
		entries[i].ServerName = serverName
		sid := entries[i].ServiceID
		if sid == 0 || s.serviceRepo == nil {
			continue
		}
		if name, ok := svcNames[sid]; ok {
			entries[i].ServiceName = name
			continue
		}
		if svc, err := s.serviceRepo.GetByID(ctx, sid); err == nil && svc != nil {
			svcNames[sid] = strings.TrimSpace(svc.Name)
			entries[i].ServiceName = svcNames[sid]
		}
	}
}

func (s *LoggieAgentService) PreviewBootstrapSources(ctx context.Context, projectID, serverID uint) ([]LoggieBootstrapSourcePreview, error) {
	if projectID == 0 || serverID == 0 {
		return nil, nil
	}
	sources, err := s.loadBootstrapSourcesFromDB(ctx, projectID, serverID)
	if err != nil {
		return nil, err
	}
	out := make([]LoggieBootstrapSourcePreview, 0, len(sources))
	for _, src := range sources {
		glob := ""
		if len(src.Paths) > 0 {
			glob = src.Paths[0]
		}
		out = append(out, LoggieBootstrapSourcePreview{
			LogSourceID: src.LogSourceID,
			ServiceID:   src.ServiceID,
			LogType:     src.LogType,
			Path:        src.Path,
			GlobPath:    glob,
		})
	}
	return out, nil
}

func bootstrapResultFromBundle(
	agent *model.LoggieAgent,
	projectID, serverID uint,
	bundle LoggiePipelineBundle,
	esCfg config.ElasticsearchConfig,
	sourceCount int,
	deployed bool,
	deployMsg string,
) *LoggieBootstrapResult {
	monitorPort := agent.MonitorPort
	if monitorPort == 0 {
		monitorPort = 9196
	}
	return &LoggieBootstrapResult{
		Token:          agent.Token,
		ProjectID:      projectID,
		ServerID:       serverID,
		ESAddresses:    esCfg.Addresses,
		ESIndexPattern: AgentIndexPattern(serverID),
		ReportURL:      "/api/v1/loggie/heartbeat/report",
		PipelineHint: fmt.Sprintf(
			"fields.project_id=%d fields.server_id=%d sink.index=%s monitor_port=%d pipelines=%d",
			projectID, serverID, AgentIndexSink(serverID), monitorPort, bundle.PipelineCount,
		),
		PipelineYAML:      bundle.PipelineYAML,
		PipelinesOnlyYAML: bundle.PipelinesOnlyYAML,
		PipelineFilename:  bundle.PipelineFilename,
		PipelinesFilename: bundle.PipelinesFilename,
		EnvFile:           bundle.EnvFile,
		EnvFilename:       bundle.EnvFilename,
		HeartbeatScript:   bundle.HeartbeatScript,
		HeartbeatFilename: bundle.HeartbeatFilename,
		StartScript:       bundle.StartScript,
		StartFilename:     bundle.StartFilename,
		MonitorPort:       monitorPort,
		PipelineCount:     bundle.PipelineCount,
		SourceCount:       sourceCount,
		Deployed:          deployed,
		DeployMessage:     deployMsg,
	}
}
