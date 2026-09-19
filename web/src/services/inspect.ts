import { getData, http } from "./http";

export interface InspectPlan {
  id: number;
  project_id: number;
  enabled: boolean;
  cron_spec: string;
  datasource_id: number;
  report_list_mode: string;
  report_template_id?: number;
  retain_days?: number;
  recipients_json?: string;
  last_run_at?: string | null;
}

export interface InspectReportTemplate {
  id: number;
  project_id: number;
  code: string;
  name: string;
  engine?: string;
  body?: string;
  is_builtin?: boolean;
  status?: number;
  remark?: string;
}

export interface InspectItem {
  id: number;
  project_id: number;
  type: string;
  name: string;
  description?: string;
  query: string;
  threshold: number;
  threshold_type: string;
  unit?: string;
  enabled: boolean;
  sort_order: number;
  linked_rule_id?: number;
}

export interface InspectRun {
  id: number;
  project_id: number;
  plan_id: number;
  status: string;
  trigger: string;
  datasource_id: number;
  datasource_name?: string;
  score: number;
  grade?: string;
  summary?: string;
  error_message?: string;
  total_count: number;
  critical_count: number;
  warning_count: number;
  normal_count: number;
  storage?: string;
  report_html_path?: string;
  report_pdf_path?: string;
  report_excel_path?: string;
  report_template_code?: string;
  email_sent_at?: string | null;
  started_at?: string | null;
  finished_at?: string | null;
  created_at?: string;
}

export interface InspectStorageInfo {
  backend: string;
  minio_ready: boolean;
  require_minio?: boolean;
  local_root?: string;
  minio_reason?: string;
}

export interface InspectRunTrendItem {
  id: number;
  score: number;
  grade: string;
  critical_count: number;
  warning_count: number;
  finished_at?: string | null;
  status: string;
}

export type InspectPlanUpdate = {
  enabled?: boolean;
  cron_spec?: string;
  datasource_id?: number;
  report_list_mode?: string;
  report_template_id?: number;
  retain_days?: number;
  recipients?: string[];
};

export type InspectItemPayload = {
  type?: string;
  name: string;
  description?: string;
  query: string;
  threshold?: number;
  threshold_type?: string;
  unit?: string;
  enabled?: boolean;
  sort_order?: number;
};

export function getInspectPlan(projectId: number) {
  return getData<InspectPlan>(http.get(`/projects/${projectId}/inspect/plan`));
}

export function getInspectStorageInfo(projectId: number) {
  return getData<InspectStorageInfo>(http.get(`/projects/${projectId}/inspect/storage-info`));
}

export function updateInspectPlan(projectId: number, payload: InspectPlanUpdate) {
  return getData<InspectPlan>(http.put(`/projects/${projectId}/inspect/plan`, payload));
}

export function listInspectItems(projectId: number) {
  return getData<InspectItem[]>(http.get(`/projects/${projectId}/inspect/items`));
}

export function createInspectItem(projectId: number, payload: InspectItemPayload) {
  return getData<InspectItem>(http.post(`/projects/${projectId}/inspect/items`, payload));
}

export function updateInspectItem(projectId: number, itemId: number, payload: InspectItemPayload) {
  return getData<InspectItem>(http.put(`/projects/${projectId}/inspect/items/${itemId}`, payload));
}

export function deleteInspectItem(projectId: number, itemId: number) {
  return getData<{ deleted: boolean }>(http.delete(`/projects/${projectId}/inspect/items/${itemId}`));
}

export function syncInspectItems(projectId: number) {
  return getData<{ created: number }>(http.post(`/projects/${projectId}/inspect/items/sync-template`));
}

export function resetInspectItems(projectId: number) {
  return getData<{ created: number }>(http.post(`/projects/${projectId}/inspect/items/reset-template`));
}

export function promoteInspectItemToAlert(
  projectId: number,
  itemId: number,
  payload?: {
    datasource_id?: number;
    for_seconds?: number;
    eval_interval_seconds?: number;
    severity?: string;
    enabled?: boolean;
  },
) {
  return getData<{ id: number; name: string; expr: string; origin?: string }>(
    http.post(`/projects/${projectId}/inspect/items/${itemId}/promote-alert`, payload ?? {}),
  );
}

export function listInspectRuns(projectId: number, params?: { page?: number; page_size?: number }) {
  return getData<{ list: InspectRun[]; total: number; page: number; page_size: number }>(
    http.get(`/projects/${projectId}/inspect/runs`, { params }),
  );
}

export function startInspectRun(projectId: number, datasourceId?: number) {
  return getData<InspectRun>(
    http.post(`/projects/${projectId}/inspect/runs`, datasourceId ? { datasource_id: datasourceId } : {}),
  );
}

export function getInspectRun(projectId: number, runId: number) {
  return getData<InspectRun>(http.get(`/projects/${projectId}/inspect/runs/${runId}`));
}

export function resendInspectEmail(projectId: number, runId: number) {
  return getData<{ sent: boolean }>(http.post(`/projects/${projectId}/inspect/runs/${runId}/resend-email`));
}

export function listInspectReportTemplates(projectId: number) {
  return getData<InspectReportTemplate[]>(http.get(`/projects/${projectId}/inspect/report-templates`));
}

export function updateInspectReportTemplate(
  projectId: number,
  templateId: number,
  payload: { name?: string; body?: string; remark?: string; status?: number },
) {
  return getData<InspectReportTemplate>(
    http.put(`/projects/${projectId}/inspect/report-templates/${templateId}`, payload),
  );
}

export function deleteInspectReportTemplate(projectId: number, templateId: number) {
  return getData<{ ok?: boolean }>(http.delete(`/projects/${projectId}/inspect/report-templates/${templateId}`));
}

export function copyInspectReportTemplate(
  projectId: number,
  payload: { source_id: number; code?: string; name?: string },
) {
  return getData<InspectReportTemplate>(http.post(`/projects/${projectId}/inspect/report-templates/copy`, payload));
}

export function previewInspectReportTemplate(
  projectId: number,
  payload: { template_id?: number; code?: string; body?: string },
) {
  return http.post(`/projects/${projectId}/inspect/report-templates/preview`, payload, {
    responseType: "blob",
  });
}

export function inspectReportPdfUrl(projectId: number, runId: number) {
  return `/projects/${projectId}/inspect/runs/${runId}/report.pdf`;
}

export function checkInspectReportPdf(projectId: number, runId: number) {
  return getData<{ exists: boolean; filename?: string; size?: number }>(
    http.get(`/projects/${projectId}/inspect/runs/${runId}/report.pdf/check`),
  );
}

export function inspectReportHtmlUrl(projectId: number, runId: number) {
  return `/projects/${projectId}/inspect/runs/${runId}/report.html`;
}

export function inspectReportExcelUrl(projectId: number, runId: number) {
  return `/projects/${projectId}/inspect/runs/${runId}/report.xlsx`;
}

export function inspectReportPrintUrl(projectId: number, runId: number) {
  return `/projects/${projectId}/inspect/runs/${runId}/report.print.html`;
}

export function listInspectRunTrends(projectId: number, limit = 30) {
  return getData<{ list: InspectRunTrendItem[] }>(
    http.get(`/projects/${projectId}/inspect/runs/trends`, { params: { limit } }),
  ).then((r) => r.list || []);
}

export function migrateInspectReportsToMinio(projectId: number) {
  return getData<{ migrated: number }>(http.post(`/projects/${projectId}/inspect/migrate-reports-to-minio`));
}
