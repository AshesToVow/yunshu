// @ts-nocheck
/**
 * 告警监控平台：监控规则 / 值班 / 处理人 / 指标浏览器（rules Tab）状态（RF-03 第四步拆分产物）
 *
 * 从 `use-alert-monitor-platform-state.tsx` 原地搬迁，逐字保留语义。
 * `loadRules` 仍由主 Hook 的 Tab 副作用统一调用（与 datasources 一并拉取），
 * 因此这里只暴露方法、不自建 Tab 级 effect。
 *
 * `promqlLabelKeyOpts` 由主 Hook 传入（静默 Tab 也用同一字典，避免重复请求）。
 */
import {
  DeleteOutlined,
  EditOutlined,
  CalendarOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import type { TreeSelectProps } from "antd";
import { Button, Form, Popconfirm, Space, Tag, message } from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useDictOptions } from "../../../hooks/use-dict-options";
import { extractApiErrorMessage } from "../../../services/http";
import {
  createAlertMonitorRule,
  createDutyBlock,
  deleteAlertMonitorRule,
  deleteDutyBlock,
  getMonitorRuleAssignees,
  listAlertMonitorRules,
  listDutyBlocks,
  promInstantQuery,
  updateAlertMonitorRule,
  updateDutyBlock,
  upsertMonitorRuleAssignees,
  type AlertDutyBlockItem,
  type AlertDatasourceItem,
  type AlertMonitorRuleItem,
} from "../../../services/alert-platform";
import { stringifyPrettyJSON } from "../../../services/alert-mappers";
import type { ProjectItem } from "../../../services/projects";
import type { UserUpdatePayload } from "../../../types/api";
import { getUser, updateUser } from "../../../services/users";
import { formatDateTime } from "../../../utils/format";
import { DEFAULT_PAGE_SIZE } from "../../../utils/table-pagination";

import type {
  MetricLabelFilter,
  RuleBuilderCondition,
  RuleBuilderLogic,
  RuleComparator,
} from "../platform-provider-types";
import {
  buildPromSelectorExpr,
  detectPromFunctionKeyFromExpr,
  isValidPromLabelKey,
  parsePromSelectorExpr,
  unwrapPrometheusQueryData,
} from "../prom-parse";
import { normalizeCloudExpiryLabelsJSON } from "../platform-helpers";
import { buildRuleExprByConditions, parseRuleBuilderExpr, parseTemplatePresetPair } from "../rule-parse";

