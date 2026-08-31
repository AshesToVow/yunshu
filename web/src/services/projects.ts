import type { ApiResponse, PageData } from "../types/api";
import { getData, http } from "./http";

export const PROJECT_TYPE_OPTIONS = [
  { label: "业务项目", value: "business" },
  { label: "平台项目", value: "platform" },
  { label: "基础设施", value: "infra" },
  { label: "研发试验", value: "research" },
] as const;

export const PROJECT_LIFECYCLE_OPTIONS = [
  { label: "规划中", value: "planning" },
  { label: "进行中", value: "active" },
  { label: "已暂停", value: "suspended" },
  { label: "已归档", value: "archived" },
] as const;

export interface ProjectItem {
  id: number;
  name: string;
  code: string;
  description?: string | null;
  status: number;
  project_type: string;
  lifecycle_status: string;
  /** 项目 Harbor 地址，空则用全局 cicd 配置 */
  harbor_url?: string;
  /** 项目 Harbor 项目名（PROJECT_GROUP），空则用全局 */
  harbor_project?: string;
  /** Apollo Meta 地址（可逗号分隔多个），发布时注入 APOLLO_META */
  apollo_meta?: string;
  /** Apollo 环境（DEV/FAT/PRO），空则按发布 Tenv 推导 */
  apollo_env?: string;
  /** Apollo namespaces（逗号分隔） */
  apollo_namespaces?: string;
  /** 可选归属部门 */
  owner_department_id?: number | null;
  /** 当前登录用户在该项目中的成员角色（owner/admin/member/readonly）；超管列表可能为空 */
  my_project_role?: string | null;
  created_at: string;
}

export interface ProjectCreatePayload {
  name: string;
  code: string;
  description?: string;
  status: number;
  project_type?: string;
  lifecycle_status?: string;
  harbor_url?: string;
  harbor_project?: string;
  apollo_meta?: string;
  apollo_env?: string;
  apollo_namespaces?: string;
  owner_department_id?: number;
}

export interface ProjectUpdatePayload {
  name?: string;
  code?: string;
  description?: string | null;
  status?: number;
  project_type?: string;
  lifecycle_status?: string;
  harbor_url?: string;
  harbor_project?: string;
  apollo_meta?: string;
  apollo_env?: string;
  apollo_namespaces?: string;
  /** 传 0 表示清空归属部门 */
  owner_department_id?: number;
}

export async function getProjects(params: {
  keyword?: string;
  project_type?: string;
  lifecycle_status?: string;
  page?: number;
  page_size?: number;
}) {
  return await getData(http.get<any, ApiResponse<PageData<ProjectItem>>>("/projects", { params }));
}

export async function createProject(payload: ProjectCreatePayload) {
  return await getData(http.post<any, ApiResponse<ProjectItem>>("/projects", payload));
}

export async function updateProject(id: number, payload: ProjectUpdatePayload) {
  return await getData(http.put<any, ApiResponse<ProjectItem>>(`/projects/${id}`, payload));
}

export async function deleteProject(id: number) {
  return await getData(http.delete<any, ApiResponse<{ message: string }>>(`/projects/${id}`));
}

export async function archiveProject(id: number) {
  return await getData(http.post<any, ApiResponse<ProjectItem>>(`/projects/${id}/archive`, {}));
}

export async function restoreProject(id: number) {
  return await getData(http.post<any, ApiResponse<ProjectItem>>(`/projects/${id}/restore`, {}));
}

/** 项目成员（project_members），与监控规则 project_id、告警通知收件人联动 */
export interface ProjectMemberItem {
  id: number;
  user_id: number;
  username: string;
  nickname: string;
  email?: string | null;
  /** owner / admin / member / readonly，与后端 project_members.role 一致 */
  role?: string;
  created_at: string;
}

export async function listProjectMembers(projectId: number) {
  return await getData<{ list: ProjectMemberItem[] }>(http.get(`/projects/${projectId}/members`));
}

export async function addProjectMember(projectId: number, payload: { user_id: number; role?: string }) {
  return await getData<ProjectMemberItem>(http.post(`/projects/${projectId}/members`, payload));
}

export async function updateProjectMember(projectId: number, memberId: number, payload: { role: string }) {
  return await getData<ProjectMemberItem>(http.put(`/projects/${projectId}/members/${memberId}`, payload));
}

export async function removeProjectMember(projectId: number, memberId: number) {
  return await getData<{ message: string }>(http.delete(`/projects/${projectId}/members/${memberId}`));
}

