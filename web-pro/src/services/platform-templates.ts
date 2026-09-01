// @ts-nocheck
import { getData, http } from "./http";
import type { ApiResponse } from "../types/api";

export type PlatformTemplate = {
  id: number;
  template_key: string;
  category: string;
  name: string;
  format: string;
  description?: string;
  published_version: number;
  status: number;
  is_builtin: boolean;
  published_checksum?: string;
  has_minio_mirror?: boolean;
  created_at?: string;
  updated_at?: string;
};

export type PlatformTemplateVersion = {
  id: number;
  template_id: number;
  version: number;
  content_inline?: string;
  storage_key?: string;
  checksum?: string;
  remark?: string;
  content_preview?: string;
  created_by?: number;
  created_at?: string;
};

export type PlatformTemplateResolve = {
  template_key: string;
  version: number;
  format: string;
  content: string;
  source: string;
};

export type PlatformTemplateListQuery = {
  category?: string;
  keyword?: string;
  status?: number;
  page?: number;
  page_size?: number;
};

type ListResult = { list: PlatformTemplate[]; total: number; page: number; page_size: number };

function asApi<T>(p: unknown) {
  return p as Promise<ApiResponse<T>>;
}

export async function listPlatformTemplates(params: PlatformTemplateListQuery = {}) {
  return getData<ListResult>(asApi(http.get("/platform-templates", { params })));
}

export async function getPlatformTemplate(id: number) {
  return getData<PlatformTemplate>(asApi(http.get(`/platform-templates/${id}`)));
}

export async function createPlatformTemplate(payload: {
  template_key: string;
  category: string;
  name: string;
  format?: string;
  description?: string;
  status?: number;
}) {
  return getData<PlatformTemplate>(asApi(http.post("/platform-templates", payload)));
}

export async function updatePlatformTemplate(
  id: number,
  payload: { category: string; name: string; format?: string; description?: string; status?: number },
) {
  return getData<PlatformTemplate>(asApi(http.put(`/platform-templates/${id}`, payload)));
}

export async function deletePlatformTemplate(id: number) {
  return getData(asApi(http.delete(`/platform-templates/${id}`)));
}

export async function listPlatformTemplateVersions(id: number) {
  return getData<PlatformTemplateVersion[]>(asApi(http.get(`/platform-templates/${id}/versions`)));
}

export async function getPlatformTemplateVersion(id: number, version: number) {
  return getData<PlatformTemplateVersion>(asApi(http.get(`/platform-templates/${id}/versions/${version}`)));
}

export async function savePlatformTemplateDraft(id: number, content: string, remark?: string) {
  return getData<PlatformTemplateVersion>(asApi(http.post(`/platform-templates/${id}/drafts`, { content, remark })));
}

export async function publishPlatformTemplate(id: number, version?: number) {
  return getData<PlatformTemplate>(asApi(http.post(`/platform-templates/${id}/publish`, { version: version ?? 0 })));
}

export async function resolvePlatformTemplate(templateKey: string) {
  return getData<PlatformTemplateResolve>(
    asApi(http.get(`/platform-templates/resolve/${encodeURIComponent(templateKey)}`)),
  );
}
