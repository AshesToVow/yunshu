#!/bin/sh
# 安装 wkhtmltopdf（巡检 HTML→PDF）。不依赖 Docker Hub 多阶段镜像。
# 优先级：离线包 > Alpine 3.14 apk（CDN 通常可达）> 跳过（Go 结构化 PDF 降级）
set -u

BUNDLE_DIR="${1:-/tmp/wkhtmltopdf-bundle}"
WKHTML_BIN=/usr/local/bin/wkhtmltopdf

if [ -f "$BUNDLE_DIR/wkhtmltopdf" ]; then
  install -m 755 "$BUNDLE_DIR/wkhtmltopdf" "$WKHTML_BIN"
  echo "wkhtmltopdf: installed from offline bundle"
  exit 0
fi

# Alpine 3.15+ 官方仓库已移除；v3.14 community 在多数环境仍可通过 dl-cdn 拉取
if apk add --no-cache \
  --repository=https://dl-cdn.alpinelinux.org/alpine/v3.14/community \
  --repository=https://dl-cdn.alpinelinux.org/alpine/v3.14/main \
  wkhtmltopdf 2>/dev/null; then
  echo "wkhtmltopdf: installed via alpine v3.14 apk"
  exit 0
fi

echo "wkhtmltopdf: skipped (inspect PDF will use Go structured fallback)"
exit 0
