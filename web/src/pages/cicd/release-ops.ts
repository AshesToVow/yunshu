// CI/CD 发布操作与 K8s 部署模板枚举。
// 由 web/src/pages/cicd-services-page.tsx 原样搬迁（RF-09），未改动任何 value/label 取值。
// 注意：value 与后端流水线参数一一对应（含中文 value，如「使用deployment模板」），不得「顺手英文化」。

export const FRONTEND_RELEASE_OPS = [
  { value: "frontend_online", label: "服务上线" },
  { value: "frontend_rollback", label: "服务回滚" },
] as const;

export const BACKEND_RELEASE_OPS = [
  { value: "backend_initial", label: "服务初次部署" },
  { value: "backend_update", label: "服务更新" },
] as const;

export const CONTAINER_RELEASE_OPS = [
  { value: "service_online", label: "服务上线" },
  { value: "pod_update", label: "POD 更新" },
  { value: "container_rollback", label: "回滚" },
] as const;

export const K8S_DEPLOY_CONFIG_TYPES = [
  { value: "使用deployment模板", label: "Deployment" },
  { value: "使用statefulset模板", label: "StatefulSet" },
  { value: "使用daemonset模板", label: "DaemonSet" },
] as const;

export const K8S_DEPLOY_TEMPLATES = [
  { value: "基础模板", label: "基础模板" },
  { value: "通用微服务含skywalking", label: "通用微服务含 SkyWalking" },
] as const;

export function releaseOpLabel(op: string) {
  return [...FRONTEND_RELEASE_OPS, ...BACKEND_RELEASE_OPS, ...CONTAINER_RELEASE_OPS].find((o) => o.value === op)?.label ?? op;
}
