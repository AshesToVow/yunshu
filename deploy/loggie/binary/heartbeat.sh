#!/usr/bin/env bash

# Loggie 心跳 + 监控端口状态采集（/api/v1/help/log、/metrics）

# 用法：

#   source loggie-heartbeat.env

#   ./heartbeat.sh



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

fi



json_escape() {

  python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$1" 2>/dev/null || printf '%s' '""'

}



payload=$(cat <<EOF

{"token":"$TOKEN","version":"$VERSION","health_status":"$HEALTH","pipeline_status":"$PIPELINE","es_sink_ok":$ES_OK,"lines_per_min":$LINES,"last_error":"$ERR","monitor_reachable":$MONITOR_REACHABLE,"monitor_port":$MONITOR_PORT,"active_fd_count":$ACTIVE_FD,"active_pipeline_count":$ACTIVE_PIPELINES,"monitor_detail":$(json_escape "$MONITOR_DETAIL")}

EOF

)



curl -sf -X POST "$YUNSHU_URL/api/v1/loggie/heartbeat/report" \

  -H "Content-Type: application/json" \

  -d "$payload"

echo

