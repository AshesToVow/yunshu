// @ts-nocheck
import { getData, http } from "./http";

export type K8sMyAccess = {
  cluster_id: number;
  access_rank: number;
  access_preset?: string;
  capabilities?: string[];
};

/** 当前登录用户在指定集群上的有效档位（仅需登录） */
export function getMyK8sAccess(clusterId: number) {
  return getData<K8sMyAccess>(http.get("/k8s-policies/my-access", { params: { cluster_id: String(clusterId) } }));
}

export const K8S_ACCESS_RANK = {
  none: 0,
  readonly: 1,
  readonlyExec: 2,
  admin: 3,
} as const;
