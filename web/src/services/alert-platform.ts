import { getData, http } from "./http";
import { normalizePagedPayload, parseNumberArray, parseStringArray, parseStringMap } from "./alert-mappers";

export interface AlertDatasourceItem {
  id: number;
  project_id: number;
  project_name?: string;
  name: string;
  type: string;
  base_url: string;
  alertmanager_url?: string;
  bearer_token?: string;
  /** 编辑时后端可能返回；新建后列表接口可能脱敏 */
  basic_user?: string;
  basic_password?: string;
  skip_tls_verify: boolean;
  enabled: boolean;
  remark?: string;
  last_health_status?: string;
  last_health_at?: string | null;
  last_health_latency_ms?: number;
  last_health_error?: string;
  last_up_total?: number;
  last_up_down?: number;
  created_at: string;
  updated_at: string;
}

export interface AlertSilenceItem {
  id: number;
  name: string;
  matchers_json: string;
  matchers?: Array<{ name: string; value: string; is_regex: boolean }>;
  starts_at: string;
  ends_at: string;
  comment?: string;
  created_by: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface AlertMonitorRuleItem {
  id: number;
  datasource_id: number;
  project_id?: number;
  project_name?: string;
  datasource_name?: string;
  name: string;
  rule_kind?: string;
  origin?: string;
  origin_inspect_item_id?: number;
  expr: string;
  for_seconds: number;
  eval_interval_seconds: number;
  severity: string;
  threshold_unit?: string;
  labels_json?: string;
  annotations_json?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface AlertRuleAssigneeItem {
  id: number;
  monitor_rule_id: number;
  user_ids_json?: string;
  department_ids_json?: string;
  extra_emails_json?: string;
  user_ids?: number[];
  department_ids?: number[];
  extra_emails?: string[];
  recipient_mode?: string;
  notify_on_resolved: boolean;
  remark?: string;
}

export interface AlertDutyBlockItem {
  id: number;
  monitor_rule_id: number;
  starts_at: string;
  ends_at: string;
  title?: string;
  user_ids_json?: string;
  department_ids_json?: string;
  extra_emails_json?: string;
  user_ids?: number[];
  department_ids?: number[];
  extra_emails?: string[];
  remark?: string;
  created_at: string;
  updated_at: string;
}

export interface CloudExpiryRuleItem {
  id: number;
  project_id: number;
  project_name?: string;
  name: string;
  provider: string;
  region_scope: string;
  advance_days: number;
  severity: string;
  labels_json?: string;
  labels?: Record<string, string>;
  /** 六段含秒、五段或 @every；启用定时评估时必填 */
  eval_cron_spec?: string;
  schedule_enabled?: boolean;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export type Paged<T> = { list: T[]; total: number; page: number; page_size: number };

function parseSilenceMatchers(raw?: string): Array<{ name: string; value: string; is_regex: boolean }> {
  const s = String(raw || "").trim();
  if (!s) return [];
  try {
    const parsed = JSON.parse(s) as Array<Record<string, unknown>>;
    if (!Array.isArray(parsed)) return [];
    return parsed
      .map((item) => ({
        name: String(item?.name ?? "").trim(),
        value: String(item?.value ?? "").trim(),
        is_regex: Boolean(item?.is_regex),
      }))
      .filter((item) => item.name);
  } catch {
    return [];
  }
}

function mapMonitorRule(item: AlertMonitorRuleItem): AlertMonitorRuleItem {
  return {
    ...item,
    labels: parseStringMap(item.labels_json),
    annotations: parseStringMap(item.annotations_json),
  };
}

function mapAssignee(item: AlertRuleAssigneeItem): AlertRuleAssigneeItem {
  return {
    ...item,
    user_ids: parseNumberArray(item.user_ids_json),
    department_ids: parseNumberArray(item.department_ids_json),
    extra_emails: parseStringArray(item.extra_emails_json),
  };
}

function mapDutyBlock(item: AlertDutyBlockItem): AlertDutyBlockItem {
  return {
    ...item,
    user_ids: parseNumberArray(item.user_ids_json),
    department_ids: parseNumberArray(item.department_ids_json),
    extra_emails: parseStringArray(item.extra_emails_json),
  };
}

export function listAlertDatasources(params?: {
  project_id?: number;
  keyword?: string;
  page?: number;
  page_size?: number;
}) {
  return getData<{
    list?: AlertDatasourceItem[];
    items?: AlertDatasourceItem[];
    total: number;
    page: number;
    page_size: number;
  }>(http.get("/alerts/datasources", { params })).then((payload) => normalizePagedPayload(payload));
}

export function createAlertDatasource(payload: Record<string, unknown>) {
  return getData<AlertDatasourceItem>(http.post("/alerts/datasources", payload));
}

export function updateAlertDatasource(id: number, payload: Record<string, unknown>) {
  return getData<AlertDatasourceItem>(http.put(`/alerts/datasources/${id}`, payload));
}

export function deleteAlertDatasource(id: number) {
  return getData<void>(http.delete(`/alerts/datasources/${id}`));
}

export interface DatasourcePingResult {
  ok: boolean;
  message: string;
  latency_ms: number;
}

export interface DatasourceHealthResult {
  datasource_id: number;
  status: string;
  message: string;
  latency_ms: number;
  up_total: number;
  up_down: number;
  checked_at?: string;
  stale_seconds?: number;
}

/** Prometheus 连通性检测（会写回健康缓存） */
export function pingAlertDatasource(id: number) {
  return getData<DatasourcePingResult>(http.get(`/alerts/datasources/${id}/ping`));
}

/** 即时健康探测（连通 + up 覆盖） */
export function checkAlertDatasourceHealth(id: number) {
  return getData<DatasourceHealthResult>(http.post(`/alerts/datasources/${id}/health-check`));
}

export function getAlertDatasourceHealth(id: number) {
  return getData<DatasourceHealthResult>(http.get(`/alerts/datasources/${id}/health`));
}

export function listAlertDatasourceHealth(projectId: number) {
  return getData<{ list?: DatasourceHealthResult[]; items?: DatasourceHealthResult[] }>(
    http.get("/alerts/datasources/health", { params: { project_id: projectId } }),
  );
}

/** GET Prometheus /api/v1/alerts，返回原始 JSON（含 data.alerts）。 */
export function promActiveAlerts(id: number) {
  return getData<{ data: unknown }>(http.get(`/alerts/datasources/${id}/prometheus-alerts`)).then((r) => r.data);
}

/** @deprecated Alertmanager 已下线 */
export function alertmanagerSilences(_id: number): Promise<unknown> {
  return Promise.resolve([]);
}

export interface AlertConsulEndpointItem {
  id: number;
  project_id: number;
  name: string;
  address: string;
  token?: string;
  datacenter?: string;
  service_tag?: string;
  enabled: boolean;
  remark?: string;
  last_sync_at?: string | null;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface AlertMonitorObjectItem {
  id: number;
  endpoint_id: number;
  project_id: number;
  service_name: string;
  service_id: string;
  node?: string;
  address?: string;
  port?: number;
  tags_json?: string;
  meta_json?: string;
  exporter_role?: string;
  yunshu_project?: string;
  health?: string;
  probe_url?: string;
  synced_at?: string;
}

export function listConsulEndpoints(params?: {
  project_id?: number;
  keyword?: string;
  page?: number;
  page_size?: number;
}) {
  return getData<{
    list?: AlertConsulEndpointItem[];
    items?: AlertConsulEndpointItem[];
    total: number;
    page: number;
    page_size: number;
  }>(http.get("/alerts/consul-endpoints", { params })).then((payload) => normalizePagedPayload(payload));
}

export function createConsulEndpoint(payload: Record<string, unknown>) {
  return getData<AlertConsulEndpointItem>(http.post("/alerts/consul-endpoints", payload));
}

export function updateConsulEndpoint(id: number, payload: Record<string, unknown>) {
  return getData<AlertConsulEndpointItem>(http.put(`/alerts/consul-endpoints/${id}`, payload));
}

export function deleteConsulEndpoint(id: number) {
  return getData<{ message: string }>(http.delete(`/alerts/consul-endpoints/${id}`));
}

export function pingConsulEndpoint(id: number) {
  return getData<{ ok: boolean; message: string }>(http.get(`/alerts/consul-endpoints/${id}/ping`));
}

export function syncConsulEndpoint(id: number) {
  return getData<{ endpoint_id: number; upserted: number; removed: number; message: string }>(
    http.post(`/alerts/consul-endpoints/${id}/sync`),
  );
}

export function listMonitorObjects(params?: {
  project_id?: number;
  endpoint_id?: number;
  exporter_role?: string;
  keyword?: string;
  page?: number;
  page_size?: number;
}) {
  return getData<{
    list?: AlertMonitorObjectItem[];
    items?: AlertMonitorObjectItem[];
    total: number;
    page: number;
    page_size: number;
  }>(http.get("/alerts/monitor-objects", { params })).then((payload) => normalizePagedPayload(payload));
}

export function promInstantQuery(id: number, payload: { query: string; time?: string }) {
  return getData<{ data: unknown }>(http.post(`/alerts/datasources/${id}/query`, payload));
}

export function promRangeQuery(id: number, payload: { query: string; start: string; end: string; step: string }) {
  return getData<{ data: unknown }>(http.post(`/alerts/datasources/${id}/query_range`, payload));
}

export function listAlertSilences(params?: {
  project_id?: number;
  keyword?: string;
  page?: number;
  page_size?: number;
}) {
  return getData<{
    list?: AlertSilenceItem[];
    items?: AlertSilenceItem[];
    total: number;
    page: number;
    page_size: number;
  }>(http.get("/alerts/silences", { params })).then((payload) =>
    normalizePagedPayload(payload, (item) => ({
      ...item,
      matchers: parseSilenceMatchers(item.matchers_json),
    })),
  );
}

export function createAlertSilence(payload: Record<string, unknown>) {
  return getData<AlertSilenceItem>(http.post("/alerts/silences", payload));
}

export interface AlertSilenceBatchItem {
  name: string;
  matchers_json: string;
  starts_at: string;
  ends_at: string;
  comment?: string;
  enabled?: boolean;
}

export function createAlertSilencesBatch(items: AlertSilenceBatchItem[]) {
  return getData<{ created: number }>(http.post("/alerts/silences/batch", { items }));
}

export function updateAlertSilence(id: number, payload: Record<string, unknown>) {
  return getData<AlertSilenceItem>(http.put(`/alerts/silences/${id}`, payload));
}

export function deleteAlertSilence(id: number) {
  return getData<void>(http.delete(`/alerts/silences/${id}`));
}

export function listAlertMonitorRules(params?: {
  datasource_id?: number;
  project_id?: number;
  keyword?: string;
  enabled?: boolean;
  page?: number;
  page_size?: number;
}) {
  return getData<{
    list?: AlertMonitorRuleItem[];
    items?: AlertMonitorRuleItem[];
    total: number;
    page: number;
    page_size: number;
  }>(http.get("/alerts/monitor-rules", { params })).then((payload) => normalizePagedPayload(payload, mapMonitorRule));
}

export function createAlertMonitorRule(payload: Record<string, unknown>) {
  return getData<AlertMonitorRuleItem>(http.post("/alerts/monitor-rules", payload));
}

export function importPrometheusYAML(payload: {
  datasource_id: number;
  project_id?: number;
  yaml: string;
  enabled?: boolean;
  dry_run?: boolean;
}) {
  return getData<{
    created: number;
    skipped: number;
    preview?: Array<{ group_name: string; name: string; expr: string; for_seconds: number; severity: string }>;
    errors?: string[];
  }>(http.post("/alerts/monitor-rules/import-prometheus-yaml", payload));
}

export interface AlertRuleTemplateItem {
  id: string;
  group: string;
  name: string;
  description: string;
  expr_template: string;
  for_seconds: number;
  eval_interval_seconds: number;
  severity: string;
  threshold_unit: string;
  default_params?: Record<string, string>;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

export async function listAlertRuleTemplates(group?: string) {
  const data = await getData<{ list?: AlertRuleTemplateItem[] }>(
    http.get("/alerts/rule-templates", { params: group ? { group } : undefined }),
  );
  return data.list ?? [];
}

export function createAlertMonitorRuleFromTemplate(payload: {
  template_id: string;
  datasource_id?: number;
  project_id?: number;
  name?: string;
  severity?: string;
  params?: Record<string, string>;
  enabled?: boolean;
}) {
  return getData<AlertMonitorRuleItem>(http.post("/alerts/monitor-rules/from-template", payload));
}

export function updateAlertMonitorRule(id: number, payload: Record<string, unknown>) {
  return getData<AlertMonitorRuleItem>(http.put(`/alerts/monitor-rules/${id}`, payload));
}

export function deleteAlertMonitorRule(id: number) {
  return getData<void>(http.delete(`/alerts/monitor-rules/${id}`));
}

export function listCloudExpiryRules(params?: {
  project_id?: number;
  provider?: string;
  keyword?: string;
  page?: number;
  page_size?: number;
}) {
  return getData<{
    list?: CloudExpiryRuleItem[];
    items?: CloudExpiryRuleItem[];
    total: number;
    page: number;
    page_size: number;
  }>(http.get("/alerts/cloud-expiry-rules", { params })).then((payload) =>
    normalizePagedPayload(payload, (item) => ({
      ...item,
      labels: parseStringMap(item.labels_json),
    })),
  );
}

export function createCloudExpiryRule(payload: Record<string, unknown>) {
  return getData<CloudExpiryRuleItem>(http.post("/alerts/cloud-expiry-rules", payload));
}

export function updateCloudExpiryRule(id: number, payload: Record<string, unknown>) {
  return getData<CloudExpiryRuleItem>(http.put(`/alerts/cloud-expiry-rules/${id}`, payload));
}

export function deleteCloudExpiryRule(id: number) {
  return getData<void>(http.delete(`/alerts/cloud-expiry-rules/${id}`));
}

export function evaluateCloudExpiryRulesNow() {
  return getData<{ message: string }>(http.post("/alerts/cloud-expiry-rules/evaluate-now", {}));
}

export function getMonitorRuleAssignees(ruleId: number) {
  return getData<{ list?: AlertRuleAssigneeItem[]; items?: AlertRuleAssigneeItem[] }>(
    http.get(`/alerts/monitor-rules/${ruleId}/assignees`),
  ).then((payload) => {
    const source = Array.isArray(payload.items) ? payload.items : Array.isArray(payload.list) ? payload.list : [];
    const items = source.map(mapAssignee);
    return { items, list: items };
  });
}

export function upsertMonitorRuleAssignees(ruleId: number, payload: Record<string, unknown>) {
  return getData<AlertRuleAssigneeItem>(http.put(`/alerts/monitor-rules/${ruleId}/assignees`, payload));
}

export function listDutyBlocks(params?: {
  monitor_rule_id?: number;
  project_id?: number;
  page?: number;
  page_size?: number;
}) {
  return getData<{
    list?: AlertDutyBlockItem[];
    items?: AlertDutyBlockItem[];
    total: number;
    page: number;
    page_size: number;
  }>(http.get("/alerts/duty-blocks", { params })).then((payload) => normalizePagedPayload(payload, mapDutyBlock));
}

export function createDutyBlock(payload: Record<string, unknown>) {
  return getData<AlertDutyBlockItem>(http.post("/alerts/duty-blocks", payload));
}

export function updateDutyBlock(id: number, payload: Record<string, unknown>) {
  return getData<AlertDutyBlockItem>(http.put(`/alerts/duty-blocks/${id}`, payload));
}

export function deleteDutyBlock(id: number) {
  return getData<{ message: string }>(http.delete(`/alerts/duty-blocks/${id}`));
}
