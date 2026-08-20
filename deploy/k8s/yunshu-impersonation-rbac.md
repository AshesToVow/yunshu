# Yunshu Impersonation RBAC 模板说明

## 什么时候需要？

| 场景 | 是否 apply 本模板 |
|------|-------------------|
| 只开平台「集群授权」，**不开** Impersonation | **不需要** |
| 集群编辑里打开了 **启用 Impersonation** | **需要** |

日常交付推荐：关闭 Impersonation，平台内授权即可。本模板仅给「要在 apiserver 审计到真实 Yunshu 用户」的客户。

## 一键安装

```bash
# 1) 按需改 YAML：角色码 ops/viewer 若平台没有，可删对应 ClusterRoleBinding
kubectl apply -f deploy/k8s/yunshu-impersonation-rbac.yaml

# 2) 取 Token（须解码）与 CA（secret 里已是 base64，直接贴到平台「CA 证书」）
kubectl -n yunshu-system get secret yunshu-gateway-token \
  -o jsonpath='{.data.token}' | base64 -d; echo
kubectl -n yunshu-system get secret yunshu-gateway-token \
  -o jsonpath='{.data.ca\.crt}'; echo

# 3) Yunshu 集群编辑（直连）：
#    - API Server：https://<apiserver>:6443
#    - Token：上面解码结果（以 eyJ 开头）；不要贴 yaml 里未解码的 data.token
#    - CA 证书：上面 ca.crt 那串 base64；或临时打开「跳过 TLS 验证」
#    - 开启 Impersonation、前缀 yunshu:
# 4) 保存后再点列表上的「测试」
```

### 平台报 500 常见原因

| 现象 | 原因 | 处理 |
|------|------|------|
| 500 / 心跳失败 x509 | 直连未填 CA 且未跳过 TLS | 填 CA，或开「跳过 TLS 验证」 |
| Token 无效 | 粘贴了 Secret 里 **未解码** 的 `data.token` | 用上面 `base64 -d` 后再贴 |
| 403 列举命名空间 | 网关 SA 只有 impersonate | 重新 `kubectl apply` 本模板（已含 namespaces get/list） |

> 安全：若 Token 曾出现在聊天/工单中，请删除 Secret 重建并轮换后再写入平台。

## 身份对照

| Yunshu | 集群侧 |
|--------|--------|
| 用户名 `admin` | User `yunshu:admin` |
| 角色码 `super-admin` | Group `yunshu:role:super-admin` |
| （固定） | Group `yunshu:authenticated` |

平台「集群档位授权」与本 YAML **互相独立**：  
- 平台授权：拦控制台按钮  
- 本 YAML：拦 apiserver  

两边都过才能在开启 Impersonation 时真正操作。

## 自检

```bash
kubectl --as=yunshu:admin \
  --as-group=yunshu:authenticated \
  --as-group=yunshu:role:super-admin \
  get ns,crd
```

失败则检查：Group 名是否与平台角色码一致、前缀是否仍是 `yunshu:`。
