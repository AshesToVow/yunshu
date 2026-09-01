/**
 * 告警监控平台：URL/路由派生状态（RF-03 第一步拆分产物）
 *
 * 从 `use-alert-monitor-platform-state.tsx` 原地搬迁，逐字保留语义：
 * - Tab 由路由 path 参数决定（`normalizeAlertMonitorTab` 兜底非法值）
 * - 项目上下文与历史事件分类由 query string 派生
 * - 保留 `?tab=xxx`（含 `cfg=history`）的历史 URL 兼容重定向
 *
 * 注意：`policies`（告警策略）Tab 为全局配置，不接受 project_id，
 * 因此 setTab 与 effect 两处都会主动清除该参数——这是刻意的双保险
 * （setTab 处理点击切换，effect 处理直接输入 URL 进入），不要「顺手合并」。
 */
import { useCallback, useEffect, useMemo } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import type { AlertEventCategory } from "../../../utils/alert-event-reasons";
import { normalizeAlertMonitorTab, tabPathForKey, type AlertMonitorTabKey } from "../tab-config";

/** 历史 Tab 允许的事件分类白名单，与后端 alert_event.category 取值一致 */
const HISTORY_EVENT_CATEGORIES: AlertEventCategory[] = [
  "delivery",
  "routing",
  "silence",
  "inhibition",
  "timing",
  "resolved",
  "failure",
  "other",
];

export function useAlertMonitorUrlState() {
  const navigate = useNavigate();
  const { tab: tabParam } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();

  const projectContextId = useMemo(() => {
    const raw = String(searchParams.get("project_id") || "").trim();
    if (!raw) return undefined;
    const n = Number(raw);
    if (!Number.isFinite(n) || n <= 0) return undefined;
    return n;
  }, [searchParams]);

  const historyEventCategory = useMemo((): AlertEventCategory | undefined => {
    const raw = String(searchParams.get("event_category") || "").trim().toLowerCase();
    return (HISTORY_EVENT_CATEGORIES as string[]).includes(raw) ? (raw as AlertEventCategory) : undefined;
  }, [searchParams]);

  const tab: AlertMonitorTabKey = useMemo(() => normalizeAlertMonitorTab(tabParam), [tabParam]);

  // 历史 URL 兼容：`?tab=xxx` / `?tab=config&cfg=history` → 路径式 Tab
  useEffect(() => {
    const legacyTab = searchParams.get("tab");
    if (!legacyTab || tabParam) return;
    let next = normalizeAlertMonitorTab(legacyTab);
    if (legacyTab === "config" && searchParams.get("cfg") === "history") {
      next = "history";
    }
    const qs = new URLSearchParams(searchParams);
    qs.delete("tab");
    qs.delete("cfg");
    const tail = qs.toString();
    const path = tabPathForKey(next);
    navigate(tail ? `${path}?${tail}` : path, { replace: true });
  }, [navigate, searchParams, tabParam]);

  const setTab = useCallback(
    (key: AlertMonitorTabKey) => {
      const qs = new URLSearchParams(searchParams);
      if (key === "policies") {
        qs.delete("project_id");
      }
      const tail = qs.toString();
      const path = tabPathForKey(key);
      navigate(tail ? `${path}?${tail}` : path, { replace: true });
    },
    [navigate, searchParams],
  );

  // 直接以带 project_id 的 URL 进入 policies Tab 时同样清除该参数
  useEffect(() => {
    if (tab !== "policies") return;
    if (!searchParams.has("project_id")) return;
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        p.delete("project_id");
        return p;
      },
      { replace: true },
    );
  }, [tab, searchParams, setSearchParams]);

  const setProjectContext = useCallback(
    (projectID?: number) => {
      setSearchParams(
        (prev) => {
          const p = new URLSearchParams(prev);
          if (projectID && Number.isFinite(projectID) && projectID > 0) p.set("project_id", String(projectID));
          else p.delete("project_id");
          return p;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  const openHistoryTab = useCallback(() => {
    setTab("history");
  }, [setTab]);

  return {
    tab,
    setTab,
    projectContextId,
    setProjectContext,
    historyEventCategory,
    openHistoryTab,
    searchParams,
    setSearchParams,
  };
}