export interface ServerItem {
  id: number;
  project_id: number;
  group_id?: number | null;
  name: string;
  host: string;
  port: number;
  os_type: string;
  os_arch: string;
  tags: string;
  source_type: string;
  provider: string;
  cloud_instance_id: string;
  cloud_region: string;
  cloud_zone: string;
  cloud_spec: string;
  cloud_config_info: string;
  cloud_os_name: string;
  cloud_network_info: string;
  cloud_charge_type: string;
  cloud_network_charge_type: string;
  cloud_tags_json: string;
  cloud_public_ip: string;
  cloud_private_ip: string;
  cloud_status_text: string;
  status: number;
  created_at: string;
  last_seen_at?: string | null;
  last_test_at?: string | null;
  last_test_error?: string | null;
}

export interface ServerUpsertPayload {
  id?: number;
  project_id: number;
  group_id?: number;
  name: string;
  host: string;
  port?: number;
  os_type?: string;
  tags?: string;
  status: number;
  source_type?: string;
  provider?: string;
  cloud_instance_id?: string;
  cloud_region?: string;
  cloud_zone?: string;
  cloud_spec?: string;
  cloud_config_info?: string;
  cloud_os_name?: string;
  cloud_network_info?: string;
  cloud_charge_type?: string;
  cloud_network_charge_type?: string;
  cloud_tags_json?: string;
  cloud_public_ip?: string;
  cloud_private_ip?: string;
  cloud_status_text?: string;

  auth_type?: "password" | "key";
  username?: string;
  password?: string;
  private_key?: string;
  passphrase?: string;
  /** 数据字典模板标签，便于编辑回显；与 username 可同时存在 */
  username_dict_label?: string;
  password_dict_label?: string;
  private_key_dict_label?: string;
}

/** GET 单台服务器详情（含 SSH 凭据元数据，不含密钥明文） */
export interface ServerDetailItem extends ServerItem {
  auth_type?: string;
  username?: string;
  password_set?: boolean;
  private_key_set?: boolean;
  username_dict_label?: string | null;
  password_dict_label?: string | null;
  private_key_dict_label?: string | null;
}

export async function getProjectServers(
  projectId: number,
  params: {
    keyword?: string;
    page?: number;
    page_size?: number;
    group_id?: number;
    source_type?: string;
    provider?: string;
    cloud_account_id?: number;
  },
) {
  return await getData(
    http.get<any, ApiResponse<PageData<ServerItem>>>(`/projects/${projectId}/servers`, {
      params: { ...params, project_id: projectId },
    }),
  );
}

export async function upsertProjectServer(projectId: number, payload: ServerUpsertPayload) {
  return await getData(http.post<any, ApiResponse<ServerItem>>(`/projects/${projectId}/servers`, payload));
}

export async function deleteProjectServer(projectId: number, serverId: number) {
  return await getData(
    http.delete<any, ApiResponse<{ message: string }>>(`/projects/${projectId}/servers/${serverId}`),
  );
}

export async function getProjectServerDetail(
  projectId: number,
  serverId: number,
  opts?: { silentErrorToast?: boolean },
) {
  return await getData<ServerDetailItem>(
    http.get(`/projects/${projectId}/servers/${serverId}`, opts?.silentErrorToast ? { silentErrorToast: true } : {}),
  );
}

export interface ServerExecPayload {
  command: string;
  timeout_sec?: number;
}

export interface ServerExecResult {
  server_id: number;
  command: string;
  stdout: string;
  stderr: string;
  exit_code: number;
  duration_ms: number;
  truncated: boolean;
}

export async function execProjectServerCommand(projectId: number, serverId: number, payload: ServerExecPayload) {
  return await getData<ServerExecResult>(http.post(`/projects/${projectId}/servers/${serverId}/exec`, payload));
}

export interface ServerRemoteFileItem {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  mode?: string;
  mod_time?: string;
}

export async function listProjectServerFiles(projectId: number, serverId: number, path = "/") {
  return await getData<{ list: ServerRemoteFileItem[]; path?: string; max_transfer_mb?: number }>(
    http.get(`/projects/${projectId}/servers/${serverId}/files`, { params: { path } }),
  );
}

export async function uploadProjectServerFile(projectId: number, serverId: number, path: string, file: File) {
  const form = new FormData();
  form.append("path", path || "/");
  form.append("file", file);
  return await getData<{ message: string; max_transfer_mb?: number }>(
    http.post(`/projects/${projectId}/servers/${serverId}/files/upload`, form, {
      headers: { "Content-Type": "multipart/form-data" },
      timeout: 600000,
    }),
  );
}

export async function downloadProjectServerFile(projectId: number, serverId: number, path: string) {
  return (await http.get(`/projects/${projectId}/servers/${serverId}/files/download`, {
    params: { path },
    responseType: "blob",
    timeout: 600000,
  })) as unknown as Blob;
}

export async function deleteProjectServerFile(projectId: number, serverId: number, path: string) {
  return await getData<{ message: string }>(
    http.post(`/projects/${projectId}/servers/${serverId}/files/delete`, { path }),
  );
}

