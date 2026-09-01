// @ts-nocheck
/** K8s involvedObject kind → 控制台菜单路径（与 dynamic-menu-page PATH_COMPONENT_FALLBACK 对齐） */
const KIND_PATH: Record<string, string> = {
  Pod: "/pods",
  Deployment: "/deployments",
  StatefulSet: "/statefulsets",
  DaemonSet: "/daemonsets",
  ReplicaSet: "/deployments",
  Job: "/jobs",
  CronJob: "/cronjobs",
  Service: "/k8s-services",
  Ingress: "/ingresses",
  ConfigMap: "/configmaps",
  Secret: "/secrets",
  Node: "/nodes",
  Namespace: "/namespaces",
  PersistentVolume: "/persistentvolumes",
  PersistentVolumeClaim: "/persistentvolumeclaims",
  HorizontalPodAutoscaler: "/horizontal-pod-autoscalers",
};

export function k8sResourceMenuPath(kind?: string): string | undefined {
  const k = String(kind ?? "").trim();
  if (!k) return undefined;
  return KIND_PATH[k];
}

/** 构建带 cluster/namespace/keyword 的资源页链接 query */
export function buildK8sResourceLink(params: {
  kind?: string;
  name?: string;
  clusterId?: number;
  namespace?: string;
}): string | undefined {
  const path = k8sResourceMenuPath(params.kind);
  if (!path) return undefined;
  const qs = new URLSearchParams();
  if (params.clusterId && params.clusterId > 0) qs.set("cluster_id", String(params.clusterId));
  if (params.namespace) qs.set("namespace", params.namespace);
  if (params.name) qs.set("keyword", params.name);
  const q = qs.toString();
  return q ? `${path}?${q}` : path;
}
