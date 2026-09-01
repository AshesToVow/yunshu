// @ts-nocheck
import { Navigate, useSearchParams } from '@umijs/max';

/** 已并入「告警监控平台」；保留路由以兼容旧书签与菜单。 */
export function AlertConfigCenterPage() {
  const [searchParams] = useSearchParams();
  const legacy = searchParams.get("tab");
  const cfg = legacy === "history" || legacy === "templates" ? legacy : "policies";
  const path = cfg === "history" ? "/alert-monitor-platform/history" : "/alert-monitor-platform/policies";
  const qs = searchParams.toString();
  return <Navigate to={qs ? `${path}?${qs}` : path} replace />;
}
