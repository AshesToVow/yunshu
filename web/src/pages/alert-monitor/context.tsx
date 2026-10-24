import { createContext, useContext } from "react";
import type { useAlertMonitorPlatformState } from "./use-alert-monitor-platform-state";

/** Tab 面板与 Modals 共享的平台上下文（由 platform-provider 注入，类型随 state hook 单一来源） */
export type AlertMonitorContextValue = ReturnType<typeof useAlertMonitorPlatformState>;

const AlertMonitorContext = createContext<AlertMonitorContextValue | null>(null);

export function AlertMonitorProvider({
  value,
  children,
}: {
  value: AlertMonitorContextValue;
  children: React.ReactNode;
}) {
  return <AlertMonitorContext.Provider value={value}>{children}</AlertMonitorContext.Provider>;
}

export function useAlertMonitor(): AlertMonitorContextValue {
  const ctx = useContext(AlertMonitorContext);
  if (!ctx) {
    throw new Error("useAlertMonitor must be used within AlertMonitorProvider");
  }
  return ctx;
}
