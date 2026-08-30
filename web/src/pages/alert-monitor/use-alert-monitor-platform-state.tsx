import type { TreeSelectProps } from "antd";
import dayjs from "dayjs";
import "dayjs/locale/zh-cn";
import { useCallback, useEffect, useRef, useState } from "react";
import type { Dispatch, SetStateAction } from "react";
import { getDepartmentTree } from "../../services/departments";
import { getProjects, type ProjectItem } from "../../services/projects";
import { getUsers } from "../../services/users";
import { useDictOptions } from "../../hooks/use-dict-options";

import { useAlertMonitorUrlState } from "./state/use-alert-monitor-url-state";
import { useAlertMonitorCloudExpiryState } from "./state/use-cloud-expiry-state";
import { useAlertMonitorDatasourceState } from "./state/use-datasource-state";
import { useAlertMonitorPromqlConsoleState } from "./state/use-promql-console-state";
import { useAlertMonitorSilenceState } from "./state/use-silence-state";
import { useAlertMonitorRulesState } from "./state/use-rules-state";
import { deptToTreeData, normalizeCloudExpiryLabelsJSON } from "./platform-helpers";

dayjs.locale("zh-cn");


export function useAlertMonitorPlatformState() {
  // URL/路由派生状态（Tab、项目上下文、历史事件分类、历史 URL 兼容）
  // 已抽到 state/use-alert-monitor-url-state.ts，返回字段名保持不变
  const {
    tab,
    setTab,
    projectContextId,
    setProjectContext,
    historyEventCategory,
    openHistoryTab,
  } = useAlertMonitorUrlState();


  // 数据源 Hook 与 PromQL 控制台 Hook 互为依赖（前者加载完要为控制台预选数据源，
  // 后者又要按 dsList 做「当前选中项已失效」兜底），用 ref 转发 setter 打破 Hook
  // 调用顺序上的环；行为与拆分前主 Hook 内直接调用 setPromDsId 完全一致。
  const setPromDsIdRef = useRef<Dispatch<SetStateAction<number | undefined>> | null>(null);
  const applyDefaultPromDatasource = useCallback((firstDatasourceId?: number) => {
    setPromDsIdRef.current?.((prev) => prev ?? firstDatasourceId);
  }, []);

  const [loading, setLoading] = useState(false);
  const [users, setUsers] = useState<Array<{ label: string; value: number }>>([]);
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [deptTree, setDeptTree] = useState<TreeSelectProps["treeData"]>([]);

  const {
    cloudExpiryColumns,
    cloudExpiryCurrent,
    cloudExpiryEvaluating,
    cloudExpiryForm,
    cloudExpiryKeyword,
    cloudExpiryList,
    cloudExpiryModalOpen,
    cloudExpiryProviderFilter,
    cloudExpirySubmitting,
    loadCloudExpiryRules,
    openCloudExpiryCreate,
    runCloudExpiryEvalNow,
    setCloudExpiryKeyword,
    setCloudExpiryModalOpen,
    setCloudExpiryProviderFilter,
    submitCloudExpiryRule,
  } = useAlertMonitorCloudExpiryState({ projectContextId, projects });

  const promqlLabelKeyOpts = useDictOptions("alert_promql_label_key");

  // 数据源（datasources Tab）：状态与操作已下沉到 state/use-datasource-state.tsx
  // 返回字段名保持不变；`loadDatasources` 仍由本文件的 Tab 副作用统一调用
  const {
    dsBasicUserAutoOpts,
    dsColumns,
    dsCurrent,
    dsForm,
    dsList,
    dsModalOpen,
    dsSubmitting,
    dsUrlAutoOpts,
    loadDatasources,
    openDsCreate,
    setDsModalOpen,
    submitDs,
  } = useAlertMonitorDatasourceState({ projectContextId, projects, applyDefaultPromDatasource });

  // 静默（silences Tab）：状态与操作已下沉到 state/use-silence-state.tsx
  // 返回字段名保持不变；`loadSilences` / `loadAmSilences` 仍由本文件的 Tab 副作用调用
  const {
    amSilencesLoading,
    loadAmSilences,
    loadNativeSilAlerts,
    loadSilences,
    nativeAlertsColumns,
    nativeAlertsLoading,
    nativeAlertsRows,
    openQuickSilence,
    openSilCreate,
    openSilenceForEvent,
    openSilenceForMonitorRule,
    quickSilenceComment,
    quickSilenceOpen,
    quickSilenceSubmitting,
    quickSilenceTargets,
    releaseSelectedSilences,
    selectedNativeAlertKeys,
    selectedSilenceIds,
    setQuickSilenceComment,
    setQuickSilenceOpen,
    setQuickSilenceTargets,
    setSelectedNativeAlertKeys,
    setSelectedSilenceIds,
    setSilModalOpen,
    silColumns,
    silCurrent,
    silForm,
    silModalOpen,
    silSubmitting,
    silenceDatasource,
    silenceDatasourceId,
    silenceDisplayList,
    silenceList,
    silenceMatcherNameOptions,
    submitQuickSilence,
    submitSil,
  } = useAlertMonitorSilenceState({ projectContextId, dsList, promqlLabelKeyOpts });

  // PromQL 查询控制台（promql Tab）：状态与操作已下沉到 state/use-promql-console-state.ts
  // 返回字段名保持不变；数据源兜底 effect 也随之搬迁（依赖 tab + dsList）
  const {
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
  } = useAlertMonitorPromqlConsoleState({ tab, dsList });
  // 转发给数据源 Hook 用于「首次加载预选第一条」，见上方 applyDefaultPromDatasource 注释
  setPromDsIdRef.current = setPromDsId;

  // 监控规则 / 值班 / 处理人 / 指标浏览器（rules Tab）
  // 状态与操作已下沉到 state/use-rules-state.tsx；`loadRules` 仍由本文件 Tab 副作用统一调用
  const rules = useAlertMonitorRulesState({
    projectContextId,
    projects,
    users,
    deptTree,
    setTab: setTab as (tab: string) => void,
    dsList,
    promDsId,
    promqlLabelKeyOpts,
    openSilenceForMonitorRule,
  });
  const {
    activeProjectName,
    alertSeverityOpts,
    applyMetricSelectorToRuleExpr,
    applyRuleAnnotationPreset,
    applyRuleBuilderToExpr,
    applyStepwisePromQL,
    assignForm,
    assignOpen,
    assignSubmitting,
    assignUserIds,
    assignUsersHint,
    blkColumns,
    blkCurrent,
    blkForm,
    blkModalOpen,
    blkSubmitting,
    blkUserIds,
    blockList,
    commonLabelKeyOptions,
    copyDutyBlocksFromSelectedRule,
    copyDutyLoading,
    copyDutyRuleOptions,
    copySourceRuleId,
    dutyModalOpen,
    dutyRuleId,
    dutyUsersHint,
    insertPromFunctionToExpr,
    labelValueLoading,
    labelValueOptions,
    loadLabelValuesForRule,
    loadMetricOptionsForRule,
    loadRules,
    onRuleTableChange,
    metricKeyword,
    metricLabelFilters,
    metricLoading,
    metricOptions,
    openBlkCreate,
    openRuleCreate,
    openRuleCreateFromObject,
    projectOptions,
    promFunctionTemplates,
    ruleColumns,
    ruleComparatorOptions,
    ruleConditions,
    ruleCurrent,
    ruleDisplayList,
    ruleEnabledFilter,
    ruleEnabledStats,
    rulePage,
    rulePageSize,
    ruleTotal,
    rulesLoading,
    ruleForm,
    ruleLogic,
    ruleLogicOptions,
    ruleModalOpen,
    ruleSeverityOptions,
    ruleSubmitting,
    ruleTemplatePresetOptions,
    selectedMetric,
    selectedPromFunc,
    selectedPromFuncMeta,
    setAssignOpen,
    setBlkModalOpen,
    setCopySourceRuleId,
    setDutyModalOpen,
    setMetricKeyword,
    setMetricLabelFilters,
    setRuleConditions,
    setRuleEnabledFilter,
    setRuleLogic,
    setRuleModalOpen,
    setSelectedMetric,
    setSelectedPromFunc,
    submitAssign,
    submitBlk,
    submitRule,
    thresholdUnit,
    thresholdUnitOptions,
    usePromFunctionAsConditionMetric,
  } = rules;

  useEffect(() => {
    void (async () => {
      try {
        const [tree, u, projRes] = await Promise.all([getDepartmentTree(), getUsers({ page: 1, page_size: 500 }), getProjects({ page: 1, page_size: 500 })]);
        setDeptTree(deptToTreeData(tree ?? []));
        setUsers(
          (u.list ?? []).map((it) => ({
            value: it.id,
            label: `${it.nickname || it.username} (${it.email || "-"})`,
          })),
        );
        setProjects(projRes.list ?? []);
      } catch {
        /* ignore */
      }
    })();
  }, []);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      setLoading(true);
      try {
        if (tab === "datasources") await loadDatasources(projectContextId);
        if (tab === "promql") await loadDatasources(projectContextId);
        if (tab === "silences") {
          await Promise.all([loadSilences(), loadDatasources(projectContextId)]);
        }
        if (tab === "rules") {
          // 项目或启停筛选变化时回到第 1 页；翻页走 onRuleTableChange，避免整卡 loading 打掉表格状态
          await Promise.all([
            loadDatasources(projectContextId),
            loadRules(projectContextId, { page: 1, enabledFilter: ruleEnabledFilter }),
          ]);
        }
        if (tab === "cloud-expiry") {
          await loadCloudExpiryRules(projectContextId, cloudExpiryProviderFilter, cloudExpiryKeyword);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [
    tab,
    projectContextId,
    ruleEnabledFilter,
    loadDatasources,
    loadSilences,
    loadRules,
    loadCloudExpiryRules,
    cloudExpiryProviderFilter,
    cloudExpiryKeyword,
  ]);

  useEffect(() => {
    if (tab !== "silences") return;
    void loadAmSilences();
  }, [tab, silenceDatasourceId, loadAmSilences]);

  return {
    activeProjectName,
    alertSeverityOpts,
    applyMetricSelectorToRuleExpr,
    applyRuleAnnotationPreset,
    applyRuleBuilderToExpr,
    applyStepwisePromQL,
    assignForm,
    assignOpen,
    assignSubmitting,
    assignUserIds,
    assignUsersHint,
    blkColumns,
    blkCurrent,
    blkForm,
    blkModalOpen,
    blkSubmitting,
    blkUserIds,
    blockList,
    cloudExpiryColumns,
    cloudExpiryCurrent,
    cloudExpiryEvaluating,
    cloudExpiryForm,
    cloudExpiryKeyword,
    cloudExpiryList,
    cloudExpiryModalOpen,
    cloudExpiryProviderFilter,
    cloudExpirySubmitting,
    commonLabelKeyOptions,
    copyDutyBlocksFromSelectedRule,
    copyDutyLoading,
    copyDutyRuleOptions,
    copySourceRuleId,
    deptTree,
    dsBasicUserAutoOpts,
    dsColumns,
    dsCurrent,
    dsForm,
    dsList,
    dsModalOpen,
    dsSubmitting,
    dsUrlAutoOpts,
    dutyModalOpen,
    dutyRuleId,
    dutyUsersHint,
    fillPromRangeLastHour,
    fillPromTimeNow,
    historyEventCategory,
    insertPromFunctionToExpr,
    labelValueLoading,
    labelValueOptions,
    loadCloudExpiryRules,
    loadDatasources,
    loadLabelValuesForRule,
    loadMetricOptionsForRule,
    loadNativeSilAlerts,
    loadRules,
    onRuleTableChange,
    loadAmSilences,
    loadSilences,
    loading,
    metricKeyword,
    metricLabelFilters,
    metricLoading,
    metricOptions,
    nativeAlertsColumns,
    nativeAlertsLoading,
    nativeAlertsRows,
    normalizeCloudExpiryLabelsJSON,
    openBlkCreate,
    openCloudExpiryCreate,
    openDsCreate,
    openHistoryTab,
    openQuickSilence,
    openSilCreate,
    openSilenceForEvent,
    openRuleCreate,
    openRuleCreateFromObject,
    projectContextId,
    projectOptions,
    promDsId,
    promEnd,
    promFunctionTemplates,
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
    quickSilenceComment,
    quickSilenceOpen,
    quickSilenceSubmitting,
    quickSilenceTargets,
    releaseSelectedSilences,
    ruleColumns,
    ruleComparatorOptions,
    ruleConditions,
    ruleCurrent,
    ruleDisplayList,
    ruleEnabledFilter,
    ruleEnabledStats,
    rulePage,
    rulePageSize,
    ruleTotal,
    rulesLoading,
    ruleForm,
    ruleLogic,
    ruleLogicOptions,
    ruleModalOpen,
    ruleSeverityOptions,
    ruleSubmitting,
    ruleTemplatePresetOptions,
    runCloudExpiryEvalNow,
    runProm,
    selectedMetric,
    selectedNativeAlertKeys,
    selectedPromFunc,
    selectedPromFuncMeta,
    selectedSilenceIds,
    setAssignOpen,
    setBlkModalOpen,
    setCloudExpiryKeyword,
    setCloudExpiryModalOpen,
    setCloudExpiryProviderFilter,
    setCopySourceRuleId,
    setDsModalOpen,
    setDutyModalOpen,
    setMetricKeyword,
    setMetricLabelFilters,
    setProjectContext,
    setPromDsId,
    setPromEnd,
    setPromMode,
    setPromQuery,
    setPromStart,
    setPromStep,
    setPromTime,
    setPromViewMode,
    setQuickSilenceComment,
    setQuickSilenceOpen,
    setQuickSilenceTargets,
    setRuleConditions,
    setRuleEnabledFilter,
    setRuleLogic,
    setRuleModalOpen,
    setSelectedMetric,
    setSelectedNativeAlertKeys,
    setSelectedPromFunc,
    setSelectedSilenceIds,
    setSilModalOpen,
    setTab,
    silColumns,
    silCurrent,
    silForm,
    silModalOpen,
    silSubmitting,
    silenceDatasource,
    silenceDatasourceId,
    amSilencesLoading,
    silenceDisplayList,
    silenceList,
    silenceMatcherNameOptions,
    submitAssign,
    submitBlk,
    submitCloudExpiryRule,
    submitDs,
    submitQuickSilence,
    submitRule,
    submitSil,
    tab,
    thresholdUnit,
    thresholdUnitOptions,
    usePromFunctionAsConditionMetric,
    users,
  };
}
