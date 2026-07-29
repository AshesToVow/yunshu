#!/bin/sh
set -eu

cd /app

# 可选：等待 MySQL 可连（宿主机或容器 MySQL）
wait_mysql() {
  if [ "${WAIT_MYSQL:-1}" != "1" ]; then
    return 0
  fi
  host="${MYSQL_HOST:-host.docker.internal}"
  port="${MYSQL_PORT:-3306}"
  echo "waiting for mysql ${host}:${port} ..."
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
