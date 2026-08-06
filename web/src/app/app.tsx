import { App as AntdApp, ConfigProvider, theme } from "antd";
import zhCN from "antd/locale/zh_CN";
import { useEffect, useMemo, useState } from "react";
import { BrowserRouter } from "react-router-dom";
import { AppRoutes } from "./app-routes";
import { ErrorBoundary } from "../components/error-boundary";
import { BRAND_PRIMARY } from "../constants/brand";
import { AuthProvider } from "../contexts/auth-context";
import { PluginProvider } from "../contexts/plugin-context";
import { WorkloadProgressProvider } from "../contexts/workload-progress-context";
import { WorkloadProgressFloat } from "../components/workload-progress-float";

export function App() {
  const [mode, setMode] = useState<"dark" | "light">(() => {
    const saved = window.localStorage.getItem("admin-theme-mode");
    return saved === "light" ? "light" : "dark";
  });
  const [accent, setAccent] = useState<string>(() => {
    return window.localStorage.getItem("admin-theme-accent") ?? BRAND_PRIMARY;
  });

  useEffect(() => {
    document.documentElement.dataset.theme = mode;
    document.documentElement.style.setProperty("--ys-brand", accent);
    document.documentElement.style.setProperty("--admin-accent", accent);
  }, [mode, accent]);

  useEffect(() => {
    const onModeChange = (event: Event) => {
      const detail = (event as CustomEvent<{ mode?: "dark" | "light" }>).detail;
      const next = detail?.mode === "light" ? "light" : "dark";
      setMode(next);
    };
    const onStorage = (event: StorageEvent) => {
      if (event.key === "admin-theme-mode") {
        setMode(event.newValue === "light" ? "light" : "dark");
      }
      if (event.key === "admin-theme-accent" && event.newValue) {
        setAccent(event.newValue);
      }
    };
    const onAccentChange = (event: Event) => {
      const detail = (event as CustomEvent<{ accent?: string }>).detail;
      if (!detail?.accent) return;
      setAccent(detail.accent);
    };
    window.addEventListener("admin-theme-mode-change", onModeChange as EventListener);
    window.addEventListener("admin-theme-accent-change", onAccentChange as EventListener);
    window.addEventListener("storage", onStorage);
    return () => {
      window.removeEventListener("admin-theme-mode-change", onModeChange as EventListener);
      window.removeEventListener("admin-theme-accent-change", onAccentChange as EventListener);
      window.removeEventListener("storage", onStorage);
    };
  }, []);

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
          fontFamily:
            '"IBM Plex Sans", "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif',
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
