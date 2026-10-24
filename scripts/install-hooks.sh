#!/bin/sh
# 安装仓库 Git 钩子（commit-msg 规范 + pre-commit gofmt 检查）。
# 用法：sh scripts/install-hooks.sh
set -e

ROOT=$(cd "$(dirname "$0")/.." && pwd)

if [ ! -d "$ROOT/githooks" ]; then
  echo "githooks 目录不存在：$ROOT/githooks" >&2
  exit 1
fi

git -C "$ROOT" config core.hooksPath githooks
chmod +x "$ROOT/githooks/commit-msg" "$ROOT/githooks/pre-commit" 2>/dev/null || true
echo "已启用仓库钩子：core.hooksPath=githooks"
echo "验证：$(git -C "$ROOT" config --get core.hooksPath)"
