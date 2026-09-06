/** 日志查看器展示偏好（localStorage）。 */
const PREF_KEY = "yunshu.log-viewer.prefs";

export type LogViewerPrefs = {
  listMode: "stacked" | "compact";
  visibleFields: string[];
  chartMode: "timeline" | "pie";
};

const DEFAULT_PREFS: LogViewerPrefs = {
  listMode: "stacked",
  visibleFields: ["level", "service_name", "host", "trace_id", "namespace", "pod"],
  chartMode: "timeline",
};

export function loadLogViewerPrefs(): LogViewerPrefs {
  try {
    const raw = localStorage.getItem(PREF_KEY);
    if (!raw) return { ...DEFAULT_PREFS };
    const parsed = JSON.parse(raw) as Partial<LogViewerPrefs>;
    return {
      listMode: parsed.listMode === "compact" ? "compact" : "stacked",
      visibleFields: Array.isArray(parsed.visibleFields) && parsed.visibleFields.length
        ? parsed.visibleFields
        : [...DEFAULT_PREFS.visibleFields],
      chartMode: parsed.chartMode === "pie" ? "pie" : "timeline",
    };
  } catch {
    return { ...DEFAULT_PREFS };
  }
}

export function saveLogViewerPrefs(prefs: LogViewerPrefs) {
  try {
    localStorage.setItem(PREF_KEY, JSON.stringify(prefs));
  } catch {
    /* ignore */
  }
}

export const LEVEL_STACK_COLORS: Record<string, string> = {
  ERROR: "#ff4d4f",
  FATAL: "#a8071a",
  WARN: "#faad14",
  WARNING: "#faad14",
  INFO: "#1677ff",
  DEBUG: "#8c8c8c",
  TRACE: "#bfbfbf",
};
