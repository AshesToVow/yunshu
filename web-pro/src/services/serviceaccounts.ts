// @ts-nocheck
import { getData, http } from "./http";
import { pickK8sDeleteOpts, type K8sDeleteOptions } from "./service-factory";

export type ServiceAccountItem = {
  name: string;
  namespace: string;
  secrets_count: number;
  image_pull_secrets_count: number;
  creation_time: string;
};

export type ServiceAccountBindingRef = {
  name: string;
  namespace?: string;
  role_ref: string;
};

export type ServiceAccountDetail = {
  yaml: string;
  secrets?: string[];
  image_pull_secrets?: string[];
  role_bindings?: ServiceAccountBindingRef[];
  cluster_role_bindings?: ServiceAccountBindingRef[];
};

export function listServiceAccounts(clusterId: number, namespace: string, keyword?: string) {
  return getData<ServiceAccountItem[]>(
    http.get("/serviceaccounts", { params: { cluster_id: clusterId, namespace, keyword } }),
  );
}

export function getServiceAccountDetail(clusterId: number, namespace: string, name: string) {
  return getData<ServiceAccountDetail>(
    http.get("/serviceaccounts/detail", { params: { cluster_id: clusterId, namespace, name } }),
  );
}

export function applyServiceAccount(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/serviceaccounts/apply", { cluster_id: clusterId, manifest }));
}

export function deleteServiceAccount(
  clusterId: number,
  namespace: string,
  name: string,
  deleteOpts?: K8sDeleteOptions,
) {
  return getData<boolean>(
    http.delete("/serviceaccounts", {
      params: { cluster_id: clusterId, namespace, name, ...pickK8sDeleteOpts(deleteOpts ?? {}) },
    }),
  );
}
