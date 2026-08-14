#!/bin/sh
# 样例：从 stdin 读 JSON，输出磁盘使用（只读）
path="/"
if command -v python >/dev/null 2>&1; then
  path=$(python -c 'import json,sys; d=json.load(sys.stdin); print(d.get("path") or "/")' 2>/dev/null || echo "/")
elif command -v python2.7 >/dev/null 2>&1; then
  path=$(python2.7 -c 'import json,sys; d=json.load(sys.stdin); print d.get("path") or "/"' 2>/dev/null || echo "/")
else
  cat >/dev/null
fi
out=$(df -h "$path" 2>/dev/null | tail -n +2 | head -n 5)
printf '{"ok":true,"path":"%s","df":%s}\n' "$path" "$(printf '%s' "$out" | python -c 'import json,sys; print(json.dumps(sys.stdin.read()))' 2>/dev/null || printf '""')"
