# ImagePullBackOff 排障剧本

## 目标
定位镜像拉取失败原因（仓库、鉴权、标签、网络）。

## 检查步骤
1. Events 中查找 Failed、ErrImagePull、ImagePullBackOff 消息。
2. 核对 image 字段：仓库地址、tag/digest 是否存在。
3. 核对 imagePullSecrets / ServiceAccount 是否绑定正确。
4. 节点到仓库网络、DNS、TLS、Harbor 项目权限。
5. 私有仓库是否缺少 registry-secret。

## 输出要求
- 区分「镜像不存在」与「鉴权失败」与「网络超时」
- 给出检查清单，不自动创建 Secret
