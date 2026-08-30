// 巡检页面预设常量与展示映射（RF-10）。
// 由 web/src/pages/project-inspect-page.tsx 原样搬迁，取值与分支顺序逐字保留。
import { CHART_BRAND, CHART_ERROR, CHART_SUCCESS, CHART_WARNING } from "../../constants/chart-colors";

/** 巡检计划 cron 预设（6 段式，与后端 robfig/cron 秒级精度一致） */
export const CRON_PRESETS = [
  { label: "每天 09:00", value: "0 0 9 * * *" },
  { label: "每天 02:00", value: "0 0 2 * * *" },
  { label: "每天 18:00", value: "0 0 18 * * *" },
  { label: "每 6 小时", value: "0 0 */6 * * *" },
  { label: "每周一 09:00", value: "0 0 9 * * 1" },
  { label: "每周一 02:00", value: "0 0 2 * * 1" },
];

export const THRESHOLD_TYPE_OPTIONS = [
  { label: "大于 (>)", value: "greater" },
  { label: "大于等于 (≥)", value: "greater_equal" },
  { label: "小于 (<)", value: "less" },
  { label: "小于等于 (≤)", value: "less_equal" },
  { label: "等于 (=)", value: "equal" },
  { label: "不等于 (≠)", value: "not_equal" },
];

export const THRESHOLD_TYPE_LABEL: Record<string, string> = Object.fromEntries(
  THRESHOLD_TYPE_OPTIONS.map((o) => [o.value, o.label]),
);

export function statusMeta(status?: string): { color: string; label: string } {
  switch (status) {
    case "success":
      return { color: "success", label: "成功" };
    case "failed":
      return { color: "error", label: "失败" };
    case "running":
      return { color: "processing", label: "执行中" };
    case "pending":
      return { color: "default", label: "排队中" };
    default:
      return { color: "default", label: status || "-" };
  }
}

export function triggerLabel(trigger?: string) {
  if (trigger === "cron") return "定时";
  if (trigger === "manual") return "手动";
  return trigger || "-";
}

export function gradeColor(grade?: string) {
  switch (grade) {
    case "A":
      return CHART_SUCCESS;
    case "B":
      return CHART_BRAND;
    case "C":
      return CHART_WARNING;
    case "D":
      return CHART_ERROR;
    default:
      return CHART_BRAND;
  }
}

/** 收件人既可能是 JSON 数组，也可能是历史遗留的逗号分隔字符串 */
export function parseRecipients(raw?: string): string[] {
  if (!raw) return [];
  try {
    const v = JSON.parse(raw);
    return Array.isArray(v) ? v.map(String) : [];
  } catch {
    return raw.split(",").map((s) => s.trim()).filter(Boolean);
  }
}

export function storageLabel(storage?: string) {
  const s = (storage || "local").toLowerCase();
  if (s === "minio") return "MinIO";
  return "本地";
}

export function storageColor(storage?: string) {
  return (storage || "local").toLowerCase() === "minio" ? "blue" : "default";
}
