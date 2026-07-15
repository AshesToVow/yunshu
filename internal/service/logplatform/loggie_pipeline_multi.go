package logplatform

import (
	"fmt"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/model"
)

const (
	defaultLoggieDeployDir = "/export/loggie"
	pipelinesOnlyFilename  = "pipelines.yml"
)

// LoggiePipelineSourceEntry 单个日志源对应一条 Loggie pipeline。
type LoggiePipelineSourceEntry struct {
	ServiceID    uint
	LogSourceID  uint
	LogType      string
	Paths        []string
	PipelineName string
	ParseProfile pipelineParseProfile
	ServiceName  string
	ProjectCode  string
	ProjectName  string
	ServerHost   string
	ServerName   string
	ExcludeRegex string
	Encoding     string
}

func logSourceToGlobPath(src model.ServiceLogSource) string {
	path := strings.TrimSpace(src.Path)
	if path == "" {
		return path
	}
	inc := ""
	if src.IncludeRegex != nil {
		inc = strings.TrimSpace(*src.IncludeRegex)
	}
	if inc != "" && !strings.Contains(path, "*") {
		if strings.HasSuffix(path, "/") {
			return path + inc
		}
		return strings.TrimRight(path, "/") + "/" + inc
	}
	// 目录型路径自动追加 /*.log，否则 file source 不会匹配目录下的滚动文件
	if !strings.ContainsAny(path, "*?") && !strings.HasSuffix(strings.ToLower(path), ".log") {
		return strings.TrimRight(path, "/") + "/*.log"
	}
	return path
}

func pipelineNameForSource(projectID, serverID, logSourceID uint, serviceID uint) string {
	if logSourceID > 0 {
		return fmt.Sprintf("yunshu-p%d-s%d-ls%d", projectID, serverID, logSourceID)
	}
	if serviceID > 0 {
		return fmt.Sprintf("yunshu-p%d-s%d-svc%d", projectID, serverID, serviceID)
	}
	return fmt.Sprintf("yunshu-p%d-s%d", projectID, serverID)
}

