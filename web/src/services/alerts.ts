import { getData, http } from "./http";
import { normalizePagedPayload, parseCommaSeparatedList, parseCommaSeparatedNumbers } from "./alert-mappers";

export interface AlertChannelItem {
  id: number;
  name: string;
  type: string;
  url: string;
  secret?: string;
  headers_json?: string;
  enabled: boolean;
  timeout_ms: number;
  created_at: string;
  updated_at: string;
}

export interface AlertEventItem {
  id: number;
  source: string;
  title: string;
  severity: string;
  status: string;
  cluster?: string;
  alertIP?: string;
  alertStartedAt?: string;
  /** 路由 slug：ds:<id>、alertmanager、platform_monitor、cloud_expiry；历史可能为 prometheus/platform */
  monitorPipeline?: string;
  datasourceId?: number;
  datasourceName?: string;
  datasourceType?: string;
  groupKey?: string;
  labelsDigest?: string;
  fingerprint?: string;
  matchedPolicyIds?: string;
  matchedPolicyNames?: string;
  matchedPolicyIdList?: number[];
  matchedPolicyNameList?: string[];
  receiverList?: string[];
  channelId: number;
  channelName: string;
  success: boolean;
  httpStatusCode: number;
  errorMessage?: string;
  requestPayload?: string;
  responsePayload?: string;
  createdAt: string;
  /** 通知载荷中的 current（触发/最近一次快照） */
  metricCurrent?: string;
  /** 仅 resolved 且成功二次查询 Prom 时有值 */
  metricResolved?: string;
}

export function listAlertChannels(params?: { keyword?: string }) {
  return getData<{ list: AlertChannelItem[] }>(http.get("/alerts/channels", { params }));
}

export function createAlertChannel(payload: {
  name: string;
  type?: string;
  url: string;
  secret?: string;
  headers_json?: string;
  enabled?: boolean;
  timeout_ms?: number;
}) {
  return getData<AlertChannelItem>(http.post("/alerts/channels", payload));
}

export function updateAlertChannel(
  id: number,
  payload: {
    name: string;
    type?: string;
    url: string;
    secret?: string;
    headers_json?: string;
    enabled?: boolean;
    timeout_ms?: number;
  },
) {
  return getData<AlertChannelItem>(http.put(`/alerts/channels/${id}`, payload));
}

export function deleteAlertChannel(id: number) {
  return getData<void>(http.delete(`/alerts/channels/${id}`));
}

export interface AlertChannelTestResult {
  success: boolean;
  http_status_code?: number;
  response_body?: string;
  error_message?: string;
}

export function testAlertChannel(
  id: number,
  payload?: { title?: string; content?: string; severity?: string; status?: "firing" | "resolved" },
) {
  return getData<AlertChannelTestResult>(http.post(`/alerts/channels/${id}/test`, payload ?? {}));
}

export interface AlertRoutingDebugRequest {
  project_id: number;
  labels: Record<string, string>;
  severity?: string;
  status?: string;
}

export interface AlertRoutingDebugResult {
  matched: boolean;
  matched_path?: string;
  matched_node_names?: string[];
  receiver_group_ids?: number[];
  silence_seconds?: number;
  channels?: Array<{ id: number; name: string; type: string }>;
  silenced?: boolean;
  silence_id?: number;
  maintenance_suppressed?: boolean;
  maintenance_id?: number;
}

export function debugAlertRouting(payload: AlertRoutingDebugRequest) {
  return getData<AlertRoutingDebugResult>(http.post("/alerts/routing/debug", payload));
}

export interface AlertEventGroupItem {
  group_key: string;
  title: string;
  count: number;
  last_at: string;
  status: string;
  severity: string;
  cluster: string;
}

export function listAlertEventsGrouped(params: {
  page: number;
  page_size: number;
  keyword?: string;
  cluster?: string;
  projectId?: number;
}) {
  return getData<{ list?: AlertEventGroupItem[]; items?: AlertEventGroupItem[]; total: number; page: number; page_size: number }>(
    http.get("/alerts/events/grouped", { params }),
  ).then((payload) => normalizePagedPayload(payload));
}

