import type { ApiResponse, PageData } from "../types/api";
import { getData, http } from "./http";

export interface CicdServiceItem {
  id: number;
  project_id: number;
  identifier: string;
  name: string;
  service_type: string;
  owner?: string;
  product_line?: string;
  remark?: string;
  status: number;
  jenkins_job: string;
  has_ci_config: boolean;
  deploy_config_count: number;
  last_build_result?: string;
  last_build_at?: string;
}

export interface CicdCiConfig {
  id: number;
  service_id: number;
  git_url: string;
  ref_type: string;
  ref_name: string;
  build_type: string;
  build_shell?: string;
  build_path?: string;
  project_name?: string;
  version?: string;
  node_version?: string;
  npm_install_mode?: string;
  clean_npm_cache: boolean;
  clean_node_modules: boolean;
  java_tool_name?: string;
  server_port?: string;
  pack_config_paths?: string;
  description?: string;
}

export interface CicdDeployConfig {
  id: number;
  service_id: number;
  name: string;
  deploy_kind: string;
  tenv: string;
  audit_enabled: boolean;
  importance?: string;
  dest_path?: string;
  server_ids_json?: string;
  deploy_user?: string;
  deploy_group?: string;
  artifact_retain_count: number;
  run_user?: string;
  start_script_type?: string;
  custom_script_content?: string;
  clean_deploy_dir: boolean;
  jvm_opts?: string;
  server_port: number;
  deploy_method?: string;
  deploy_action?: string;
  deploy_config_type?: string;
  deploy_config_template?: string;
  k8s_namespace?: string;
  k8s_cluster_id?: number;
  image_name?: string;
  image_tag?: string;
  replicas: number;
  container_port: number;
  status: number;
  server_count?: number;
  nodes_status?: string;
}

export interface CicdBuildRun {
  id: number;
  project_id: number;
  service_id: number;
  build_number: number;
  branch_name?: string;
  publish_mode: string;
  tenv?: string;
  build_result: string;
  builder_name?: string;
  version?: string;
  package_path?: string;
  image_address?: string;
  download_url?: string;
  jenkins_build_url?: string;
  params_json?: string;
  started_at?: string;
  finished_at?: string;
  service_name?: string;
  service_identifier?: string;
}

export interface CicdReleaseRun {
  id: number;
  project_id: number;
  service_id: number;
  deploy_config_id?: number;
  title: string;
  release_kind: string;
  release_type?: string;
  tenv?: string;
  status: string;
  current_stage_key?: string;
  current_stage_name?: string;
  submitter_name?: string;
  image_address?: string;
  artifact_name?: string;
  audit_enabled?: boolean;
  reviewer_name?: string;
  review_comment?: string;
  reviewed_at?: string;
  jenkins_build_number?: number;
  jenkins_build_url?: string;
  params_json?: string;
  started_at?: string;
  finished_at?: string;
  service_name?: string;
  service_identifier?: string;
  project_name?: string;
}

function projectPath(projectId: number, suffix: string) {
  return `/projects/${projectId}/cicd${suffix}`;
}

export async function listCicdServices(projectId: number, params?: Record<string, unknown>) {
  return getData<PageData<CicdServiceItem>>(
    http.get(projectPath(projectId, "/services"), { params }) as Promise<ApiResponse<PageData<CicdServiceItem>>>,
  );
}

export async function createCicdService(projectId: number, payload: Record<string, unknown>) {
  return getData<CicdServiceItem>(
    http.post(projectPath(projectId, "/services"), payload) as Promise<ApiResponse<CicdServiceItem>>,
  );
}

export async function updateCicdService(projectId: number, serviceId: number, payload: Record<string, unknown>) {
  return getData<CicdServiceItem>(
    http.put(`${projectPath(projectId, "/services")}/${serviceId}`, payload) as Promise<ApiResponse<CicdServiceItem>>,
  );
}

export async function deleteCicdService(projectId: number, serviceId: number) {
  return getData<{ deleted: boolean }>(
    http.delete(`${projectPath(projectId, "/services")}/${serviceId}`) as Promise<ApiResponse<{ deleted: boolean }>>,
  );
}

export interface CicdCiConfigView {
  configured: boolean;
  config?: CicdCiConfig | null;
}

export async function getCiConfig(projectId: number, serviceId: number) {
  return getData<CicdCiConfigView>(
    http.get(`${projectPath(projectId, "/services")}/${serviceId}/ci-config`) as Promise<ApiResponse<CicdCiConfigView>>,
  );
}

export interface CicdCiConfigUpsertResult {
  config: CicdCiConfig;
  jenkins_sync?: { job_name: string; script_path: string; created: boolean; updated: boolean };
  jenkins_sync_error?: string;
}

