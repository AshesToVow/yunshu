import { App as AntdApp, ConfigProvider, theme } from "antd";
import zhCN from "antd/locale/zh_CN";
import { useEffect, useMemo } from "react";
import { BrowserRouter } from "react-router-dom";
import { AppRoutes } from "./app-routes";
import { ErrorBoundary } from "../components/error-boundary";
import { AuthProvider } from "../contexts/auth-context";
import { PluginProvider } from "../contexts/plugin-context";
import { WorkloadProgressProvider } from "../contexts/workload-progress-context";
import { WorkloadProgressFloat } from "../components/workload-progress-float";
import { useAdminThemeStore } from "../stores/admin-theme-store";

export function App() {
  const mode = useAdminThemeStore((s) => s.mode);
  const accent = useAdminThemeStore((s) => s.accent);

  useEffect(() => {
    document.documentElement.dataset.theme = mode;
    document.documentElement.style.setProperty("--ys-brand", accent);
    document.documentElement.style.setProperty("--admin-accent", accent);
  }, [mode, accent]);

  const isDark = mode === "dark";
  const algorithm = useMemo(() => (isDark ? theme.darkAlgorithm : theme.defaultAlgorithm), [isDark]);

  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm,
        token: {
          colorPrimary: accent,
          colorSuccess: "#389e0d",
          colorWarning: "#d48806",
          colorError: "#cf1322",
          colorInfo: "#0958d9",
          borderRadius: 8,
          fontFamily: '"IBM Plex Sans", "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif',
          fontFamilyCode: '"JetBrains Mono", "IBM Plex Mono", "Consolas", monospace',
          colorBgLayout: isDark ? "#141414" : "#f4f6f8",
          colorText: isDark ? "rgba(255,255,255,0.88)" : "#0f172a",
          colorTextSecondary: isDark ? "rgba(255,255,255,0.55)" : "#64748b",
          colorBorder: isDark ? "#303030" : "#e2e8f0",
        },
        components: {
          Layout: {
            headerBg: "transparent",
            siderBg: isDark ? "#1f1f1f" : "#ffffff",
            bodyBg: isDark ? "#141414" : "#f4f6f8",
          },
          Menu: {
            darkItemBg: "#1f1f1f",
            darkSubMenuItemBg: "#1f1f1f",
            darkItemSelectedBg: "rgba(13, 148, 136, 0.18)",
            darkItemHoverBg: "rgba(13, 148, 136, 0.08)",
            itemSelectedColor: accent,
          },
          Card: {
            boxShadow: "none",
          },
          Button: {
            controlHeightLG: 44,
          },
          Table: {
            headerBg: isDark ? "#1f1f1f" : "#f8fafc",
            rowHoverBg: isDark ? "rgba(255,255,255,0.04)" : "rgba(15,23,42,0.03)",
          },
        },
      }}
    >
      <AntdApp>
        <AuthProvider>
          <PluginProvider>
            <WorkloadProgressProvider>
              <BrowserRouter>
                <ErrorBoundary>
                  <AppRoutes />
                  <WorkloadProgressFloat />
                </ErrorBoundary>
              </BrowserRouter>
            </WorkloadProgressProvider>
          </PluginProvider>
        </AuthProvider>
      </AntdApp>
    </ConfigProvider>
  );
}