func sourcesFromLogSources(projectID, serverID uint, list []model.ServiceLogSource) []LoggiePipelineSourceEntry {
	out := make([]LoggiePipelineSourceEntry, 0, len(list))
	for _, src := range list {
		if strings.EqualFold(strings.TrimSpace(src.LogType), "journal") {
			continue
		}
		glob := logSourceToGlobPath(src)
		if glob == "" {
			continue
		}
		out = append(out, LoggiePipelineSourceEntry{
			ServiceID:    src.ServiceID,
			LogSourceID:  src.ID,
			LogType:      src.LogType,
			Paths:        []string{glob},
			PipelineName: pipelineNameForSource(projectID, serverID, src.ID, src.ServiceID),
			ParseProfile: parseProfileForLogSource(src),
			ExcludeRegex: derefStr(src.ExcludeRegex),
			Encoding:     derefStr(src.Encoding),
		})
	}
	return out
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func renderPipelineEntry(projectID, serverID uint, entry LoggiePipelineSourceEntry, hostsBlock, indexSink, sinkAuth string) string {
	parseProfile := entry.ParseProfile
	if !parseProfile.hasTransformer() {
		parseProfile = pipelineParseProfileFor(entry.Paths)
	}
	if parseProfile.multilinePattern == "" {
		parseProfile.multilinePattern = profileSpringLog().multilinePattern
	}
	var pathsLines strings.Builder
	for _, p := range entry.Paths {
		pathsLines.WriteString("          - ")
		pathsLines.WriteString(quoteYAML(p))
		pathsLines.WriteByte('\n')
	}
	fieldsExtra := ""
	if entry.ServiceID > 0 {
		fieldsExtra += fmt.Sprintf("\n          service_id: %q", fmt.Sprintf("%d", entry.ServiceID))
	}
	if entry.LogSourceID > 0 {
		fieldsExtra += fmt.Sprintf("\n          log_source_id: %q", fmt.Sprintf("%d", entry.LogSourceID))
	}
	if sn := strings.TrimSpace(entry.ServiceName); sn != "" {
		fieldsExtra += fmt.Sprintf("\n          service_name: %q", sn)
	}
	if pc := strings.TrimSpace(entry.ProjectCode); pc != "" {
		fieldsExtra += fmt.Sprintf("\n          project_code: %q", pc)
	}
	if pn := strings.TrimSpace(entry.ProjectName); pn != "" {
		fieldsExtra += fmt.Sprintf("\n          project_name: %q", pn)
	}
	if sh := strings.TrimSpace(entry.ServerHost); sh != "" {
		fieldsExtra += fmt.Sprintf("\n          server_host: %q", sh)
	}
	if sn := strings.TrimSpace(entry.ServerName); sn != "" {
		fieldsExtra += fmt.Sprintf("\n          server_name: %q", sn)
	}
	sourceName := "logs"
	if entry.LogSourceID > 0 {
		sourceName = fmt.Sprintf("ls-%d", entry.LogSourceID)
	}
	excludeBlock := ""
	if er := strings.TrimSpace(entry.ExcludeRegex); er != "" {
		excludeBlock = fmt.Sprintf("\n        exclude: [%s]", quoteYAML(er))
	}
	encodingBlock := ""
	if enc := strings.TrimSpace(entry.Encoding); enc != "" {
		encodingBlock = fmt.Sprintf("\n        charset: %s", quoteYAML(enc))
	}
	interceptorBlock := ""
	if parseProfile.hasTransformer() {
		interceptorBlock = parseProfile.renderTransformerActions()
	}
	return fmt.Sprintf(`  - name: %s
    sources:
      - type: file
        name: %s
        paths:
%s        addonMeta: true
        readFromTail: false%s%s
        fields:
          project_id: %q
          server_id: %q%s
        multiline:
          pattern: '%s'
          negate: true
          match: after
%s    sink:
      type: elasticsearch
      hosts:
%s      index: %s
      codec:
        type: json
        beatsFormat: true%s
`,
		entry.PipelineName,
		sourceName,
		pathsLines.String(),
		excludeBlock,
		encodingBlock,
		fmt.Sprintf("%d", projectID),
		fmt.Sprintf("%d", serverID),
		fieldsExtra,
		parseProfile.multilinePattern,
		ensureFileMetaActions(interceptorBlock, entry.Paths),
		hostsBlock,
		quoteYAML(indexSink),
		sinkAuth,
	)
}

// ensureFileMetaActions 保证 transformer 提升 file_path / host，并对 K8s pod 路径抽 typePodFields 风格元信息。
func ensureFileMetaActions(interceptorBlock string, paths []string) string {
	fileMeta := `          - action: copy(state.filename, file_path)
            ignoreError: true
          - action: copy(state.hostname, host)
            ignoreError: true
`
	k8sMeta := ""
	if isK8sPodLogPath(paths) {
		k8sMeta = `          - action: regex(file_path)
            pattern: '.*/pods/(?P<namespace>[^_]+)_(?P<podname>.+)_(?P<pod_uid>[0-9a-fA-F-]{32,36})/(?P<containername>[^/]+)/(?P<log_file>[^/]+)$'
            ignoreError: true
`
	}
	if strings.TrimSpace(interceptorBlock) == "" {
		return `    interceptors:
      - type: transformer
        actions:
` + fileMeta + k8sMeta
	}
	if strings.Contains(interceptorBlock, "copy(state.filename, file_path)") {
		out := interceptorBlock
		if !strings.Contains(out, "copy(state.hostname, host)") {
			out += `          - action: copy(state.hostname, host)
            ignoreError: true
`
		}
		if k8sMeta != "" && !strings.Contains(out, "podname") {
			out += k8sMeta
		}
		return out
	}
	return interceptorBlock + fileMeta + k8sMeta
}

func isK8sPodLogPath(paths []string) bool {
	for _, p := range paths {
		if strings.Contains(strings.ToLower(p), "/var/log/pods/") || strings.Contains(strings.ToLower(p), "/pods/") {
			return true
		}
	}
	return false
}

func renderPipelinesYAML(projectID, serverID uint, entries []LoggiePipelineSourceEntry, esCfg config.ElasticsearchConfig) string {
	esCfg = esCfg.Normalized()
	indexSink := AgentIndexSink(serverID)

	var hostsLines strings.Builder
	for _, h := range esCfg.Addresses {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		hostsLines.WriteString("        - ")
		hostsLines.WriteString(quoteYAML(h))
		hostsLines.WriteByte('\n')
	}
	if hostsLines.Len() == 0 {
		hostsLines.WriteString("        - \"http://127.0.0.1:9200\"\n")
	}
	hostsBlock := hostsLines.String()

	var sinkAuth strings.Builder
	if u := strings.TrimSpace(esCfg.Username); u != "" {
		sinkAuth.WriteString(fmt.Sprintf("\n      username: %s", quoteYAML(u)))
	}
	if p := strings.TrimSpace(esCfg.Password); p != "" {
		sinkAuth.WriteString(fmt.Sprintf("\n      password: %s", quoteYAML(p)))
	}

	var body strings.Builder
	body.WriteString("# Generated by Yunshu Loggie Bootstrap\n")
	body.WriteString(fmt.Sprintf("# project_id=%d server_id=%d pipelines=%d index=%s\n", projectID, serverID, len(entries), AgentIndexPattern(serverID)))
	body.WriteString("pipelines:\n")
	for _, e := range entries {
		body.WriteString(renderPipelineEntry(projectID, serverID, e, hostsBlock, indexSink, sinkAuth.String()))
	}
	return body.String()
}

func renderFullPipelineYAML(projectID, serverID uint, monitorPort int, entries []LoggiePipelineSourceEntry, esCfg config.ElasticsearchConfig) string {
	pipelines := renderPipelinesYAML(projectID, serverID, entries, esCfg)
	return fmt.Sprintf(`# Generated by Yunshu Loggie Bootstrap
# project_id=%d server_id=%d
loggie:
  reload:
    enabled: true
    period: 10s
  http:
    enabled: true
    host: "127.0.0.1"
    port: %d
  defaults:
    interceptors:
      - type: schema
        name: global
        order: 700
        addMeta:
          timestamp:
            key: "@timestamp"
  monitor:
    enabled: true
    listeners:
      filesource: ~
      filewatcher: ~
      sink: ~
      queue: ~

%s
`, projectID, serverID, defaultMonitorPort(monitorPort), strings.TrimPrefix(pipelines, "# Generated by Yunshu Loggie Bootstrap\n"))
}

// BuildMultiPipelineBundle 按日志源生成多 pipeline（每个源独立 service_id/log_source_id）。
func BuildMultiPipelineBundle(
	projectID, serverID uint,
	entries []LoggiePipelineSourceEntry,
	monitorPort int,
	esCfg config.ElasticsearchConfig,
	token, yunshuBaseURL, deployDir string,
) LoggiePipelineBundle {
	if len(entries) == 0 {
		opts := LoggiePipelineOptions{ProjectID: projectID, ServerID: serverID, MonitorPort: monitorPort}
		return BuildPipelineBundle(opts, esCfg, token, yunshuBaseURL)
	}
	monitorPort = defaultMonitorPort(monitorPort)
	deployDir = strings.TrimRight(strings.TrimSpace(deployDir), "/")
	if deployDir == "" {
		deployDir = defaultLoggieDeployDir
	}
	pipelinesOnly := renderPipelinesYAML(projectID, serverID, entries, esCfg)
	fullYAML := renderFullPipelineYAML(projectID, serverID, monitorPort, entries, esCfg)

	baseURL := strings.TrimRight(strings.TrimSpace(yunshuBaseURL), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	envFile := fmt.Sprintf(`# Yunshu Loggie heartbeat environment
YUNSHU_URL=%s
LOGGIE_TOKEN=%s
LOGGIE_VERSION=unknown
LOGGIE_MONITOR_PORT=%d
LOGGIE_DEPLOY_DIR=%s
`, baseURL, token, monitorPort, deployDir)

	return LoggiePipelineBundle{
		PipelineYAML:      fullYAML,
		PipelinesOnlyYAML: pipelinesOnly,
		PipelineFilename:  "pipeline.yml",
		PipelinesFilename: pipelinesOnlyFilename,
		DeployDir:         deployDir,
		EnvFile:           envFile,
		EnvFilename:       "loggie-heartbeat.env",
		HeartbeatScript:   heartbeatScriptTemplate(),
		HeartbeatFilename: "heartbeat.sh",
		StartScript:       renderStartScript(),
		StartFilename:     startScriptFilename,
		PipelineCount:     len(entries),
	}
}