export async function testProjectServer(projectId: number, serverId: number) {
  return await getData(
    http.post<any, ApiResponse<{ ok: boolean; message: string }>>(`/projects/${projectId}/servers/test`, {
      server_id: serverId,
    }),
  );
}

export interface CloudServerActionPayload {
  action: "reset_password" | "reboot" | "shutdown";
  new_password?: string;
}

export interface CloudServerActionResult {
  server_id: number;
  action: string;
  message: string;
}

export async function runProjectCloudServerAction(
  projectId: number,
  serverId: number,
  payload: CloudServerActionPayload,
) {
  return await getData<CloudServerActionResult>(
    http.post(`/projects/${projectId}/servers/${serverId}/cloud-actions`, payload),
  );
}

export interface BatchServerTestResult {
  server_id: number;
  ok: boolean;
  message: string;
}

export interface BatchServerTestResponse {
  total: number;
  success: number;
  failed: number;
  results: BatchServerTestResult[];
}

export async function batchTestProjectServers(projectId: number, serverIds: number[], parallel = 5) {
  return await getData(
    http.post<any, ApiResponse<BatchServerTestResponse>>(`/projects/${projectId}/servers/test/batch`, {
      project_id: projectId,
      server_ids: serverIds,
      parallel,
    }),
  );
}

export interface ServerSyncResult {
  total: number;
  online: number;
  offline: number;
  updated_at: string;
  test_results: BatchServerTestResult[];
}

export async function syncProjectServers(projectId: number, serverIds?: number[], parallel = 8) {
  return await getData<ServerSyncResult>(
    http.post(`/projects/${projectId}/servers/sync`, {
      project_id: projectId,
      server_ids: serverIds?.length ? serverIds : undefined,
      parallel,
    }),
  );
}

export async function exportProjectServers(projectId: number, params?: { keyword?: string }): Promise<Blob> {
  return (await http.get(`/projects/${projectId}/servers/export`, { params, responseType: "blob" })) as unknown as Blob;
}

export interface ServerImportRowError {
  row: number;
  name: string;
  host: string;
  message: string;
}

export interface ServerImportResult {
  imported: number;
  skipped: number;
  errors?: ServerImportRowError[];
}

export async function importProjectServers(projectId: number, file: File) {
  const form = new FormData();
  form.append("file", file);
  return await getData<ServerImportResult>(
    http.post(`/projects/${projectId}/servers/import`, form, { headers: { "Content-Type": "multipart/form-data" } }),
  );
}

export async function downloadProjectServersImportTemplate(projectId: number): Promise<Blob> {
  return (await http.get(`/projects/${projectId}/servers/import-template`, {
    responseType: "blob",
  })) as unknown as Blob;
}

export interface ServerGroupItem {
  id: number;
  project_id: number;
  parent_id?: number | null;
  name: string;
  category: "self_hosted" | "cloud" | string;
  provider: string;
  sort: number;
  status: number;
  children?: ServerGroupItem[];
}

export interface ServerGroupUpsertPayload {
  id?: number;
  project_id?: number;
  parent_id?: number | null;
  name: string;
  category: "self_hosted" | "cloud" | string;
  provider?: string;
  sort?: number;
  status?: number;
}

export async function getProjectServerGroupTree(projectId: number) {
  return await getData<ServerGroupItem[]>(
    http.get(`/projects/${projectId}/server-groups/tree`, { params: { project_id: projectId } }),
  );
}

export async function upsertProjectServerGroup(projectId: number, payload: ServerGroupUpsertPayload) {
  return await getData<ServerGroupItem>(
    http.post(`/projects/${projectId}/server-groups`, { ...payload, project_id: projectId }),
  );
}

export async function deleteProjectServerGroup(projectId: number, groupId: number) {
  return await getData<{ message: string }>(http.delete(`/projects/${projectId}/server-groups/${groupId}`));
}

export interface CloudAccountItem {
  id: number;
  project_id: number;
  group_id: number;
  provider: string;
  account_name: string;
  region_scope: string;
  ak_dict_label?: string | null;
  sk_dict_label?: string | null;
  status: number;
  last_sync_at?: string | null;
  last_sync_error?: string | null;
  created_at: string;
}

export interface CloudAccountUpsertPayload {
  id?: number;
  group_id: number;
  provider: string;
  account_name: string;
  region_scope?: string;
  ak?: string;
  sk?: string;
  ak_dict_label?: string;
  sk_dict_label?: string;
  status?: number;
}

export async function getProjectCloudAccounts(projectId: number, groupId?: number) {
  return await getData<CloudAccountItem[]>(
    http.get(`/projects/${projectId}/cloud-accounts`, { params: { project_id: projectId, group_id: groupId } }),
  );
}

