import { getData, http } from "./http";

/** 对应 Kubernetes DeleteOptions 的可选字段 */
export type K8sDeleteOptions = {
  grace_period_seconds?: number;
  propagation_policy?: "Background" | "Foreground" | "Orphan";
};

export function pickK8sDeleteOpts(args: K8sDeleteOptions): K8sDeleteOptions | undefined {
  const opts: K8sDeleteOptions = {};
  if (args.grace_period_seconds != null && args.grace_period_seconds >= 0) {
    opts.grace_period_seconds = args.grace_period_seconds;
  }
  if (args.propagation_policy) {
    opts.propagation_policy = args.propagation_policy;
  }
  return Object.keys(opts).length > 0 ? opts : undefined;
}

export function k8sParams(clusterId: number, params?: Record<string, any>, deleteOpts?: K8sDeleteOptions) {
  return { cluster_id: clusterId, ...(params ?? {}), ...(deleteOpts ?? {}) };
}

export function createK8sResourceService<Item, Detail>(basePath: string) {
  return {
    list: (params: Record<string, any>) => getData<Item[]>(http.get(basePath, { params })),
    detail: (params: Record<string, any>) => getData<Detail>(http.get(`${basePath}/detail`, { params })),
    apply: (body: Record<string, any>) => getData<boolean>(http.post(`${basePath}/apply`, body)),
    remove: (params: Record<string, any>) => getData<boolean>(http.delete(basePath, { params })),
    get: <T>(subPath: string, params: Record<string, any>) => getData<T>(http.get(`${basePath}${subPath}`, { params })),
    post: <T>(subPath: string, body: Record<string, any>) => getData<T>(http.post(`${basePath}${subPath}`, body)),
  };
}
