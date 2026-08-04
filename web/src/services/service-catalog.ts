import type { ApiResponse, PageData } from "../types/api";
import { getData, http } from "./http";

export interface ServiceLinkItem {
  id: number;
  service_id: number;
  link_type: string;
  ref_id?: number | null;
  ref_key?: string;
}

export interface ServiceCatalogItem {
  id: number;
  project_id: number;
  identifier: string;
  name: string;
  owner?: string;
  product_line?: string;
  criticality?: string;
  status: number;
  remark?: string;
  links?: ServiceLinkItem[];
}

export async function listServiceCatalog(
  projectId: number,
  params?: { keyword?: string; status?: number; page?: number; page_size?: number },
) {
  return getData<PageData<ServiceCatalogItem>>(
    http.get(`/projects/${projectId}/service-catalog`, { params }),
  );
}

export async function upsertServiceCatalog(
  projectId: number,
  payload: Partial<ServiceCatalogItem> & { identifier: string; name: string },
) {
  return getData<ServiceCatalogItem>(http.post(`/projects/${projectId}/service-catalog`, payload));
}

export async function deleteServiceCatalog(projectId: number, catalogId: number) {
  return getData<{ message: string }>(
    http.delete(`/projects/${projectId}/service-catalog/${catalogId}`) as Promise<ApiResponse<{ message: string }>>,
  );
}

export async function addServiceCatalogLink(
  projectId: number,
  catalogId: number,
  payload: { link_type: string; ref_id?: number; ref_key?: string },
) {
  return getData<ServiceLinkItem>(
    http.post(`/projects/${projectId}/service-catalog/${catalogId}/links`, payload),
  );
}

export async function deleteServiceCatalogLink(projectId: number, catalogId: number, linkId: number) {
  return getData(
    http.delete(`/projects/${projectId}/service-catalog/${catalogId}/links/${linkId}`) as Promise<
      ApiResponse<unknown>
    >,
  );
}
