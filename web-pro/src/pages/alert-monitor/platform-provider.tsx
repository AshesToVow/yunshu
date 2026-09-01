// @ts-nocheck
import { lazy, Suspense } from "react";
import { Spin } from "antd";
import { AlertMonitorProvider } from "./context";
import { AlertMonitorLayout } from "./layout";
import { useAlertMonitorPlatformState } from "./use-alert-monitor-platform-state";

export type { AlertMonitorTabKey } from "./tab-config";

const AlertMonitorModals = lazy(async () => {
  const mod = await import("./modals");
  return { default: mod.AlertMonitorModals };
});

export function AlertMonitorPlatformRoot() {
  const state = useAlertMonitorPlatformState();
  return (
    <AlertMonitorProvider value={state as never}>
      <AlertMonitorLayout />
      <Suspense
        fallback={
          <div style={{ position: "fixed", right: 16, bottom: 16 }}>
            <Spin size="small" />
          </div>
        }
      >
        <AlertMonitorModals />
      </Suspense>
    </AlertMonitorProvider>
  );
}
