// @ts-nocheck
/**
 * 告警监控平台：PromQL 查询控制台（promql Tab）状态（RF-03 第二步拆分产物）
 *
 * 从 `use-alert-monitor-platform-state.tsx` 原地搬迁，逐字保留语义：
 * - 即时/区间两种查询模式共用一个结果区，`promResult` 存外层原始 JSON 字符串，
 *   `promDataInner` 存剥掉 Prometheus 包裹后的数据（表格视图依赖它）
 * - 每次查询成功都强制回到表格视图；查询失败时清空表格数据只留错误文本
 * - 数据源默认值：进入 promql Tab 且当前选中项不在列表中时，取首个已启用数据源
 *
 * 注意：规则弹窗里的「指标浏览器」也调用 promInstantQuery，但它依赖 ruleForm，
 * 与本控制台无共享状态，仍留在主 Hook，不要合并到这里。
 */
import { message } from "antd";
import dayjs from "dayjs";
import { useEffect, useMemo, useState } from "react";

import type { AlertDatasourceItem } from "../../../services/alert-platform";
import { promInstantQuery, promRangeQuery } from "../../../services/alert-platform";
import { extractApiErrorMessage } from "../../../services/http";
import { buildPromTableView, formatPromScalarSummary, unwrapPrometheusQueryData } from "../prom-parse";
import type { AlertMonitorTabKey } from "../tab-config";

export function useAlertMonitorPromqlConsoleState(params: {
  /** 当前 Tab：仅 promql Tab 需要兜底选中数据源 */
  tab: AlertMonitorTabKey;
  /** 数据源列表（由主 Hook 按项目上下文加载） */
  dsList: AlertDatasourceItem[];
}) {
  const { tab, dsList } = params;

  const [promDsId, setPromDsId] = useState<number | undefined>();
  const [promMode, setPromMode] = useState<"instant" | "range">("instant");
  const [promQuery, setPromQuery] = useState("up");
  const [promTime, setPromTime] = useState("");
  const [promStart, setPromStart] = useState("");
  const [promEnd, setPromEnd] = useState("");
  const [promStep, setPromStep] = useState("30s");
  const [promResult, setPromResult] = useState<string>("");
  const [promDataInner, setPromDataInner] = useState<unknown>(null);
  const [promViewMode, setPromViewMode] = useState<"table" | "json">("table");
  const [promLoading, setPromLoading] = useState(false);

  const promTableView = useMemo(() => buildPromTableView(promDataInner), [promDataInner]);
  const promScalarText = useMemo(() => formatPromScalarSummary(promDataInner), [promDataInner]);

  useEffect(() => {
    if (tab !== "promql") return;
    if (promDsId != null && dsList.some((d) => d.id === promDsId)) return;
    const first = dsList.find((d) => d.enabled)?.id ?? dsList[0]?.id;
    setPromDsId(first);
  }, [tab, dsList, promDsId]);

  async function runProm() {
    if (!promDsId) {
      message.warning("请选择数据源");
      return;
    }
    setPromLoading(true);
    setPromResult("");
    setPromDataInner(null);
    try {
      if (promMode === "instant") {
        const r = await promInstantQuery(promDsId, { query: promQuery, time: promTime.trim() || undefined });
        const outer = (r as { data?: unknown }).data ?? r;
        const inner = unwrapPrometheusQueryData(outer);
        setPromDataInner(inner);
        setPromResult(JSON.stringify(outer, null, 2));
      } else {
        const r = await promRangeQuery(promDsId, {
          query: promQuery,
          start: promStart.trim(),
          end: promEnd.trim(),
          step: promStep.trim() || "30s",
        });
        const outer = (r as { data?: unknown }).data ?? r;
        const inner = unwrapPrometheusQueryData(outer);
        setPromDataInner(inner);
        setPromResult(JSON.stringify(outer, null, 2));
      }
      setPromViewMode("table");
    } catch (e) {
      setPromResult(extractApiErrorMessage(e, "操作失败"));
      setPromDataInner(null);
    } finally {
      setPromLoading(false);
    }
  }

  function fillPromTimeNow() {
    setPromTime(dayjs().toISOString());
  }

  function fillPromRangeLastHour() {
    const end = dayjs();
    const start = end.subtract(1, "hour");
    setPromStart(start.toISOString());
    setPromEnd(end.toISOString());
    setPromStep("30s");
  }

  return {
    fillPromRangeLastHour,
    fillPromTimeNow,
    promDsId,
    promEnd,
    promLoading,
    promMode,
    promQuery,
    promResult,
    promScalarText,
    promStart,
    promStep,
    promTableView,
    promTime,
    promViewMode,
    runProm,
    setPromDsId,
    setPromEnd,
    setPromMode,
    setPromQuery,
    setPromStart,
    setPromStep,
    setPromTime,
    setPromViewMode,
  };
}
