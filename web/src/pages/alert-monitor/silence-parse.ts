import dayjs from "dayjs";
import type {
  PromNativeAlertRow,
  QuickSilenceTarget,
  SilenceMatcherForm,
} from "./platform-provider-types";

export function parseSilenceMatchersForForm(raw?: string): SilenceMatcherForm[] {
  const s = raw?.trim();
  if (!s) return [{ name: "alertname", value: "", is_regex: false }];
  try {
    const v = JSON.parse(s) as unknown;
    if (!Array.isArray(v)) return [{ name: "alertname", value: "", is_regex: false }];
    return v.map((row: unknown) => {
      const o = row as Record<string, unknown>;
      return {
        name: String(o?.name ?? "").trim(),
        value: String(o?.value ?? "").trim(),
        is_regex: Boolean(o?.is_regex),
      };
    });
  } catch {
    return [{ name: "alertname", value: "", is_regex: false }];
  }
}

export function toQuickSilenceTarget(row: PromNativeAlertRow): QuickSilenceTarget {
  const now = dayjs();
  const n = String(row.alertname || "").trim() || "未命名告警";
  return {
    key: row.key,
    name: `静默 ${n}`,
    labels: row.labels ?? {},
    startsAt: now,
    endsAt: now.add(2, "hour"),
  };
}

export function buildMatchersByLabels(labels: Record<string, string>): SilenceMatcherForm[] {
  return Object.entries(labels ?? {})
    .map(([name, value]) => ({ name: String(name || "").trim(), value: String(value || "").trim(), is_regex: false }))
    .filter((m) => m.name && m.value);
}
