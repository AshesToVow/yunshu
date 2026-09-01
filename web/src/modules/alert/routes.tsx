import { lazy, Suspense } from "react";
import { Navigate, type RouteObject } from "react-router-dom";
import { Spin } from "antd";
import { AlertMonitorLegacyRedirect } from "../../pages/alert-monitor/legacy-redirect";

export const ALERT_PLUGIN = "alert";

const AlertMonitorPlatformRoot = lazy(async () => {
  const mod = await import("../../pages/alert-monitor/platform-root");
  return { default: mod.AlertMonitorPlatformRoot };
});

function TabFallback() {
  return (
    <div style={{ padding: 48, textAlign: "center" }}>
      <Spin tip="加载告警平台..." />
    </div>
  );
}

const alertPlatformElement = (
  <Suspense fallback={<TabFallback />}>
    <AlertMonitorPlatformRoot />
  </Suspense>
);

export const alertRoutes: RouteObject[] = [
  {
    path: "alert-monitor-platform",
    element: <Navigate to="/alert-monitor-platform/history" replace />,
  },
  {
    path: "alert-monitor-platform/:tab",
    element: alertPlatformElement,
  },
  {
    path: "alert-monitor-platform/",
    element: <AlertMonitorLegacyRedirect />,
  },
];
