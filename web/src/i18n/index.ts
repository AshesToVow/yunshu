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
  interpolation: { escapeValue: false },
});

export default i18n;
