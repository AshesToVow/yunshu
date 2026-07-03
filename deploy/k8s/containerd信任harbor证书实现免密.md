# containerd 信任 Harbor 证书（节点拉镜像免证书报错）

> 适用：K8s 节点使用 **containerd** 作为 CRI，Harbor 为 **HTTPS 自签/私有 CA**（如 `harbor.deploy.local`）。  
> 目标：节点 `crictl pull` / kubelet 拉 Pod 镜像时 **不再出现 x509 证书错误**。  
> **所有 Master / Worker 节点均需执行**（Jenkins Agent 若走 containerd/CRI 也需配置）。

---

## 变量（按环境修改）

| 变量 | 示例值 | 说明 |
|------|--------|------|
| `HARBOR_DOMAIN` | `harbor.deploy.local` | Harbor 域名（与 `/etc/hosts` 或 DNS 一致） |
| `HARBOR_CERT_DIR` | `/export/server/harbor/certs` | Harbor 安装目录下的证书路径 |
| `PAUSE_IMAGE` | `harbor.deploy.local/registry/pause:3.9` | Pod 沙箱 pause 镜像（需在 Harbor 中存在） |

---

## 一、系统级信任 Harbor CA

让 `curl`、部分工具能校验 Harbor HTTPS。

```bash
HARBOR_DOMAIN=harbor.deploy.local
HARBOR_CERT_DIR=/export/server/harbor/certs   # 或 /root/harbor/certs

cd "${HARBOR_CERT_DIR}"
cp ca.crt /etc/pki/ca-trust/source/anchors/harbor-ca.crt
update-ca-trust extract
```

验证（不应再出现 certificate error）：

```bash
curl -vk "https://${HARBOR_DOMAIN}/v2/" 2>&1 | grep -iE 'SSL|subject|issuer|error'
```

---

## 二、配置 containerd `certs.d`（核心）

containerd **不会**自动使用系统 CA 信任库，必须在 `certs.d` 为每个 registry 域名单独配置。

### 2.1 仅信任 CA（Harbor 项目已设为公开或走 imagePullSecret）

```bash
HARBOR_DOMAIN=harbor.deploy.local
HARBOR_CERT_DIR=/export/server/harbor/certs

mkdir -p "/etc/containerd/certs.d/${HARBOR_DOMAIN}"
cp "${HARBOR_CERT_DIR}/ca.crt" "/etc/containerd/certs.d/${HARBOR_DOMAIN}/ca.crt"

cat > "/etc/containerd/certs.d/${HARBOR_DOMAIN}/hosts.toml" <<EOF
server = "https://${HARBOR_DOMAIN}"

[host."https://${HARBOR_DOMAIN}"]
  capabilities = ["pull", "resolve", "push"]
  ca = "/etc/containerd/certs.d/${HARBOR_DOMAIN}/ca.crt"
EOF
```

### 2.2 同时写入 Harbor 账号（节点级免 docker/crictl login）

若 Harbor 项目需登录，可在 `hosts.toml` 增加 `username` / `password`（**明文，注意权限**）：

```bash
cat > "/etc/containerd/certs.d/${HARBOR_DOMAIN}/hosts.toml" <<EOF
server = "https://${HARBOR_DOMAIN}"

[host."https://${HARBOR_DOMAIN}"]
  capabilities = ["pull", "resolve", "push"]
  ca = "/etc/containerd/certs.d/${HARBOR_DOMAIN}/ca.crt"
  username = "admin"
  password = "你的Harbor密码"
EOF

chmod 600 "/etc/containerd/certs.d/${HARBOR_DOMAIN}/hosts.toml"
```

> K8s 业务 Pod 仍建议使用 `imagePullSecrets`（`registry-secret`）；节点级 `hosts.toml` 主要方便 kubelet 拉 pause、系统组件及 `crictl` 调试。

---

## 三、修改 `/etc/containerd/config.toml`

### 3.1 启用 `certs.d` 目录

containerd **≥ 1.5** 支持；需显式设置 `config_path`：

```bash
# 若 config.toml 不存在，先生成默认配置
mkdir -p /etc/containerd
containerd config default | tee /etc/containerd/config.toml

# 启用 certs.d
sed -i 's|config_path = ""|config_path = "/etc/containerd/certs.d"|' /etc/containerd/config.toml
grep config_path /etc/containerd/config.toml
```

### 3.2 sandbox_image 指向 Harbor 中的 pause

```bash
PAUSE_IMAGE=harbor.deploy.local/registry/pause:3.9

sed -i "s|sandbox_image = \".*\"|sandbox_image = \"${PAUSE_IMAGE}\"|" /etc/containerd/config.toml
grep sandbox_image /etc/containerd/config.toml
```

若配置里仍有旧域名（如 `harbor.jdicity.local`），统一替换：

```bash
sed -i 's|harbor.jdicity.local|harbor.deploy.local|g' /etc/containerd/config.toml
```

### 3.3 systemd cgroup（与 kubelet 一致）

kubelet 使用 `cgroupDriver: systemd` 时，runc 选项须为 `SystemdCgroup = true`：

```toml
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
  SystemdCgroup = true
```

一键修改（在默认生成的 config 上）：

```bash
sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml
```

