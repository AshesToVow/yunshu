// @ts-nocheck
import { Navigate, useLocation } from '@umijs/max';

/** 已合并到「告警监控平台 → 历史记录」，保留路由以兼容旧书签与权限路径。 */
export function AlertEventsPage() {
  const { search } = useLocation();
  return <Navigate to={`/alert-monitor-platform/history${search}`} replace />;
}
