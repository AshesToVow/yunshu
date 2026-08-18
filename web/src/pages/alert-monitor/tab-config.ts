export const ALERT_MONITOR_TABS = [
  { key: "history", label: "事件台", path: "history" },
  { key: "rules", label: "规则中心", path: "rules" },
  { key: "datasources", label: "数据源", path: "datasources" },
  { key: "objects", label: "监控对象", path: "objects" },
  { key: "quality", label: "质量", path: "quality" },
  { key: "policies", label: "通知与路由", path: "policies" },
  { key: "silences", label: "降噪·静默", path: "silences" },
  { key: "inhibition", label: "降噪·抑制", path: "inhibition" },
  { key: "cloud-expiry", label: "云到期规则", path: "cloud-expiry" },
  { key: "promql", label: "PromQL 调试", path: "promql" },
] as const;

export type AlertMonitorTabKey = (typeof ALERT_MONITOR_TABS)[number]["key"];

export const DEFAULT_ALERT_MONITOR_TAB: AlertMonitorTabKey = "history";

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
