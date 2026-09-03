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

export interface ESIndexStatItem {
  name: string;
  docs_count: number;
  store_bytes: number;
  store_human: string;
  matched_pattern: boolean;
}

export interface ESStorageStats {
  index_pattern: string;
  index_count: number;
  document_count: number;
  store_bytes: number;
  store_human: string;
  pattern_index_count?: number;
  pattern_document_count?: number;
  pattern_store_bytes?: number;
  pattern_store_human?: string;
  indices?: ESIndexStatItem[];
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

export async function deleteESIndex(index: string) {
  return await getData(
    http.delete<any, ApiResponse<{ message: string; index: string }>>(
      `/log-platform/es-indices/${encodeURIComponent(index)}`,
    ),
  );
}

export interface KafkaPartitionLag {
  topic?: string;
  partition: number;
  high_water_mark: number;
  consumer_offset: number;
  lag: number;
}

export interface KafkaQueueStats {
  enabled: boolean;
  sink_via_kafka: boolean;
  brokers: string[];
  topic_prefix?: string;
  topics?: string[];
  consumer_group: string;
  consumer_running: boolean;
  /** 本进程内仍在跑的 Kafka→ES 消费协程数 */
  consumer_workers?: number;
  lag_total: number;
  partitions?: KafkaPartitionLag[];
  consumed_total: number;
  written_total: number;
  error_total: number;
  last_consume_at?: string;
  last_error?: string;
  message?: string;
  has_sasl?: boolean;
}

export interface KafkaConfigPreview {
  enabled: boolean;
  brokers: string[];
  topic_prefix?: string;
  topic_example?: string;
  consumer_group: string;
  username?: string;
  has_password: boolean;
  sasl_mechanism?: string;
  batch_size: number;
  topic_partitions?: number;
  workers: number;
  sink_via_kafka: boolean;
}

export async function getKafkaQueueStats() {
  return await getData(http.get<any, ApiResponse<KafkaQueueStats>>("/log-platform/kafka-stats"));
}

export async function getKafkaConfigPreview() {
  return await getData(http.get<any, ApiResponse<KafkaConfigPreview>>("/log-platform/kafka-config"));
}

export async function deleteKafkaTopic(topic: string) {
  return await getData(
    http.delete<any, ApiResponse<{ message: string; topic: string }>>(
      `/log-platform/kafka-topics/${encodeURIComponent(topic)}`,
    ),
  );
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
  connection_id?: number;
  connection_name?: string;
  source?: "managed" | "dict" | string;
}

export interface LoggieStatusItem {
  server_id: number;
  server_name: string;
  server_host: string;
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
  inactive_fd_count?: number;
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
  start_script?: string;
  start_filename?: string;
  monitor_port: number;
  pipeline_count?: number;
  source_count?: number;
  deployed?: boolean;
  deploy_message?: string;
}

export interface LoggieBootstrapPayload {
  server_id?: number;
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
  downloadText(bundle.pipeline_yaml, bundle.pipeline_filename || "pipeline.yml", "application/x-yaml;charset=utf-8");
  if (bundle.pipelines_only_yaml) {
    downloadText(
      bundle.pipelines_only_yaml,
      bundle.pipelines_filename || "pipelines.yml",
      "application/x-yaml;charset=utf-8",
    );
  }
  if (bundle.env_file) {
    downloadText(bundle.env_file, bundle.env_filename || "loggie-heartbeat.env");
  }
  if (bundle.heartbeat_script) {
    downloadText(bundle.heartbeat_script, bundle.heartbeat_filename || "heartbeat.sh", "text/x-sh;charset=utf-8");
  }
  if (bundle.start_script) {
    downloadText(bundle.start_script, bundle.start_filename || "start.sh", "text/x-sh;charset=utf-8");
  }
}

export async function downloadLoggieFile(
  projectId: number,
  serverId: number,
  file: "pipeline" | "pipelines" | "env" | "heartbeat" | "start" = "pipeline",
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
    start: "start.sh",
  };
  downloadText(await blob.text(), names[file] ?? "pipeline.yml");
}

export async function getESConfigPreview() {
  return await getData(http.get<any, ApiResponse<ESConfigPreview>>("/log-platform/es-config"));
}

export async function setLogPlatformESConnection(connectionId: number) {
  return await getData(
    http.put<any, ApiResponse<ESConfigPreview>>("/log-platform/es-connection", {
      connection_id: connectionId,
    }),
  );
}

export async function getLoggieStatus(projectId: number) {
  return await getData(
    http.get<any, ApiResponse<{ list: LoggieStatusItem[] }>>(`/projects/${projectId}/loggie/status`),
  );
}

export async function getLoggieBootstrapSources(projectId: number, serverId: number) {
  return await getData(
    http.get<any, ApiResponse<{ list: LoggieBootstrapSourcePreview[] }>>(
      `/projects/${projectId}/loggie/bootstrap-sources`,
      {
        params: { server_id: serverId },
      },
    ),
  );
}

export async function bootstrapLoggie(projectId: number, payload: LoggieBootstrapPayload) {
  return await getData(
    http.post<any, ApiResponse<LoggieBootstrapResult>>(`/projects/${projectId}/loggie/bootstrap`, payload),
  );
}