/** 与后端 alertdispatch 模板变量说明一致（通道 Go template {{.Name}}） */
export interface AlertTemplateVariableDoc {
  name: string;
  description: string;
}

export interface AlertTemplatePreviewResult {
  rendered: string;
  sample_payload: Record<string, unknown>;
  available_fields: string[];
  raw_payload_fields: string[];
  combined_fields: string[];
  suggested_label_keys: string[];
  /** 固定模板变量及含义（WatchAlert 式「通知模板」文档化） */
  template_variables?: AlertTemplateVariableDoc[];
}

export function previewAlertChannelTemplate(payload: {
  template_firing?: string;
  template_resolved?: string;
  status?: "firing" | "resolved";
  title?: string;
  content?: string;
  severity?: string;
  project_id?: number;
  raw_payload_json?: string;
}) {
  return getData<AlertTemplatePreviewResult>(http.post("/alerts/channels/preview-template", payload));
}

export function listAlertEvents(params: {
  page: number;
  page_size: number;
  keyword?: string;
  cluster?: string;
  alertIP?: string;
  status?: string;
  monitorPipeline?: string;
  /** 与后端 datasourceId 一致，按已配置 Prometheus 数据源筛选 */
  datasourceId?: number;
  groupKey?: string;
  /** 告警指纹；兼容旧数据会同时匹配 group_key / payload */
  fingerprint?: string;
  /** 策略分类，与后端 category 一致：inhibition|silence|timing|… */
  category?: string;
  /** 告警监控平台顶栏项目筛选 */
  projectId?: number;
}) {
  return getData<{ list?: AlertEventItem[]; items?: AlertEventItem[]; total: number; page: number; page_size: number }>(
    http.get("/alerts/events", { params }),
  ).then((payload) =>
    normalizePagedPayload(payload, (item) => ({
      ...item,
      matchedPolicyIdList: parseCommaSeparatedNumbers(item.matchedPolicyIds),
      matchedPolicyNameList: parseCommaSeparatedList(item.matchedPolicyNames),
    })),
  );
}

export function getAlertHistoryStats(params?: { project_id?: number }) {
  return getData<{
    total: number;
    firing: number;
    resolved: number;
    success: number;
    failed: number;
    today_created: number;
    /** K8s / Prometheus external_labels.cluster 等（历史记录中去重） */
    cluster_values?: string[];
    /** 历史 monitor_pipeline slug 去重（含 ds:N、alertmanager 等） */
    monitor_pipeline_values?: string[];
    /** 历史事件中已出现的数据源，用于筛选下拉 */
    datasource_filter_options?: Array<{ id: number; name: string }>;
  }>(http.get("/alerts/history/stats", { params }));
}

export type FingerprintDeliveryExplain = {
  fingerprint: string;
  firing_delivered: boolean;
  firing_delivered_source?: string;
  events: Array<{
    id: number;
    created_at: string;
    status: string;
    title: string;
    channel_name: string;
    success: boolean;
    error_message: string;
    category: string;
    reason_hint: string;
    response_snippet?: string;
  }>;
  skip_summary: Array<{
    error_message: string;
    category: string;
    count: number;
    hint: string;
  }>;
};

/** 按 fingerprint 追溯投递成功/跳过原因 */
export function explainAlertByFingerprint(fingerprint: string) {
  return getData<FingerprintDeliveryExplain>(
    http.get("/alerts/events/by-fingerprint", { params: { fingerprint } }),
  );
}

export function sendAlertmanagerWebhook(payload: Record<string, unknown>, token?: string) {
  const headers: Record<string, string> = {};
  if ((token || "").trim()) {
    headers["X-Webhook-Token"] = String(token).trim();
  }
  return getData<{ message: string }>(http.post("/alerts/webhook/alertmanager", payload, { headers }));
}
