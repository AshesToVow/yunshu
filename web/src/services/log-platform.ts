import type { ApiResponse } from "../types/api";
import { getData, http } from "./http";
import { getProjects, type ProjectItem } from "./projects";

export type { ProjectItem };

export interface LogRetentionItem {
  id?: number;
  project_id: number;
  server_id?: number;
  retention_days: number;
  max_store_bytes?: number;
  max_index_count?: number;
  enabled: boolean;
  index_pattern?: string;
  remark?: string;
  source?: string;
  updated_at?: string;
}

export interface ESStorageStats {
  index_pattern: string;
  index_count: number;
  document_count: number;
  store_bytes: number;
  store_human: string;
}

export interface LogRetentionCleanupResult {
  deleted_indices: string[];
  deleted_documents: number;
  message: string;
}

export async function getGlobalLogRetention() {
  return await getData(http.get<any, ApiResponse<LogRetentionItem>>("/log-platform/retention"));
}

export async function upsertGlobalLogRetention(payload: {
  retention_days: number;
  enabled: boolean;
  index_pattern?: string;
  remark?: string;
}) {
  return await getData(http.put<any, ApiResponse<LogRetentionItem>>("/log-platform/retention", payload));
}

export async function listLogRetentionPolicies() {
  return await getData(http.get<any, ApiResponse<{ list: LogRetentionItem[] }>>("/log-platform/retention/list"));
}

export async function getESStorageStats() {
  return await getData(http.get<any, ApiResponse<ESStorageStats>>("/log-platform/es-storage"));
}

export async function runLogRetentionCleanup() {
  return await getData(http.post<any, ApiResponse<LogRetentionCleanupResult>>("/log-platform/retention/cleanup"));
}

export async function upsertProjectLogRetention(
  projectId: number,
  payload: { retention_days: number; enabled: boolean; index_pattern?: string; remark?: string },
) {
  return await getData(http.put<any, ApiResponse<LogRetentionItem>>(`/projects/${projectId}/log-retention`, payload));
}

export async function deleteProjectLogRetention(projectId: number) {
  return await getData(http.delete<any, ApiResponse<{ message: string }>>(`/projects/${projectId}/log-retention`));
}

export interface ESConfigPreview {
  enabled: boolean;
  addresses: string[];
  username?: string;
  index_pattern: string;
  has_password: boolean;
}

export interface LoggieStatusItem {
  server_id: number;
  server_name: string;
  server_host: string;
  deploy_mode?: "binary" | "k8s" | string;
  cluster_id?: number;
  k8s_namespace?: string;
  daemonset_name?: string;
  registered: boolean;
  online: boolean;
  recent_ingest: boolean;
  version?: string;
  health_status?: string;
  pipeline_status?: string;
  es_sink_ok: boolean;
  lines_per_min: number;
  last_seen_at?: string;
  last_ingest_at?: string;
  last_error?: string;
  recent_doc_count: number;
  monitor_port?: number;
  monitor_reachable?: boolean;
  active_pipeline_count?: number;
  active_fd_count?: number;
  monitor_detail?: string;
  live_probe?: {
    reachable: boolean;
    active_fd_count: number;
    inactive_fd_count: number;
    active_pipeline_count: number;
    pipeline_names?: string[];
    error?: string;
  };
}

export interface LoggieBootstrapSourcePreview {
  log_source_id: number;
  service_id: number;
  log_type: string;
  path: string;
  glob_path: string;
}

export interface LoggieBootstrapResult {
  token: string;
  project_id: number;
  server_id: number;
  deploy_mode?: "binary" | "k8s" | string;
  cluster_id?: number;
  k8s_namespace?: string;
  daemonset_name?: string;
  es_addresses: string[];
  es_index_pattern: string;
  report_url: string;
  pipeline_hint: string;
  pipeline_yaml: string;
  pipelines_only_yaml?: string;
  pipeline_filename: string;
  pipelines_filename?: string;
  env_file?: string;
  env_filename?: string;
  heartbeat_script?: string;
  heartbeat_filename?: string;
  monitor_port: number;
  pipeline_count?: number;
  source_count?: number;
  deployed?: boolean;
  deploy_message?: string;
  k8s_manifest?: string;
}

