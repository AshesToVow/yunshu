#!/bin/bash
# K8s Pod → Consul 同步包装
# 安装：
#   cp consul_k8s_pods_sync.py consul-k8s-pods.json consul-k8s-pods-ctl.sh \
#      /export/server/monitor/consul/
#   chmod 700 consul-k8s-pods-ctl.sh consul_k8s_pods_sync.py
# cron 示例（每 2 分钟）：
#   */2 * * * * /export/server/monitor/consul/consul-k8s-pods-ctl.sh sync >>/var/log/consul-k8s-pods.log 2>&1

set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIG="${CONSUL_K8S_PODS_CONFIG:-${DIR}/consul-k8s-pods.json}"
PY="${CONSUL_K8S_PODS_PY:-${DIR}/consul_k8s_pods_sync.py}"

export CONSUL_TOKEN="${CONSUL_TOKEN:-}"
if [[ -z "${CONSUL_TOKEN}" ]] && [[ -f "${DIR}/.consul_token" ]]; then
  CONSUL_TOKEN="$(tr -d ' \n\r' < "${DIR}/.consul_token")"
  export CONSUL_TOKEN
fi

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
