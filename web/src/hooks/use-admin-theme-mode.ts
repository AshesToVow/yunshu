import { useEffect, useState } from "react";

export type AdminThemeMode = "dark" | "light";

function readThemeMode(): AdminThemeMode {
  return window.localStorage.getItem("admin-theme-mode") === "light" ? "light" : "dark";
}

export function useAdminThemeMode(): AdminThemeMode {
  const [mode, setMode] = useState<AdminThemeMode>(readThemeMode);

  useEffect(() => {
    const onModeChange = (event: Event) => {
      const detail = (event as CustomEvent<{ mode?: string }>).detail;
      if (detail?.mode === "light" || detail?.mode === "dark") {
        setMode(detail.mode);
        return;
      }
      setMode(readThemeMode());
    };

    window.addEventListener("admin-theme-mode-change", onModeChange as EventListener);
    return () => window.removeEventListener("admin-theme-mode-change", onModeChange as EventListener);
  }, []);

  return mode;
}