export interface LoggieBootstrapPayload {
  server_id?: number;
  deploy_mode?: "binary" | "k8s";
  cluster_id?: number;
  k8s_namespace?: string;
  daemonset_name?: string;
  k8s_require_pod_label?: boolean;
  log_paths?: string[];
  service_id?: number;
  log_source_id?: number;
  monitor_port?: number;
  yunshu_url?: string;
  deploy_dir?: string;
  auto_from_log_sources?: boolean;
  deploy_after_bootstrap?: boolean;
}

export interface LoggieDeployPayload {
  server_id?: number;
  deploy_mode?: "binary" | "k8s";
  cluster_id?: number;
  sync_from_db?: boolean;
  restart_loggie?: boolean;
}

export interface LoggieDeployResult {
  success: boolean;
  message: string;
  stdout?: string;
  stderr?: string;
  pipeline_count?: number;
  source_count?: number;
  deployed_at?: string;
  deploy_mode?: string;
}

function downloadText(content: string, filename: string, mime = "text/plain;charset=utf-8") {
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export function downloadLoggieBundle(bundle: LoggieBootstrapResult) {
  const manifest = bundle.k8s_manifest || (bundle.deploy_mode === "k8s" ? bundle.pipeline_yaml : "");
  if (manifest) {
    downloadText(manifest, bundle.pipeline_filename || "loggie-k8s-manifest.yaml", "application/x-yaml;charset=utf-8");
    if (bundle.pipelines_only_yaml) {
      downloadText(bundle.pipelines_only_yaml, bundle.pipelines_filename || "clusterlogconfig.yaml", "application/x-yaml;charset=utf-8");
    }
    return;
  }
  downloadText(bundle.pipeline_yaml, bundle.pipeline_filename || "pipeline.yml", "application/x-yaml;charset=utf-8");
  if (bundle.pipelines_only_yaml) {
    downloadText(bundle.pipelines_only_yaml, bundle.pipelines_filename || "pipelines.yml", "application/x-yaml;charset=utf-8");
  }
  if (bundle.env_file) {
    downloadText(bundle.env_file, bundle.env_filename || "loggie-heartbeat.env");
  }
  if (bundle.heartbeat_script) {
    downloadText(bundle.heartbeat_script, bundle.heartbeat_filename || "heartbeat.sh", "text/x-sh;charset=utf-8");
  }
}

export async function downloadLoggieFile(
  projectId: number,
  serverId: number,
  file: "pipeline" | "pipelines" | "env" | "heartbeat" = "pipeline",
) {
  const blob = (await http.get(`/projects/${projectId}/loggie/pipeline/download`, {
    params: { server_id: serverId, file },
    responseType: "blob",
  })) as unknown as Blob;
  const names: Record<string, string> = {
    pipeline: "pipeline.yml",
    pipelines: "pipelines.yml",
    env: "loggie-heartbeat.env",
    heartbeat: "heartbeat.sh",
  };
  downloadText(await blob.text(), names[file] ?? "pipeline.yml");
}

export async function getESConfigPreview() {
  return await getData(http.get<any, ApiResponse<ESConfigPreview>>("/log-platform/es-config"));
}

export async function getLoggieStatus(projectId: number) {
  return await getData(http.get<any, ApiResponse<{ list: LoggieStatusItem[] }>>(`/projects/${projectId}/loggie/status`));
}

export async function getLoggieBootstrapSources(projectId: number, serverId: number) {
  return await getData(
    http.get<any, ApiResponse<{ list: LoggieBootstrapSourcePreview[] }>>(`/projects/${projectId}/loggie/bootstrap-sources`, {
      params: { server_id: serverId },
    }),
  );
}

export async function bootstrapLoggie(projectId: number, payload: LoggieBootstrapPayload) {
  return await getData(http.post<any, ApiResponse<LoggieBootstrapResult>>(`/projects/${projectId}/loggie/bootstrap`, payload));
}

export async function deployLoggieConfig(projectId: number, payload: LoggieDeployPayload) {
  return await getData(http.post<any, ApiResponse<LoggieDeployResult>>(`/projects/${projectId}/loggie/deploy`, payload));
}

export async function restartLoggie(projectId: number, payload: LoggieDeployPayload) {
  return await getData(http.post<any, ApiResponse<LoggieDeployResult>>(`/projects/${projectId}/loggie/restart`, payload));
}

export async function syncLoggieFromLogSources(projectId: number, payload: LoggieDeployPayload) {
  return await getData(http.post<any, ApiResponse<LoggieDeployResult>>(`/projects/${projectId}/loggie/sync`, payload));
}

export { getProjects };
