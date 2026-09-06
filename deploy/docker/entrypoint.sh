#!/bin/sh
set -eu

cd /app

# 业务配置唯一入口：configs/config.yaml（密钥/库地址/开关均在 YAML）。
# 启动期安全闸门由 Go config.Validate 负责；此处不再要求 ENCRYPTION_KEY/JWT_SECRET 环境变量。

# 从 config.yaml 读取 mysql.host / mysql.port（供 wait 使用）
yaml_mysql_field() {
  field="$1"
  awk -v field="$field" '
    /^mysql:[[:space:]]*$/ { in_mysql=1; next }
    in_mysql && /^[^[:space:]#]/ { exit }
    in_mysql && $1 == field":" {
      val=$2
      gsub(/"/, "", val)
      gsub(/'\''/, "", val)
      print val
      exit
    }
  ' configs/config.yaml 2>/dev/null || true
}

wait_mysql() {
  if [ "${WAIT_MYSQL:-1}" != "1" ]; then
    return 0
  fi
  host="${MYSQL_HOST:-$(yaml_mysql_field host)}"
  port="${MYSQL_PORT:-$(yaml_mysql_field port)}"
  host="${host:-172.17.0.1}"
  port="${port:-3306}"
  echo "waiting for mysql ${host}:${port} (from config.yaml or MYSQL_* override) ..."
  i=0
  while [ "$i" -lt 60 ]; do
    if nc -z "$host" "$port" 2>/dev/null; then
      echo "mysql is reachable"
      return 0
    fi
    i=$((i + 1))
    sleep 2
  done
  echo "warning: mysql not reachable after wait; continuing anyway" >&2
}

wait_mysql

# RUN_MIGRATE=1 时先执行 migrate（生产推荐；与 AutoMigrate 关闭配合）
if [ "${RUN_MIGRATE:-0}" = "1" ]; then
  echo "running: /app/yunshu migrate"
  /app/yunshu migrate
fi

# RUN_SEED=1（默认）时先执行 seed；设为 0 可跳过
if [ "${RUN_SEED:-1}" = "1" ]; then
  echo "running: /app/yunshu seed"
  /app/yunshu seed
fi

# 无参数时默认起 server；也可 docker run ... migrate / seed
if [ "$#" -eq 0 ]; then
  set -- server
fi

exec /app/yunshu "$@"
