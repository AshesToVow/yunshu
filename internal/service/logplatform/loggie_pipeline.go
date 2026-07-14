package logplatform

import (
	"fmt"
	"strings"

	"yunshu/internal/config"
)

// LoggiePipelineOptions 生成 pipeline.yml 所需参数。
type LoggiePipelineOptions struct {
	ProjectID    uint
	ServerID     uint
	ServiceID    uint
	LogSourceID  uint
	LogPaths     []string
	MonitorPort  int
	PipelineName string
}

type LoggiePipelineBundle struct {
	PipelineYAML      string `json:"pipeline_yaml"`
	PipelinesOnlyYAML string `json:"pipelines_only_yaml"`
	PipelineFilename  string `json:"pipeline_filename"`
	PipelinesFilename string `json:"pipelines_filename"`
	DeployDir         string `json:"deploy_dir"`
	EnvFile           string `json:"env_file"`
	EnvFilename       string `json:"env_filename"`
	HeartbeatScript   string `json:"heartbeat_script"`
	HeartbeatFilename string `json:"heartbeat_filename"`
	PipelineCount     int    `json:"pipeline_count"`
}

func defaultLogPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"/var/log/myapp/*.log"}
	}
	return out
}

func defaultMonitorPort(port int) int {
	if port <= 0 {
		return 9196
	}
	return port
}

func defaultPipelineName(projectID, serverID uint, name string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return fmt.Sprintf("yunshu-p%d-s%d", projectID, serverID)
}

func quoteYAML(s string) string {
	return fmt.Sprintf("%q", s)
}

// BuildPipelineBundle 生成完整 Loggie 部署文件（pipeline + 心跳脚本 + env）。
func BuildPipelineBundle(opts LoggiePipelineOptions, esCfg config.ElasticsearchConfig, token, yunshuBaseURL string) LoggiePipelineBundle {
	opts.LogPaths = defaultLogPaths(opts.LogPaths)
	opts.MonitorPort = defaultMonitorPort(opts.MonitorPort)
	opts.PipelineName = defaultPipelineName(opts.ProjectID, opts.ServerID, opts.PipelineName)

	entry := LoggiePipelineSourceEntry{
		ServiceID:    opts.ServiceID,
		LogSourceID:  opts.LogSourceID,
		Paths:        opts.LogPaths,
		PipelineName: opts.PipelineName,
		ParseProfile: pipelineParseProfileFor(opts.LogPaths),
	}
	pipelinesOnly := renderPipelinesYAML(opts.ProjectID, opts.ServerID, []LoggiePipelineSourceEntry{entry}, esCfg)
	pipelineYAML := renderFullPipelineYAML(opts.ProjectID, opts.ServerID, opts.MonitorPort, []LoggiePipelineSourceEntry{entry}, esCfg)

	baseURL := strings.TrimRight(strings.TrimSpace(yunshuBaseURL), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}

	envFile := fmt.Sprintf(`# Yunshu Loggie heartbeat environment
YUNSHU_URL=%s
LOGGIE_TOKEN=%s
LOGGIE_VERSION=unknown
LOGGIE_MONITOR_PORT=%d
`, baseURL, token, opts.MonitorPort)

	heartbeatScript := heartbeatScriptTemplate()

	return LoggiePipelineBundle{
		PipelineYAML:      pipelineYAML,
		PipelinesOnlyYAML: pipelinesOnly,
		PipelineFilename:  "pipeline.yml",
		PipelinesFilename: pipelinesOnlyFilename,
		DeployDir:         defaultLoggieDeployDir,
		EnvFile:           envFile,
		EnvFilename:       "loggie-heartbeat.env",
		HeartbeatScript:   heartbeatScript,
		HeartbeatFilename: "heartbeat.sh",
		PipelineCount:     1,
	}
}

func heartbeatScriptTemplate() string {
	return `#!/usr/bin/env bash
# Loggie 心跳 + 监控端口状态采集（/api/v1/help/log、/metrics）
set -euo pipefail

YUNSHU_URL="${YUNSHU_URL:-http://127.0.0.1:8080}"
TOKEN="${LOGGIE_TOKEN:-}"
VERSION="${LOGGIE_VERSION:-unknown}"
MONITOR_PORT="${LOGGIE_MONITOR_PORT:-9196}"

if [[ -z "$TOKEN" ]]; then
  echo "LOGGIE_TOKEN required" >&2
  exit 1
fi

HEALTH="running"
PIPELINE="running"
ES_OK=false
LINES=0
ERR=""
MONITOR_REACHABLE=false
ACTIVE_FD=0
ACTIVE_PIPELINES=0
MONITOR_DETAIL=""

if ! pgrep -x loggie >/dev/null 2>&1 && ! pgrep -f "loggie -config" >/dev/null 2>&1; then
  HEALTH="stopped"
  PIPELINE="stopped"
  ERR="loggie process not found"
fi

HELP_JSON="$(mktemp)"
METRICS_TMP="$(mktemp)"
trap 'rm -f "$HELP_JSON" "$METRICS_TMP"' EXIT

MONITOR_URL="http://127.0.0.1:${MONITOR_PORT}"
if curl -sf --max-time 5 "${MONITOR_URL}/api/v1/help/log" -o "$HELP_JSON"; then
  MONITOR_REACHABLE=true
  if command -v jq >/dev/null 2>&1; then
    ACTIVE_FD="$(jq -r '.fdStatus.activeFdCount // 0' "$HELP_JSON" 2>/dev/null || echo 0)"
    ACTIVE_PIPELINES="$(jq -r '.fileStatus.pipeline | length // 0' "$HELP_JSON" 2>/dev/null || echo 0)"
    MONITOR_DETAIL="$(jq -c '{active_fd:.fdStatus.activeFdCount,inactive_fd:.fdStatus.inActiveFdCount,pipelines:(.fileStatus.pipeline|keys)}' "$HELP_JSON" 2>/dev/null || echo "")"
  else
    ACTIVE_FD="$(grep -o '"activeFdCount"[[:space:]]*:[[:space:]]*[0-9]*' "$HELP_JSON" | head -1 | grep -o '[0-9]*' || echo 0)"
  fi
  PIPELINE="running"
fi

if curl -sf --max-time 5 "${MONITOR_URL}/metrics" -o "$METRICS_TMP"; then
  if grep -qE 'loggie_sink.*success|sink.*success' "$METRICS_TMP" 2>/dev/null; then
    ES_OK=true
  fi
  RATE="$(grep -E 'loggie_source_lines_total|loggie_eventbus' "$METRICS_TMP" | head -3 | tr '\n' ';' || true)"
  if [[ -n "$RATE" && -z "$MONITOR_DETAIL" ]]; then
    MONITOR_DETAIL="{\"metrics_snippet\":\"${RATE}\"}"
  fi
fi

payload=$(cat <<EOF
{"token":"$TOKEN","version":"$VERSION","health_status":"$HEALTH","pipeline_status":"$PIPELINE","es_sink_ok":$ES_OK,"lines_per_min":$LINES,"last_error":"$ERR","monitor_reachable":$MONITOR_REACHABLE,"monitor_port":$MONITOR_PORT,"active_fd_count":$ACTIVE_FD,"active_pipeline_count":$ACTIVE_PIPELINES,"monitor_detail":$(python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$MONITOR_DETAIL" 2>/dev/null || echo '""')}
EOF
)

curl -sf -X POST "$YUNSHU_URL/api/v1/loggie/heartbeat/report" \
  -H "Content-Type: application/json" \
  -d "$payload"
echo
`
}