export async function deployLoggieConfig(projectId: number, payload: LoggieDeployPayload) {
  return await getData(
    http.post<any, ApiResponse<LoggieDeployResult>>(`/projects/${projectId}/loggie/deploy`, payload),
  );
}

export async function installLoggie(
  projectId: number,
  payload: LoggieDeployPayload & { deploy_dir?: string; yunshu_url?: string; monitor_port?: number },
) {
  return await getData(
    http.post<any, ApiResponse<LoggieDeployResult>>(`/projects/${projectId}/loggie/install`, payload, {
      timeout: 300000,
    }),
  );
}

export async function uninstallLoggie(
  projectId: number,
  payload: LoggieDeployPayload & { skip_remote?: boolean; keep_files?: boolean; force_local?: boolean },
) {
  return await getData(
    http.post<any, ApiResponse<LoggieDeployResult>>(`/projects/${projectId}/loggie/uninstall`, payload, {
      timeout: 120000,
    }),
  );
}

export async function startLoggie(projectId: number, payload: LoggieDeployPayload) {
  return await getData(http.post<any, ApiResponse<LoggieDeployResult>>(`/projects/${projectId}/loggie/start`, payload));
}

export async function stopLoggie(projectId: number, payload: LoggieDeployPayload) {
  return await getData(http.post<any, ApiResponse<LoggieDeployResult>>(`/projects/${projectId}/loggie/stop`, payload));
}

export async function restartLoggie(projectId: number, payload: LoggieDeployPayload) {
  return await getData(
    http.post<any, ApiResponse<LoggieDeployResult>>(`/projects/${projectId}/loggie/restart`, payload),
  );
}

export async function syncLoggieFromLogSources(projectId: number, payload: LoggieDeployPayload) {
  return await getData(http.post<any, ApiResponse<LoggieDeployResult>>(`/projects/${projectId}/loggie/sync`, payload));
}

export interface ClusterLogRule {
  id: number;
  project_id: number;
  cluster_id: number;
  name: string;
  match_namespaces?: string;
  match_workloads?: string;
  exclude_namespaces?: string;
  parse_profile?: string;
  rate_limit_qps?: number;
  allocated_qps?: number;
  enabled: boolean;
  remark?: string;
}

export interface ClusterLogAgent {
  id: number;
  project_id: number;
  cluster_id: number;
  namespace: string;
  status: string;
  deploy_revision?: number;
  desired_replicas?: number;
  ready_replicas?: number;
  rate_limit_qps?: number;
  last_error?: string;
  last_sync_at?: string;
}

export function listClusterLogRules(projectId: number, clusterId?: number) {
  return getData<{ list: ClusterLogRule[] }>(
    http.get(`/projects/${projectId}/cluster-log/rules`, {
      params: clusterId ? { cluster_id: clusterId } : undefined,
    }),
  );
}

export function createClusterLogRule(
  projectId: number,
  payload: {
    cluster_id: number;
    name: string;
    match_namespaces?: string[];
    match_workloads?: string[];
    exclude_namespaces?: string[];
    parse_profile?: string;
    rate_limit_qps?: number;
    enabled?: boolean;
    remark?: string;
  },
) {
  return getData<ClusterLogRule>(http.post(`/projects/${projectId}/cluster-log/rules`, payload));
}

export function updateClusterLogRule(projectId: number, ruleId: number, payload: Record<string, unknown>) {
  return getData<ClusterLogRule>(http.put(`/projects/${projectId}/cluster-log/rules/${ruleId}`, payload));
}

export function deleteClusterLogRule(projectId: number, ruleId: number) {
  return getData<{ ok: boolean }>(http.delete(`/projects/${projectId}/cluster-log/rules/${ruleId}`));
}

export function listClusterLogAgents(projectId: number) {
  return getData<{ list: ClusterLogAgent[] }>(http.get(`/projects/${projectId}/cluster-log/agents`));
}

export function deployClusterLog(
  projectId: number,
  payload: { cluster_id: number; namespace?: string; rate_limit_qps?: number },
) {
  return getData<ClusterLogAgent>(http.post(`/projects/${projectId}/cluster-log/deploy`, payload));
}

export function previewClusterLogPipelines(projectId: number, clusterId: number) {
  return getData<{ pipelines_yml: string; generated_yml: string; is_custom: boolean }>(
    http.get(`/projects/${projectId}/cluster-log/pipelines/preview`, { params: { cluster_id: clusterId } }),
  );
}

export function saveClusterLogPipelines(
  projectId: number,
  payload: {
    cluster_id: number;
    pipelines_yml?: string;
    reset?: boolean;
    apply?: boolean;
    namespace?: string;
    rate_limit_qps?: number;
  },
) {
  return getData<{ pipelines_yml: string; generated_yml: string; is_custom: boolean }>(
    http.put(`/projects/${projectId}/cluster-log/pipelines`, payload),
  );
}

export function refreshClusterLogStatus(projectId: number, clusterId: number) {
  return getData<ClusterLogAgent>(
    http.get(`/projects/${projectId}/cluster-log/status`, { params: { cluster_id: clusterId } }),
  );
}

export { getProjects };
