import { createContext, useContext } from "react";
import type { AlertMonitorTabKey } from "./tab-config";

/** Tab 面板与 Modals 共享的平台上下文（由 platform-provider 注入） */
export type AlertMonitorContextValue = Record<string, unknown> & {
  tab: AlertMonitorTabKey;
  setTab: (key: AlertMonitorTabKey) => void;
  projectContextId?: number;
  loading: boolean;
  activeProjectName: string;
  projectOptions: { label: string; value: number }[];
  setProjectContext: (projectID?: number) => void;
  openHistoryTab: () => void;
};

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

export function useAlertMonitor(): any {
  const ctx = useContext(AlertMonitorContext);
  if (!ctx) {
    throw new Error("useAlertMonitor must be used within AlertMonitorProvider");
  }
  return ctx;
}