export async function upsertProjectCloudAccount(projectId: number, payload: CloudAccountUpsertPayload) {
  return await getData<CloudAccountItem>(http.post(`/projects/${projectId}/cloud-accounts`, payload));
}

export interface CloudSyncResult {
  total: number;
  added: number;
  updated: number;
  disabled: number;
  unchanged: number;
  message: string;
}

export async function syncProjectCloudAccount(projectId: number, accountId: number) {
  return await getData<CloudSyncResult>(http.put(`/projects/${projectId}/cloud-accounts/${accountId}/sync`, {}));
}

export async function deleteProjectCloudAccount(projectId: number, accountId: number) {
  return await getData(
    http.delete<any, ApiResponse<{ message: string }>>(`/projects/${projectId}/cloud-accounts/${accountId}`),
  );
}

export interface ServiceItem {
  id: number;
  server_id: number;
  name: string;
  env?: string | null;
  labels?: string | null;
  remark?: string | null;
  status: number;
  created_at: string;
}

export async function getProjectServices(
  projectId: number,
  params: { server_id?: number; keyword?: string; page?: number; page_size?: number },
) {
  return await getData(
    http.get<any, ApiResponse<PageData<ServiceItem>>>(`/projects/${projectId}/services`, {
      params: { ...params, project_id: projectId },
    }),
  );
}

export async function upsertProjectService(
  projectId: number,
  payload: {
    id?: number;
    server_id: number;
    name: string;
    env?: string;
    labels?: string;
    remark?: string;
    status: number;
  },
) {
  return await getData(http.post<any, ApiResponse<ServiceItem>>(`/projects/${projectId}/services`, payload));
}

export async function deleteProjectService(projectId: number, serviceId: number) {
  return await getData(
    http.delete<any, ApiResponse<{ message: string }>>(`/projects/${projectId}/services/${serviceId}`),
  );
}

export interface LogSourceItem {
  id: number;
  service_id: number;
  log_type: "file" | "journal" | string;
  path: string;
  encoding?: string | null;
  timezone?: string | null;
  multiline_rule?: string | null;
  include_regex?: string | null;
  exclude_regex?: string | null;
  status: number;
  created_at: string;
}

export async function getProjectLogSources(
  projectId: number,
  params: { service_id?: number; page?: number; page_size?: number },
) {
  return await getData(
    http.get<any, ApiResponse<PageData<LogSourceItem>>>(`/projects/${projectId}/log-sources`, {
      params: { ...params, project_id: projectId },
    }),
  );
}

export async function upsertProjectLogSource(
  projectId: number,
  payload: {
    id?: number;
    service_id: number;
    log_type?: string;
    path: string;
    encoding?: string;
    timezone?: string;
    multiline_rule?: string;
    include_regex?: string;
    exclude_regex?: string;
    status: number;
  },
) {
  return await getData(http.post<any, ApiResponse<LogSourceItem>>(`/projects/${projectId}/log-sources`, payload));
}

export async function deleteProjectLogSource(projectId: number, logSourceId: number) {
  return await getData(
    http.delete<any, ApiResponse<{ message: string }>>(`/projects/${projectId}/log-sources/${logSourceId}`),
  );
}

export interface LogSearchItem {
  timestamp: string;
  message: string;
  highlight?: string;
  level?: string;
  file_path?: string;
  server_id?: number;
  service_id?: number;
  log_source_id?: number;
  service_name?: string;
  server_host?: string;
  host?: string;
  collector_mode?: string;
  cluster_id?: number;
  namespace?: string;
  pod?: string;
  podname?: string;
  container?: string;
  containername?: string;
}

export async function searchProjectLogs(
  projectId: number,
  params: {
    server_id?: number;
    service_id?: number;
    log_source_id?: number;
    collector_mode?: string;
    cluster_id?: number;
    namespace?: string;
    pod?: string;
    container?: string;
    keyword?: string;
    level?: string;
    file_path?: string;
    from?: string;
    to?: string;
    page?: number;
    page_size?: number;
  },
) {
  return await getData(
    http.get<any, ApiResponse<PageData<LogSearchItem>>>(`/projects/${projectId}/logs/search`, {
      params: { ...params, project_id: projectId },
    }),
  );
}

export async function exportProjectLogs(
  projectId: number,
  params: {
    server_id?: number;
    service_id?: number;
    log_source_id?: number;
    collector_mode?: string;
    cluster_id?: number;
    namespace?: string;
    pod?: string;
    container?: string;
    keyword?: string;
    level?: string;
    file_path?: string;
    from?: string;
    to?: string;
    page_size?: number;
  },
): Promise<Blob> {
  // http 拦截器已经返回 response.data（Blob），不可再取 .data
  return (await http.get(`/projects/${projectId}/logs/export`, {
    params: { ...params, project_id: projectId },
    responseType: "blob",
  })) as unknown as Blob;
}
