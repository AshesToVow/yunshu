import { getData, http } from "./http";

export interface K8sCrTemplateItem {
  id: number;
  project_id: number;
  name: string;
  gvk_group?: string;
  gvk_version: string;
  gvk_kind: string;
  body: string;
  sort_order: number;
  created_at: string;
  updated_at?: string;
}

export function listK8sCrTemplates(params?: { project_id?: number; kind?: string }) {
  return getData<{ list: K8sCrTemplateItem[] }>(http.get("/k8s-cr-templates", { params })).then((r) => r.list || []);
}

export function createK8sCrTemplate(payload: {
  project_id?: number;
  name: string;
  gvk_group?: string;
  gvk_version?: string;
  gvk_kind: string;
  body: string;
  sort_order?: number;
}) {
  return getData<K8sCrTemplateItem>(http.post("/k8s-cr-templates", payload));
}

export function updateK8sCrTemplate(
  id: number,
  payload: {
    project_id?: number;
    name: string;
    gvk_group?: string;
    gvk_version?: string;
    gvk_kind: string;
    body: string;
    sort_order?: number;
  },
) {
  return getData<K8sCrTemplateItem>(http.put(`/k8s-cr-templates/${id}`, payload));
}

export function deleteK8sCrTemplate(id: number) {
  return getData<{ deleted?: boolean }>(http.delete(`/k8s-cr-templates/${id}`));
}