export async function upsertCiConfig(projectId: number, serviceId: number, payload: Record<string, unknown>) {
  return getData<CicdCiConfigUpsertResult>(
    http.put(`${projectPath(projectId, "/services")}/${serviceId}/ci-config`, payload) as Promise<
      ApiResponse<CicdCiConfigUpsertResult>
    >,
  );
}

export async function listDeployConfigs(projectId: number, serviceId: number) {
  return getData<CicdDeployConfig[]>(
    http.get(`${projectPath(projectId, "/services")}/${serviceId}/deploy-configs`) as Promise<ApiResponse<CicdDeployConfig[]>>,
  );
}

export interface CicdArtifactItem {
  name: string;
  size: number;
  last_modified?: string;
  object_key: string;
}

export async function listCicdArtifacts(projectId: number, serviceId: number) {
  return getData<CicdArtifactItem[]>(
    http.get(`${projectPath(projectId, "/services")}/${serviceId}/artifacts`) as Promise<ApiResponse<CicdArtifactItem[]>>,
  );
}

export async function createDeployConfig(projectId: number, serviceId: number, payload: Record<string, unknown>) {
  return getData<CicdDeployConfig>(
    http.post(`${projectPath(projectId, "/services")}/${serviceId}/deploy-configs`, payload) as Promise<
      ApiResponse<CicdDeployConfig>
    >,
  );
}

export async function updateDeployConfig(
  projectId: number,
  serviceId: number,
  configId: number,
  payload: Record<string, unknown>,
) {
  return getData<CicdDeployConfig>(
    http.put(`${projectPath(projectId, "/services")}/${serviceId}/deploy-configs/${configId}`, payload) as Promise<
      ApiResponse<CicdDeployConfig>
    >,
  );
}

export async function deleteDeployConfig(projectId: number, serviceId: number, configId: number) {
  return getData<{ deleted: boolean }>(
    http.delete(`${projectPath(projectId, "/services")}/${serviceId}/deploy-configs/${configId}`) as Promise<
      ApiResponse<{ deleted: boolean }>
    >,
  );
}

export async function triggerBuild(projectId: number, serviceId: number, payload: Record<string, unknown>) {
  return getData<CicdBuildRun>(
    http.post(`${projectPath(projectId, "/services")}/${serviceId}/builds`, payload) as Promise<ApiResponse<CicdBuildRun>>,
  );
}

export async function triggerRelease(projectId: number, serviceId: number, payload: Record<string, unknown>) {
  return getData<CicdReleaseRun>(
    http.post(`${projectPath(projectId, "/services")}/${serviceId}/releases`, payload) as Promise<
      ApiResponse<CicdReleaseRun>
    >,
  );
}

export async function listBuildRuns(projectId: number, params?: Record<string, unknown>) {
  return getData<PageData<CicdBuildRun>>(
    http.get(projectPath(projectId, "/build-runs"), { params }) as Promise<ApiResponse<PageData<CicdBuildRun>>>,
  );
}

export async function getBuildRun(projectId: number, runId: number) {
  return getData<CicdBuildRun>(
    http.get(`${projectPath(projectId, "/build-runs")}/${runId}`) as Promise<ApiResponse<CicdBuildRun>>,
  );
}

export async function getBuildRunLog(projectId: number, runId: number) {
  return getData<{ log: string }>(
    http.get(`${projectPath(projectId, "/build-runs")}/${runId}/log`) as Promise<ApiResponse<{ log: string }>>,
  );
}

export async function deleteBuildRun(projectId: number, runId: number) {
  return getData<{ deleted: boolean }>(
    http.delete(`${projectPath(projectId, "/build-runs")}/${runId}`) as Promise<ApiResponse<{ deleted: boolean }>>,
  );
}

export async function listReleaseRuns(projectId: number, params?: Record<string, unknown>) {
  return getData<PageData<CicdReleaseRun>>(
    http.get(projectPath(projectId, "/release-runs"), { params }) as Promise<ApiResponse<PageData<CicdReleaseRun>>>,
  );
}

export async function getReleaseRunDetail(projectId: number, runId: number) {
  return getData<CicdReleaseRunDetail>(
    http.get(`${projectPath(projectId, "/release-runs")}/${runId}`) as Promise<ApiResponse<CicdReleaseRunDetail>>,
  );
}

export async function getReleaseRunLog(projectId: number, runId: number) {
  return getData<{ log: string }>(
    http.get(`${projectPath(projectId, "/release-runs")}/${runId}/log`) as Promise<ApiResponse<{ log: string }>>,
  );
}

