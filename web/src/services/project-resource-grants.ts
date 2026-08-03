import { getData, http } from "./http";

export interface ServerAccessGrantItem {
  id: number;
  project_id: number;
  server_id: number;
  principal_kind: string;
  principal_ref: string;
  can_view: boolean;
  can_exec: boolean;
  can_manage: boolean;
  remark?: string;
  server_name?: string;
  server_host?: string;
  username?: string;
  nickname?: string;
}

export interface CicdAccessGrantItem {
  id: number;
  project_id: number;
  service_id: number;
  principal_kind: string;
  principal_ref: string;
  can_view: boolean;
  can_build: boolean;
  can_release: boolean;
  can_manage: boolean;
  remark?: string;
  service_name?: string;
  username?: string;
  nickname?: string;
}

export async function getMyServerAccess(projectId: number, serverId: number) {
  return getData<{ can_view: boolean; can_exec: boolean; can_manage: boolean }>(
    http.get(`/projects/${projectId}/servers/${serverId}/my-access`),
  );
}

export async function listServerAccessGrants(projectId: number, params?: { user_id?: number; server_id?: number }) {
  const data = await getData<{ list?: ServerAccessGrantItem[] }>(
    http.get(`/projects/${projectId}/server-access-grants`, { params }),
  );
  return data.list ?? [];
}

export async function upsertServerAccessGrant(
  projectId: number,
  body: { server_id: number; user_id: number; can_view?: boolean; can_exec?: boolean; can_manage?: boolean; remark?: string },
) {
  return getData<ServerAccessGrantItem>(http.post(`/projects/${projectId}/server-access-grants`, body));
}

export async function bulkUpsertServerAccessGrants(
  projectId: number,
  body: { user_id: number; server_ids: number[]; can_view?: boolean; can_exec?: boolean; can_manage?: boolean },
) {
  return getData<{ upserted: number }>(http.post(`/projects/${projectId}/server-access-grants/bulk`, body));
}

export async function deleteServerAccessGrant(projectId: number, grantId: number) {
  return getData<{ message: string }>(http.delete(`/projects/${projectId}/server-access-grants/${grantId}`));
}

export async function bootstrapServerAccessGrants(projectId: number) {
  return getData<Record<string, number>>(http.post(`/projects/${projectId}/server-access-grants/bootstrap`, {}));
}

export async function listCicdAccessGrants(projectId: number, params?: { user_id?: number; service_id?: number }) {
  const data = await getData<{ list?: CicdAccessGrantItem[] }>(
    http.get(`/projects/${projectId}/cicd-access-grants`, { params }),
  );
  return data.list ?? [];
}

export async function upsertCicdAccessGrant(
  projectId: number,
  body: {
    service_id: number;
    user_id: number;
    can_view?: boolean;
    can_build?: boolean;
    can_release?: boolean;
    can_manage?: boolean;
    remark?: string;
  },
) {
  return getData<CicdAccessGrantItem>(http.post(`/projects/${projectId}/cicd-access-grants`, body));
}

export async function bulkUpsertCicdAccessGrants(
  projectId: number,
  body: {
    user_id: number;
    service_ids: number[];
    can_view?: boolean;
    can_build?: boolean;
    can_release?: boolean;
    can_manage?: boolean;
  },
) {
  return getData<{ upserted: number }>(http.post(`/projects/${projectId}/cicd-access-grants/bulk`, body));
}

export async function deleteCicdAccessGrant(projectId: number, grantId: number) {
  return getData<{ message: string }>(http.delete(`/projects/${projectId}/cicd-access-grants/${grantId}`));
}

export async function bootstrapCicdAccessGrants(projectId: number) {
  return getData<Record<string, number>>(http.post(`/projects/${projectId}/cicd-access-grants/bootstrap`, {}));
}
