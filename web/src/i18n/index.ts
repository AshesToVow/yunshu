import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import zhCN from "../locales/zh-CN.json";
import enUS from "../locales/en-US.json";

const saved = typeof window !== "undefined" ? window.localStorage.getItem("app-locale") : null;

void i18n.use(initReactI18next).init({
  resources: {
    "zh-CN": { translation: zhCN },
    "en-US": { translation: enUS },
  },
  lng: saved === "en-US" ? "en-US" : "zh-CN",
  fallbackLng: "zh-CN",
  supportedLngs: ["zh-CN", "en-US"],
  interpolation: { escapeValue: false },
  react: { useSuspense: false },
});

if (typeof document !== "undefined") {
  document.documentElement.lang = saved === "en-US" ? "en" : "zh-CN";
}

export function resolveAppLocale(language?: string): "zh-CN" | "en-US" {
  return language?.startsWith("en") ? "en-US" : "zh-CN";
}

export default i18n;
