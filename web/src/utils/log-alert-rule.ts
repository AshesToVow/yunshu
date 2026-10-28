import type { Dayjs } from "dayjs";

export type LogAlertMode = "error_count" | "error_spike";

export interface LogAlertRuleDraft {
  projectId?: number;
  name: string;
  mode: LogAlertMode;
  level?: string;
  keyword?: string;
  serviceName?: string;
  namespace?: string;
  windowMinutes?: number;
  threshold?: number;
  baselineMinutes?: number;
  spikeRatio?: number;
  severity?: string;
  evalIntervalSeconds?: number;
}

export const LOG_ALERT_WINDOW_MAX_MINUTES = 120;

export function clampLogAlertWindow(minutes?: number): number {
  const n = Math.floor(Number(minutes));
  if (!Number.isFinite(n) || n <= 0) return 5;
  return Math.min(LOG_ALERT_WINDOW_MAX_MINUTES, Math.max(1, n));
}

/** 把日志检索时间范围折算为告警评估窗口（分钟），超出后端上限时截断。 */
export function inferLogAlertWindow(range?: [Dayjs, Dayjs] | null): number {
  if (!range?.[0] || !range?.[1]) return 5;
  const minutes = range[1].diff(range[0], "minute");
  return clampLogAlertWindow(minutes);
}

export function buildLogAlertRulePayload(draft: LogAlertRuleDraft): Record<string, unknown> {
  const projectId = Number(draft.projectId || 0);
  if (!projectId) throw new Error("请先在检索页选择项目");
  const name = String(draft.name || "").trim();
  if (!name) throw new Error("请填写规则名称");

  const windowMinutes = clampLogAlertWindow(draft.windowMinutes);
  const threshold = Math.max(1, Math.floor(Number(draft.threshold) || 10));
  const mode: LogAlertMode = draft.mode === "error_spike" ? "error_spike" : "error_count";
  const level = String(draft.level || "").trim().toUpperCase();
  const keyword = String(draft.keyword || "").trim();
  const serviceName = String(draft.serviceName || "").trim();
  const namespace = String(draft.namespace || "").trim();

  const config: Record<string, unknown> = {
    mode,
    level,
    keyword,
    service_name: serviceName,
    namespace,
    window_minutes: windowMinutes,
    threshold,
  };
  if (mode === "error_spike") {
    config.baseline_minutes = Math.max(windowMinutes, Math.floor(Number(draft.baselineMinutes) || windowMinutes * 6));
    config.spike_ratio = Math.max(1.1, Number(draft.spikeRatio) || 3);
  }

  const scope: string[] = [];
  if (level) scope.push(`level=${level}`);
  if (serviceName) scope.push(`service=${serviceName}`);
  if (namespace) scope.push(`namespace=${namespace}`);
  if (keyword) scope.push(`keyword=${keyword}`);
  const condition =
    mode === "error_spike"
      ? `近 ${windowMinutes} 分钟日志量 ≥ 基线 ${config.baseline_minutes} 分钟均值 × ${config.spike_ratio} 且 ≥ ${threshold} 条`
      : `近 ${windowMinutes} 分钟匹配日志 ≥ ${threshold} 条`;

  return {
    datasource_id: 0,
    project_id: projectId,
    name,
    rule_kind: "log",
    expr: JSON.stringify(config),
    for_seconds: 0,
    eval_interval_seconds: Math.max(5, Math.floor(Number(draft.evalIntervalSeconds) || 60)),
    severity: String(draft.severity || "").trim() || "warning",
    threshold_unit: "count",
    labels_json: JSON.stringify({ category: "log", rule_kind: "log" }),
    annotations_json: JSON.stringify({
      summary: `${name}：${condition}`,
      description: `日志告警（${scope.length ? scope.join(", ") : "全量日志"}）`,
    }),
    enabled: true,
  };
}
