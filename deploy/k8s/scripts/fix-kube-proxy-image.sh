#!/bin/bash
# 修复 containerd 中 kube-proxy 缺 /usr/sbin/iptables 的问题。
# 根因: 渡渡鸟/Harbor 多层镜像 unpack 为 8b72ce21 错包；flat 镜像可绕过 layer dedupe。
# 用法: bash deploy/k8s/scripts/fix-kube-proxy-image.sh
set -euo pipefail

VER="${KUBE_PROXY_VER:-v1.28.15}"
SRC="${KUBE_PROXY_SRC:-swr.cn-north-4.myhuaweicloud.com/ddn-k8s/registry.k8s.io/kube-proxy@sha256:78891bdd6b9063822b0f0cd6f2d6ad5f8cdc34ab3afcbb931bc025bcd531a546}"
FLAT="localhost/kube-proxy:flat-${VER}"

echo "=== stop kubelet (if running) ==="
systemctl stop kubelet 2>/dev/null || true

echo "=== clean containerd refs ==="
for ref in $(ctr -n k8s.io images ls -q 2>/dev/null | grep -E 'kube-proxy|localhost/kube' || true); do
  ctr -n k8s.io images rm "$ref" 2>/dev/null || true
done

echo "=== docker pull & verify ==="
docker pull "$SRC"
docker run --rm --entrypoint /usr/sbin/iptables "$SRC" --version

echo "=== docker export | import flat ==="
CID=$(docker create --entrypoint="" "$SRC" /usr/local/bin/kube-proxy --help)
docker export "$CID" | tar -t | grep 'usr/sbin/iptables' >/dev/null
docker export "$CID" | docker import - "$FLAT"
docker rm "$CID"
docker run --rm --entrypoint /usr/sbin/iptables "$FLAT" --version

echo "=== import flat into containerd ==="
docker save "$FLAT" -o /tmp/kube-proxy-flat.tar
ctr -n k8s.io images import /tmp/kube-proxy-flat.tar
ctr -n k8s.io run --rm "$FLAT" iptables-test /usr/sbin/iptables --version

echo "=== done: use image ${FLAT} with imagePullPolicy: Never ==="
systemctl start kubelet 2>/dev/null || true
