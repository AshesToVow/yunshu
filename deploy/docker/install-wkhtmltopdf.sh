#!/bin/sh
# 安装 wkhtmltopdf（巡检 HTML→PDF）。
# 必须使用静态链接二进制（surnet 镜像或 deploy/wkhtmltopdf/wkhtmltopdf 离线包）。
# 禁止 apk 安装 Alpine 3.14 的 wkhtmltopdf —— 在 Alpine 3.19 上会 Qt 库冲突崩溃。
set -u

BUNDLE_DIR="${1:-/tmp/wkhtmltopdf-bundle}"
WKHTML_BIN=/usr/local/bin/wkhtmltopdf

install_bundle() {
  if [ ! -f "$BUNDLE_DIR/wkhtmltopdf" ]; then
    return 1
  fi
  install -m 755 "$BUNDLE_DIR/wkhtmltopdf" "$WKHTML_BIN"
  echo "wkhtmltopdf: installed from offline bundle"
  return 0
}

remove_broken_apk() {
  # 清理 apk 误装的动态链接版（与 Alpine 3.19 Qt 不兼容）
  if [ -x /usr/bin/wkhtmltopdf ] && [ "$(readlink -f /usr/bin/wkhtmltopdf 2>/dev/null || echo /usr/bin/wkhtmltopdf)" != "$WKHTML_BIN" ]; then
    rm -f /usr/bin/wkhtmltopdf
    echo "wkhtmltopdf: removed incompatible /usr/bin/wkhtmltopdf (apk)"
  fi
}

smoke_test() {
  if [ ! -x "$WKHTML_BIN" ] && ! command -v wkhtmltopdf >/dev/null 2>&1; then
    echo "wkhtmltopdf: not installed — inspect PDF will use structured fallback"
    return 1
  fi
  local bin="${WKHTML_BIN}"
  if ! command -v "$bin" >/dev/null 2>&1; then
    bin="$(command -v wkhtmltopdf)"
  fi
  if ! "$bin" --version >/dev/null 2>&1; then
    echo "wkhtmltopdf: --version failed (Qt mismatch?)"
    rm -f "$WKHTML_BIN" /usr/bin/wkhtmltopdf 2>/dev/null || true
    return 1
  fi
  echo '<html><body><p>wkhtmltopdf smoke</p></body></html>' > /tmp/wkhtml-smoke.html
  if ! "$bin" --quiet --encoding utf-8 /tmp/wkhtml-smoke.html /tmp/wkhtml-smoke.pdf 2>/dev/null; then
    echo "wkhtmltopdf: smoke PDF failed (Qt mismatch?) — removing binary"
    rm -f "$WKHTML_BIN" /usr/bin/wkhtmltopdf 2>/dev/null || true
    rm -f /tmp/wkhtml-smoke.html /tmp/wkhtml-smoke.pdf
    return 1
  fi
  if [ ! -s /tmp/wkhtml-smoke.pdf ] || ! head -c 4 /tmp/wkhtml-smoke.pdf | grep -q '%PDF'; then
    echo "wkhtmltopdf: invalid smoke PDF output"
    rm -f "$WKHTML_BIN" /usr/bin/wkhtmltopdf 2>/dev/null || true
    return 1
  fi
  rm -f /tmp/wkhtml-smoke.html /tmp/wkhtml-smoke.pdf
  "$bin" --version 2>/dev/null | head -1
  echo "wkhtmltopdf: smoke test OK ($bin)"
  return 0
}

remove_broken_apk
install_bundle || true

if smoke_test; then
  exit 0
fi

# 离线包优先于镜像拷贝时再试一次
if install_bundle && smoke_test; then
  exit 0
fi

echo "wkhtmltopdf: unavailable — inspect PDF will use structured fallback"
exit 0
