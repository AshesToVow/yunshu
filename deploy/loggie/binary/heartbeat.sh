#!/usr/bin/env bash
# Loggie 心跳 + 监控端口状态采集（/api/v1/help/log、/metrics）
# 生产环境以 Yunshu 引导/热更下发的脚本为准（见 loggie_pipeline.go 模板）
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
INACTIVE_FD=0
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
    INACTIVE_FD="$(jq -r '.fdStatus.inActiveFdCount // 0' "$HELP_JSON" 2>/dev/null || echo 0)"
    ACTIVE_PIPELINES="$(jq -r '(.fileStatus.pipeline // {}) | length' "$HELP_JSON" 2>/dev/null || echo 0)"
    MONITOR_DETAIL="$(jq -c '{active_fd:(.fdStatus.activeFdCount // 0),inactive_fd:(.fdStatus.inActiveFdCount // 0),pipelines:((.fileStatus.pipeline // {})|keys)}' "$HELP_JSON" 2>/dev/null || echo "")"
  else
    ACTIVE_FD="$(grep -oE '"activeFdCount"[[:space:]]*:[[:space:]]*[0-9]+' "$HELP_JSON" | head -1 | grep -oE '[0-9]+' || echo 0)"
    INACTIVE_FD="$(grep -oE '"inActiveFdCount"[[:space:]]*:[[:space:]]*[0-9]+' "$HELP_JSON" | head -1 | grep -oE '[0-9]+' || echo 0)"
  fi
  PIPELINE="running"
fi

if curl -sf --max-time 5 "${MONITOR_URL}/metrics" -o "$METRICS_TMP"; then
  if grep -qE 'loggie_sink.*success|sink.*success' "$METRICS_TMP" 2>/dev/null; then
    ES_OK=true
  fi
fi

ACTIVE_FD="${ACTIVE_FD:-0}"
INACTIVE_FD="${INACTIVE_FD:-0}"
ACTIVE_PIPELINES="${ACTIVE_PIPELINES:-0}"

payload=$(cat <<EOF
{"token":"$TOKEN","version":"$VERSION","health_status":"$HEALTH","pipeline_status":"$PIPELINE","es_sink_ok":$ES_OK,"lines_per_min":$LINES,"last_error":"$ERR","monitor_reachable":$MONITOR_REACHABLE,"monitor_port":$MONITOR_PORT,"active_fd_count":$ACTIVE_FD,"inactive_fd_count":$INACTIVE_FD,"active_pipeline_count":$ACTIVE_PIPELINES,"monitor_detail":$(python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$MONITOR_DETAIL" 2>/dev/null || echo '""')}
EOF
)

curl -sf -X POST "$YUNSHU_URL/api/v1/loggie/heartbeat/report" \
  -H "Content-Type: application/json" \
  -d "$payload"
echo
