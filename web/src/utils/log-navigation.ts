/** 构建日志检索页 URL（支持关联上下文参数）。 */
export function buildProjectLogsUrl(params: {
  project_id: number;
  anchor_time?: string;
  alert_id?: number;
  change_id?: number;
  fingerprint?: string;
  window_minutes?: number;
  tab?: "logs" | "patterns" | "anomalies";
  log_source_id?: number;
  service_id?: number;
  level?: string;
}) {
  const q = new URLSearchParams();
  q.set("project_id", String(params.project_id));
  if (params.anchor_time) q.set("anchor_time", params.anchor_time);
  if (params.alert_id) q.set("alert_id", String(params.alert_id));
  if (params.change_id) q.set("change_id", String(params.change_id));
  if (params.fingerprint) q.set("fingerprint", params.fingerprint);
  if (params.window_minutes) q.set("window_minutes", String(params.window_minutes));
  if (params.tab) q.set("tab", params.tab);
  if (params.log_source_id) q.set("log_source_id", String(params.log_source_id));
  if (params.service_id) q.set("service_id", String(params.service_id));
  if (params.level) q.set("level", params.level);
  const qs = q.toString();
  return `/project-logs${qs ? `?${qs}` : ""}`;
}

/** 从告警详情跳转关联日志。 */
export function buildAlertLogContextUrl(input: {
  project_id?: number;
  alert_id?: number;
  fingerprint?: string;
  starts_at?: string;
  window_minutes?: number;
}) {
  if (!input.project_id) return "/project-logs";
  return buildProjectLogsUrl({
    project_id: input.project_id,
    alert_id: input.alert_id,
    fingerprint: input.fingerprint,
    anchor_time: input.starts_at,
    window_minutes: input.window_minutes ?? 5,
    level: "ERROR",
  });
}

/** 从变更事件跳转关联日志。 */
export function buildChangeLogContextUrl(input: {
  project_id: number;
  change_id: number;
  started_at?: string;
  window_minutes?: number;
}) {
  return buildProjectLogsUrl({
    project_id: input.project_id,
    change_id: input.change_id,
    anchor_time: input.started_at,
    window_minutes: input.window_minutes ?? 5,
    level: "ERROR",
  });
}
