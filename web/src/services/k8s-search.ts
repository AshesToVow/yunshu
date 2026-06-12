import { getData, http } from "./http";

export interface K8sSearchItem {
  type: string;
  cluster_id: number;
  cluster_name: string;
  namespace: string;
  name: string;
  extra?: string;
  status?: string;
  link_path?: string;
}

export function searchK8s(params: { q: string; types?: string; limit?: number }) {
  return getData<K8sSearchItem[]>(http.get("/k8s/search", { params }));
}
