import {
  ApiOutlined,
  DeleteOutlined,
  EditOutlined,
  CalendarOutlined,
  MinusCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import type { TreeSelectProps } from "antd";
import {
  Alert,
  AutoComplete,
  Badge,
  Button,
  Calendar,
  Card,
  Col,
  Collapse,
  DatePicker,
  Drawer,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Radio,
  Row,
  Segmented,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  TreeSelect,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import "dayjs/locale/zh-cn";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { DepartmentItem } from "../../types/api";
import { getDepartmentTree } from "../../services/departments";
import { getProjects, type ProjectItem } from "../../services/projects";
import { getUsers } from "../../services/users";
import {
  createAlertDatasource,
  createAlertMonitorRule,
  createAlertSilence,
  createAlertSilencesBatch,
  createCloudExpiryRule,
  createDutyBlock,
  deleteAlertDatasource,
  deleteAlertMonitorRule,
  deleteAlertSilence,
  deleteCloudExpiryRule,
  deleteDutyBlock,
  evaluateCloudExpiryRulesNow,
  getMonitorRuleAssignees,
  listAlertDatasources,
  listAlertMonitorRules,
  listAlertSilences,
  listCloudExpiryRules,
  listDutyBlocks,
  pingAlertDatasource,
  promActiveAlerts,
  promInstantQuery,
  promRangeQuery,
  updateAlertDatasource,
  updateAlertMonitorRule,
  updateAlertSilence,
  updateCloudExpiryRule,
  updateDutyBlock,
  upsertMonitorRuleAssignees,
  type AlertDatasourceItem,
  type AlertDutyBlockItem,
  type AlertMonitorRuleItem,
  type AlertSilenceItem,
  type CloudExpiryRuleItem,
} from "../../services/alert-platform";
import { stringifyPrettyJSON } from "../../services/alert-mappers";
import { useDictOptions } from "../../hooks/use-dict-options";
import type { UserUpdatePayload } from "../../types/api";
import { getUser, updateUser } from "../../services/users";
import { formatDateTime } from "../../utils/format";
import type { AlertEventCategory } from "../../utils/alert-event-reasons";

dayjs.locale("zh-cn");

type TabKey = "datasources" | "policies" | "silences" | "inhibition" | "rules" | "history" | "cloud-expiry" | "promql";

type SilenceMatcherForm = { name: string; value: string; is_regex: boolean };

type PromNativeAlertRow = {
  key: string;
  alertname: string;
  state: string;
  labelsShort: string;
  activeAt?: string;
  labels: Record<string, string>;
};
type QuickSilenceTarget = {
  key: string;
  name: string;
  labels: Record<string, string>;
  startsAt: Dayjs;
  endsAt: Dayjs;
};
type RuleComparator = ">" | ">=" | "<" | "<=" | "==" | "!=";
type RuleBuilderLogic = "and" | "or";
type RuleBuilderCondition = { metric: string; comparator: RuleComparator; threshold: number | null };
type MetricLabelFilter = { key: string; op: "=" | "!=" | "=~" | "!~"; value: string };

function parseTemplatePresetPair(raw: string): { summary: string; description: string } | null {
  const s = String(raw || "").trim();
  if (!s) return null;
  try {
    const parsed = JSON.parse(s) as { summary?: string; description?: string };
    const summary = String(parsed.summary || "").trim();
    const description = String(parsed.description || "").trim();
    if (!summary || !description) return null;
    return { summary, description };
  } catch {
    return null;
  }
}

function parseSilenceMatchersForForm(raw?: string): SilenceMatcherForm[] {
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

function parsePrometheusActiveAlertsTable(body: unknown): PromNativeAlertRow[] {
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

function parseUintArrayJSON(raw?: string): number[] {
  const s = raw?.trim();
  if (!s) return [];
  try {
    const v = JSON.parse(s) as unknown;
    if (!Array.isArray(v)) return [];
    return v
      .map((x) => (typeof x === "number" ? x : typeof x === "string" && /^\d+$/.test(x) ? Number(x) : NaN))
      .filter((n) => !Number.isNaN(n));
  } catch {
    return [];
  }
}

function deptToTreeData(nodes: DepartmentItem[]): TreeSelectProps["treeData"] {
  return nodes.map((n) => ({
    title: n.name,
    value: n.id,
    children: n.children?.length ? deptToTreeData(n.children) : undefined,
  }));
}

function sortMetricKeys(a: string, b: string): number {
  if (a === "__name__") return -1;
  if (b === "__name__") return 1;
  return a.localeCompare(b);
}

function formatPromTimestampLocal(raw: string): string {
  const n = Number(raw);
  if (!Number.isFinite(n)) return raw;
  const ms = n > 1e12 ? n : n * 1000;
  return dayjs(ms).format("YYYY-MM-DD HH:mm:ss");
}

function isValidPromLabelKey(s: string): boolean {
  return /^[a-zA-Z_][a-zA-Z0-9_]*$/.test(String(s || "").trim());
}

function buildPromSelectorExpr(metric: string, filters: MetricLabelFilter[]): string {
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

function parsePromSelectorExpr(raw: string): { metric: string; filters: MetricLabelFilter[] } | null {
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

function detectPromFunctionKeyFromExpr(exprRaw: string): string | null {
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

type PromTableView = {
  columns: ColumnsType<Record<string, string>>;
  dataSource: Record<string, string>[];
};

/** 将 Prometheus instant/range 的 data 段解析为表格（vector / matrix）。 */
function buildPromTableView(data: unknown): PromTableView | null {
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

function formatPromScalarSummary(data: unknown): string | null {
  if (!data || typeof data !== "object") return null;
  const o = data as Record<string, unknown>;
  if (String(o.resultType) !== "string") return null;
  const r = o.result;
  if (Array.isArray(r) && r.length >= 2) return `结果值：${String(r[1])}（时间戳 ${r[0]}）`;
  return null;
}

/** 后端返回的 Prometheus JSON 可能为 { status, data:{ resultType, ... } }，表格解析取内层 data。 */
function unwrapPrometheusQueryData(body: unknown): unknown {
  if (!body || typeof body !== "object") return body;
  const o = body as Record<string, unknown>;
  if (o.data && typeof o.data === "object") {
    const d = o.data as Record<string, unknown>;
    if (typeof d.resultType === "string" || Array.isArray(d.result)) return o.data;
  }
  return body;
}


import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { AlertMonitorProvider } from "./context";
import { AlertMonitorLayout } from "./layout";
import { AlertMonitorModals } from "./modals";
import { normalizeAlertMonitorTab, tabPathForKey, type AlertMonitorTabKey } from "./tab-config";

export type { AlertMonitorTabKey };

export function AlertMonitorPlatformRoot() {
  const state = useAlertMonitorPlatformState();
  return (
    <AlertMonitorProvider value={state as never}>
      <AlertMonitorLayout />
      <AlertMonitorModals />
    </AlertMonitorProvider>
  );
}

function useAlertMonitorPlatformState() {
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
    const allowed: AlertEventCategory[] = [
      "delivery",
      "routing",
      "silence",
      "inhibition",
      "timing",
      "resolved",
      "failure",
      "other",
    ];
    return (allowed as string[]).includes(raw) ? (raw as AlertEventCategory) : undefined;
  }, [searchParams]);

  const tab: AlertMonitorTabKey = useMemo(() => normalizeAlertMonitorTab(tabParam), [tabParam]);

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

  function setTab(key: AlertMonitorTabKey) {
    const qs = searchParams.toString();
    const path = tabPathForKey(key);
    navigate(qs ? `${path}?${qs}` : path, { replace: true });
  }

  function setProjectContext(projectID?: number) {
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (projectID && Number.isFinite(projectID) && projectID > 0) p.set("project_id", String(projectID));
        else p.delete("project_id");
        return p;
      },
      { replace: true },
    );
  }

  function openHistoryTab() {
    setTab("history");
  }

  const [dsList, setDsList] = useState<AlertDatasourceItem[]>([]);
  const [silenceList, setSilenceList] = useState<AlertSilenceItem[]>([]);
  const [ruleList, setRuleList] = useState<AlertMonitorRuleItem[]>([]);
  /** 监控规则列表：全部 / 仅启用 / 仅停用 */
  const [ruleEnabledFilter, setRuleEnabledFilter] = useState<"all" | "enabled" | "disabled">("all");
  const [cloudExpiryList, setCloudExpiryList] = useState<CloudExpiryRuleItem[]>([]);
  const [blockList, setBlockList] = useState<AlertDutyBlockItem[]>([]);
  const [dutyRuleId, setDutyRuleId] = useState<number | null>(null);
  const [dutyModalOpen, setDutyModalOpen] = useState(false);
  /** 规则值班弹窗：从其他规则复制班次时的来源规则 ID */
  const [copySourceRuleId, setCopySourceRuleId] = useState<number | undefined>();
  const [copyDutyLoading, setCopyDutyLoading] = useState(false);

  const [loading, setLoading] = useState(false);
  const [users, setUsers] = useState<Array<{ label: string; value: number }>>([]);
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [deptTree, setDeptTree] = useState<TreeSelectProps["treeData"]>([]);

  const [dsModalOpen, setDsModalOpen] = useState(false);
  const [dsCurrent, setDsCurrent] = useState<AlertDatasourceItem | null>(null);
  const [dsForm] = Form.useForm();
  const [dsSubmitting, setDsSubmitting] = useState(false);
  const [dsPingId, setDsPingId] = useState<number | null>(null);

  const [silModalOpen, setSilModalOpen] = useState(false);
  const [silCurrent, setSilCurrent] = useState<AlertSilenceItem | null>(null);
  const [silForm] = Form.useForm();
  const [silSubmitting, setSilSubmitting] = useState(false);

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

  const [cloudExpiryModalOpen, setCloudExpiryModalOpen] = useState(false);
  const [cloudExpiryCurrent, setCloudExpiryCurrent] = useState<CloudExpiryRuleItem | null>(null);
  const [cloudExpiryForm] = Form.useForm();
  const [cloudExpirySubmitting, setCloudExpirySubmitting] = useState(false);
  const [cloudExpiryEvaluating, setCloudExpiryEvaluating] = useState(false);
  const [cloudExpiryProviderFilter, setCloudExpiryProviderFilter] = useState<string>("");
  const [cloudExpiryKeyword, setCloudExpiryKeyword] = useState<string>("");

  const alertSeverityOpts = useDictOptions("alert_severity");
  const dsUrlDictOpts = useDictOptions("alert_datasource_base_url");
  const dsBasicUserDictOpts = useDictOptions("alert_datasource_basic_user");
  const promqlLabelKeyOpts = useDictOptions("alert_promql_label_key");
  const thresholdUnitDictOpts = useDictOptions("alert_threshold_unit");
  const ruleTemplatePresetDictOpts = useDictOptions("alert_rule_template_preset");
  const dsUrlAutoOpts = useMemo(
    () => dsUrlDictOpts.map((o) => ({ label: o.label, value: String(o.value) })),
    [dsUrlDictOpts],
  );
  const dsBasicUserAutoOpts = useMemo(
    () => dsBasicUserDictOpts.map((o) => ({ label: o.label, value: String(o.value) })),
    [dsBasicUserDictOpts],
  );
  const silenceMatcherNameOptions = useMemo(() => {
    const platformKeys = [
      { label: "monitor_rule_id（平台监控规则 ID）", value: "monitor_rule_id" },
      { label: "alertname（规则/告警名）", value: "alertname" },
      { label: "project_id（项目）", value: "project_id" },
      { label: "source（来源，如 prometheus_monitor）", value: "source" },
    ];
    const fromProm = promqlLabelKeyOpts
      .map((o) => {
        const value = String(o.value || "").trim();
        const label = String(o.label || "").trim() || value;
        return { label: `${label} (${value})`, value };
      })
      .filter((o) => o.value);
    const seen = new Set<string>();
    const out = [...platformKeys, ...fromProm].filter((o) => {
      if (seen.has(o.value)) return false;
      seen.add(o.value);
      return true;
    });
    out.sort((a, b) => a.value.localeCompare(b.value, "zh-CN"));
    return out;
  }, [promqlLabelKeyOpts]);
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
  const [metricKeyword, setMetricKeyword] = useState("");
  const [metricLoading, setMetricLoading] = useState(false);
  const [metricOptions, setMetricOptions] = useState<string[]>([]);
  const [selectedMetric, setSelectedMetric] = useState("");
  const [metricLabelFilters, setMetricLabelFilters] = useState<MetricLabelFilter[]>([{ key: "instance", op: "=", value: "" }]);
  const [labelValueLoading, setLabelValueLoading] = useState(false);
  const [labelValueOptions, setLabelValueOptions] = useState<string[]>([]);
  const [selectedPromFunc, setSelectedPromFunc] = useState<string>("none");

  const [nativeAlertsLoading, setNativeAlertsLoading] = useState(false);
  const [nativeAlertsRows, setNativeAlertsRows] = useState<PromNativeAlertRow[]>([]);
  const [selectedNativeAlertKeys, setSelectedNativeAlertKeys] = useState<string[]>([]);
  const [selectedSilenceIds, setSelectedSilenceIds] = useState<number[]>([]);
  const [quickSilenceOpen, setQuickSilenceOpen] = useState(false);
  const [quickSilenceSubmitting, setQuickSilenceSubmitting] = useState(false);
  const [quickSilenceTargets, setQuickSilenceTargets] = useState<QuickSilenceTarget[]>([]);
  /** 批量静默（从活跃告警勾选）时共用的说明，写入每条 alert_silences.comment */
  const [quickSilenceComment, setQuickSilenceComment] = useState("");
  const projectOptions = useMemo(() => projects.map((p) => ({ label: `${p.name} (${p.code})`, value: p.id })), [projects]);
  const activeProjectName = useMemo(() => {
    if (!projectContextId) return "";
    const p = projects.find((it) => it.id === projectContextId);
    return p ? `${p.name} (${p.code})` : `项目 ${projectContextId}`;
  }, [projects, projectContextId]);

  /** 平台静默：Prometheus 活跃告警跟随顶栏项目，默认取首个已启用数据源 */
  const silenceDatasource = useMemo(() => {
    const enabled = dsList.filter((d) => d.enabled !== false);
    return (enabled.length ? enabled : dsList)[0];
  }, [dsList]);
  const silenceDatasourceId = silenceDatasource?.id;

  const promTableView = useMemo(() => buildPromTableView(promDataInner), [promDataInner]);
  const promScalarText = useMemo(() => formatPromScalarSummary(promDataInner), [promDataInner]);
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
  function parseRuleBuilderExpr(exprRaw: string): { conditions: RuleBuilderCondition[]; logic: RuleBuilderLogic } | null {
    const expr = String(exprRaw || "").trim();
    if (!expr) return null;
    const hasOr = /\s+or\s+/i.test(expr);
    const hasAnd = /\s+and\s+/i.test(expr);
    if (hasOr && hasAnd) return null;
    const logic: RuleBuilderLogic = hasOr ? "or" : "and";
    const parts = (hasOr ? expr.split(/\s+or\s+/i) : expr.split(/\s+and\s+/i)).map((p) => p.trim()).filter(Boolean);
    const parsed: RuleBuilderCondition[] = [];
    for (const p0 of parts) {
      const p = p0.replace(/^\((.*)\)$/, "$1").trim();
      const m = p.match(/^(.+?)\s*(>=|<=|==|!=|>|<)\s*(-?\d+(?:\.\d+)?)\s*$/);
      if (!m) return null;
      parsed.push({
        metric: String(m[1] || "").trim(),
        comparator: (m[2] as RuleComparator) || ">",
        threshold: Number(m[3]),
      });
    }
    if (!parsed.length) return null;
    return { conditions: parsed, logic };
  }

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

  function buildRuleExprByConditions(conditions: RuleBuilderCondition[], logic: RuleBuilderLogic): string {
    const valid = conditions
      .map((c) => ({
        metric: String(c.metric || "").trim(),
        comparator: c.comparator,
        threshold: c.threshold,
      }))
      .filter((c) => c.metric && c.threshold !== null && !Number.isNaN(c.threshold));
    if (!valid.length) return "";
    if (valid.length === 1) {
      return `${valid[0].metric} ${valid[0].comparator} ${valid[0].threshold}`;
    }
    const joiner = logic === "or" ? " or " : " and ";
    return valid.map((c) => `(${c.metric} ${c.comparator} ${c.threshold})`).join(joiner);
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

  const loadDatasources = useCallback(async (projectID?: number) => {
    const r = await listAlertDatasources({ project_id: projectID, page: 1, page_size: 200 });
    setDsList(r.list ?? []);
    setPromDsId((prev) => prev ?? r.list?.[0]?.id);
  }, []);

  const loadSilences = useCallback(async () => {
    const r = await listAlertSilences({
      page: 1,
      page_size: 200,
      project_id: projectContextId && projectContextId > 0 ? projectContextId : undefined,
    });
    setSilenceList(r.list ?? []);
  }, [projectContextId]);

  const loadNativeSilAlerts = useCallback(async () => {
    if (!silenceDatasourceId) {
      message.warning(
        projectContextId ? "当前项目下暂无 Prometheus 数据源，请先在「数据源」Tab 创建并启用" : "请先在顶栏选择项目",
      );
      return;
    }
    setNativeAlertsLoading(true);
    try {
      const raw = await promActiveAlerts(silenceDatasourceId);
      const rows = parsePrometheusActiveAlertsTable(raw);
      setNativeAlertsRows(rows);
      setSelectedNativeAlertKeys((prev) => prev.filter((k) => rows.some((r) => r.key === k)));
    } catch {
      setNativeAlertsRows([]);
      setSelectedNativeAlertKeys([]);
    } finally {
      setNativeAlertsLoading(false);
    }
  }, [silenceDatasourceId, projectContextId]);

  const loadRules = useCallback(async (projectID?: number) => {
    const r = await listAlertMonitorRules({ project_id: projectID, page: 1, page_size: 500 });
    setRuleList(r.list ?? []);
  }, []);

  const ruleEnabledStats = useMemo(() => {
    let enabled = 0;
    let disabled = 0;
    for (const r of ruleList) {
      if (r.enabled === false) disabled++;
      else enabled++;
    }
    return { total: ruleList.length, enabled, disabled };
  }, [ruleList]);

  const ruleDisplayList = useMemo(() => {
    if (ruleEnabledFilter === "enabled") return ruleList.filter((r) => r.enabled !== false);
    if (ruleEnabledFilter === "disabled") return ruleList.filter((r) => r.enabled === false);
    return ruleList;
  }, [ruleList, ruleEnabledFilter]);
  const loadCloudExpiryRules = useCallback(async (projectID?: number, provider?: string, keyword?: string) => {
    const r = await listCloudExpiryRules({
      project_id: projectID,
      provider: String(provider || "").trim() || undefined,
      keyword: String(keyword || "").trim() || undefined,
      page: 1,
      page_size: 200,
    });
    setCloudExpiryList(r.list ?? []);
  }, []);
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
          await Promise.all([loadDatasources(projectContextId), loadRules(projectContextId)]);
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
  }, [tab, projectContextId, loadDatasources, loadSilences, loadRules, loadCloudExpiryRules, cloudExpiryProviderFilter, cloudExpiryKeyword]);

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
      setPromResult(e instanceof Error ? e.message : String(e));
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

  async function runDsPing(id: number) {
    setDsPingId(id);
    try {
      const res = await pingAlertDatasource(id);
      if (res.ok) {
        message.success(`连通正常，耗时 ${res.latency_ms} ms`);
      } else {
        message.error(res.message || "连通失败");
      }
    } catch (e) {
      message.error(e instanceof Error ? e.message : String(e));
    } finally {
      setDsPingId(null);
    }
  }

  const dsColumns = [
    { title: "ID", dataIndex: "id", width: 70 },
    { title: "项目", dataIndex: "project_name", width: 160, render: (v: string, r: AlertDatasourceItem) => v || String(r.project_id || "-") },
    { title: "名称", dataIndex: "name" },
    { title: "地址", dataIndex: "base_url", ellipsis: true },
    { title: "启用", dataIndex: "enabled", width: 80, render: (v: boolean) => (v ? <Tag color="green">是</Tag> : <Tag>否</Tag>) },
    {
      title: "操作",
      width: 240,
      render: (_: unknown, r: AlertDatasourceItem) => (
        <Space wrap>
          <Button
            type="link"
            size="small"
            icon={<ApiOutlined />}
            loading={dsPingId === r.id}
            onClick={() => void runDsPing(r.id)}
          >
            连通检测
          </Button>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openDsEdit(r)}>
            编辑
          </Button>
          <Popconfirm title="删除数据源？" onConfirm={() => void removeDs(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  function openDsCreate() {
    setDsCurrent(null);
    dsForm.resetFields();
    const fallbackProjectID = projectContextId ?? projects[0]?.id;
    dsForm.setFieldsValue({ project_id: fallbackProjectID, type: "prometheus", skip_tls_verify: false, enabled: true });
    setDsModalOpen(true);
  }

  function openDsEdit(r: AlertDatasourceItem) {
    setDsCurrent(r);
    dsForm.setFieldsValue({
      project_id: r.project_id,
      name: r.name,
      type: r.type,
      base_url: r.base_url,
      basic_user: r.basic_user ?? "",
      skip_tls_verify: r.skip_tls_verify,
      enabled: r.enabled,
      remark: r.remark,
    });
    setDsModalOpen(true);
  }

  async function submitDs() {
    setDsSubmitting(true);
    try {
      const v = await dsForm.validateFields();
      if (dsCurrent) {
        await updateAlertDatasource(dsCurrent.id, v);
        message.success("已更新");
      } else {
        await createAlertDatasource(v);
        message.success("已创建");
      }
      setDsModalOpen(false);
      await loadDatasources(projectContextId);
    } finally {
      setDsSubmitting(false);
    }
  }

  async function removeDs(id: number) {
    await deleteAlertDatasource(id);
    message.success("已删除");
    await loadDatasources(projectContextId);
  }

  const nativeAlertsColumns: ColumnsType<PromNativeAlertRow> = useMemo(
    () => [
      { title: "告警名", dataIndex: "alertname", width: 160, ellipsis: true },
      {
        title: "状态",
        dataIndex: "state",
        width: 120,
        render: (v: string) => {
          const s = String(v || "").toLowerCase();
          const firing = s === "firing";
          const resolved = s === "resolved";
          return (
            <Space size={6}>
              <Badge status={firing ? "error" : resolved ? "success" : "default"} />
              <Typography.Text>{v || "-"}</Typography.Text>
            </Space>
          );
        },
      },
      { title: "Labels", dataIndex: "labelsShort", ellipsis: true },
      { title: "activeAt", dataIndex: "activeAt", width: 180, ellipsis: true },
      {
        title: "操作",
        width: 110,
        render: (_: unknown, r: PromNativeAlertRow) => (
          <Button type="link" size="small" onClick={() => openQuickSilence([r])}>
            静默
          </Button>
        ),
      },
    ],
    [],
  );

  const silColumns = [
    { title: "ID", dataIndex: "id", width: 70 },
    { title: "名称", dataIndex: "name" },
    {
      title: "说明",
      dataIndex: "comment",
      width: 140,
      ellipsis: true,
      render: (c: string) => (c && String(c).trim() ? c : "—"),
    },
    {
      title: "匹配摘要",
      key: "m",
      width: 200,
      ellipsis: true,
      render: (_: unknown, r: AlertSilenceItem) => {
        if (r.matchers?.length) {
          return r.matchers.map((x) => `${x.name ?? ""}=${x.value ?? ""}`).join(", ");
        }
        return r.matchers_json?.slice(0, 80) ?? "—";
      },
    },
    { title: "开始", dataIndex: "starts_at", width: 170, render: (t: string) => formatDateTime(t) },
    { title: "结束", dataIndex: "ends_at", width: 170, render: (t: string) => formatDateTime(t) },
    {
      title: "状态",
      key: "status",
      width: 100,
      render: (_: unknown, r: AlertSilenceItem) => {
        const expired = dayjs(r.ends_at).isBefore(dayjs());
        if (expired) return <Tag color="red">已过期</Tag>;
        return r.enabled ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>;
      },
    },
    {
      title: "操作",
      width: 230,
      render: (_: unknown, r: AlertSilenceItem) => (
        <Space>
          <Button type="link" size="small" disabled={!r.enabled} onClick={() => void releaseSingleSilence(r)}>
            解除静默
          </Button>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openSilEdit(r)}>
            编辑
          </Button>
          <Popconfirm title="删除静默？" onConfirm={() => void removeSil(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  function openSilCreate() {
    setSilCurrent(null);
    silForm.resetFields();
    silForm.setFieldsValue({
      name: "",
      matchers: [{ name: "alertname", value: "", is_regex: false }],
      comment: "",
      enabled: true,
      starts_at: dayjs(),
      ends_at: dayjs().add(2, "hour"),
    });
    setSilModalOpen(true);
  }

  /** 为平台内置监控规则预填静默（monitor_rule_id + alertname），与 Prometheus 活跃告警列表无关。 */
  function openSilenceForMonitorRule(r: AlertMonitorRuleItem) {
    setSilCurrent(null);
    silForm.resetFields();
    const ruleName = String(r.name || "").trim();
    silForm.setFieldsValue({
      name: ruleName ? `静默规则 ${ruleName}` : `静默 monitor_rule ${r.id}`,
      matchers: [
        { name: "monitor_rule_id", value: String(r.id), is_regex: false },
        ...(ruleName ? [{ name: "alertname", value: ruleName, is_regex: false }] : []),
      ],
      comment: "平台监控规则一键静默",
      enabled: true,
      starts_at: dayjs(),
      ends_at: dayjs().add(2, "hour"),
    });
    setSilModalOpen(true);
  }

  function toQuickSilenceTarget(row: PromNativeAlertRow): QuickSilenceTarget {
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

  function openQuickSilence(rows: PromNativeAlertRow[]) {
    setQuickSilenceComment("");
    const targets = rows.map(toQuickSilenceTarget);
    if (!targets.length) {
      message.warning("请先选择需要静默的告警");
      return;
    }
    setQuickSilenceTargets(targets);
    setQuickSilenceOpen(true);
  }

  function buildMatchersByLabels(labels: Record<string, string>): SilenceMatcherForm[] {
    return Object.entries(labels ?? {})
      .map(([name, value]) => ({ name: String(name || "").trim(), value: String(value || "").trim(), is_regex: false }))
      .filter((m) => m.name && m.value);
  }

  async function submitQuickSilence() {
    if (!quickSilenceTargets.length) {
      setQuickSilenceOpen(false);
      return;
    }
    for (const it of quickSilenceTargets) {
      if (!it.endsAt.isAfter(it.startsAt)) {
        message.error(`「${it.name}」结束时间必须晚于开始时间`);
        return;
      }
    }
    setQuickSilenceSubmitting(true);
    try {
      const comment = quickSilenceComment.trim();
      const items = quickSilenceTargets.map((it) => ({
        name: it.name,
        matchers_json: JSON.stringify(buildMatchersByLabels(it.labels)),
        comment,
        enabled: true,
        starts_at: it.startsAt.toISOString(),
        ends_at: it.endsAt.toISOString(),
      }));
      const { created } = await createAlertSilencesBatch(items);
      message.success(`已创建 ${created} 条静默`);
      setQuickSilenceOpen(false);
      await loadSilences();
    } finally {
      setQuickSilenceSubmitting(false);
    }
  }

  function openSilEdit(r: AlertSilenceItem) {
    setSilCurrent(r);
    silForm.setFieldsValue({
      name: r.name,
      matchers: r.matchers?.length ? r.matchers : parseSilenceMatchersForForm(r.matchers_json),
      comment: r.comment,
      enabled: r.enabled,
      starts_at: dayjs(r.starts_at),
      ends_at: dayjs(r.ends_at),
    });
    setSilModalOpen(true);
  }

  async function submitSil() {
    setSilSubmitting(true);
    try {
      const v = await silForm.validateFields();
      const rawMatchers = (v.matchers ?? []) as SilenceMatcherForm[];
      const matchers = rawMatchers
        .map((m) => ({
          name: String(m?.name ?? "").trim(),
          value: String(m?.value ?? "").trim(),
          is_regex: Boolean(m?.is_regex),
        }))
        .filter((m) => m.name !== "");
      if (matchers.length === 0) {
        message.error("至少添加一条匹配器，并填写名称（如 alertname）");
        return;
      }
      const payload = {
        name: v.name,
        matchers_json: JSON.stringify(matchers),
        comment: v.comment,
        enabled: v.enabled,
        starts_at: (v.starts_at as Dayjs).toISOString(),
        ends_at: (v.ends_at as Dayjs).toISOString(),
        project_id: projectContextId && projectContextId > 0 ? projectContextId : undefined,
      };
      if (silCurrent) {
        await updateAlertSilence(silCurrent.id, payload);
        message.success("已更新");
      } else {
        await createAlertSilence(payload);
        message.success("已创建");
      }
      setSilModalOpen(false);
      await loadSilences();
    } finally {
      setSilSubmitting(false);
    }
  }

  async function removeSil(id: number) {
    await deleteAlertSilence(id);
    message.success("已删除");
    await loadSilences();
  }

  async function releaseSilenceNow(row: AlertSilenceItem) {
    await updateAlertSilence(row.id, {
      name: row.name,
      matchers_json: row.matchers_json,
      comment: row.comment ?? "",
      enabled: false,
      starts_at: row.starts_at,
      ends_at: row.ends_at,
    });
  }

  async function releaseSingleSilence(row: AlertSilenceItem) {
    await releaseSilenceNow(row);
    message.success("已解除静默");
    await loadSilences();
  }

  async function releaseSelectedSilences() {
    const rows = silenceList.filter((it) => selectedSilenceIds.includes(it.id) && it.enabled);
    if (!rows.length) {
      message.warning("请选择需要解除的启用静默");
      return;
    }
    const results = await Promise.allSettled(rows.map((r) => releaseSilenceNow(r)));
    const ok = results.filter((r) => r.status === "fulfilled").length;
    const fail = rows.length - ok;
    if (ok > 0) message.success(`已解除 ${ok} 条静默`);
    if (fail > 0) message.warning(`${fail} 条静默解除失败`);
    setSelectedSilenceIds([]);
    await loadSilences();
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
      message.error(`加载指标失败：${e instanceof Error ? e.message : String(e)}`);
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
      message.error(`加载标签值失败：${e instanceof Error ? e.message : String(e)}`);
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
    } finally {
      setRuleSubmitting(false);
    }
  }

  async function removeRule(id: number) {
    await deleteAlertMonitorRule(id);
    message.success("已删除");
    await loadRules(projectContextId);
  }

  const cloudExpiryColumns: ColumnsType<CloudExpiryRuleItem> = [
    { title: "ID", dataIndex: "id", width: 70 },
    { title: "项目", dataIndex: "project_id", width: 90 },
    { title: "规则名", dataIndex: "name", width: 180 },
    {
      title: "厂商",
      dataIndex: "provider",
      width: 110,
      render: (v: string) => {
        const p = String(v || "").trim();
        if (!p) return "全部";
        if (p === "alibaba") return "阿里云";
        if (p === "tencent") return "腾讯云";
        if (p === "jd") return "京东云";
        return p;
      },
    },
    { title: "地域范围", dataIndex: "region_scope", width: 180, render: (v: string) => String(v || "").trim() || "全部" },
    { title: "提前天数", dataIndex: "advance_days", width: 100 },
    { title: "级别", dataIndex: "severity", width: 90 },
    { title: "定时", dataIndex: "schedule_enabled", width: 80, render: (v: boolean) => (v !== false ? <Tag color="blue">开</Tag> : <Tag>关</Tag>) },
    {
      title: "Cron",
      dataIndex: "eval_cron_spec",
      width: 160,
      ellipsis: true,
      render: (v: string) => {
        const s = String(v || "").trim();
        return s ? <span title={s}>{s}</span> : <span style={{ color: "#999" }}>—</span>;
      },
    },
    { title: "启用", dataIndex: "enabled", width: 80, render: (v: boolean) => (v ? <Tag color="green">是</Tag> : <Tag>否</Tag>) },
    { title: "创建时间", dataIndex: "created_at", width: 170, render: (v: string) => (v ? formatDateTime(v) : "-") },
    { title: "更新时间", dataIndex: "updated_at", width: 170, render: (v: string) => (v ? formatDateTime(v) : "-") },
    {
      title: "操作",
      width: 180,
      fixed: "right",
      render: (_: unknown, r: CloudExpiryRuleItem) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openCloudExpiryEdit(r)}>
            编辑
          </Button>
          <Popconfirm title="删除云到期规则？" onConfirm={() => void removeCloudExpiryRule(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  function openCloudExpiryCreate() {
    setCloudExpiryCurrent(null);
    cloudExpiryForm.resetFields();
    cloudExpiryForm.setFieldsValue({
      project_id: projectContextId,
      provider: "",
      region_scope: "",
      advance_days: 7,
      severity: "warning",
      eval_cron_spec: "0 9 * * *",
      schedule_enabled: true,
      labels_json: "{}",
      enabled: true,
    });
    setCloudExpiryModalOpen(true);
  }

  function openCloudExpiryEdit(row: CloudExpiryRuleItem) {
    setCloudExpiryCurrent(row);
    cloudExpiryForm.setFieldsValue({
      project_id: row.project_id,
      name: row.name,
      provider: row.provider || "",
      region_scope: row.region_scope || "",
      advance_days: row.advance_days,
      severity: row.severity || "warning",
      eval_cron_spec: row.eval_cron_spec ?? "",
      schedule_enabled: row.schedule_enabled !== false,
      labels_json: stringifyPrettyJSON(row.labels ?? {}, "{}"),
      enabled: row.enabled,
    });
    setCloudExpiryModalOpen(true);
  }

  async function submitCloudExpiryRule() {
    setCloudExpirySubmitting(true);
    try {
      const v = await cloudExpiryForm.validateFields();
      const payload = {
        ...v,
        provider: String(v.provider || "").trim(),
        region_scope: String(v.region_scope || "").trim(),
        labels_json: String(v.labels_json || "{}").trim() || "{}",
        eval_cron_spec: String(v.eval_cron_spec ?? "").trim(),
      };
      if (cloudExpiryCurrent) {
        await updateCloudExpiryRule(cloudExpiryCurrent.id, payload);
        message.success("已更新云到期规则");
      } else {
        await createCloudExpiryRule(payload);
        message.success("已创建云到期规则");
      }
      setCloudExpiryModalOpen(false);
      await loadCloudExpiryRules(projectContextId, cloudExpiryProviderFilter, cloudExpiryKeyword);
    } finally {
      setCloudExpirySubmitting(false);
    }
  }

  async function removeCloudExpiryRule(id: number) {
    await deleteCloudExpiryRule(id);
    message.success("已删除");
    await loadCloudExpiryRules(projectContextId, cloudExpiryProviderFilter, cloudExpiryKeyword);
  }

  async function runCloudExpiryEvalNow() {
    setCloudExpiryEvaluating(true);
    try {
      await evaluateCloudExpiryRulesNow();
      message.success({
        content:
          "评估已完成。历史记录仅在存在「剩余天数 ≤ 提前天数」的实例时产生 firing；无到期实例则不会有新记录。请在历史记录中搜索规则名，或数据源选「云资源到期」；未配置 encryption_key 时接口会报错。",
        duration: 9,
      });
    } finally {
      setCloudExpiryEvaluating(false);
    }
  }

  function normalizeCloudExpiryLabelsJSON(raw: string): string | null {
    const s = String(raw || "").trim();
    if (!s) return "{}";
    try {
      const parsed = JSON.parse(s) as unknown;
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return null;
      return stringifyPrettyJSON(parsed, "{}");
    } catch {
      return null;
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
        notify_on_resolved: row?.notify_on_resolved ?? false,
        remark: row?.remark ?? "",
      });
    } catch {
      assignForm.setFieldsValue({ user_ids: [], department_ids: [], notify_on_resolved: false, remark: "" });
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
            message.warning(`处理人已保存，但写回用户资料失败：${e instanceof Error ? e.message : String(e)}`);
          }
        }
      }
      await upsertMonitorRuleAssignees(assignRuleId, {
        user_ids_json: JSON.stringify(userIds),
        department_ids_json: JSON.stringify(deptIds),
        extra_emails_json: "[]",
        notify_on_resolved: v.notify_on_resolved,
        remark: v.remark ?? "",
      });
      message.success("处理人已保存");
      setAssignOpen(false);
    } finally {
      setAssignSubmitting(false);
    }
  }

  const copyDutyRuleOptions = useMemo(() => {
    if (!dutyRuleId) return [];
    return ruleList
      .filter((r) => r.id !== dutyRuleId)
      .map((r) => ({ label: r.name, value: r.id }));
  }, [ruleList, dutyRuleId]);

  async function openDuty(ruleId: number) {
    setDutyRuleId(ruleId);
    setCopySourceRuleId(undefined);
    setBlockList([]);
    try {
      const r = await listDutyBlocks({ monitor_rule_id: ruleId, page: 1, page_size: 500 });
      setBlockList(r.list ?? []);
    } catch {
      setBlockList([]);
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
      message.error(e instanceof Error ? e.message : String(e));
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
            message.warning(`班次已保存，但写回用户资料失败：${e instanceof Error ? e.message : String(e)}`);
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
    openRuleCreate,
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
