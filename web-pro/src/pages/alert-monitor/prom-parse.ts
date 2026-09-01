// @ts-nocheck
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import type { MetricLabelFilter, PromNativeAlertRow, PromTableView } from "./platform-provider-types";

export function parsePrometheusActiveAlertsTable(body: unknown): PromNativeAlertRow[] {
  if (!body || typeof body !== "object") return [];
  const root = body as { data?: { alerts?: unknown[] } };
  const alerts = root.data?.alerts;
  if (!Array.isArray(alerts)) return [];
  return alerts.map((a, i) => {
    const row = (a ?? {}) as { labels?: Record<string, string>; state?: string; activeAt?: string };
    const labels = row.labels ?? {};
    const name = labels.alertname ?? "";
    const short = JSON.stringify(labels);
    return {
      key: String(i),
      alertname: String(name),
      state: String(row.state ?? ""),
      labelsShort: short.length > 140 ? `${short.slice(0, 140)}…` : short,
      activeAt: row.activeAt,
      labels,
    };
  });
}

export function sortMetricKeys(a: string, b: string): number {
  if (a === "__name__") return -1;
  if (b === "__name__") return 1;
  return a.localeCompare(b);
}

export function formatPromTimestampLocal(raw: string): string {
  const n = Number(raw);
  if (!Number.isFinite(n)) return raw;
  const ms = n > 1e12 ? n : n * 1000;
  return dayjs(ms).format("YYYY-MM-DD HH:mm:ss");
}

export function isValidPromLabelKey(s: string): boolean {
  return /^[a-zA-Z_][a-zA-Z0-9_]*$/.test(String(s || "").trim());
}

export function buildPromSelectorExpr(metric: string, filters: MetricLabelFilter[]): string {
  const m = String(metric || "").trim();
  if (!m) return "";
  const parts = filters
    .map((f) => ({
      key: String(f.key || "").trim(),
      op: f.op,
      value: String(f.value || "").trim(),
    }))
    .filter((f) => isValidPromLabelKey(f.key) && f.value !== "")
    .map((f) => `${f.key}${f.op}"${f.value.replace(/"/g, '\\"')}"`);
  if (!parts.length) return m;
  return `${m}{${parts.join(",")}}`;
}

export function parsePromSelectorExpr(raw: string): { metric: string; filters: MetricLabelFilter[] } | null {
  const s = String(raw || "").trim();
  if (!s) return null;
  const m = s.match(/^([a-zA-Z_:][a-zA-Z0-9_:]*)(?:\{([\s\S]*)\})?$/);
  if (!m) return null;
  const metric = String(m[1] || "").trim();
  if (!metric) return null;
  const body = String(m[2] || "").trim();
  if (!body) return { metric, filters: [{ key: "instance", op: "=", value: "" }] };
  const filters: MetricLabelFilter[] = [];
  const re = /([a-zA-Z_][a-zA-Z0-9_]*)\s*(=~|!~|!=|=)\s*"((?:\\.|[^"\\])*)"\s*(?:,|$)/g;
  let match: RegExpExecArray | null;
  while ((match = re.exec(body)) !== null) {
    const key = String(match[1] || "").trim();
    const op = (match[2] as MetricLabelFilter["op"]) || "=";
    const value = String(match[3] || "").replace(/\\"/g, '"').trim();
    filters.push({ key, op, value });
  }
  return { metric, filters: filters.length ? filters : [{ key: "instance", op: "=", value: "" }] };
}

