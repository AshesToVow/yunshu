# Yunshu 纳管 Kubernetes 集群（连接说明）

## 产品模型（推荐）

1. 在平台配置集群凭证（能连通 API Server 即可）
2. 超管在 **集群授权** 给用户档位（只读 / 只读+Exec / 管理）与可选命名空间限制
3. 用户在控制台操作；**无需**在目标集群再为每个用户 apply RBAC YAML

> Impersonation（按 Yunshu 用户伪装到 apiserver）已下线，避免客户在集群侧重复维护 YAML。

## 两种连接方式

| 模式 | 适用 | 填写内容 |
|------|------|----------|
| **kubeconfig** | 已有完整 kubeconfig | 粘贴 YAML；可选再配「只读 kubeconfig」 |
| **direct（直连）** | 只有 API Server + Token | Server + Token + **集群根 CA**（或临时跳过 TLS） |

### 直连必看：CA 与 TLS

| 正确 | 错误 |
|------|------|
| `/etc/kubernetes/pki/ca.crt` 的 base64 | ServiceAccount Secret 里的 `ca.crt`（常与 APIServer 证书不匹配 → x509） |
| 或临时开启「跳过 TLS 验证」 | 既无 CA 又不跳过 TLS |

```bash
# 推荐：集群根 CA
base64 -w0 /etc/kubernetes/pki/ca.crt; echo

# 或从管理员 kubeconfig
kubectl config view --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}'; echo
```

Token 须为 **解码后的 JWT**（一般以 `eyJ` 开头）：

```bash
kubectl -n <ns> get secret <sa-token> -o jsonpath='{.data.token}' | base64 -d; echo
```

### 凭证权限建议

入库 Token / kubeconfig 对应身份在目标集群至少能：

- 读 `/version`
- `get/list/watch` namespaces（连接测试需要）
- 控制台实际用到的资源权限（建议用运维专用 SA，权限按环境裁剪）

可选：单独配置 **只读 kubeconfig**，只读 API 优先走该凭证。

## 授权（平台内完成）

路径：**集群列表 → 授权**（或授权矩阵）

- 档位：`readonly` / `readonly_exec` / `admin`
- 可限制命名空间黑/白名单
- 与 Casbin API 权限叠加；**不要求**在 K8s 再绑 `yunshu:<用户>`

## 连接测试

列表「测试」→ `GET /clusters/:id/status`：用入库凭证测版本 + 列命名空间。  
失败时接口返回 `connection_state=degraded` 与可读的 `last_error`（TLS / Token / 超时等）。

## 自检命令（在 Yunshu 能访问 API Server 的网络位置）

```bash
TOKEN='eyJ...'   # 解码后的 Token
# 联调
kubectl --server=https://<apiserver>:6443 --token="$TOKEN" --insecure-skip-tls-verify get ns
# 正式（替换为真实 ca.crt）
kubectl --server=https://<apiserver>:6443 --token="$TOKEN" --certificate-authority=/etc/kubernetes/pki/ca.crt get ns
```
