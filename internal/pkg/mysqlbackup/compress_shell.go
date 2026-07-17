package mysqlbackup

// BackupCompletedMarker 写入备份日志末行，用于与 xtrabackup 自身的 "completed OK!" 区分。
const BackupCompletedMarker = "yunshu backup completed OK!"

// shellResolveGzip 解析 gzip（兼容非交互 SSH 下 PATH 不含 /usr/bin）。
const shellResolveGzip = `
resolve_gzip() {
  if command -v gzip >/dev/null 2>&1; then command -v gzip; return 0; fi
  if [ -x /usr/bin/gzip ]; then echo /usr/bin/gzip; return 0; fi
  return 1
}
`

// shellTarGzFromDir 用 tar cf - | gzip -c 打 .tar.gz（勿 export GZIP=路径：GNU 里 GZIP 是压缩选项不是可执行文件）。
const shellTarGzFromDir = shellResolveGzip + `
export PATH="/usr/bin:/bin:${PATH:-}"
GZIP_BIN=$(resolve_gzip) || {
  echo "ERROR: 未找到 gzip，无法生成 .tar.gz。请执行: yum install -y gzip" >>"$LOG"
  exit 127
}
echo "[$(date '+%%F %%T')] packing with tar|gzip (gzip=$GZIP_BIN)" >>"$LOG"
echo "[$(date '+%%F %%T')] tmp dir size: $(du -sh "$TMP" 2>/dev/null | awk '{print $1}')" >>"$LOG"
rm -f "$ARCHIVE"
_pack_exit=0
tar -cf - -C "$TMP" . 2>>"$LOG" | "$GZIP_BIN" -c > "$ARCHIVE" 2>>"$LOG" || _pack_exit=$?
if [ "$_pack_exit" -ne 0 ]; then
  echo "ERROR: tar|gzip pack failed, exit=$_pack_exit" >>"$LOG"
  rm -f "$ARCHIVE"
  exit 1
fi
if [ ! -s "$ARCHIVE" ]; then
  echo "ERROR: 打包失败，归档为空: $ARCHIVE" >>"$LOG"
  rm -f "$ARCHIVE"
  exit 1
fi
`

// shellResolveMysqldump 解析 mysqldump 可执行文件（非交互 SSH 下 PATH 常不含自定义安装目录）。
const shellResolveMysqldump = `
resolve_mysqldump() {
  if [ -n "${MYSQLDUMP_BIN_PRESET:-}" ] && [ -x "$MYSQLDUMP_BIN_PRESET" ]; then
    echo "$MYSQLDUMP_BIN_PRESET"
    return 0
  fi
  if command -v mysqldump >/dev/null 2>&1; then
    command -v mysqldump
    return 0
  fi
  for p in /usr/bin/mysqldump /usr/local/bin/mysqldump /usr/local/mysql/bin/mysqldump; do
    if [ -x "$p" ]; then echo "$p"; return 0; fi
  done
  for p in /export/servers/app/*/bin/mysqldump /opt/mysql/*/bin/mysqldump /export/servers/*/bin/mysqldump; do
    if [ -x "$p" ]; then echo "$p"; return 0; fi
  done
  return 1
}
`

// shellResolveMysql 解析 mysql 客户端（与 mysqldump 同目录，非交互 SSH 下 PATH 常不可用）。
const shellResolveMysql = `
resolve_mysql() {
  if [ -n "${MYSQL_BIN_PRESET:-}" ] && [ -x "$MYSQL_BIN_PRESET" ]; then
    echo "$MYSQL_BIN_PRESET"
    return 0
  fi
  if [ -n "${MYSQLDUMP_BIN_PRESET:-}" ]; then
    _derived="${MYSQLDUMP_BIN_PRESET/mysqldump/mysql}"
    if [ -x "$_derived" ]; then echo "$_derived"; return 0; fi
  fi
  if command -v mysql >/dev/null 2>&1; then
    command -v mysql
    return 0
  fi
  for p in /usr/bin/mysql /usr/local/bin/mysql /usr/local/mysql/bin/mysql; do
    if [ -x "$p" ]; then echo "$p"; return 0; fi
  done
  for p in /export/servers/app/*/bin/mysql /opt/mysql/*/bin/mysql /export/servers/*/bin/mysql; do
    if [ -x "$p" ]; then echo "$p"; return 0; fi
  done
  return 1
}
`

// shellMysqldumpToGz 将 mysqldump 输出压缩为 .sql.gz（优先 pigz，管道用 stdbuf 降低首包等待）。
const shellMysqldumpToGz = shellResolveGzip + shellResolveMysqldump + `
resolve_pigz() {
  if command -v pigz >/dev/null 2>&1; then command -v pigz; return 0; fi
  if [ -x /usr/bin/pigz ]; then echo /usr/bin/pigz; return 0; fi
  return 1
}
export PATH="/usr/bin:/bin:${PATH:-}"
GZIP_BIN=$(resolve_gzip) || {
  echo "ERROR: 未找到 gzip，无法生成 .sql.gz。请执行: yum install -y gzip" >>"$LOG"
  exit 127
}
if PIGZ_BIN=$(resolve_pigz); then
  COMPRESS_BIN="$PIGZ_BIN"
  COMPRESS_ARGS="-1 -c"
  echo "[$(date '+%%F %%T')] compress with pigz ($PIGZ_BIN)" >>"$LOG"
else
  COMPRESS_BIN="$GZIP_BIN"
  COMPRESS_ARGS="-1 -c"
  echo "[$(date '+%%F %%T')] compress with gzip ($GZIP_BIN)" >>"$LOG"
fi
MYSQLDUMP_BIN=$(resolve_mysqldump) || {
  echo "ERROR: 未找到 mysqldump。请在实例中配置 mysqldump_bin（如 /export/servers/app/mysql-5.7.22/bin/mysqldump）" >>"$LOG"
  exit 127
}
echo "[$(date '+%%F %%T')] mysqldump bin=$MYSQLDUMP_BIN" >>"$LOG"
DUMP_CMD=("$MYSQLDUMP_BIN" %s %s %s)
echo "[$(date '+%%F %%T')] connect=%s target=%s flags=%s" >>"$LOG"
_dump_ec=0
if command -v stdbuf >/dev/null 2>&1; then
  stdbuf -oL "${DUMP_CMD[@]}" 2>>"$LOG" | "$COMPRESS_BIN" $COMPRESS_ARGS > "$SQL" || _dump_ec=$?
else
  "${DUMP_CMD[@]}" 2>>"$LOG" | "$COMPRESS_BIN" $COMPRESS_ARGS > "$SQL" || _dump_ec=$?
fi
EC=$_dump_ec
if [ "$EC" -ne 0 ]; then
  echo "ERROR: mysqldump pipeline failed exit=$EC" >>"$LOG"
  kill "$MON" 2>/dev/null || true
  wait "$MON" 2>/dev/null || true
  tail -n 120 "$LOG" 2>/dev/null || true
  exit "$EC"
fi
`
