export const ALERT_MONITOR_TABS = [
  { key: "datasources", label: "数据源", path: "datasources" },
  { key: "policies", label: "告警路由", path: "policies" },
  { key: "history", label: "历史记录", path: "history" },
  { key: "inhibition", label: "告警抑制", path: "inhibition" },
  { key: "silences", label: "平台静默", path: "silences" },
  { key: "rules", label: "监控规则与值班", path: "rules" },
  { key: "cloud-expiry", label: "云到期规则", path: "cloud-expiry" },
  { key: "promql", label: "PromQL 查询", path: "promql" },
] as const;

export type AlertMonitorTabKey = (typeof ALERT_MONITOR_TABS)[number]["key"];

export const DEFAULT_ALERT_MONITOR_TAB: AlertMonitorTabKey = "datasources";

export function normalizeAlertMonitorTab(raw?: string | null): AlertMonitorTabKey {
  const t = String(raw || "").trim().toLowerCase();
  if (t === "config") return "policies";
  if (ALERT_MONITOR_TABS.some((x) => x.key === t)) {
    return t as AlertMonitorTabKey;
  }
  return DEFAULT_ALERT_MONITOR_TAB;
}

export function tabPathForKey(key: AlertMonitorTabKey): string {
  return `/alert-monitor-platform/${key}`;
}