### 3.4 配置参考片段

```toml
[plugins."io.containerd.grpc.v1.cri"]
  sandbox_image = "harbor.deploy.local/registry/pause:3.9"

  [plugins."io.containerd.grpc.v1.cri".containerd]
    config_path = "/etc/containerd/certs.d"

    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
      runtime_type = "io.containerd.runc.v2"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
        BinaryName = "/usr/bin/runc"
        SystemdCgroup = true
```

---

## 四、重启并验证

```bash
systemctl daemon-reload
systemctl restart containerd
systemctl restart kubelet

# CRI 插件应 ok
ctr plugins ls | grep -i cri

# 拉取测试（替换为实际业务镜像）
crictl pull harbor.deploy.local/registry/pause:3.9
crictl pull harbor.deploy.local/registry/springbootdemo:latest_prod_xxx

crictl images | grep harbor.deploy.local
```

K8s 侧验证：

```bash
kubectl run test-harbor --image=harbor.deploy.local/registry/pause:3.9 --restart=Never
kubectl get pod test-harbor
kubectl delete pod test-harbor
```

---

## 五、排错

| 现象 | 处理 |
|------|------|
| `x509: certificate signed by unknown authority` | 检查 `certs.d/.../ca.crt` 是否为 Harbor **CA**（不是 server.crt）；`config_path` 是否已启用并 restart containerd |
| `401 Unauthorized` | 证书已通，属鉴权问题：在 `hosts.toml` 加账号密码，或配置 K8s `imagePullSecrets` |
| `manifest unknown` / `not found` | 镜像名/tag 错误或 Harbor 中不存在；与证书无关 |
| Pod 一直 `ImagePullBackOff` | `kubectl describe pod` 看 Events；在**该 Pod 所在节点**用 `crictl pull` 同镜像复现 |
| `sandbox_image` 拉失败 | pause 镜像未推到 Harbor，或 `sandbox_image` 地址写错 |
| 改 config 不生效 | 确认 `systemctl restart containerd` 且无 config 语法错误：`containerd config dump` |

查看日志：

```bash
journalctl -u containerd -n 80 --no-pager
journalctl -u kubelet -n 80 --no-pager
```

---

## 附录 A：升级 containerd 至 1.7.x（旧集群可选）

仅当节点 containerd 版本过旧（如 1.3.x）、不支持 `config_path` / `certs.d` 时执行。  
**在 Master 上操作前务必评估停机窗口**；生产环境建议先在 Worker 验证。

```bash
# 0. 停服务并清理沙箱
systemctl stop kubelet
systemctl stop docker 2>/dev/null
crictl stopp $(crictl pods -q) 2>/dev/null
crictl rmp $(crictl pods -q) 2>/dev/null

# 1. 备份
cp -a /usr/bin/containerd /usr/bin/containerd.bak 2>/dev/null
cp -a /usr/bin/ctr /usr/bin/ctr.bak 2>/dev/null
cp -a /usr/bin/runc /usr/bin/runc.bak 2>/dev/null
cp -a /etc/containerd/config.toml /etc/containerd/config.toml.bak 2>/dev/null

# 2. 安装 containerd 1.7.18
cd /tmp
wget https://github.com/containerd/containerd/releases/download/v1.7.18/containerd-1.7.18-linux-amd64.tar.gz
tar Cxzvf /usr/local containerd-1.7.18-linux-amd64.tar.gz
cp /usr/local/bin/containerd /usr/local/bin/ctr /usr/local/bin/containerd-shim-runc-v2 /usr/bin/
chmod +x /usr/bin/containerd /usr/bin/ctr /usr/bin/containerd-shim-runc-v2

# 3. 安装 runc 1.1.14
wget https://github.com/opencontainers/runc/releases/download/v1.1.14/runc.amd64
install -m 755 runc.amd64 /usr/bin/runc

# 4. CNI 插件（若缺失）
mkdir -p /opt/cni/bin
if [ ! -f /opt/cni/bin/bridge ]; then
  wget https://github.com/containernetworking/plugins/releases/download/v1.4.1/cni-plugins-linux-amd64-v1.4.1.tgz
  tar Cxzvf cni-plugins-linux-amd64-v1.4.1.tgz -C /opt/cni/bin
fi

# 5. 生成配置 → 再按本文「二、三」章配置 certs.d 与 sandbox_image
containerd config default | tee /etc/containerd/config.toml
sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml

# 6. 验证并启动
containerd -v
runc --version
systemctl daemon-reload
systemctl restart containerd
systemctl start kubelet
sleep 40
crictl ps | grep -E 'etcd|apiserver|controller|scheduler'
```

升级完成后，**必须**重新完成本文第二、三章（`certs.d` + `config_path` + `sandbox_image`）。

---

## 附录 B：与 Docker / Jenkins 的区别

| 组件 | 配置位置 |
|------|----------|
| containerd（kubelet） | `/etc/containerd/certs.d/<域名>/hosts.toml` |
| Docker（Jenkins 构建） | `/etc/docker/daemon.json` + `docker login` 或 `~/.docker/config.json` |
| K8s Pod | `imagePullSecrets` + 命名空间内 `registry-secret` |

三者互不影响，需分别配置。
