import { App as AntdApp, ConfigProvider, theme } from "antd";
import enUS from "antd/locale/en_US";
import zhCN from "antd/locale/zh_CN";
import { useEffect, useMemo, useState } from "react";
import { BrowserRouter } from "react-router-dom";
import { AppRoutes } from "./app-routes";
import { ErrorBoundary } from "../components/error-boundary";
import { AuthProvider } from "../contexts/auth-context";
import { PluginProvider } from "../contexts/plugin-context";
import { WorkloadProgressProvider } from "../contexts/workload-progress-context";
import { WorkloadProgressFloat } from "../components/workload-progress-float";
import i18n, { resolveAppLocale } from "../i18n";
import { useAdminThemeStore } from "../stores/admin-theme-store";

export function App() {
  const mode = useAdminThemeStore((s) => s.mode);
  const accent = useAdminThemeStore((s) => s.accent);
  const [localeKey, setLocaleKey] = useState(() => resolveAppLocale(i18n.language));

  useEffect(() => {
    document.documentElement.dataset.theme = mode;
    document.documentElement.style.setProperty("--ys-brand", accent);
    document.documentElement.style.setProperty("--admin-accent", accent);
  }, [mode, accent]);

  useEffect(() => {
    const sync = (lng: string) => setLocaleKey(resolveAppLocale(lng));
    sync(i18n.language);
    i18n.on("languageChanged", sync);
    return () => {
      i18n.off("languageChanged", sync);
    };
  }, []);

  const isDark = mode === "dark";
  const algorithm = useMemo(() => (isDark ? theme.darkAlgorithm : theme.defaultAlgorithm), [isDark]);
  const antdLocale = localeKey === "en-US" ? enUS : zhCN;
  const linkColor = isDark ? "#38bdf8" : "#0284c7";
  const linkHover = isDark ? "#7dd3fc" : "#0369a1";

  return (
    <ConfigProvider
      locale={antdLocale}
      theme={{
        algorithm,
        token: {
          colorPrimary: accent,
          colorLink: linkColor,
          colorLinkHover: linkHover,
          colorSuccess: "#389e0d",
          colorWarning: "#d48806",
          colorError: "#cf1322",
          colorInfo: "#0958d9",
          borderRadius: 8,
          fontFamily: '"IBM Plex Sans", "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif',
          fontFamilyCode: '"JetBrains Mono", "IBM Plex Mono", "Consolas", monospace',
          colorBgLayout: isDark ? "#141414" : "#f4f6f8",
          colorText: isDark ? "rgba(255,255,255,0.88)" : "#0f172a",
          colorTextSecondary: isDark ? "rgba(255,255,255,0.68)" : "#475569",
          colorTextDescription: isDark ? "rgba(255,255,255,0.62)" : "#64748b",
          colorTextPlaceholder: isDark ? "rgba(255,255,255,0.38)" : "#94a3b8",
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
            colorLink: linkColor,
            colorLinkHover: linkHover,
          },
          Table: {
            headerBg: isDark ? "#1f1f1f" : "#f8fafc",
            rowHoverBg: isDark ? "rgba(255,255,255,0.04)" : "rgba(15,23,42,0.03)",
          },
          Alert: isDark
            ? {
                colorInfoBg: "#111a2c",
                colorInfoBorder: "#15325b",
                colorSuccessBg: "#162312",
                colorSuccessBorder: "#274916",
                colorWarningBg: "#2b2111",
                colorWarningBorder: "#594214",
                colorErrorBg: "#2a1215",
                colorErrorBorder: "#5c2223",
                // 强制描述/标题用主文字色，避免浅底上继承的浅色字
                colorText: "rgba(255,255,255,0.88)",
                colorTextHeading: "rgba(255,255,255,0.92)",
              }
            : undefined,
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
