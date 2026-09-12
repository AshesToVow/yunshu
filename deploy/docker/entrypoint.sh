#!/bin/sh
set -eu

cd /app

# 业务配置唯一入口：configs/config.yaml（密钥/库地址/开关均在 YAML）。
# 启动期安全闸门由 Go config.Validate 负责；此处不再要求 ENCRYPTION_KEY/JWT_SECRET 环境变量。

CONFIG_FILE="${YUNSHU_CONFIG:-configs/config.yaml}"

# 从 YAML 指定段读取简单标量字段（host/port/driver/db_name）。
yaml_section_field() {
  section="$1"
  field="$2"
  awk -v section="$section" -v field="$field" '
    $0 ~ "^" section ":[[:space:]]*$" { in_sec=1; next }
    in_sec && /^[^[:space:]#]/ { exit }
    in_sec && $1 == field":" {
      val=$2
      gsub(/"/, "", val)
      gsub(/'\''/, "", val)
      sub(/[[:space:]]*#.*$/, "", val)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", val)
      print val
      exit
    }
  ' "$CONFIG_FILE" 2>/dev/null || true
}

# 与 Go normalizeDatabaseConfig 对齐：database 有 host 或 db_name 时优先，否则 mysql。
resolve_db_section() {
  db_host="$(yaml_section_field database host)"
  db_name="$(yaml_section_field database db_name)"
  if [ -n "${db_host}" ] || [ -n "${db_name}" ]; then
    echo database
  else
    echo mysql
  fi
}

normalize_driver() {
  d="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')"
  case "$d" in
    postgres|postgresql|pg) echo postgres ;;
    dameng|dm|dm8) echo dameng ;;
    *) echo mysql ;;
  esac
}

wait_database() {
  # WAIT_DB 优先；兼容旧 WAIT_MYSQL（设 0 可跳过）
  wait_flag="${WAIT_DB:-${WAIT_MYSQL:-1}}"
  if [ "${wait_flag}" != "1" ]; then
    return 0
  fi

  section="$(resolve_db_section)"
  driver="$(normalize_driver "$(yaml_section_field "$section" driver)")"
  host="${DATABASE_HOST:-${MYSQL_HOST:-$(yaml_section_field "$section" host)}}"
  port="${DATABASE_PORT:-${MYSQL_PORT:-$(yaml_section_field "$section" port)}}"
  host="${host:-172.17.0.1}"
  if [ -z "${port}" ] || [ "${port}" = "0" ]; then
    case "${driver}" in
      postgres) port=5432 ;;
      dameng) port=5236 ;;
      *) port=3306 ;;
    esac
  fi

  echo "waiting for ${driver} ${host}:${port} (section=${section}, from ${CONFIG_FILE} or DATABASE_*/MYSQL_* override) ..."
  i=0
  while [ "$i" -lt 60 ]; do
    if nc -z "$host" "$port" 2>/dev/null; then
      echo "${driver} is reachable"
      return 0
    fi
    i=$((i + 1))
    sleep 2
  done
  echo "warning: ${driver} not reachable after wait; continuing anyway" >&2
}

wait_database

# RUN_MIGRATE=1 时先执行 migrate（生产推荐；与 AutoMigrate 关闭配合）
if [ "${RUN_MIGRATE:-0}" = "1" ]; then
  echo "running: /app/yunshu migrate --config ${CONFIG_FILE}"
  /app/yunshu migrate --config "${CONFIG_FILE}"
fi

# RUN_SEED=1（默认）时先执行 seed；设为 0 可跳过
if [ "${RUN_SEED:-1}" = "1" ]; then
  echo "running: /app/yunshu seed --config ${CONFIG_FILE}"
  /app/yunshu seed --config "${CONFIG_FILE}"
fi

# 无参数时默认起 server；也可 docker run ... migrate / seed
if [ "$#" -eq 0 ]; then
  set -- server
fi

# 统一注入 --config（YUNSHU_CONFIG / 默认 configs/config.yaml），除非调用方已显式传入。
has_config=0
for a in "$@"; do
  case "$a" in
    --config|--config=*) has_config=1 ;;
  esac
done
if [ "${has_config}" -eq 0 ]; then
  set -- --config "${CONFIG_FILE}" "$@"
fi

exec /app/yunshu "$@"
