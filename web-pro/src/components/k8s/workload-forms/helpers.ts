// @ts-nocheck
import YAML from "yaml";


export type EnvPair = { key?: string; value?: string };

export function envPairsToMap(pairs?: EnvPair[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const p of pairs ?? []) {
    const k = String(p?.key ?? "").trim();
    if (!k) continue;
    out[k] = String(p?.value ?? "");
  }
  return out;
}

export function mapToEnvPairs(m?: Record<string, string>): EnvPair[] {
  const out: EnvPair[] = [];
  for (const [k, v] of Object.entries(m ?? {})) out.push({ key: k, value: v });
  return out.length ? out : [{ key: "", value: "" }];
}

export function safeParseYaml(yaml: string): any | null {
  try {
    return YAML.parse(yaml);
  } catch {
    return null;
  }
}

export function safeGet(obj: any, path: string): any {
  const parts = path.split(".").filter(Boolean);
  let cur = obj;
  for (const p of parts) {
    if (cur == null) return undefined;
    if (p.endsWith("]")) {
      const m = p.match(/^(\w+)\[(\d+)\]$/);
      if (!m) return undefined;
      const key = m[1];
      const idx = Number(m[2]);
      cur = cur?.[key]?.[idx];
      continue;
    }
    cur = cur?.[p];
  }
  return cur;
}

export function toNumberOrUndefined(v: any): number | undefined {
  if (typeof v === "number" && Number.isFinite(v)) return v;
  if (typeof v === "string") {
    const s = v.trim();
    if (!s) return undefined;
    const n = Number(s);
    return Number.isFinite(n) ? n : undefined;
  }
  return undefined;
}

export type ProbeType = "httpGet" | "tcpSocket" | "exec";
export type KVPair = { key?: string; value?: string };

export function kvPairsToMap(pairs?: KVPair[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const p of pairs ?? []) {
    const k = String(p?.key ?? "").trim();
    if (!k) continue;
    out[k] = String(p?.value ?? "").trim();
  }
  return out;
}

export function mapToKvPairs(m?: Record<string, string>): KVPair[] {
  const out = Object.entries(m ?? {}).map(([key, value]) => ({ key, value: String(value ?? "") }));
  return out.length ? out : [{ key: "", value: "" }];
}

export function parseIntOrStringPort(v: any): number | string | undefined {
  if (typeof v === "number" && Number.isFinite(v) && v > 0) return Math.trunc(v);
  const s = String(v ?? "").trim();
  if (!s) return undefined;
  if (/^\d+$/.test(s)) {
    const n = Number(s);
    if (Number.isFinite(n) && n > 0) return n;
  }
  return s;
}

export function parseExecCommandJson(v?: string): string[] | undefined {
  const s = String(v ?? "").trim();
  if (!s) return undefined;
  try {
    const arr = JSON.parse(s);
    if (!Array.isArray(arr)) return undefined;
    const out = arr.map((x) => String(x)).filter((x) => x.trim() !== "");
    return out.length ? out : undefined;
  } catch {
    return undefined;
  }
}

export function probeFromForm(prefix: "liveness" | "readiness" | "startup", v: any): any | undefined {
  const probeType: ProbeType | undefined = v?.[`${prefix}_probe_type`];
  if (!probeType) return undefined;

  const initialDelaySeconds = toNumberOrUndefined(v?.[`${prefix}_initial_delay_seconds`]);
  const periodSeconds = toNumberOrUndefined(v?.[`${prefix}_period_seconds`]);
  const timeoutSeconds = toNumberOrUndefined(v?.[`${prefix}_timeout_seconds`]);
  const failureThreshold = toNumberOrUndefined(v?.[`${prefix}_failure_threshold`]);
  const successThreshold = toNumberOrUndefined(v?.[`${prefix}_success_threshold`]);

  if (probeType === "httpGet") {
    const port = parseIntOrStringPort(v?.[`${prefix}_http_port`]);
    if (!port) return undefined;
    const path = String(v?.[`${prefix}_http_path`] ?? "").trim() || "/";
    const scheme = String(v?.[`${prefix}_http_scheme`] ?? "").trim() || undefined;
    return {
      httpGet: { path, port, ...(scheme ? { scheme } : {}) },
      ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
      ...(periodSeconds !== undefined ? { periodSeconds } : {}),
      ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
      ...(failureThreshold !== undefined ? { failureThreshold } : {}),
      ...(successThreshold !== undefined ? { successThreshold } : {}),
    };
  }

  if (probeType === "tcpSocket") {
    const port = parseIntOrStringPort(v?.[`${prefix}_tcp_port`]);
    if (!port) return undefined;
    return {
      tcpSocket: { port },
      ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
      ...(periodSeconds !== undefined ? { periodSeconds } : {}),
      ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
      ...(failureThreshold !== undefined ? { failureThreshold } : {}),
      ...(successThreshold !== undefined ? { successThreshold } : {}),
    };
  }

  if (probeType === "exec") {
    const command = parseExecCommandJson(v?.[`${prefix}_exec_command`]);
    if (!command?.length) return undefined;
    return {
      exec: { command },
      ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
      ...(periodSeconds !== undefined ? { periodSeconds } : {}),
      ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
      ...(failureThreshold !== undefined ? { failureThreshold } : {}),
      ...(successThreshold !== undefined ? { successThreshold } : {}),
    };
  }

  return undefined;
}

export function probeToForm(prefix: "liveness" | "readiness" | "startup", probe: any): Record<string, unknown> {
  if (!probe) return {};
  const out: Record<string, unknown> = {};
  const type: ProbeType | undefined = probe?.httpGet ? "httpGet" : probe?.tcpSocket ? "tcpSocket" : probe?.exec ? "exec" : undefined;
  if (!type) return out;
  out[`${prefix}_probe_type`] = type;
  if (type === "httpGet") {
    out[`${prefix}_http_path`] = probe?.httpGet?.path != null ? String(probe.httpGet.path) : "";
    out[`${prefix}_http_port`] = probe?.httpGet?.port != null ? String(probe.httpGet.port) : undefined;
    out[`${prefix}_http_scheme`] = probe?.httpGet?.scheme ? String(probe.httpGet.scheme) : undefined;
  }
  if (type === "tcpSocket") {
    out[`${prefix}_tcp_port`] = probe?.tcpSocket?.port != null ? String(probe.tcpSocket.port) : undefined;
  }
  if (type === "exec") {
    out[`${prefix}_exec_command`] = Array.isArray(probe?.exec?.command) ? JSON.stringify(probe.exec.command) : "";
  }
  out[`${prefix}_initial_delay_seconds`] = toNumberOrUndefined(probe?.initialDelaySeconds);
  out[`${prefix}_period_seconds`] = toNumberOrUndefined(probe?.periodSeconds);
  out[`${prefix}_timeout_seconds`] = toNumberOrUndefined(probe?.timeoutSeconds);
  out[`${prefix}_failure_threshold`] = toNumberOrUndefined(probe?.failureThreshold);
  out[`${prefix}_success_threshold`] = toNumberOrUndefined(probe?.successThreshold);
  return out;
}

