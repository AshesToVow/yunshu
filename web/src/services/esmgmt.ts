import { getData, http } from "./http";

export interface EsmgmtConnection {
  id: number;
  name: string;
  addresses: string;
  username?: string;
  has_password?: boolean;
  timeout_sec?: number;
  is_default?: boolean;
  remark?: string;
  created_at?: string;
  updated_at?: string;
}

export interface EsmgmtBackupJob {
  id: number;
  connection_id?: number;
  index_name: string;
  trigger?: string;
  status: string;
  phase?: string;
  doc_count?: number;
  minio_bucket?: string;
  minio_object?: string;
  analysis_object?: string;
  mapping_object?: string;
  data_object?: string;
  error_message?: string;
  created_at?: string;
  updated_at?: string;
}

export interface EsmgmtRestoreJob {
  id: number;
  backup_job_id: number;
  connection_id?: number;
  source_index?: string;
  target_index: string;
  delete_existing?: boolean;
  status: string;
  phase?: string;
  doc_count?: number;
  error_message?: string;
  created_at?: string;
}

export interface EsmgmtBackupSchedule {
  id: number;
  connection_id: number;
  index_name: string;
  enabled: boolean;
  cron_spec: string;
  max_docs?: number;
  last_scheduled_at?: string;
  remark?: string;
}

export function listEsmgmtConnections(params?: { include_log_platform?: boolean }) {
  return getData<EsmgmtConnection[]>(
    http.get("/esmgmt/connections", {
      params: params?.include_log_platform ? { include_log_platform: true } : undefined,
    }),
  );
}

export function createEsmgmtConnection(payload: Record<string, unknown>) {
  return getData<EsmgmtConnection>(http.post("/esmgmt/connections", payload));
}

export function updateEsmgmtConnection(id: number, payload: Record<string, unknown>) {
  return getData<EsmgmtConnection>(http.put(`/esmgmt/connections/${id}`, payload));
}

export function deleteEsmgmtConnection(id: number) {
  return getData<{ ok: boolean }>(http.delete(`/esmgmt/connections/${id}`));
}

export function pingEsmgmtConnection(id: number) {
  return getData<{ ok: boolean; message?: string }>(http.post(`/esmgmt/connections/${id}/ping`, {}));
}

export function testEsmgmtConnection(payload: {
  addresses?: string;
  username?: string;
  password?: string;
  timeout_sec?: number;
  connection_id?: number;
}) {
  return getData<{ ok: boolean; message?: string }>(http.post("/esmgmt/connections/test", payload));
}

function connParams(connectionId?: number) {
  return typeof connectionId === "number" ? { connection_id: connectionId } : undefined;
}

export function getEsmgmtClusterHealth(connectionId?: number) {
  return getData<Record<string, unknown>>(
    http.get("/esmgmt/cluster/health", { params: connParams(connectionId) }),
  );
}

export function listEsmgmtIndices(params?: { connection_id?: number; pattern?: string }) {
  return getData<Array<{ name: string; store_bytes?: number; docs_count?: number }>>(
    http.get("/esmgmt/indices", { params }),
  );
}

export function createEsmgmtIndex(payload: {
  connection_id?: number;
  name: string;
  settings?: Record<string, unknown>;
  mappings?: Record<string, unknown>;
}) {
  return getData<{ ok: boolean; name: string }>(http.post("/esmgmt/indices", payload));
}

export function deleteEsmgmtIndex(name: string, force?: boolean, connectionId?: number) {
  return getData<{ ok: boolean }>(
    http.delete(`/esmgmt/indices/${encodeURIComponent(name)}`, {
      params: { force: force ? "true" : undefined, ...connParams(connectionId) },
    }),
  );
}

export function openEsmgmtIndex(name: string, connectionId?: number) {
  return getData<{ ok: boolean }>(
    http.post(`/esmgmt/indices/${encodeURIComponent(name)}/open`, {}, { params: connParams(connectionId) }),
  );
}

export function closeEsmgmtIndex(name: string, connectionId?: number) {
  return getData<{ ok: boolean }>(
    http.post(`/esmgmt/indices/${encodeURIComponent(name)}/close`, {}, { params: connParams(connectionId) }),
  );
}

export function listEsmgmtNodes(connectionId?: number) {
  return getData<Record<string, unknown>[]>(
    http.get("/esmgmt/nodes", { params: connParams(connectionId) }),
  );
}

export function proxyEsmgmtREST(payload: {
  connection_id?: number;
  method: string;
  path: string;
  body?: string;
}) {
  return getData<{ status: number; body: unknown }>(http.post("/esmgmt/proxy", payload));
}

export function createEsmgmtBackup(payload: { connection_id?: number; index: string; max_docs?: number }) {
  return getData<EsmgmtBackupJob>(http.post("/esmgmt/backups", payload));
}

export function listEsmgmtBackups(params?: { connection_id?: number; limit?: number }) {
  return getData<EsmgmtBackupJob[]>(http.get("/esmgmt/backups", { params }));
}

export function getEsmgmtBackup(id: number) {
  return getData<EsmgmtBackupJob>(http.get(`/esmgmt/backups/${id}`));
}

export function downloadEsmgmtBackup(id: number, artifact: "zip" | "analysis" | "mapping" | "data" = "zip") {
  return getData<{ url: string; artifact: string; object_key?: string; expires_in_sec?: number }>(
    http.get(`/esmgmt/backups/${id}/download`, { params: { artifact } }),
  );
}

export function createEsmgmtRestore(payload: {
  backup_job_id: number;
  connection_id?: number;
  target_index?: string;
  delete_existing?: boolean;
}) {
  return getData<EsmgmtRestoreJob>(http.post("/esmgmt/restores", payload));
}

export function listEsmgmtRestores(params?: { connection_id?: number; limit?: number }) {
  return getData<EsmgmtRestoreJob[]>(http.get("/esmgmt/restores", { params }));
}

export function listEsmgmtSchedules(connectionId?: number) {
  return getData<EsmgmtBackupSchedule[]>(
    http.get("/esmgmt/schedules", { params: connParams(connectionId) }),
  );
}

export function createEsmgmtSchedule(payload: Record<string, unknown>) {
  return getData<EsmgmtBackupSchedule>(http.post("/esmgmt/schedules", payload));
}

export function updateEsmgmtSchedule(id: number, payload: Record<string, unknown>) {
  return getData<EsmgmtBackupSchedule>(http.put(`/esmgmt/schedules/${id}`, payload));
}

export function deleteEsmgmtSchedule(id: number) {
  return getData<{ ok: boolean }>(http.delete(`/esmgmt/schedules/${id}`));
}
