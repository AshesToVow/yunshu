// @ts-nocheck
import { create } from "zustand";
import { persist } from "zustand/middleware";
import { BRAND_PRIMARY } from "../constants/brand";

export type AdminThemeMode = "dark" | "light";

/** 主题色以 CSS 色值存储（与设置抽屉色板一致）；登录页命名色映射为 hex 后再写入。 */
export type AdminThemeAccent = string;

type ThemeState = {
  mode: AdminThemeMode;
  accent: AdminThemeAccent;
  setMode: (mode: AdminThemeMode) => void;
  toggleMode: () => void;
  setAccent: (accent: AdminThemeAccent) => void;
};

function emitMode(mode: AdminThemeMode) {
  window.dispatchEvent(new CustomEvent("admin-theme-mode-change", { detail: { mode } }));
}

function emitAccent(accent: AdminThemeAccent) {
  window.dispatchEvent(new CustomEvent("admin-theme-accent-change", { detail: { accent } }));
  document.documentElement.style.setProperty("--admin-accent", accent);
  document.documentElement.style.setProperty("--ys-brand", accent);
}

function applyModeToDom(mode: AdminThemeMode) {
  document.documentElement.dataset.theme = mode;
}

export const useAdminThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      mode: "dark",
      accent: BRAND_PRIMARY,
      setMode: (mode) => {
        set({ mode });
        applyModeToDom(mode);
        emitMode(mode);
      },
      toggleMode: () => {
        const mode = get().mode === "light" ? "dark" : "light";
        set({ mode });
        applyModeToDom(mode);
        emitMode(mode);
      },
      setAccent: (accent) => {
        set({ accent });
        emitAccent(accent);
      },
    }),
    {
      name: "admin-theme",
      partialize: (s) => ({ mode: s.mode, accent: s.accent }),
      // 兼容旧键 admin-theme-mode / admin-theme-accent
      merge: (persisted, current) => {
        const p = (persisted ?? {}) as Partial<ThemeState>;
        const legacyMode = window.localStorage.getItem("admin-theme-mode");
        const legacyAccent = window.localStorage.getItem("admin-theme-accent");
        const mode = p.mode ?? (legacyMode === "light" ? "light" : current.mode);
        const accent = p.accent ?? legacyAccent ?? current.accent;
        return { ...current, ...p, mode, accent };
      },
      onRehydrateStorage: () => (state) => {
        if (!state) return;
        applyModeToDom(state.mode);
        document.documentElement.style.setProperty("--admin-accent", state.accent);
        document.documentElement.style.setProperty("--ys-brand", state.accent);
      },
    },
  ),
);

/** 兼容旧 hook 名：读模式 */
export function useAdminThemeMode(): AdminThemeMode {
  return useAdminThemeStore((s) => s.mode);
}
