package logplatform

import (
	"context"
	"encoding/json"
	"fmt"
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
	var esCfg config.ElasticsearchConfig
	if s.esProvider != nil {
		esCfg, _ = s.esProvider.Resolve(ctx)
	}
	bundle := BuildMultiPipelineBundle(projectID, serverID, entries, stored.MonitorPort, esCfg, agent.Token, stored.YunshuURL, stored.DeployDir)
	return bundle, sources, nil
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
		Token:             agent.Token,
		ProjectID:         projectID,
		ServerID:          serverID,
		ESAddresses:       esCfg.Addresses,
		ESIndexPattern:    esCfg.IndexPattern,
		ReportURL:         "/api/v1/loggie/heartbeat/report",
		PipelineHint: fmt.Sprintf(
			"fields.project_id=%d fields.server_id=%d sink.index=%s monitor_port=%d pipelines=%d",
			projectID, serverID, strings.TrimSuffix(esCfg.IndexPattern, "*")+"${+YYYY.MM.DD}", monitorPort, bundle.PipelineCount,
		),
		PipelineYAML:      bundle.PipelineYAML,
		PipelinesOnlyYAML: bundle.PipelinesOnlyYAML,
		PipelineFilename:  bundle.PipelineFilename,
		PipelinesFilename: bundle.PipelinesFilename,
		EnvFile:           bundle.EnvFile,
		EnvFilename:       bundle.EnvFilename,
		HeartbeatScript:   bundle.HeartbeatScript,
		HeartbeatFilename: bundle.HeartbeatFilename,
		MonitorPort:       monitorPort,
		PipelineCount:     bundle.PipelineCount,
		SourceCount:       sourceCount,
		Deployed:          deployed,
		DeployMessage:     deployMsg,
	}
}
