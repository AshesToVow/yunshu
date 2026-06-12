import { App as AntdApp, ConfigProvider, theme } from "antd";
import zhCN from "antd/locale/zh_CN";
import { useEffect, useMemo, useState } from "react";
import { BrowserRouter } from "react-router-dom";
import { AppRoutes } from "./app-routes";
import { ErrorBoundary } from "../components/error-boundary";
import { AuthProvider } from "../contexts/auth-context";
import { PluginProvider } from "../contexts/plugin-context";

export function App() {
  const [mode, setMode] = useState<"dark" | "light">(() => {
    const saved = window.localStorage.getItem("admin-theme-mode");
    return saved === "light" ? "light" : "dark";
  });
  const [accent, setAccent] = useState<string>(() => {
    return window.localStorage.getItem("admin-theme-accent") ?? "#e61919";
  });

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
          colorSuccess: "#4af626",
          colorWarning: "#f59e0b",
          colorError: accent,
          borderRadius: 0,
          fontFamily:
            '"IBM Plex Sans", "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif',
          fontFamilyCode: '"JetBrains Mono", "IBM Plex Mono", "Consolas", monospace',
          colorBgLayout: isDark ? "#0a0a0a" : "#f4f4f0",
          colorText: isDark ? "#eaeaea" : "#050505",
          colorTextSecondary: isDark ? "rgba(234,234,234,0.58)" : "rgba(5,5,5,0.56)",
          colorBorder: isDark ? "rgba(234,234,234,0.14)" : "rgba(5,5,5,0.14)",
        },
        components: {
          Layout: {
            headerBg: "transparent",
            siderBg: isDark ? "#121212" : "#ffffff",
            bodyBg: isDark ? "#0a0a0a" : "#f4f4f0",
          },
          Menu: {
            darkItemBg: "#121212",
            darkSubMenuItemBg: "#121212",
            darkItemSelectedBg: "rgba(230, 25, 25, 0.18)",
            darkItemHoverBg: "rgba(230, 25, 25, 0.08)",
            itemSelectedColor: accent,
          },
          Card: {
            boxShadow: "none",
          },
          Button: {
            controlHeightLG: 44,
          },
          Table: {
            headerBg: isDark ? "#1a1a1a" : "#eae8e3",
            rowHoverBg: isDark ? "rgba(230, 25, 25, 0.08)" : "rgba(230, 25, 25, 0.06)",
          },
        },
      }}
    >
      <AntdApp>
        <AuthProvider>
          <PluginProvider>
            <BrowserRouter>
              <ErrorBoundary>
                <AppRoutes />
              </ErrorBoundary>
            </BrowserRouter>
          </PluginProvider>
        </AuthProvider>
      </AntdApp>
    </ConfigProvider>
  );
}