export async function deleteReleaseRun(projectId: number, runId: number) {
  return getData<{ deleted: boolean }>(
    http.delete(`${projectPath(projectId, "/release-runs")}/${runId}`) as Promise<ApiResponse<{ deleted: boolean }>>,
  );
}

export async function approveReleaseRun(projectId: number, runId: number, comment?: string) {
  return getData<CicdReleaseRun>(
    http.post(`${projectPath(projectId, "/release-runs")}/${runId}/approve`, { comment }) as Promise<
      ApiResponse<CicdReleaseRun>
    >,
  );
}

export async function rejectReleaseRun(projectId: number, runId: number, comment?: string) {
  return getData<CicdReleaseRun>(
    http.post(`${projectPath(projectId, "/release-runs")}/${runId}/reject`, { comment }) as Promise<
      ApiResponse<CicdReleaseRun>
    >,
  );
}

export async function executeReleaseRun(projectId: number, runId: number) {
  return getData<CicdReleaseRun>(
    http.post(`${projectPath(projectId, "/release-runs")}/${runId}/execute`) as Promise<ApiResponse<CicdReleaseRun>>,
  );
}

export async function terminateReleaseRun(projectId: number, runId: number, comment?: string) {
  return getData<CicdReleaseRun>(
    http.post(`${projectPath(projectId, "/release-runs")}/${runId}/terminate`, { comment }) as Promise<
      ApiResponse<CicdReleaseRun>
    >,
  );
}

export async function batchApproveReleaseRuns(projectId: number, ids: number[], comment?: string) {
  return getData<{ count: number }>(
    http.post(`${projectPath(projectId, "/release-runs")}/batch-approve`, { ids, comment }) as Promise<
      ApiResponse<{ count: number }>
    >,
  );
}

export async function batchRejectReleaseRuns(projectId: number, ids: number[], comment?: string) {
  return getData<{ count: number }>(
    http.post(`${projectPath(projectId, "/release-runs")}/batch-reject`, { ids, comment }) as Promise<
      ApiResponse<{ count: number }>
    >,
  );
}

export async function batchExecuteReleaseRuns(projectId: number, ids: number[]) {
  return getData<{ count: number }>(
    http.post(`${projectPath(projectId, "/release-runs")}/batch-execute`, { ids }) as Promise<
      ApiResponse<{ count: number }>
    >,
  );
}

export async function batchTerminateReleaseRuns(projectId: number, ids: number[], comment?: string) {
  return getData<{ count: number }>(
    http.post(`${projectPath(projectId, "/release-runs")}/batch-terminate`, { ids, comment }) as Promise<
      ApiResponse<{ count: number }>
    >,
  );
}

export interface CicdApprovalFlowStage {
  stage_key: string;
  stage_name: string;
  sort_order: number;
  enabled: boolean;
  user_group_id?: number;
  user_group_name?: string;
}

export interface CicdApprovalFlow {
  project_id: number;
  stages: CicdApprovalFlowStage[];
}

export interface CicdReleaseApprovalStep {
  id: number;
  stage_key: string;
  stage_name: string;
  sort_order: number;
  status: string;
  user_group_id?: number;
  user_group_name?: string;
  reviewer_user_id?: number;
  reviewer_name?: string;
  review_comment?: string;
  reviewed_at?: string;
}

export interface CicdReleaseHandler {
  user_id: number;
  username: string;
  nickname: string;
}

export interface CicdReleaseOperationLog {
  action: string;
  actor_name: string;
  operated_at: string;
  message: string;
}

export interface CicdReleaseRunDetail extends CicdReleaseRun {
  approval_steps?: CicdReleaseApprovalStep[];
  approval_flow_text?: string;
  current_handlers?: CicdReleaseHandler[];
  operation_logs?: CicdReleaseOperationLog[];
  dest_hosts?: string[];
  deploy_config_name?: string;
  dest_path?: string;
}

export async function getApprovalFlow(projectId: number) {
  return getData<CicdApprovalFlow>(
    http.get(projectPath(projectId, "/approval-flow")) as Promise<ApiResponse<CicdApprovalFlow>>,
  );
}

export async function saveApprovalFlow(
  projectId: number,
  stages: { stage_key: string; enabled: boolean; user_group_id?: number }[],
) {
  return getData<CicdApprovalFlow>(
    http.put(projectPath(projectId, "/approval-flow"), { stages }) as Promise<ApiResponse<CicdApprovalFlow>>,
  );
}

export async function listReleaseApprovalSteps(projectId: number, runId: number) {
  return getData<CicdReleaseApprovalStep[]>(
    http.get(`${projectPath(projectId, "/release-runs")}/${runId}/approval-steps`) as Promise<
      ApiResponse<CicdReleaseApprovalStep[]>
    >,
  );
}
