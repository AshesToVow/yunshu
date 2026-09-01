export const ALERT_MONITOR_TABS = [
  { key: "history", label: "事件台", path: "history", group: "duty" },
  { key: "silences", label: "降噪·静默", path: "silences", group: "duty" },
  { key: "rules", label: "规则中心", path: "rules", group: "config" },
  { key: "policies", label: "通知与路由", path: "policies", group: "config" },
  { key: "objects", label: "监控对象", path: "objects", group: "config" },
  { key: "datasources", label: "数据源", path: "datasources", group: "config" },
  { key: "quality", label: "质量", path: "quality", group: "expert" },
  { key: "inhibition", label: "降噪·抑制", path: "inhibition", group: "expert" },
  { key: "cloud-expiry", label: "云到期规则", path: "cloud-expiry", group: "expert" },
  { key: "promql", label: "PromQL 调试", path: "promql", group: "expert" },
] as const;

export type AlertMonitorTabKey = (typeof ALERT_MONITOR_TABS)[number]["key"];
export type AlertMonitorTabGroup = "duty" | "config" | "expert";

export const ALERT_MONITOR_TAB_GROUPS: Array<{ key: AlertMonitorTabGroup; label: string }> = [
  { key: "duty", label: "值班处置" },
  { key: "config", label: "规则与配置" },
  { key: "expert", label: "专家工具" },
];

export const DEFAULT_ALERT_MONITOR_TAB: AlertMonitorTabKey = "history";

export function groupForTab(key: AlertMonitorTabKey): AlertMonitorTabGroup {
  const hit = ALERT_MONITOR_TABS.find((x) => x.key === key);
  return (hit?.group ?? "duty") as AlertMonitorTabGroup;
}

export function tabsInGroup(group: AlertMonitorTabGroup) {
  return ALERT_MONITOR_TABS.filter((x) => x.group === group);
}

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