export function detectPromFunctionKeyFromExpr(exprRaw: string): string | null {
  const s = String(exprRaw || "").trim().toLowerCase();
  if (!s) return null;
  if (/^histogram_quantile\s*\(/.test(s)) return "histogram_quantile";
  if (/^sum\s+by\s*\(/.test(s)) return "sum_by";
  if (/^avg_over_time\s*\(/.test(s)) return "avg_over_time";
  if (/^max_over_time\s*\(/.test(s)) return "max_over_time";
  if (/^min_over_time\s*\(/.test(s)) return "min_over_time";
  if (/^increase\s*\(/.test(s)) return "increase";
  if (/^irate\s*\(/.test(s)) return "irate";
  if (/^rate\s*\(/.test(s)) return "rate";
  if (/^ceil\s*\(/.test(s)) return "ceil";
  if (/^floor\s*\(/.test(s)) return "floor";
  if (/^round\s*\(/.test(s)) return "round";
  return null;
}

/** 将 Prometheus instant/range 的 data 段解析为表格（vector / matrix）。 */
export function buildPromTableView(data: unknown): PromTableView | null {
  if (!data || typeof data !== "object") return null;
  const obj = data as Record<string, unknown>;
  const rt = String(obj.resultType ?? "");
  const result = obj.result;
  if (!Array.isArray(result) || result.length === 0) return null;

  if (rt === "vector") {
    const rows: Record<string, string>[] = [];
    const keySet = new Set<string>();
    let k = 0;
    for (const item of result as Array<{ metric?: Record<string, string>; value?: [string, string] }>) {
      const m = item.metric ?? {};
      const val = item.value;
      const row: Record<string, string> = { key: String(k++) };
      for (const [mk, mv] of Object.entries(m)) {
        keySet.add(mk);
        row[mk] = mv;
      }
      row.__timestamp__ = val?.[0] ?? "";
      row.__time_local__ = formatPromTimestampLocal(val?.[0] ?? "");
      row.__value__ = val?.[1] ?? "";
      keySet.add("__timestamp__");
      keySet.add("__time_local__");
      keySet.add("__value__");
      rows.push(row);
    }
    const metricKeys = [...keySet]
      .filter((x) => x !== "__timestamp__" && x !== "__time_local__" && x !== "__value__")
      .sort(sortMetricKeys);
    const columns: ColumnsType<Record<string, string>> = [
      { title: "时间", dataIndex: "__time_local__", width: 180, ellipsis: true },
      { title: "时间戳", dataIndex: "__timestamp__", width: 150, ellipsis: true },
      ...metricKeys.map((name) => ({ title: name, dataIndex: name, ellipsis: true })),
      { title: "Value", dataIndex: "__value__", width: 120 },
    ];
    return { columns, dataSource: rows };
  }

  if (rt === "matrix") {
    const rows: Record<string, string>[] = [];
    const keySet = new Set<string>();
    let k = 0;
    for (const item of result as Array<{ metric?: Record<string, string>; values?: [string, string][] }>) {
      const m = item.metric ?? {};
      const vals = item.values ?? [];
      for (const pair of vals) {
        const row: Record<string, string> = { key: String(k++) };
        for (const [mk, mv] of Object.entries(m)) {
          keySet.add(mk);
          row[mk] = mv;
        }
        row.__timestamp__ = pair?.[0] ?? "";
        row.__time_local__ = formatPromTimestampLocal(pair?.[0] ?? "");
        row.__value__ = pair?.[1] ?? "";
        keySet.add("__timestamp__");
        keySet.add("__time_local__");
        keySet.add("__value__");
        rows.push(row);
      }
    }
    const metricKeys = [...keySet]
      .filter((x) => x !== "__timestamp__" && x !== "__time_local__" && x !== "__value__")
      .sort(sortMetricKeys);
    const columns: ColumnsType<Record<string, string>> = [
      { title: "时间", dataIndex: "__time_local__", width: 180, ellipsis: true },
      { title: "时间戳", dataIndex: "__timestamp__", width: 150, ellipsis: true },
      ...metricKeys.map((name) => ({ title: name, dataIndex: name, ellipsis: true })),
      { title: "Value", dataIndex: "__value__", width: 120 },
    ];
    return { columns, dataSource: rows };
  }

  return null;
}

export function formatPromScalarSummary(data: unknown): string | null {
  if (!data || typeof data !== "object") return null;
  const o = data as Record<string, unknown>;
  if (String(o.resultType) !== "string") return null;
  const r = o.result;
  if (Array.isArray(r) && r.length >= 2) return `结果值：${String(r[1])}（时间戳 ${r[0]}）`;
  return null;
}

/** 后端返回的 Prometheus JSON 可能为 { status, data:{ resultType, ... } }，表格解析取内层 data。 */
export function unwrapPrometheusQueryData(body: unknown): unknown {
  if (!body || typeof body !== "object") return body;
  const o = body as Record<string, unknown>;
  if (o.data && typeof o.data === "object") {
    const d = o.data as Record<string, unknown>;
    if (typeof d.resultType === "string" || Array.isArray(d.result)) return o.data;
  }
  return body;
}
