#!/bin/bash
# 包装：供 telegraf systemd / cron 调用（Python 2.7 / 3 均可）
# 安装：
#   cp consul_targets_sync.py consul-targets.json consul-targets-ctl.sh \
#      /export/server/monitor/telegraf/scripts/
#   chmod 700 consul-targets-ctl.sh consul_targets_sync.py

set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIG="${CONSUL_TARGETS_CONFIG:-${DIR}/consul-targets.json}"
PY="${CONSUL_SYNC_PY:-${DIR}/consul_targets_sync.py}"

export CONSUL_TOKEN="${CONSUL_TOKEN:-}"
if [[ -z "${CONSUL_TOKEN}" ]] && [[ -f "${DIR}/.consul_token" ]]; then
  CONSUL_TOKEN="$(tr -d ' \n\r' < "${DIR}/.consul_token")"
  export CONSUL_TOKEN
fi

# 优先 python2.7 / python2 / python（你们机房是 2.7）
if [[ -n "${PYTHON_BIN:-}" ]]; then
  :
elif command -v python2.7 >/dev/null 2>&1; then
  PYTHON_BIN=python2.7
elif command -v python2 >/dev/null 2>&1; then
  PYTHON_BIN=python2
elif command -v python >/dev/null 2>&1; then
  PYTHON_BIN=python
elif command -v python3 >/dev/null 2>&1; then
  PYTHON_BIN=python3
else
  echo "ERROR: python not found" >&2
  exit 1
fi

exec "${PYTHON_BIN}" "${PY}" -c "${CONFIG}" "$@"
