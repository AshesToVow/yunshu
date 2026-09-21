#!/usr/bin/env bash
# PostgreSQL 冒烟：migrate → seed → server(health)。
# 依赖：Docker（临时拉起 postgres + redis）、Go toolchain。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PG_NAME="yunshu-smoke-pg"
REDIS_NAME="yunshu-smoke-redis"
CFG_DIR="$(mktemp -d "${TMPDIR:-/tmp}/yunshu-smoke-pg.XXXXXX")"
CFG="${CFG_DIR}/config.yaml"
BIN="${CFG_DIR}/yunshu"
SERVER_LOG="${CFG_DIR}/server.log"
SERVER_PID=""

cleanup() {
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
  docker rm -f "${PG_NAME}" "${REDIS_NAME}" >/dev/null 2>&1 || true
  rm -rf "${CFG_DIR}"
}
trap cleanup EXIT

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

need_cmd docker
need_cmd go
need_cmd curl

echo "==> start postgres + redis"
docker rm -f "${PG_NAME}" "${REDIS_NAME}" >/dev/null 2>&1 || true
docker run -d --name "${PG_NAME}" \
  -e POSTGRES_USER=yunshu \
  -e POSTGRES_PASSWORD=yunshu \
  -e POSTGRES_DB=yunshu \
  -p 55432:5432 \
  postgres:16-alpine >/dev/null
docker run -d --name "${REDIS_NAME}" \
  -p 56379:6379 \
  redis:7-alpine redis-server --requirepass 1234 >/dev/null

echo "==> wait for postgres"
for _ in $(seq 1 60); do
  if docker exec "${PG_NAME}" pg_isready -U yunshu -d yunshu >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker exec "${PG_NAME}" pg_isready -U yunshu -d yunshu >/dev/null

# casbin model 用仓库相对路径；工作目录为 ROOT
cat >"${CFG}" <<EOF
app:
  name: yunshu-smoke-pg
  env: dev
  port: 18080
plugins:
  enabled: [core, k8s, alert, project, cmdb, backup, cicd, dbmgmt, inspect, ai, esmgmt]
ai:
  enabled: false
database:
  driver: postgres
  host: 127.0.0.1
  port: 55432
  user: yunshu
  password: "yunshu"
  db_name: yunshu
  sslmode: disable
  timezone: Asia/Shanghai
redis:
  addr: 127.0.0.1:56379
  password: "1234"
  db: 0
auth:
  jwt_secret: "YunshuJWTSecretKey2026MustBeAtLeast32BytesLong!!"
security:
  encryption_key: "dGVzdC1lbmNyeXB0aW9uLWtleS0zMmJ5dGVzISE="
casbin:
  model_path: configs/casbin_model.conf
  auto_load_interval_seconds: 0
http:
  read_header_timeout_seconds: 5
  read_timeout_seconds: 30
  write_timeout_seconds: 30
  idle_timeout_seconds: 30
  shutdown_timeout_seconds: 5
log:
  level: warn
  format: json
  output: console
swagger:
  enabled: false
EOF

echo "==> build"
go build -o "${BIN}" .

echo "==> migrate"
"${BIN}" migrate --config "${CFG}"

echo "==> seed"
"${BIN}" seed --config "${CFG}"

echo "==> server"
"${BIN}" server --config "${CFG}" >"${SERVER_LOG}" 2>&1 &
SERVER_PID=$!

ok=0
for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:18080/api/v1/health" >/dev/null 2>&1; then
    ok=1
    break
  fi
  if ! kill -0 "${SERVER_PID}" 2>/dev/null; then
    echo "server exited early; log:" >&2
    cat "${SERVER_LOG}" >&2 || true
    exit 1
  fi
  sleep 1
done

if [[ "${ok}" -ne 1 ]]; then
  echo "health check failed; server log:" >&2
  cat "${SERVER_LOG}" >&2 || true
  exit 1
fi

echo "PG smoke OK: migrate → seed → /api/v1/health"
