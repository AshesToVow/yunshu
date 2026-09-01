// @ts-nocheck
import { Navigate, useSearchParams } from '@umijs/max';
import { normalizeAlertMonitorTab, tabPathForKey } from "./tab-config";

/** 兼容 ?tab=rules、?tab=config&cfg=history 等旧链接 */
export function AlertMonitorLegacyRedirect() {
  const [searchParams] = useSearchParams();
  const tabParam = searchParams.get("tab");
  if (!tabParam) {
    return <Navigate to="/alert-monitor-platform/datasources" replace />;
  }
  let tab = normalizeAlertMonitorTab(tabParam);
  if (tabParam === "config" && searchParams.get("cfg") === "history") {
    tab = "history";
  }
  const qs = new URLSearchParams(searchParams);
  qs.delete("tab");
  qs.delete("cfg");
  const tail = qs.toString();
  const base = tabPathForKey(tab);
  return <Navigate to={tail ? `${base}?${tail}` : base} replace />;
}
