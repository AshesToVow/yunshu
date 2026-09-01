// @ts-nocheck
import dayjs from "dayjs";
import type {
  AlertmanagerSilenceRow,
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

export function parseAlertmanagerSilences(raw: unknown): AlertmanagerSilenceRow[] {
  if (!Array.isArray(raw)) return [];
  return raw.map((item) => {
    const row = (item ?? {}) as {
      id?: string;
      comment?: string;
      createdBy?: string;
      startsAt?: string;
      endsAt?: string;
      status?: { state?: string; comment?: string; createdBy?: string; startsAt?: string; endsAt?: string };
      matchers?: Array<{ name?: string; value?: string; isRegex?: boolean; is_regex?: boolean }>;
    };
    const state = String(row.status?.state ?? "").toLowerCase();
    const matchers = Array.isArray(row.matchers)
      ? row.matchers.map((m) => ({
          name: String(m?.name ?? ""),
          value: String(m?.value ?? ""),
          is_regex: Boolean(m?.isRegex ?? m?.is_regex),
        }))
      : [];
    const amId = String(row.id ?? "");
    const comment = String(row.comment ?? row.status?.comment ?? "").trim();
    return {
      rowKey: `am-${amId}`,
      source: "alertmanager" as const,
      amId,
      name: comment || String(row.createdBy ?? row.status?.createdBy ?? `Alertmanager #${amId}`),
      comment: comment || undefined,
      matchers,
      starts_at: String(row.startsAt ?? row.status?.startsAt ?? ""),
      ends_at: String(row.endsAt ?? row.status?.endsAt ?? ""),
      state: state || "unknown",
      enabled: state === "active" || state === "pending",
    };
  });
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