export function useAlertMonitorRulesState(params: {
  projectContextId?: number;
  projects: ProjectItem[];
  users: Array<{ label: string; value: number }>;
  deptTree: TreeSelectProps["treeData"];
  setTab: (tab: string) => void; // AlertMonitorTabKey at call site
  dsList: AlertDatasourceItem[];
  promDsId?: number;
  /** 与静默 Tab 共用，由主 Hook 请求后传入 */
  promqlLabelKeyOpts: Array<{ label: string; value: string | number }>;
  /** 规则列表操作列「静默」入口，来自 silences Hook */
  openSilenceForMonitorRule: (r: AlertMonitorRuleItem) => void;
}) {
  const {
    projectContextId,
    projects,
    setTab,
    dsList,
    promqlLabelKeyOpts,
    openSilenceForMonitorRule,
  } = params;
  // users / deptTree / promDsId 由调用方传入以保持与搬迁前闭包一致；表单 JSX 在主 Provider 消费方读取，本 Hook 内不直接引用。
  void params.users;
  void params.deptTree;
  void params.promDsId;

  const alertSeverityOpts = useDictOptions("alert_severity");
  const thresholdUnitDictOpts = useDictOptions("alert_threshold_unit");
  const ruleTemplatePresetDictOpts = useDictOptions("alert_rule_template_preset");

  const [ruleList, setRuleList] = useState<AlertMonitorRuleItem[]>([]);
  /** 监控规则列表：全部 / 仅启用 / 仅停用 */
  const [ruleEnabledFilter, setRuleEnabledFilter] = useState<"all" | "enabled" | "disabled">("all");
  const [rulePage, setRulePage] = useState(1);
  const [rulePageSize, setRulePageSize] = useState(DEFAULT_PAGE_SIZE);
  const [ruleTotal, setRuleTotal] = useState(0);
  const [ruleEnabledStats, setRuleEnabledStats] = useState({ total: 0, enabled: 0, disabled: 0 });
  const [rulesLoading, setRulesLoading] = useState(false);
  const ruleQueryRef = useRef({
    page: 1,
    pageSize: DEFAULT_PAGE_SIZE,
    filter: "all" as "all" | "enabled" | "disabled",
  });
  ruleQueryRef.current = { page: rulePage, pageSize: rulePageSize, filter: ruleEnabledFilter };
  const [blockList, setBlockList] = useState<AlertDutyBlockItem[]>([]);
  const [dutyRuleId, setDutyRuleId] = useState<number | null>(null);
  const [dutyModalOpen, setDutyModalOpen] = useState(false);
  /** 规则值班弹窗：从其他规则复制班次时的来源规则 ID */
  const [copySourceRuleId, setCopySourceRuleId] = useState<number | undefined>();
  const [copyDutyLoading, setCopyDutyLoading] = useState(false);
  const [dutyCopyRuleOptions, setDutyCopyRuleOptions] = useState<Array<{ label: string; value: number }>>([]);


  const [ruleModalOpen, setRuleModalOpen] = useState(false);
  const [ruleCurrent, setRuleCurrent] = useState<AlertMonitorRuleItem | null>(null);
  const [ruleForm] = Form.useForm();
  const [ruleSubmitting, setRuleSubmitting] = useState(false);
  const [ruleLogic, setRuleLogic] = useState<RuleBuilderLogic>("and");
  const [ruleConditions, setRuleConditions] = useState<RuleBuilderCondition[]>([{ metric: "", comparator: ">", threshold: null }]);

  const [assignOpen, setAssignOpen] = useState(false);
  const [assignRuleId, setAssignRuleId] = useState<number | null>(null);
  const [assignForm] = Form.useForm();
  const [assignSubmitting, setAssignSubmitting] = useState(false);
  const assignUserIds = Form.useWatch("user_ids", assignForm) as number[] | undefined;
  const assignSyncedKeyRef = useRef("");
  const [assignProfileOriginal, setAssignProfileOriginal] = useState<{ email: string; department_id?: number } | null>(null);
  const [assignUsersHint, setAssignUsersHint] = useState("");

  const [blkModalOpen, setBlkModalOpen] = useState(false);
  const [blkCurrent, setBlkCurrent] = useState<AlertDutyBlockItem | null>(null);
  const [blkForm] = Form.useForm();
  const [blkSubmitting, setBlkSubmitting] = useState(false);
  const blkUserIds = Form.useWatch("user_ids", blkForm) as number[] | undefined;
  const dutySyncedKeyRef = useRef<string>("");
  const [dutyProfileOriginal, setDutyProfileOriginal] = useState<{ email: string; department_id?: number } | null>(null);
  const [dutyUsersHint, setDutyUsersHint] = useState<string>("");

  const ruleComparatorOptions = useMemo(
    () => [
      { label: "大于 (>)", value: ">" },
      { label: "大于等于 (>=)", value: ">=" },
      { label: "小于 (<)", value: "<" },
      { label: "小于等于 (<=)", value: "<=" },
      { label: "等于 (==)", value: "==" },
      { label: "不等于 (!=)", value: "!=" },
    ],
    [],
  );
  const ruleLogicOptions = useMemo(
    () => [
      { label: "AND（且）", value: "and" },
      { label: "OR（或）", value: "or" },
    ],
    [],
  );

  const [metricKeyword, setMetricKeyword] = useState("");
  const [metricLoading, setMetricLoading] = useState(false);
  const [metricOptions, setMetricOptions] = useState<string[]>([]);
  const [selectedMetric, setSelectedMetric] = useState("");
  const [metricLabelFilters, setMetricLabelFilters] = useState<MetricLabelFilter[]>([{ key: "instance", op: "=", value: "" }]);
  const [labelValueLoading, setLabelValueLoading] = useState(false);
  const [labelValueOptions, setLabelValueOptions] = useState<string[]>([]);
  const [selectedPromFunc, setSelectedPromFunc] = useState<string>("none");

  const projectOptions = useMemo(() => projects.map((p) => ({ label: `${p.name} (${p.code})`, value: p.id })), [projects]);
  const activeProjectName = useMemo(() => {
    if (!projectContextId) return "";
    const p = projects.find((it) => it.id === projectContextId);
    return p ? `${p.name} (${p.code})` : `项目 ${projectContextId}`;
  }, [projects, projectContextId]);

  const ruleSeverityOptions = useMemo(() => {
    const s = ruleCurrent?.severity?.trim();
    const base = alertSeverityOpts;
    if (!s || base.some((o) => String(o.value) === s)) return base;
    return [...base, { label: `${s}（当前规则）`, value: s }];
  }, [alertSeverityOpts, ruleCurrent?.severity]);
  const commonLabelKeyOptions = useMemo(() => {
    const defaults = ["instance", "job", "cluster", "namespace", "pod", "service", "node", "severity", "alertname", "path", "device", "fstype", "mountpoint"];
    const merged = new Set<string>(defaults);
    promqlLabelKeyOpts.forEach((o) => merged.add(String(o.value || "").trim()));
    return Array.from(merged)
      .filter(Boolean)
      .sort((a, b) => a.localeCompare(b))
      .map((k) => ({ label: k, value: k }));
  }, [promqlLabelKeyOpts]);
  const thresholdUnitOptions = useMemo(() => {
    const defaults = [
      { label: "原始值", value: "raw" },
      { label: "百分比 (%)", value: "percent" },
      { label: "字节 (bytes)", value: "bytes" },
      { label: "毫秒 (ms)", value: "ms" },
      { label: "计数 (count)", value: "count" },
    ];
    const merged = [...defaults];
    thresholdUnitDictOpts.forEach((o) => {
      const v = String(o.value || "").trim();
      if (!v) return;
      if (!merged.some((it) => it.value === v)) {
        merged.push({ label: String(o.label || v), value: v });
      }
    });
    if (!merged.some((it) => it.value === "precent")) {
      merged.push({ label: "百分比（兼容旧拼写 precent）", value: "precent" });
    }
    return merged;
  }, [thresholdUnitDictOpts]);
  const thresholdUnit = Form.useWatch("threshold_unit", ruleForm) as string | undefined;
  const ruleTemplatePresetOptions = useMemo(() => {
    const base = [
      { label: "智能推荐（按规则名/PromQL）", value: "smart" },
      { label: "通用阈值告警", value: "generic" },
      { label: "可用性/组件状态", value: "availability" },
      { label: "错误率/失败率", value: "error_rate" },
      { label: "延迟/响应时间", value: "latency" },
      { label: "队列积压/消费延迟", value: "queue_lag" },
      { label: "流量突增/突降", value: "traffic" },
      { label: "证书到期", value: "certificate" },
      { label: "CPU 资源", value: "cpu" },
      { label: "内存资源", value: "memory" },
      { label: "磁盘资源", value: "disk" },
    ];
    const custom = ruleTemplatePresetDictOpts
      .map((o) => ({ label: `自定义 · ${String(o.label || "").trim() || String(o.value || "").trim()}`, value: `custom:${String(o.value || "")}` }))
      .filter((o) => String(o.value).trim() !== "custom:");
    return [...base, ...custom];
  }, [ruleTemplatePresetDictOpts]);
  const promFunctionTemplates = useMemo(
    () => [
      { key: "none", label: "不使用函数", template: "__METRIC__", desc: "直接使用指标与标签过滤，不包裹 Prometheus 函数。" },
      { key: "rate", label: "rate()", template: "rate(__METRIC__[5m])", desc: "计算窗口内每秒增长率，常用于 counter 指标。" },
      { key: "irate", label: "irate()", template: "irate(__METRIC__[5m])", desc: "基于最近两点计算瞬时速率，波动更灵敏。" },
      { key: "increase", label: "increase()", template: "increase(__METRIC__[5m])", desc: "计算窗口内增长总量。" },
      { key: "ceil", label: "ceil()", template: "ceil(__METRIC__)", desc: "向上取整。" },
      { key: "floor", label: "floor()", template: "floor(__METRIC__)", desc: "向下取整。" },
      { key: "round", label: "round()", template: "round(__METRIC__, 0.1)", desc: "按给定精度四舍五入。" },
      { key: "avg_over_time", label: "avg_over_time()", template: "avg_over_time(__METRIC__[5m])", desc: "时间窗口平均值，适合平滑抖动。" },
      { key: "max_over_time", label: "max_over_time()", template: "max_over_time(__METRIC__[5m])", desc: "时间窗口最大值。" },
      { key: "min_over_time", label: "min_over_time()", template: "min_over_time(__METRIC__[5m])", desc: "时间窗口最小值。" },
      { key: "sum_by", label: "sum by()", template: "sum by (instance) (__METRIC__)", desc: "按标签聚合求和。" },
      { key: "histogram_quantile", label: "histogram_quantile()", template: "histogram_quantile(0.95, sum by (le) (__METRIC__))", desc: "直方图分位数计算（如 P95）。" },
    ],
    [],
  );
  const selectedPromFuncMeta = useMemo(
    () => promFunctionTemplates.find((it) => it.key === selectedPromFunc) ?? promFunctionTemplates[0],
    [promFunctionTemplates, selectedPromFunc],
  );
  function tryFillRuleBuilderFromExpr(exprRaw: string) {
    const parsed = parseRuleBuilderExpr(exprRaw);
    if (!parsed) {
      setRuleLogic("and");
      setRuleConditions([{ metric: "", comparator: ">", threshold: null }]);
      return;
    }
    setRuleLogic(parsed.logic);
    setRuleConditions(parsed.conditions);
  }

  function applyRuleBuilderToExpr() {
    if (!ruleConditions.length) {
      message.warning("请至少添加一个条件");
      return;
    }
    if (ruleConditions.some((c) => !String(c.metric || "").trim() || c.threshold === null || Number.isNaN(c.threshold))) {
      message.warning("请完善每个条件的指标表达式和阈值");
      return;
    }
    ruleForm.setFieldValue("expr", buildRuleExprByConditions(ruleConditions, ruleLogic));
  }

  const fillAssignFromUserIds = useCallback(
    async (ids: number[] | undefined) => {
      if (!ids?.length) {
        assignForm.setFieldsValue({ profile_email: undefined });
        setAssignProfileOriginal(null);
        setAssignUsersHint("");
        return;
      }
      try {
        const details = await Promise.all(ids.map((id) => getUser(id)));
        setAssignUsersHint(details.map((u) => `${u.nickname || u.username}：${u.email || "（无邮箱）"}`).join("；"));
        if (ids.length === 1) {
          const u = details[0];
          const em = (u.email ?? "").trim();
          assignForm.setFieldsValue({ profile_email: em });
          setAssignProfileOriginal({ email: em, department_id: u.department_id });
        } else {
          assignForm.setFieldsValue({ profile_email: undefined });
          setAssignProfileOriginal(null);
        }
      } catch {
        setAssignUsersHint("");
      }
    },
    [assignForm],
  );

  useEffect(() => {
    if (!assignOpen) {
      assignSyncedKeyRef.current = "";
      return;
    }
    const key = (assignUserIds ?? []).join(",");
    if (key === assignSyncedKeyRef.current) return;
    assignSyncedKeyRef.current = key;
    void fillAssignFromUserIds(assignUserIds);
  }, [assignOpen, assignUserIds, fillAssignFromUserIds]);

  const fillDutyFromUserIds = useCallback(
    async (ids: number[] | undefined) => {
      if (!ids?.length) {
        blkForm.setFieldsValue({ profile_email: undefined });
        setDutyProfileOriginal(null);
        setDutyUsersHint("");
        return;
      }
      try {
        const details = await Promise.all(ids.map((id) => getUser(id)));
        setDutyUsersHint(details.map((u) => `${u.nickname || u.username}：${u.email || "（无邮箱）"}`).join("；"));
        if (ids.length === 1) {
          const u = details[0];
          const em = (u.email ?? "").trim();
          blkForm.setFieldsValue({ profile_email: em });
          setDutyProfileOriginal({ email: em, department_id: u.department_id });
        } else {
          blkForm.setFieldsValue({ profile_email: undefined });
          setDutyProfileOriginal(null);
        }
      } catch {
        setDutyUsersHint("");
      }
    },
    [blkForm],
  );

  useEffect(() => {
    if (!blkModalOpen) {
      dutySyncedKeyRef.current = "";
      return;
    }
    const key = (blkUserIds ?? []).join(",");
    if (key === dutySyncedKeyRef.current) return;
    dutySyncedKeyRef.current = key;
    void fillDutyFromUserIds(blkUserIds);
  }, [blkModalOpen, blkUserIds, fillDutyFromUserIds]);

  const loadRules = useCallback(
    async (
      projectID?: number,
      opts?: {
        page?: number;
        pageSize?: number;
        enabledFilter?: "all" | "enabled" | "disabled";
      },
    ) => {
      const q = ruleQueryRef.current;
      const page = opts?.page ?? q.page;
      const pageSize = Math.min(Math.max(opts?.pageSize ?? q.pageSize, 1), 100);
      const filter = opts?.enabledFilter ?? q.filter;
      const enabled = filter === "all" ? undefined : filter === "enabled";
      const base = {
        project_id: projectID && projectID > 0 ? projectID : undefined,
      };
      setRulesLoading(true);
      try {
        const [r, allR, enR, disR] = await Promise.all([
          listAlertMonitorRules({ ...base, page, page_size: pageSize, enabled }),
          listAlertMonitorRules({ ...base, page: 1, page_size: 1 }),
          listAlertMonitorRules({ ...base, page: 1, page_size: 1, enabled: true }),
          listAlertMonitorRules({ ...base, page: 1, page_size: 1, enabled: false }),
        ]);
        setRuleList(r.list ?? r.items ?? []);
        setRuleTotal(Number(r.total) || 0);
        setRulePage(page);
        setRulePageSize(pageSize);
        setRuleEnabledStats({
          total: Number(allR.total) || 0,
          enabled: Number(enR.total) || 0,
          disabled: Number(disR.total) || 0,
        });
      } finally {
        setRulesLoading(false);
      }
    },
    [],
  );

  const onRuleTableChange = useCallback(
    (page: number, pageSize: number) => {
      void loadRules(projectContextId, { page, pageSize });
    },
    [loadRules, projectContextId],
  );

  function openRuleCreate() {
    setRuleCurrent(null);
    setRuleLogic("and");
    setRuleConditions([{ metric: "", comparator: ">", threshold: null }]);
    ruleForm.resetFields();
    const firstDs = dsList[0]?.id;
    ruleForm.setFieldsValue({
      datasource_id: firstDs,
      for_seconds: 0,
      eval_interval_seconds: 30,
      severity: "warning",
      threshold_unit: "percent",
      labels_json: "{}",
      summary_template: "{{$labels.instance}}: {{.RuleName}} 告警触发，当前值 {{$value}}",
      description_template: "规则 {{.RuleName}} 触发，PromQL={{.Expr}}，实例={{$labels.instance}}，当前值={{$value}}",
      rule_template_preset: "generic",
      enabled: true,
    });
    setMetricKeyword("");
    setMetricOptions([]);
    setSelectedMetric("");
    setMetricLabelFilters([{ key: "instance", op: "=", value: "" }]);
    setSelectedPromFunc("none");
    setLabelValueOptions([]);
    setRuleModalOpen(true);
  }

  function openRuleCreateFromObject(obj: {
    service_name?: string;
    service_id?: string;
    address?: string;
    port?: number;
    exporter_role?: string;
    yunshu_project?: string;
  }) {
    openRuleCreate();
    const instance = obj.address && obj.port ? `${obj.address}:${obj.port}` : obj.address || "";
    const labels: Record<string, string> = {};
    if (obj.service_name) labels.job = String(obj.service_name);
    if (instance) labels.instance = instance;
    if (obj.exporter_role) labels.exporter_role = String(obj.exporter_role);
    if (obj.yunshu_project) labels.yunshu_project = String(obj.yunshu_project);
    const nameHint = [obj.service_name, instance].filter(Boolean).join(" · ") || "监控对象";
    ruleForm.setFieldsValue({
      name: `${nameHint} 可用性`,
      labels_json: JSON.stringify(labels, null, 2),
      summary_template: `${nameHint}：{{.RuleName}} 触发，当前值 {{$value}}`,
    });
    if (instance) {
      setMetricLabelFilters([{ key: "instance", op: "=", value: instance }]);
    }
    setTab("rules");
  }

  function detectRulePresetByContext(): string {
    const name = String(ruleForm.getFieldValue("name") || "").toLowerCase();
    const expr = String(ruleForm.getFieldValue("expr") || "").toLowerCase();
    const text = `${name} ${expr}`;
    if (/cpu|load/.test(text)) return "cpu";
    if (/mem|memory|oom/.test(text)) return "memory";
    if (/disk|inode|filesystem|fs_/.test(text)) return "disk";
    if (/latency|duration|response|p95|p99/.test(text)) return "latency";
    if (/error|5xx|4xx|fail|exception/.test(text)) return "error_rate";
    if (/queue|lag|backlog|consumer/.test(text)) return "queue_lag";
    if (/traffic|qps|rps|throughput|request/.test(text)) return "traffic";
    if (/cert|certificate|ssl|tls|x509/.test(text)) return "certificate";
    if (/up\s*==\s*0|unavailable|down|health/.test(text)) return "availability";
    return "generic";
  }

  function detectPresetKeyByTemplates(summary: string, description: string): string | undefined {
    const s = String(summary || "").trim();
    const d = String(description || "").trim();
    if (!s || !d) return undefined;

    // 1) match custom dict presets (value is JSON string)
    for (const o of ruleTemplatePresetDictOpts) {
      const raw = String(o.value || "");
      const parsed = parseTemplatePresetPair(raw);
      if (parsed && parsed.summary === s && parsed.description === d) {
        return `custom:${raw}`;
      }
    }

    // 2) match built-in presets (exact match)
    const builtinMap: Record<string, { summary: string; description: string }> = {
      cpu: {
        summary: "{{$labels.instance}} CPU 使用率过高（{{$value}}）",
        description: "实例 {{$labels.instance}} CPU 连续超过阈值，请检查负载。规则={{.RuleName}}，PromQL={{.Expr}}，当前值={{$value}}",
      },
      memory: {
        summary: "{{$labels.instance}} 内存使用率过高（{{$value}}）",
        description: "实例 {{$labels.instance}} 内存连续超过阈值，请检查进程占用。规则={{.RuleName}}，PromQL={{.Expr}}，当前值={{$value}}",
      },
      disk: {
        summary: "{{$labels.instance}} 磁盘使用率过高（{{$value}}）",
        description: "实例 {{$labels.instance}} 磁盘告警，建议检查挂载点 {{$labels.mountpoint}}。规则={{.RuleName}}，PromQL={{.Expr}}，当前值={{$value}}",
      },
      generic: {
        summary: "{{$labels.instance}}: {{.RuleName}} 告警触发（{{$value}}）",
        description: "规则 {{.RuleName}} 触发，PromQL={{.Expr}}，实例={{$labels.instance}}，当前值={{$value}}",
      },
      availability: {
        summary: "{{$labels.instance}} 可用性异常（{{$value}}）",
        description: "组件/实例可用性异常，请检查健康状态。规则={{.RuleName}}，PromQL={{.Expr}}，实例={{$labels.instance}}，当前值={{$value}}",
      },
      error_rate: {
        summary: "{{$labels.instance}} 错误率异常（{{$value}}）",
        description: "错误率超过阈值，建议检查日志与上游依赖。规则={{.RuleName}}，PromQL={{.Expr}}，实例={{$labels.instance}}，当前值={{$value}}",
      },
      latency: {
        summary: "{{$labels.instance}} 延迟异常（{{$value}}）",
        description: "响应时间超过阈值，建议检查慢请求、下游依赖和资源瓶颈。规则={{.RuleName}}，PromQL={{.Expr}}，当前值={{$value}}",
      },
      queue_lag: {
        summary: "{{$labels.instance}} 队列积压异常（{{$value}}）",
        description: "消费延迟/积压升高，请检查消费者处理能力与上游流量。规则={{.RuleName}}，PromQL={{.Expr}}，当前值={{$value}}",
      },
      traffic: {
        summary: "{{$labels.instance}} 流量波动异常（{{$value}}）",
        description: "流量突增或突降，请结合发布与入口流量排查。规则={{.RuleName}}，PromQL={{.Expr}}，当前值={{$value}}",
      },
      certificate: {
        summary: "{{$labels.instance}} 证书到期预警（{{$value}}）",
        description: "证书有效期接近阈值，请及时续期。规则={{.RuleName}}，PromQL={{.Expr}}，实例={{$labels.instance}}，当前值={{$value}}",
      },
    };
    for (const [k, v] of Object.entries(builtinMap)) {
      if (String(v.summary).trim() === s && String(v.description).trim() === d) return k;
    }
    return undefined;
  }

  function applyRuleAnnotationPreset(preset: string) {
    if (preset.startsWith("custom:")) {
      const raw = preset.slice("custom:".length);
      const parsed = parseTemplatePresetPair(raw);
      if (parsed) {
        ruleForm.setFieldsValue({
          summary_template: parsed.summary,
          description_template: parsed.description,
          rule_template_preset: preset,
        });
        message.success("已应用自定义模板预设");
        return;
      }
      message.error('自定义模板预设格式错误：请在数据字典 value 中配置 JSON，如 {"summary":"...","description":"..."}');
      return;
    }
    const selected = preset === "smart" ? detectRulePresetByContext() : preset;
    const map: Record<string, { summary: string; description: string }> = {
      cpu: {
        summary: "{{$labels.instance}} CPU 使用率过高（{{$value}}）",
        description: "实例 {{$labels.instance}} CPU 连续超过阈值，请检查负载。规则={{.RuleName}}，PromQL={{.Expr}}，当前值={{$value}}",
      },
      memory: {
        summary: "{{$labels.instance}} 内存使用率过高（{{$value}}）",
        description: "实例 {{$labels.instance}} 内存连续超过阈值，请检查进程占用。规则={{.RuleName}}，PromQL={{.Expr}}，当前值={{$value}}",
      },
      disk: {
        summary: "{{$labels.instance}} 磁盘使用率过高（{{$value}}）",
        description: "实例 {{$labels.instance}} 磁盘告警，建议检查挂载点 {{$labels.mountpoint}}。规则={{.RuleName}}，PromQL={{.Expr}}，当前值={{$value}}",
      },
      generic: {
        summary: "{{$labels.instance}}: {{.RuleName}} 告警触发（{{$value}}）",
        description: "规则 {{.RuleName}} 触发，PromQL={{.Expr}}，实例={{$labels.instance}}，当前值={{$value}}",
      },
      availability: {
        summary: "{{$labels.instance}} 可用性异常（{{$value}}）",
        description: "组件/实例可用性异常，请检查健康状态。规则={{.RuleName}}，PromQL={{.Expr}}，实例={{$labels.instance}}，当前值={{$value}}",
      },
      error_rate: {
        summary: "{{$labels.instance}} 错误率异常（{{$value}}）",
        description: "错误率超过阈值，建议检查日志与上游依赖。规则={{.RuleName}}，PromQL={{.Expr}}，实例={{$labels.instance}}，当前值={{$value}}",
      },
      latency: {
        summary: "{{$labels.instance}} 延迟异常（{{$value}}）",
        description: "响应时间超过阈值，建议检查慢请求、下游依赖和资源瓶颈。规则={{.RuleName}}，PromQL={{.Expr}}，当前值={{$value}}",
      },
      queue_lag: {
        summary: "{{$labels.instance}} 队列积压异常（{{$value}}）",
        description: "消费延迟/积压升高，请检查消费者处理能力与上游流量。规则={{.RuleName}}，PromQL={{.Expr}}，当前值={{$value}}",
      },
      traffic: {
        summary: "{{$labels.instance}} 流量波动异常（{{$value}}）",
        description: "流量突增或突降，请结合发布与入口流量排查。规则={{.RuleName}}，PromQL={{.Expr}}，当前值={{$value}}",
      },
      certificate: {
        summary: "{{$labels.instance}} 证书到期预警（{{$value}}）",
        description: "证书有效期接近阈值，请及时续期。规则={{.RuleName}}，PromQL={{.Expr}}，实例={{$labels.instance}}，当前值={{$value}}",
      },
    };
    const next = map[selected] || map.generic;
    ruleForm.setFieldsValue({
      summary_template: next.summary,
      description_template: next.description,
      rule_template_preset: preset,
    });
    message.success(preset === "smart" ? `已按规则内容推荐预设：${selected}` : "已应用模板预设");
  }

  function openRuleEdit(r: AlertMonitorRuleItem) {
    const summaryTemplate = typeof r.annotations?.summary === "string" ? r.annotations.summary : "";
    const descriptionTemplate = typeof r.annotations?.description === "string" ? r.annotations.description : "";
    setRuleCurrent(r);
    const presetKey = detectPresetKeyByTemplates(summaryTemplate, descriptionTemplate);
    ruleForm.setFieldsValue({
      datasource_id: r.datasource_id,
      name: r.name,
      expr: r.expr,
      for_seconds: r.for_seconds,
      eval_interval_seconds: r.eval_interval_seconds,
      severity: r.severity,
      threshold_unit: r.threshold_unit || "raw",
      labels_json: stringifyPrettyJSON(r.labels ?? {}, "{}"),
      summary_template: summaryTemplate || "{{$labels.instance}}: {{.RuleName}} 告警触发，当前值 {{$value}}",
      description_template: descriptionTemplate || "规则 {{.RuleName}} 触发，PromQL={{.Expr}}，实例={{$labels.instance}}，当前值={{$value}}",
      rule_template_preset: presetKey,
      enabled: r.enabled,
    });
    tryFillRuleBuilderFromExpr(r.expr);
    setMetricKeyword("");
    setMetricOptions([]);
    const parsedByExpr = parseRuleBuilderExpr(r.expr);
    const selectorSrc = parsedByExpr?.conditions?.[0]?.metric || r.expr;
    const selector = parsePromSelectorExpr(selectorSrc);
    setSelectedMetric(selector?.metric || "");
    setMetricLabelFilters(selector?.filters || [{ key: "instance", op: "=", value: "" }]);
    const funcKey = detectPromFunctionKeyFromExpr(selectorSrc);
    setSelectedPromFunc(funcKey || "none");
    setLabelValueOptions([]);
    setRuleModalOpen(true);
  }

  async function loadMetricOptionsForRule() {
    const dsID = Number(ruleForm.getFieldValue("datasource_id"));
    if (!dsID) {
      message.warning("请先选择数据源");
      return;
    }
    const kw = String(metricKeyword || "").trim();
    const re = kw ? `.*${kw.replace(/\//g, "\\/")}.*` : ".+";
    const query = `topk(300, count by (__name__)({__name__=~"${re}"}))`;
    setMetricLoading(true);
    try {
      const r = await promInstantQuery(dsID, { query });
      const outer = (r as { data?: unknown }).data ?? r;
      const inner = unwrapPrometheusQueryData(outer) as { result?: Array<{ metric?: Record<string, string> }> };
      const names = Array.from(
        new Set(
          (inner?.result ?? [])
            .map((it) => String(it?.metric?.__name__ ?? "").trim())
            .filter(Boolean),
        ),
      ).sort((a, b) => a.localeCompare(b));
      setMetricOptions(names);
      if (names.length === 0) message.warning("未检索到指标，请调整关键字");
    } catch (e) {
      message.error(`加载指标失败：${extractApiErrorMessage(e, "操作失败")}`);
    } finally {
      setMetricLoading(false);
    }
  }

  async function loadLabelValuesForRule(idx: number) {
    const dsID = Number(ruleForm.getFieldValue("datasource_id"));
    if (!dsID) {
      message.warning("请先选择数据源");
      return;
    }
    const metric = String(selectedMetric || "").trim();
    if (!metric) {
      message.warning("请先选择指标");
      return;
    }
    const key = String(metricLabelFilters[idx]?.key || "").trim();
    if (!isValidPromLabelKey(key)) {
      message.warning("标签名不合法");
      return;
    }
    const selector = buildPromSelectorExpr(
      metric,
      metricLabelFilters.filter((_, i) => i !== idx),
    );
    const query = `topk(200, count by (${key}) (${selector}))`;
    setLabelValueLoading(true);
    try {
      const r = await promInstantQuery(dsID, { query });
      const outer = (r as { data?: unknown }).data ?? r;
      const inner = unwrapPrometheusQueryData(outer) as { result?: Array<{ metric?: Record<string, string> }> };
      const vals = Array.from(
        new Set(
          (inner?.result ?? [])
            .map((it) => String(it?.metric?.[key] ?? "").trim())
            .filter(Boolean),
        ),
      ).sort((a, b) => a.localeCompare(b));
      setLabelValueOptions(vals);
      if (!vals.length) message.warning("未检索到可用标签值");
    } catch (e) {
      message.error(`加载标签值失败：${extractApiErrorMessage(e, "操作失败")}`);
    } finally {
      setLabelValueLoading(false);
    }
  }

  function applyMetricSelectorToRuleExpr() {
    const metric = String(selectedMetric || "").trim();
    if (!metric) {
      message.warning("请先选择指标");
      return;
    }
    const selector = buildPromSelectorExpr(metric, metricLabelFilters);
    ruleForm.setFieldValue("expr", selector);
    setRuleConditions((prev) => {
      if (!prev.length) return [{ metric: selector, comparator: ">", threshold: null }];
      return prev.map((it, i) => (i === 0 ? { ...it, metric: selector } : it));
    });
    message.success("已生成并带入 PromQL");
  }

  function materializePromFunctionTemplate(raw: string): string {
    const selector = buildPromSelectorExpr(String(selectedMetric || "").trim(), metricLabelFilters);
    const baseMetric = selector || String(ruleConditions[0]?.metric || "").trim() || "your_metric";
    return raw.split("__METRIC__").join(baseMetric);
  }

  function buildMetricExprBySteps(): string {
    const selector = buildPromSelectorExpr(String(selectedMetric || "").trim(), metricLabelFilters);
    const baseMetric = selector || String(ruleConditions[0]?.metric || "").trim() || "";
    if (!baseMetric) return "";
    if (selectedPromFuncMeta.key === "none") return baseMetric;
    return selectedPromFuncMeta.template.split("__METRIC__").join(baseMetric);
  }

  function insertPromFunctionToExpr() {
    const nextExpr = materializePromFunctionTemplate(selectedPromFuncMeta.template);
    const prev = String(ruleForm.getFieldValue("expr") || "").trim();
    ruleForm.setFieldValue("expr", prev ? `${prev}\n${nextExpr}` : nextExpr);
  }

  function usePromFunctionAsConditionMetric() {
    const nextExpr = materializePromFunctionTemplate(selectedPromFuncMeta.template);
    setRuleConditions((prev) => {
      if (!prev.length) return [{ metric: nextExpr, comparator: ">", threshold: null }];
      return prev.map((it, i) => (i === 0 ? { ...it, metric: nextExpr } : it));
    });
  }

  function applyStepwisePromQL() {
    const metricExpr = buildMetricExprBySteps();
    if (!metricExpr) {
      message.warning("请先完成第1步：选择指标与标签过滤（或手填条件表达式）");
      return;
    }
    const nextConditions: RuleBuilderCondition[] =
      ruleConditions.length > 0
        ? ruleConditions.map((it, i) => (i === 0 ? { ...it, metric: metricExpr } : it))
        : [{ metric: metricExpr, comparator: ">" as RuleComparator, threshold: null as number | null }];
    if (nextConditions.some((c) => !String(c.metric || "").trim() || c.threshold === null || Number.isNaN(c.threshold))) {
      setRuleConditions(nextConditions);
      message.warning("请完成第3步：填写阈值后再生成最终 PromQL");
      return;
    }
    setRuleConditions(nextConditions);
    ruleForm.setFieldValue("expr", buildRuleExprByConditions(nextConditions, ruleLogic));
  }

  async function submitRule() {
    setRuleSubmitting(true);
    try {
      const v = await ruleForm.validateFields();
      const normalizedUnit = String(v.threshold_unit || "raw").trim().toLowerCase() === "precent" ? "percent" : v.threshold_unit;
      const labelsNorm = normalizeCloudExpiryLabelsJSON(String(v.labels_json ?? "{}"));
      if (labelsNorm === null) {
        message.error("附加 Labels 须为合法 JSON 对象");
        return;
      }
      const payload = {
        ...v,
        threshold_unit: normalizedUnit,
        labels_json: labelsNorm,
        annotations_json: JSON.stringify({
          summary: String(v.summary_template || "").trim(),
          description: String(v.description_template || "").trim(),
        }),
      };
      if (ruleCurrent) {
        await updateAlertMonitorRule(ruleCurrent.id, payload);
        message.success("已更新");
      } else {
        await createAlertMonitorRule(payload);
        message.success("已创建");
      }
      setRuleModalOpen(false);
      await loadRules(projectContextId);
    } catch (e) {
      if (e && typeof e === "object" && "errorFields" in e) return;
      message.error(extractApiErrorMessage(e, "保存规则失败"));
    } finally {
      setRuleSubmitting(false);
    }
  }

  async function removeRule(id: number) {
    try {
      await deleteAlertMonitorRule(id);
      message.success("已删除");
      await loadRules(projectContextId);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "删除规则失败"));
    }
  }

  async function openAssign(ruleId: number) {
    setAssignRuleId(ruleId);
    assignForm.resetFields();
    let userIds: number[] = [];
    try {
      const { list } = await getMonitorRuleAssignees(ruleId);
      const row = list?.[0];
      userIds = row?.user_ids ?? [];
      assignForm.setFieldsValue({
        user_ids: userIds,
        department_ids: row?.department_ids ?? [],
        recipient_mode: row?.recipient_mode || "assignee_and_cc",
        notify_on_resolved: row?.notify_on_resolved ?? false,
        remark: row?.remark ?? "",
      });
    } catch {
      assignForm.setFieldsValue({
        user_ids: [],
        department_ids: [],
        recipient_mode: "assignee_and_cc",
        notify_on_resolved: false,
        remark: "",
      });
    }
    assignSyncedKeyRef.current = userIds.join(",");
    setAssignOpen(true);
    void fillAssignFromUserIds(userIds);
  }

  async function submitAssign() {
    if (!assignRuleId) return;
    setAssignSubmitting(true);
    try {
      const v = await assignForm.validateFields();
      const userIds = (v.user_ids ?? []) as number[];
      const deptIds = (v.department_ids ?? []) as number[];
      if (userIds.length === 1 && assignProfileOriginal) {
        const patch: UserUpdatePayload = {};
        const emailNew = String(v.profile_email ?? "").trim();
        if (emailNew && emailNew !== String(assignProfileOriginal.email ?? "").trim()) {
          patch.email = emailNew;
        }
        if (deptIds.length === 1 && deptIds[0] !== assignProfileOriginal.department_id) {
          patch.department_id = deptIds[0];
        }
        if (Object.keys(patch).length) {
          try {
            await updateUser(userIds[0], patch);
            message.success("已同步更新用户资料中的邮箱或部门");
          } catch (e) {
            message.warning(`处理人已保存，但写回用户资料失败：${extractApiErrorMessage(e, "操作失败")}`);
          }
        }
      }
      await upsertMonitorRuleAssignees(assignRuleId, {
        user_ids_json: JSON.stringify(userIds),
        department_ids_json: JSON.stringify(deptIds),
        extra_emails_json: "[]",
        recipient_mode: v.recipient_mode || "assignee_and_cc",
        notify_on_resolved: v.notify_on_resolved,
        remark: v.remark ?? "",
      });
      message.success("处理人已保存");
      setAssignOpen(false);
    } finally {
      setAssignSubmitting(false);
    }
  }

  const copyDutyRuleOptions = dutyCopyRuleOptions;

  async function openDuty(ruleId: number) {
    setDutyRuleId(ruleId);
    setCopySourceRuleId(undefined);
    setBlockList([]);
    try {
      const [blocks, rules] = await Promise.all([
        listDutyBlocks({ monitor_rule_id: ruleId, page: 1, page_size: 100 }),
        listAlertMonitorRules({
          project_id: projectContextId && projectContextId > 0 ? projectContextId : undefined,
          page: 1,
          page_size: 100,
        }),
      ]);
      setBlockList(blocks.list ?? []);
      setDutyCopyRuleOptions(
        (rules.list ?? [])
          .filter((r) => r.id !== ruleId)
          .map((r) => ({ label: r.name, value: r.id })),
      );
    } catch {
      setBlockList([]);
      setDutyCopyRuleOptions([]);
    }
    setDutyModalOpen(true);
  }

  /** 将「来源规则」下的全部班次复制为当前规则的班次（新建记录，互不影响原规则）。 */
  async function copyDutyBlocksFromSelectedRule() {
    if (!dutyRuleId) {
      message.warning("未识别当前规则");
      return;
    }
    if (!copySourceRuleId || copySourceRuleId === dutyRuleId) {
      message.warning("请选择一条其他监控规则作为班次来源");
      return;
    }
    setCopyDutyLoading(true);
    try {
      const r = await listDutyBlocks({ monitor_rule_id: copySourceRuleId, page: 1, page_size: 500 });
      const src = r.list ?? [];
      if (src.length === 0) {
        message.info("所选规则下暂无班次，无法复制");
        return;
      }
      for (const b of src) {
        await createDutyBlock({
          monitor_rule_id: dutyRuleId,
          starts_at: b.starts_at,
          ends_at: b.ends_at,
          title: b.title,
          user_ids_json: JSON.stringify(b.user_ids ?? []),
          department_ids_json: JSON.stringify(b.department_ids ?? []),
          extra_emails_json: JSON.stringify(b.extra_emails ?? []),
          remark: b.remark ?? "",
        });
      }
      message.success(`已复制 ${src.length} 条班次到当前规则`);
      const refreshed = await listDutyBlocks({ monitor_rule_id: dutyRuleId, page: 1, page_size: 500 });
      setBlockList(refreshed.list ?? []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "操作失败"));
    } finally {
      setCopyDutyLoading(false);
    }
  }

  const blkColumns = [
    { title: "ID", dataIndex: "id", width: 70 },
    { title: "标题", dataIndex: "title", width: 120 },
    { title: "开始", dataIndex: "starts_at", width: 160, render: (t: string) => formatDateTime(t) },
    { title: "结束", dataIndex: "ends_at", width: 160, render: (t: string) => formatDateTime(t) },
    {
      title: "操作",
      key: "actions",
      width: 120,
      fixed: "right" as const,
      render: (_: unknown, r: AlertDutyBlockItem) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openBlkEdit(r)}>
            编辑
          </Button>
          <Popconfirm title="删除班次？" onConfirm={() => void removeBlk(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  function openBlkCreate() {
    if (!dutyRuleId) {
      message.warning("请先选择一条监控规则");
      return;
    }
    dutySyncedKeyRef.current = "";
    setBlkCurrent(null);
    blkForm.resetFields();
    blkForm.setFieldsValue({
      monitor_rule_id: dutyRuleId,
      range: [dayjs(), dayjs().add(8, "hour")],
      title: "",
      user_ids: [],
      department_ids: [],
      profile_email: undefined,
      remark: "",
    });
    setBlkModalOpen(true);
  }

  function openBlkEdit(r: AlertDutyBlockItem) {
    setBlkCurrent(r);
    const userIds = r.user_ids ?? [];
    blkForm.setFieldsValue({
      monitor_rule_id: r.monitor_rule_id,
      range: [dayjs(r.starts_at), dayjs(r.ends_at)],
      title: r.title,
      user_ids: userIds,
      department_ids: r.department_ids ?? [],
      profile_email: undefined,
      remark: r.remark,
    });
    dutySyncedKeyRef.current = userIds.join(",");
    setBlkModalOpen(true);
    void fillDutyFromUserIds(userIds);
  }

  async function submitBlk() {
    setBlkSubmitting(true);
    try {
      const v = await blkForm.validateFields();
      const range = v.range as [Dayjs, Dayjs];
      const monitorRuleID = Number(v.monitor_rule_id || dutyRuleId || 0);
      if (!monitorRuleID) {
        message.error("未识别到监控规则，请关闭侧栏后重新从规则行进入值班");
        return;
      }
      if (!Array.isArray(range) || range.length !== 2 || !range[0] || !range[1]) {
        message.error("请选择完整的起止时间");
        return;
      }
      if (!range[1].isAfter(range[0])) {
        message.error("结束时间必须晚于开始时间");
        return;
      }
      const userIds = (v.user_ids ?? []) as number[];
      const deptIds = (v.department_ids ?? []) as number[];
      if (userIds.length === 1 && dutyProfileOriginal) {
        const patch: UserUpdatePayload = {};
        const emailNew = String(v.profile_email ?? "").trim();
        if (emailNew && emailNew !== String(dutyProfileOriginal.email ?? "").trim()) {
          patch.email = emailNew;
        }
        if (deptIds.length === 1 && deptIds[0] !== dutyProfileOriginal.department_id) {
          patch.department_id = deptIds[0];
        }
        if (Object.keys(patch).length) {
          try {
            await updateUser(userIds[0], patch);
            message.success("已同步更新用户资料中的邮箱或部门");
          } catch (e) {
            message.warning(`班次已保存，但写回用户资料失败：${extractApiErrorMessage(e, "操作失败")}`);
          }
        }
      }
      const payload = {
        monitor_rule_id: monitorRuleID,
        starts_at: range[0].toISOString(),
        ends_at: range[1].toISOString(),
        title: v.title,
        user_ids_json: JSON.stringify(userIds),
        department_ids_json: JSON.stringify(deptIds),
        extra_emails_json: "[]",
        remark: v.remark ?? "",
      };
      if (blkCurrent) {
        await updateDutyBlock(blkCurrent.id, payload);
        message.success("已更新");
      } else {
        await createDutyBlock(payload);
        message.success("已创建");
      }
      setBlkModalOpen(false);
      if (dutyRuleId) {
        const r = await listDutyBlocks({ monitor_rule_id: dutyRuleId, page: 1, page_size: 500 });
        setBlockList(r.list ?? []);
      }
    } finally {
      setBlkSubmitting(false);
    }
  }

  async function removeBlk(id: number) {
    await deleteDutyBlock(id);
    message.success("已删除");
    if (dutyRuleId) {
      const r = await listDutyBlocks({ monitor_rule_id: dutyRuleId, page: 1, page_size: 500 });
      setBlockList(r.list ?? []);
    }
  }


  const ruleColumns = [
    { title: "ID", dataIndex: "id", width: 70 },
    {
      title: "项目",
      dataIndex: "project_name",
      width: 160,
      render: (v: string, r: AlertMonitorRuleItem) => {
        if (String(v || "").trim()) return v;
        const ds = dsList.find((d) => d.id === r.datasource_id);
        if (String(ds?.project_name || "").trim()) return String(ds?.project_name);
        return r.project_id ? String(r.project_id) : "—";
      },
    },
    { title: "名称", dataIndex: "name", width: 160 },
    {
      title: "数据源",
      key: "ds",
      width: 200,
      render: (_: unknown, r: AlertMonitorRuleItem) => {
        const name = String(r.datasource_name || "").trim();
        if (name) return name;
        const ds = dsList.find((d) => d.id === r.datasource_id);
        return ds ? ds.name : String(r.datasource_id);
      },
    },
    { title: "级别", dataIndex: "severity", width: 90 },
    { title: "for(s)", dataIndex: "for_seconds", width: 80 },
    { title: "间隔(s)", dataIndex: "eval_interval_seconds", width: 90 },
    { title: "启用", dataIndex: "enabled", width: 70, render: (v: boolean) => (v ? <Tag color="green">是</Tag> : <Tag>否</Tag>) },
    {
      title: "操作",
      width: 320,
      fixed: "right" as const,
      render: (_: unknown, r: AlertMonitorRuleItem) => (
        <Space wrap>
          <Button type="link" size="small" onClick={() => openSilenceForMonitorRule(r)}>
            静默
          </Button>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openRuleEdit(r)}>
            规则
          </Button>
          <Button type="link" size="small" icon={<TeamOutlined />} onClick={() => void openAssign(r.id)}>
            处理人
          </Button>
          <Button type="link" size="small" icon={<CalendarOutlined />} onClick={() => void openDuty(r.id)}>
            值班
          </Button>
          <Popconfirm title="删除规则？" onConfirm={() => void removeRule(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];



  const ruleDisplayList = ruleList;

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
  };
}
