import {
  StarOutlined,
  DownloadOutlined,
  FilterOutlined,
  ReloadOutlined,
  RobotOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Checkbox,
  Col,
  Collapse,
  DatePicker,
  Drawer,
  Dropdown,
  Form,
  Input,
  List,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs, { type Dayjs } from "dayjs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import {
  analyzeProjectLogs,
  exportProjectLogs,
  getProjectLogAnomalies,
  getProjectLogContext,
  getProjectLogFields,
  getProjectLogOverview,
  getProjectLogPatterns,
  getProjectLogSources,
  getProjectLogTopN,
  getProjectServers,
  getProjectServices,
  getProjects,
  searchProjectLogs,
  updateProjectLogAnomaly,
  type LogAnalyzeResult,
  type LogAnomalyItem,
  type LogContextResult,
  type LogFieldStat,
  type LogHistogramBucket,
  type LogOverviewResult,
  type LogPatternItem,
  type LogSearchItem,
  type LogSourceItem,
  type LogTopNResult,
  type ProjectItem,
  type ServerItem,
  type ServiceItem,
} from "../services/projects";
import {
  createLogDropRule,
  createLogSavedQuery,
  deleteLogDropRule,
  deleteLogSavedQuery,
  getLoggieStatus,
  listLogDropRules,
  listLogSavedQueries,
  updateLogDropRule,
  type LogDropRuleItem,
  type LogSavedQueryItem,
} from "../services/log-platform";
import { getClusters, type ClusterItem } from "../services/clusters";
import { extractApiErrorMessage } from "../services/http";
import { formatDateTime } from "../utils/format";
import { LEVEL_STACK_COLORS, loadLogViewerPrefs, saveLogViewerPrefs } from "../utils/log-viewer-prefs";

type SearchForm = {
  project_id?: number;
  server_id?: number;
  service_id?: number;
  log_source_id?: number;
  service_name?: string;
  host?: string;
  collector_mode?: string;
  cluster_id?: number;
  namespace?: string;
  pod?: string;
  container?: string;
  keyword?: string;
  level?: string;
  file_path?: string;
  extra_field?: string;
  extra_value?: string;
  index_pattern?: string;
  time_range?: [Dayjs, Dayjs];
  page?: number;
  page_size?: number;
};

type FacetItem = { key: string; label: string; count: number };

const TIME_PRESETS: Array<{ label: string; minutes: number }> = [
  { label: "5m", minutes: 5 },
  { label: "15m", minutes: 15 },
  { label: "1h", minutes: 60 },
  { label: "6h", minutes: 360 },
  { label: "24h", minutes: 1440 },
  { label: "7d", minutes: 10080 },
];

export function ProjectLogsPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [servers, setServers] = useState<ServerItem[]>([]);
  const [services, setServices] = useState<ServiceItem[]>([]);
  const [sources, setSources] = useState<LogSourceItem[]>([]);
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [rows, setRows] = useState<LogSearchItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [overview, setOverview] = useState<LogOverviewResult | null>(null);
  const [patterns, setPatterns] = useState<LogPatternItem[]>([]);
  const [patternTotal, setPatternTotal] = useState(0);
  const [anomalies, setAnomalies] = useState<LogAnomalyItem[]>([]);
  const [anomalyTotal, setAnomalyTotal] = useState(0);
  const [activeTab, setActiveTab] = useState("logs");
  const [listSubTab, setListSubTab] = useState<"all" | "cluster">("all");
  const prefsInit = loadLogViewerPrefs();
  const [chartMode, setChartMode] = useState<"timeline" | "pie">(prefsInit.chartMode);
  const [aiOpen, setAiOpen] = useState(false);
  const [aiLoading, setAiLoading] = useState(false);
  const [aiResult, setAiResult] = useState<LogAnalyzeResult | null>(null);
  const [facetSearch, setFacetSearch] = useState("");
  const [activePreset, setActivePreset] = useState<string>("24h");
  const [fieldStats, setFieldStats] = useState<LogFieldStat[]>([]);
  const [listMode, setListMode] = useState<"stacked" | "compact">(prefsInit.listMode);
  const [visibleFields, setVisibleFields] = useState<string[]>(prefsInit.visibleFields);
  const [savedQueries, setSavedQueries] = useState<LogSavedQueryItem[]>([]);
  const [contextOpen, setContextOpen] = useState(false);
  const [contextLoading, setContextLoading] = useState(false);
  const [contextResult, setContextResult] = useState<LogContextResult | null>(null);
  const [parseOpen, setParseOpen] = useState(false);
  const [parseRow, setParseRow] = useState<LogSearchItem | null>(null);
  const [guideTips, setGuideTips] = useState<string[]>([]);
  const [dropOpen, setDropOpen] = useState(false);
  const [dropRules, setDropRules] = useState<LogDropRuleItem[]>([]);
  const [dropLoading, setDropLoading] = useState(false);
  const [topnDim, setTopnDim] = useState("service");
  const [topn, setTopn] = useState<LogTopNResult | null>(null);
  const [topnLoading, setTopnLoading] = useState(false);
  const [dropForm] = Form.useForm<{ name: string; field: string; operator: string; value: string; enabled: boolean }>();

  const [form] = Form.useForm<SearchForm>();
  const watchCollectorMode = Form.useWatch("collector_mode", form);
  const watchLevel = Form.useWatch("level", form);
  const watchServiceName = Form.useWatch("service_name", form);
  const watchHost = Form.useWatch("host", form);
  const watchNamespace = Form.useWatch("namespace", form);
  const watchPod = Form.useWatch("pod", form);
  const watchExtraField = Form.useWatch("extra_field", form);
  const watchExtraValue = Form.useWatch("extra_value", form);

  useEffect(() => {
    saveLogViewerPrefs({ listMode, visibleFields, chartMode });
  }, [listMode, visibleFields, chartMode]);

  const projectOptions = useMemo(() => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })), [projects]);
  const serverOptions = useMemo(
    () => servers.map((s) => ({ value: s.id, label: `${s.name} ${s.host}:${s.port}` })),
    [servers],
  );
  const serviceOptions = useMemo(() => services.map((s) => ({ value: s.id, label: s.name })), [services]);
  const sourceOptions = useMemo(() => sources.map((s) => ({ value: s.id, label: `${s.log_type}:${s.path}` })), [sources]);
  const clusterOptions = useMemo(() => clusters.map((c) => ({ value: c.id, label: c.name })), [clusters]);

  const reloadServers = useCallback(async (projectId?: number) => {
    if (!projectId) return;
    const data = await getProjectServers(projectId, { page: 1, page_size: 1000 });
    setServers(data.list);
  }, []);

  const reloadServices = useCallback(async (projectId?: number, serverId?: number) => {
    if (!projectId) return;
    const data = await getProjectServices(projectId, { page: 1, page_size: 1000, server_id: serverId });
    setServices(data.list);
  }, []);

  const reloadSources = useCallback(async (projectId?: number, serviceId?: number) => {
    if (!projectId) return;
    const data = await getProjectLogSources(projectId, { page: 1, page_size: 1000, service_id: serviceId });
    setSources(data.list);
  }, []);

  const [emptyHint, setEmptyHint] = useState<string>("");
  const [scannedHint, setScannedHint] = useState("");

  const buildSearchParams = useCallback((values: SearchForm, page?: number, pageSize?: number) => {
    const range = values.time_range;
    return {
      server_id: values.server_id,
      service_id: values.service_id,
      log_source_id: values.log_source_id,
      service_name: values.service_name?.trim() || undefined,
      host: values.host?.trim() || undefined,
      collector_mode: values.collector_mode || undefined,
      cluster_id: values.cluster_id,
      namespace: values.namespace?.trim() || undefined,
      pod: values.pod?.trim() || undefined,
      container: values.container?.trim() || undefined,
      keyword: values.keyword?.trim() || undefined,
      level: values.level?.trim() || undefined,
      file_path: values.file_path?.trim() || undefined,
      extra_field: values.extra_field?.trim() || undefined,
      extra_value: values.extra_value?.trim() || undefined,
      index_pattern: values.index_pattern?.trim() || undefined,
      from: range?.[0]?.toISOString(),
      to: range?.[1]?.toISOString(),
      page: page ?? values.page ?? 1,
      page_size: pageSize ?? values.page_size ?? 100,
    };
  }, []);

  const reloadSavedQueries = useCallback(async (projectId?: number) => {
    if (!projectId) {
      setSavedQueries([]);
      return;
    }
    try {
      const res = await listLogSavedQueries(projectId);
      setSavedQueries(res?.list || []);
    } catch {
      setSavedQueries([]);
    }
  }, []);

  const reloadDropRules = useCallback(async (projectId?: number) => {
    if (!projectId) {
      setDropRules([]);
      return;
    }
    try {
      const res = await listLogDropRules(projectId);
      setDropRules(res?.list || []);
    } catch {
      setDropRules([]);
    }
  }, []);

  const loadTopN = useCallback(
    async (dim?: string) => {
      const values = form.getFieldsValue();
      if (!values.project_id) return;
      const d = dim || topnDim;
      setTopnLoading(true);
      try {
        const res = await getProjectLogTopN(values.project_id, {
          ...buildSearchParams(values),
          dim: d,
          size: 15,
        });
        setTopn(res);
      } catch {
        setTopn(null);
      } finally {
        setTopnLoading(false);
      }
    },
    [form, buildSearchParams, topnDim],
  );

  const refreshEmptyGuide = useCallback(async (projectId?: number, totalHits?: number) => {
    if (!projectId || (totalHits ?? 0) > 0) {
      setGuideTips([]);
      return;
    }
    const tips: string[] = ["当前筛选无日志命中，请按链路排查："];
    try {
      const st = await getLoggieStatus(projectId);
      const list = st?.list || [];
      if (!list.length) {
        tips.push("① 未登记主机 Agent → 前往「Agent 管理」引导安装");
      } else {
        const online = list.filter((x) => x.online).length;
        tips.push(`① 主机 Agent：共 ${list.length} 台，在线 ${online} 台`);
        if (online === 0) tips.push("② Agent 全部离线 → 检查进程 / heartbeat / Token");
        else if (!list.some((x) => x.recent_ingest || x.es_sink_ok)) {
          tips.push("② Agent 在线但未见近期写入 → 检查日志源路径与 Kafka/ES");
        }
      }
    } catch {
      tips.push("① 无法拉取 Agent 状态，请确认权限与后端可用");
    }
    tips.push("③ K8s 采集请确认 DaemonSet Ready，并到 Pipeline 仓库核对解析");
    tips.push("④ 可尝试扩大时间范围，或清空级别/服务/主机筛选");
    setGuideTips(tips);
  }, []);

  const loadOverviewAndIntel = useCallback(
    async (values: SearchForm) => {
      if (!values.project_id) return;
      const params = buildSearchParams(values);
      const [ov, pat, an, fields] = await Promise.all([
        getProjectLogOverview(values.project_id, params).catch(() => null),
        getProjectLogPatterns(values.project_id, { ...params, page: 1, page_size: 20 }).catch(() => ({ list: [], total: 0 })),
        getProjectLogAnomalies(values.project_id, { page: 1, page_size: 50 }).catch(() => ({ list: [], total: 0 })),
        getProjectLogFields(values.project_id, params).catch(() => null),
      ]);
      setOverview(ov);
      setPatterns(pat?.list ?? []);
      setPatternTotal(pat?.total ?? 0);
      setAnomalies(an?.list ?? []);
      setAnomalyTotal(an?.total ?? 0);
      setFieldStats(fields?.fields ?? []);
      void loadTopN();
    },
    [buildSearchParams, loadTopN],
  );

  const runSearch = useCallback(
    async (override?: Partial<SearchForm>) => {
      const values = { ...form.getFieldsValue(), ...override };
      if (!values.project_id) {
        message.warning("请选择项目");
        return;
      }
      const page = values.page ?? 1;
      const pageSize = values.page_size ?? 100;
      const range = values.time_range;
      const filePath = values.file_path?.trim() || undefined;
      if (range?.[0] && range?.[1] && range[1].diff(range[0], "day") > 7) {
        message.warning("时间范围超过 7 天，检索可能较慢；建议缩小到 1 天内");
      }
      setLoading(true);
      setEmptyHint("");
      try {
        const params = buildSearchParams(values, page, pageSize);
        const res = await searchProjectLogs(values.project_id, params);
        setRows(res.list);
        setTotal(res.total);
        setScannedHint(`请求完成，共 ${res.total.toLocaleString()} 条（当前页 ${res.list.length} 条）`);
        form.setFieldsValue({ page, page_size: pageSize });
        void loadOverviewAndIntel({ ...values, page, page_size: pageSize });
        void refreshEmptyGuide(values.project_id, res.total);
        if ((res.total ?? 0) === 0 && filePath) {
          setEmptyHint(`按文件名「${filePath}」无命中，可清空文件名或扩大时间范围。`);
        } else if ((res.total ?? 0) === 0 && !range?.[0] && !range?.[1]) {
          setEmptyHint("未选时间范围且无数据，请确认 Agent / DaemonSet 已写入 ES。");
        } else {
          setEmptyHint("");
        }
      } catch (e: unknown) {
        message.error(extractApiErrorMessage(e));
      } finally {
        setLoading(false);
      }
    },
    [form, buildSearchParams, loadOverviewAndIntel, refreshEmptyGuide],
  );

  const runAiAnalyze = useCallback(async () => {
    const values = form.getFieldsValue();
    if (!values.project_id) {
      message.warning("请选择项目");
      return;
    }
    setAiOpen(true);
    setAiLoading(true);
    setAiResult(null);
    try {
      const params = buildSearchParams(values);
      const res = await analyzeProjectLogs(values.project_id, params);
      setAiResult(res);
    } catch (e: unknown) {
      message.error(extractApiErrorMessage(e));
    } finally {
      setAiLoading(false);
    }
  }, [form, buildSearchParams]);

  useEffect(() => {
    void (async () => {
      const [data, clusterRes] = await Promise.all([
        getProjects({ page: 1, page_size: 1000 }),
        getClusters({ page: 1, page_size: 1000 }).catch(() => ({ list: [] as ClusterItem[] })),
      ]);
      setProjects(data.list);
      setClusters(clusterRes?.list || []);

      const qPid = Number(searchParams.get("project_id") || 0);
      const defaultProject = qPid || data.list[0]?.id;
      const tab = searchParams.get("tab");
      if (tab === "patterns" || tab === "anomalies" || tab === "logs" || tab === "overview") {
        setActiveTab(tab === "overview" ? "analysis" : tab);
      }
      const anchorTime = searchParams.get("anchor_time");
      const windowMinutes = Number(searchParams.get("window_minutes") || 5) || 5;
      let timeRange: [Dayjs, Dayjs] = [dayjs().subtract(24, "hour"), dayjs()];
      if (anchorTime) {
        const anchor = dayjs(anchorTime);
        if (anchor.isValid()) {
          timeRange = [anchor.subtract(windowMinutes, "minute"), anchor.add(windowMinutes, "minute")];
          setActivePreset("");
        }
      } else {
        setActivePreset("24h");
      }
      const initial: Partial<SearchForm> = {
        project_id: defaultProject,
        page: 1,
        page_size: 100,
        time_range: timeRange,
        level: searchParams.get("level") || undefined,
        log_source_id: Number(searchParams.get("log_source_id") || 0) || undefined,
        service_id: Number(searchParams.get("service_id") || 0) || undefined,
      };
      if (defaultProject) {
        form.setFieldsValue(initial);
        await reloadServers(defaultProject);
        await reloadServices(defaultProject, initial.server_id);
        if (initial.service_id) {
          await reloadSources(defaultProject, initial.service_id);
        }
        await runSearch(initial);
        void reloadSavedQueries(defaultProject);
        void reloadDropRules(defaultProject);
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function countsToFacets(counts?: Record<string, number>, limit = 15): FacetItem[] {
    return Object.entries(counts || {})
      .map(([key, count]) => ({ key, label: key, count: Number(count) || 0 }))
      .sort((a, b) => b.count - a.count)
      .slice(0, limit);
  }

  const levelFacets = useMemo<FacetItem[]>(() => countsToFacets(overview?.level_counts), [overview]);

  const serviceFacets = useMemo<FacetItem[]>(() => {
    const fromOverview = countsToFacets(overview?.service_name_counts);
    if (fromOverview.length) return fromOverview;
    const map = new Map<string, number>();
    for (const r of rows) {
      const name = r.service_name?.trim();
      if (!name) continue;
      map.set(name, (map.get(name) || 0) + 1);
    }
    return Array.from(map.entries())
      .map(([key, count]) => ({ key, label: key, count }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 12);
  }, [overview, rows]);

  const hostFacets = useMemo<FacetItem[]>(() => {
    const fromOverview = countsToFacets(overview?.host_counts);
    if (fromOverview.length) return fromOverview;
    const map = new Map<string, number>();
    for (const r of rows) {
      const host = (r.host || r.server_host || "").trim();
      if (!host) continue;
      map.set(host, (map.get(host) || 0) + 1);
    }
    return Array.from(map.entries())
      .map(([key, count]) => ({ key, label: key, count }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 12);
  }, [overview, rows]);

  const namespaceFacets = useMemo(() => countsToFacets(overview?.namespace_counts), [overview]);
  const podFacets = useMemo(() => countsToFacets(overview?.pod_counts), [overview]);
  const containerFacets = useMemo(() => countsToFacets(overview?.container_counts), [overview]);

  const filteredFacetGroups = useMemo(() => {
    const q = facetSearch.trim().toLowerCase();
    const filter = (items: FacetItem[]) => (!q ? items : items.filter((i) => i.label.toLowerCase().includes(q)));
    return [
      { title: "日志级别", field: "level" as const, items: filter(levelFacets) },
      { title: "服务", field: "service" as const, items: filter(serviceFacets) },
      { title: "主机", field: "host" as const, items: filter(hostFacets) },
      { title: "Namespace", field: "namespace" as const, items: filter(namespaceFacets) },
      { title: "Pod", field: "pod" as const, items: filter(podFacets) },
      { title: "容器", field: "container" as const, items: filter(containerFacets) },
    ];
  }, [facetSearch, levelFacets, serviceFacets, hostFacets, namespaceFacets, podFacets, containerFacets]);

  function applyFacet(
    field: "level" | "service" | "host" | "namespace" | "pod" | "container",
    value: string,
  ) {
    const toggle = (cur: string | undefined) => (cur === value ? undefined : value);
    if (field === "level") {
      const next = toggle(watchLevel);
      form.setFieldsValue({ level: next });
      void runSearch({ page: 1, level: next });
      return;
    }
    if (field === "service") {
      const next = toggle(watchServiceName);
      form.setFieldsValue({ service_name: next });
      void runSearch({ page: 1, service_name: next });
      return;
    }
    if (field === "host") {
      const next = toggle(watchHost);
      form.setFieldsValue({ host: next });
      void runSearch({ page: 1, host: next });
      return;
    }
    if (field === "namespace") {
      const next = toggle(watchNamespace);
      form.setFieldsValue({ namespace: next });
      void runSearch({ page: 1, namespace: next });
      return;
    }
    if (field === "pod") {
      const next = toggle(watchPod);
      form.setFieldsValue({ pod: next });
      void runSearch({ page: 1, pod: next });
      return;
    }
    const cur = form.getFieldValue("container") as string | undefined;
    const next = toggle(cur);
    form.setFieldsValue({ container: next });
    void runSearch({ page: 1, container: next });
  }

  function applyFieldClick(name: string, value: string) {
    const n = name.trim();
    const v = value.trim();
    if (!n || !v) return;
    if (n === "level" || n === "status") {
      applyFacet("level", v.toUpperCase());
      return;
    }
    if (n === "service_name" || n === "service") {
      applyFacet("service", v);
      return;
    }
    if (n === "host" || n === "server_host" || n === "hostname") {
      applyFacet("host", v);
      return;
    }
    if (n === "namespace") {
      applyFacet("namespace", v);
      return;
    }
    if (n === "pod" || n === "podname") {
      applyFacet("pod", v);
      return;
    }
    if (n === "container" || n === "containername") {
      applyFacet("container", v);
      return;
    }
    const clear =
      watchExtraField === n && watchExtraValue === v
        ? { extra_field: undefined, extra_value: undefined }
        : { extra_field: n, extra_value: v };
    form.setFieldsValue(clear);
    void runSearch({ page: 1, ...clear });
  }

  function zoomHistogramBucket(bucket: LogHistogramBucket) {
    const start = dayjs(bucket.time);
    if (!start.isValid()) return;
    const range = form.getFieldValue("time_range") as [Dayjs, Dayjs] | undefined;
    const spanMs = range?.[0] && range?.[1] ? Math.max(60_000, range[1].diff(range[0]) / 30) : 5 * 60_000;
    const half = Math.floor(spanMs / 2);
    const nextRange: [Dayjs, Dayjs] = [start.subtract(half, "millisecond"), start.add(half, "millisecond")];
    setActivePreset("");
    form.setFieldsValue({ time_range: nextRange });
    void runSearch({ page: 1, time_range: nextRange });
  }

  async function openLogContext(row: LogSearchItem) {
    const pid = form.getFieldValue("project_id") as number | undefined;
    if (!pid) return;
    setContextOpen(true);
    setContextLoading(true);
    setContextResult(null);
    try {
      const res = await getProjectLogContext(pid, {
        anchor_time: row.timestamp,
        window_minutes: 5,
        service_id: row.service_id || undefined,
      });
      setContextResult(res);
    } catch (e: unknown) {
      message.error(extractApiErrorMessage(e, "加载上下文失败"));
    } finally {
      setContextLoading(false);
    }
  }

  async function saveCurrentQuery() {
    const pid = form.getFieldValue("project_id") as number | undefined;
    if (!pid) {
      message.warning("请选择项目");
      return;
    }
    const name = window.prompt("收藏名称");
    if (!name?.trim()) return;
    try {
      const values = form.getFieldsValue();
      const query = {
        ...buildSearchParams(values),
        time_range: values.time_range
          ? [values.time_range[0]?.toISOString(), values.time_range[1]?.toISOString()]
          : undefined,
      };
      await createLogSavedQuery(pid, { name: name.trim(), query });
      message.success("已收藏");
      void reloadSavedQueries(pid);
    } catch (e: unknown) {
      message.error(extractApiErrorMessage(e, "收藏失败"));
    }
  }

  function applySavedQuery(item: LogSavedQueryItem) {
    try {
      const q = JSON.parse(item.query || "{}") as Record<string, unknown>;
      const patch: Partial<SearchForm> = {
        keyword: (q.keyword as string) || undefined,
        level: (q.level as string) || undefined,
        service_name: (q.service_name as string) || undefined,
        host: (q.host as string) || undefined,
        namespace: (q.namespace as string) || undefined,
        pod: (q.pod as string) || undefined,
        container: (q.container as string) || undefined,
        collector_mode: (q.collector_mode as string) || undefined,
        file_path: (q.file_path as string) || undefined,
        extra_field: (q.extra_field as string) || undefined,
        extra_value: (q.extra_value as string) || undefined,
        page: 1,
      };
      if (Array.isArray(q.time_range) && q.time_range[0] && q.time_range[1]) {
        patch.time_range = [dayjs(String(q.time_range[0])), dayjs(String(q.time_range[1]))];
        setActivePreset("");
      }
      form.setFieldsValue(patch);
      void runSearch(patch);
    } catch {
      message.error("收藏条件解析失败");
    }
  }

  function applyTimePreset(label: string, minutes: number) {
    const range: [Dayjs, Dayjs] = [dayjs().subtract(minutes, "minute"), dayjs()];
    setActivePreset(label);
    form.setFieldsValue({ time_range: range });
    void runSearch({ page: 1, time_range: range });
  }

  function goAdjustPipeline(sampleOverride?: string[]) {
    const pid = form.getFieldValue("project_id") as number | undefined;
    const samples =
      sampleOverride ||
      rows
        .slice(0, 10)
        .map((r) => (r.message || "").trim())
        .filter(Boolean);
    navigate(`/log-pipelines?project_id=${pid || ""}&ai=1`, {
      state: {
        project_id: pid,
        open_ai: true,
        sample_logs: samples,
        goal: "根据当前日志样例优化 Loggie 解析，抽出 level/service/host/trace_id 等可观察字段",
      },
    });
  }

  const logColumns: ColumnsType<LogSearchItem> = [
    {
      title: "时间",
      dataIndex: "timestamp",
      width: 168,
      fixed: "left",
      render: (v: string) => <span className="log-viewer-time">{formatDateTime(v)}</span>,
    },
    {
      title: "完整信息",
      dataIndex: "message",
      render: (_: string, r) => (
        <LogLineCell
          row={r}
          visibleFields={visibleFields}
          stacked={listMode === "stacked"}
          onFieldClick={applyFieldClick}
          onContext={() => void openLogContext(r)}
          onParseCompare={() => {
            setParseRow(r);
            setParseOpen(true);
          }}
        />
      ),
    },
  ];

  return (
    <div className="project-logs-viewer">
      <div className="project-logs-viewer-toolbar">
        <div className="project-logs-viewer-toolbar-main">
          <Typography.Title level={5} className="project-logs-viewer-title">
            日志查看器
          </Typography.Title>
          <Form form={form} layout="inline" className="project-logs-viewer-query-form">
            <Form.Item name="project_id" rules={[{ required: true }]}>
              <Select
                style={{ width: 200 }}
                placeholder="项目"
                options={projectOptions}
                onChange={(pid) => {
                  form.setFieldsValue({
                    server_id: undefined,
                    service_id: undefined,
                    log_source_id: undefined,
                    service_name: undefined,
                    host: undefined,
                  });
                  setServers([]);
                  setServices([]);
                  setSources([]);
                  void reloadServers(pid);
                  void reloadServices(pid);
                  void reloadSavedQueries(pid);
                  void reloadDropRules(pid);
                }}
              />
            </Form.Item>
            <Form.Item name="time_range">
              <DatePicker.RangePicker
                showTime
                allowClear={false}
                onChange={() => setActivePreset("")}
              />
            </Form.Item>
            <Form.Item>
              <Space.Compact size="middle">
                {TIME_PRESETS.map((p) => (
                  <Button
                    key={p.label}
                    type={activePreset === p.label ? "primary" : "default"}
                    onClick={() => applyTimePreset(p.label, p.minutes)}
                  >
                    {p.label}
                  </Button>
                ))}
              </Space.Compact>
            </Form.Item>
            <Form.Item name="keyword">
              <Tooltip title="支持简化 DQL：level:ERROR AND service:api host:node-1">
                <Input
                  allowClear
                  placeholder="关键词 / DQL：level:ERROR service:xxx"
                  style={{ width: 280 }}
                />
              </Tooltip>
            </Form.Item>
            <Form.Item name="level" hidden>
              <Input />
            </Form.Item>
            <Form.Item name="service_name" hidden>
              <Input />
            </Form.Item>
            <Form.Item name="host" hidden>
              <Input />
            </Form.Item>
            <Form.Item name="namespace" hidden>
              <Input />
            </Form.Item>
            <Form.Item name="pod" hidden>
              <Input />
            </Form.Item>
            <Form.Item name="container" hidden>
              <Input />
            </Form.Item>
            <Form.Item name="extra_field" hidden>
              <Input />
            </Form.Item>
            <Form.Item name="extra_value" hidden>
              <Input />
            </Form.Item>
            <Form.Item>
              <Button type="primary" icon={<SearchOutlined />} loading={loading} onClick={() => void runSearch({ page: 1 })}>
                查询
              </Button>
            </Form.Item>
          </Form>
        </div>
        <Space wrap className="project-logs-viewer-actions">
          <Button icon={<ReloadOutlined />} onClick={() => void runSearch()}>
            刷新
          </Button>
          <Button icon={<DownloadOutlined />} onClick={() => void exportLogs(form)}>
            导出
          </Button>
          <Button icon={<StarOutlined />} onClick={() => void saveCurrentQuery()}>
            收藏查询
          </Button>
          <Dropdown
            menu={{
              items: [
                ...(savedQueries.length
                  ? savedQueries.map((q) => ({
                      key: `q-${q.id}`,
                      label: q.name,
                      onClick: () => applySavedQuery(q),
                    }))
                  : [{ key: "empty", label: "暂无收藏", disabled: true }]),
                ...(savedQueries.length
                  ? [
                      { type: "divider" as const },
                      ...savedQueries.map((q) => ({
                        key: `del-${q.id}`,
                        label: `删除「${q.name}」`,
                        danger: true,
                        onClick: () =>
                          void (async () => {
                            const pid = form.getFieldValue("project_id") as number;
                            await deleteLogSavedQuery(pid, q.id);
                            message.success("已删除");
                            void reloadSavedQueries(pid);
                          })(),
                      })),
                    ]
                  : []),
              ],
            }}
          >
            <Button>我的收藏</Button>
          </Dropdown>
          <Button type="primary" ghost icon={<RobotOutlined />} onClick={() => void runAiAnalyze()}>
            AI 分析建议
          </Button>
          <Button onClick={() => goAdjustPipeline()}>Pipeline 仓库</Button>
          <Button
            onClick={() => {
              const pid = form.getFieldValue("project_id") as number | undefined;
              void reloadDropRules(pid);
              setDropOpen(true);
            }}
          >
            黑名单 ({dropRules.filter((r) => r.enabled).length})
          </Button>
        </Space>
      </div>

      <Collapse
        ghost
        className="project-logs-advanced-filters"
        items={[
          {
            key: "adv",
            label: (
              <Space>
                <FilterOutlined />
                高级筛选（采集源 / K8s / 文件名）
              </Space>
            ),
            children: (
              <Form form={form} layout="vertical" size="small">
                <Row gutter={12}>
                  <Col span={4}>
                    <Form.Item label="采集来源" name="collector_mode">
                      <Select
                        allowClear
                        placeholder="全部"
                        options={[
                          { value: "host", label: "主机 Agent" },
                          { value: "k8s", label: "集群 K8s" },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                  <Col span={4}>
                    <Form.Item label="服务器" name="server_id">
                      <Select
                        allowClear
                        disabled={watchCollectorMode === "k8s"}
                        options={serverOptions}
                        placeholder="全部"
                        onChange={(sid) => {
                          const pid = form.getFieldValue("project_id");
                          form.setFieldsValue({ service_id: undefined, log_source_id: undefined });
                          setServices([]);
                          setSources([]);
                          void reloadServices(pid, sid);
                        }}
                      />
                    </Form.Item>
                  </Col>
                  <Col span={4}>
                    <Form.Item label="服务" name="service_id">
                      <Select
                        allowClear
                        disabled={watchCollectorMode === "k8s"}
                        options={serviceOptions}
                        placeholder="全部"
                        onChange={(svcId) => {
                          const pid = form.getFieldValue("project_id");
                          form.setFieldsValue({ log_source_id: undefined });
                          setSources([]);
                          void reloadSources(pid, svcId);
                        }}
                      />
                    </Form.Item>
                  </Col>
                  <Col span={4}>
                    <Form.Item label="日志源" name="log_source_id">
                      <Select allowClear disabled={watchCollectorMode === "k8s"} options={sourceOptions} placeholder="全部" />
                    </Form.Item>
                  </Col>
                  <Col span={4}>
                    <Form.Item label="K8s 集群" name="cluster_id">
                      <Select allowClear disabled={watchCollectorMode === "host"} options={clusterOptions} placeholder="全部" />
                    </Form.Item>
                  </Col>
                  <Col span={4}>
                    <Form.Item label="Namespace" name="namespace">
                      <Input allowClear disabled={watchCollectorMode === "host"} />
                    </Form.Item>
                  </Col>
                  <Col span={4}>
                    <Form.Item label="Pod" name="pod">
                      <Input allowClear disabled={watchCollectorMode === "host"} />
                    </Form.Item>
                  </Col>
                  <Col span={4}>
                    <Form.Item label="容器" name="container">
                      <Input allowClear disabled={watchCollectorMode === "host"} />
                    </Form.Item>
                  </Col>
                  <Col span={8}>
                    <Form.Item label="文件名" name="file_path">
                      <Input allowClear placeholder="748.log / server.log" />
                    </Form.Item>
                  </Col>
                  <Col span={8}>
                    <Form.Item
                      label="索引 Pattern"
                      name="index_pattern"
                      tooltip="可选，覆盖默认索引（多数据流）。留空则按采集模式自动选择。"
                    >
                      <Input allowClear placeholder="例如 logs-app-* / k8s-logs-*" />
                    </Form.Item>
                  </Col>
                </Row>
              </Form>
            ),
          },
        ]}
      />

      <div className="project-logs-viewer-body">
        <aside className="project-logs-facet-panel">
          <div className="project-logs-facet-head">
            <Typography.Text strong>快捷筛选 / 字段</Typography.Text>
            <Input
              size="small"
              allowClear
              placeholder="搜索字段"
              value={facetSearch}
              onChange={(e) => setFacetSearch(e.target.value)}
            />
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              采样 {fieldStats.length ? `${fieldStats.length} 字段` : "-"} ·{" "}
              <a onClick={goAdjustPipeline}>调整 Pipeline</a>
            </Typography.Text>
          </div>
          {filteredFacetGroups.map((group) => (
            <div key={group.title} className="project-logs-facet-group">
              <Typography.Text type="secondary" className="project-logs-facet-group-title">
                {group.title}
              </Typography.Text>
              <div className="project-logs-facet-list">
                {group.items.length ? (
                  group.items.map((item) => (
                    <label
                      key={`${group.field}-${item.key}`}
                      className={`project-logs-facet-item${
                        (group.field === "level" && watchLevel === item.key) ||
                        (group.field === "service" && watchServiceName === item.key) ||
                        (group.field === "host" && watchHost === item.key) ||
                        (group.field === "namespace" && watchNamespace === item.key) ||
                        (group.field === "pod" && watchPod === item.key) ||
                        (group.field === "container" && form.getFieldValue("container") === item.key)
                          ? " is-active"
                          : ""
                      }`}
                    >
                      <Checkbox
                        checked={
                          group.field === "level"
                            ? watchLevel === item.key
                            : group.field === "service"
                              ? watchServiceName === item.key
                              : group.field === "host"
                                ? watchHost === item.key
                                : group.field === "namespace"
                                  ? watchNamespace === item.key
                                  : group.field === "pod"
                                    ? watchPod === item.key
                                    : form.getFieldValue("container") === item.key
                        }
                        onChange={() => applyFacet(group.field, item.key)}
                      />
                      <span className="project-logs-facet-label" title={item.label}>
                        {item.label}
                      </span>
                      <span className="project-logs-facet-count">{item.count}</span>
                    </label>
                  ))
                ) : (
                  <Typography.Text type="secondary" className="project-logs-facet-empty">
                    暂无
                  </Typography.Text>
                )}
              </div>
            </div>
          ))}
          <div className="project-logs-facet-group">
            <Typography.Text type="secondary" className="project-logs-facet-group-title">
              可观察字段
            </Typography.Text>
            <div className="project-logs-facet-list">
              {(facetSearch.trim()
                ? fieldStats.filter((f) => f.name.toLowerCase().includes(facetSearch.trim().toLowerCase()))
                : fieldStats
              )
                .slice(0, 40)
                .map((f) => (
                  <label key={f.name} className="project-logs-facet-item">
                    <Checkbox
                      checked={visibleFields.includes(f.name)}
                      onChange={(e) => {
                        setVisibleFields((prev) =>
                          e.target.checked ? [...prev, f.name] : prev.filter((x) => x !== f.name),
                        );
                      }}
                    />
                    <span className="project-logs-facet-label" title={f.sample_values?.join(", ")}>
                      {f.name}
                    </span>
                    <span className="project-logs-facet-count">{f.count}</span>
                  </label>
                ))}
              {!fieldStats.length ? (
                <Typography.Text type="secondary" className="project-logs-facet-empty">
                  检索后展示字段
                </Typography.Text>
              ) : null}
            </div>
          </div>
        </aside>

        <main className="project-logs-main-panel">
          {emptyHint ? <Alert type="warning" showIcon message={emptyHint} style={{ marginBottom: 8 }} /> : null}

          <div className="project-logs-chart-strip">
            <div className="project-logs-chart-strip-head">
              <Space>
                <Typography.Text strong>状态分布图</Typography.Text>
                <Tag>{overview ? `${overview.total.toLocaleString()} 条` : "-"}</Tag>
                {watchLevel ? <Tag color="blue">level={watchLevel}</Tag> : null}
                {watchServiceName ? <Tag color="blue">service={watchServiceName}</Tag> : null}
                {watchHost ? <Tag color="blue">host={watchHost}</Tag> : null}
                {watchNamespace ? <Tag color="blue">ns={watchNamespace}</Tag> : null}
                {watchPod ? <Tag color="blue">pod={watchPod}</Tag> : null}
                {watchExtraField && watchExtraValue ? (
                  <Tag color="purple">
                    {watchExtraField}={watchExtraValue}
                  </Tag>
                ) : null}
              </Space>
              <Space size={4}>
                <Button size="small" type={chartMode === "timeline" ? "primary" : "default"} onClick={() => setChartMode("timeline")}>
                  时序图
                </Button>
                <Button size="small" type={chartMode === "pie" ? "primary" : "default"} onClick={() => setChartMode("pie")}>
                  饼图
                </Button>
              </Space>
            </div>
            {chartMode === "timeline" ? (
              <LogHistogramChart buckets={overview?.histogram || []} onBucketClick={zoomHistogramBucket} />
            ) : (
              <LogLevelPieChart levelCounts={overview?.level_counts || {}} />
            )}
          </div>

          {guideTips.length ? (
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 8 }}
              message="空结果排查引导"
              description={
                <ul style={{ margin: 0, paddingLeft: 18 }}>
                  {guideTips.map((t) => (
                    <li key={t}>{t}</li>
                  ))}
                </ul>
              }
            />
          ) : null}

          <Tabs
            activeKey={activeTab}
            onChange={setActiveTab}
            className="project-logs-main-tabs"
            items={[
              {
                key: "logs",
                label: `列表 (${total.toLocaleString()})`,
                children: (
                  <>
                    {scannedHint ? (
                      <div className="project-logs-scan-hint">
                        <Typography.Text type="secondary">{scannedHint}</Typography.Text>
                        <Space size={4}>
                          <Button
                            size="small"
                            type={listMode === "stacked" ? "primary" : "default"}
                            onClick={() => setListMode("stacked")}
                          >
                            堆叠列表
                          </Button>
                          <Button
                            size="small"
                            type={listMode === "compact" ? "primary" : "default"}
                            onClick={() => setListMode("compact")}
                          >
                            紧凑
                          </Button>
                          <Button type="link" size="small" icon={<RobotOutlined />} onClick={() => void runAiAnalyze()}>
                            AI 分析建议
                          </Button>
                          <Button type="link" size="small" onClick={() => goAdjustPipeline()}>
                            调整解析
                          </Button>
                        </Space>
                      </div>
                    ) : null}
                    <Tabs
                      size="small"
                      activeKey={listSubTab}
                      onChange={(k) => setListSubTab(k as "all" | "cluster")}
                      items={[
                        {
                          key: "all",
                          label: "全部日志",
                          children: (
                            <Table
                              rowKey={(r, i) => `${r.timestamp}-${i}`}
                              loading={loading}
                              columns={logColumns}
                              dataSource={rows}
                              size="small"
                              className={`project-logs-compact-table${listMode === "stacked" ? " is-stacked" : ""}`}
                              pagination={{
                                current: form.getFieldValue("page") ?? 1,
                                pageSize: form.getFieldValue("page_size") ?? 100,
                                total,
                                showSizeChanger: true,
                                pageSizeOptions: ["50", "100", "200", "500"],
                                showTotal: (t) => `共 ${t.toLocaleString()} 条`,
                                onChange: (page, pageSize) => void runSearch({ page, page_size: pageSize }),
                              }}
                              scroll={{ x: 900 }}
                            />
                          ),
                        },
                        {
                          key: "cluster",
                          label: `聚类分析 (${patternTotal})`,
                          children: (
                            <Table
                              rowKey="id"
                              size="small"
                              dataSource={patterns}
                              pagination={false}
                              columns={[
                                { title: "级别", dataIndex: "level", width: 80, render: (v: string) => <Tag>{v || "-"}</Tag> },
                                { title: "命中", dataIndex: "hit_count", width: 80 },
                                { title: "服务", dataIndex: "service_name", width: 120 },
                                {
                                  title: "签名",
                                  dataIndex: "signature",
                                  render: (v: string) => (
                                    <Typography.Link
                                      onClick={() => {
                                        form.setFieldsValue({ keyword: v });
                                        setListSubTab("all");
                                        void runSearch({ page: 1, keyword: v });
                                      }}
                                    >
                                      <Typography.Text code>{v}</Typography.Text>
                                    </Typography.Link>
                                  ),
                                },
                                { title: "样例", dataIndex: "sample", ellipsis: true },
                              ]}
                            />
                          ),
                        },
                      ]}
                    />
                  </>
                ),
              },
              {
                key: "analysis",
                label: "智能分析",
                children: overview ? (
                  <LogAnalysisPanel
                    overview={overview}
                    topn={topn}
                    topnDim={topnDim}
                    topnLoading={topnLoading}
                    onAi={() => void runAiAnalyze()}
                    onDimChange={(d) => {
                      setTopnDim(d);
                      void loadTopN(d);
                    }}
                    onTopNClick={(dim, key) => {
                      if (dim === "service" || dim === "service_name") {
                        form.setFieldsValue({ service_name: key, page: 1 });
                        void runSearch({ page: 1, service_name: key });
                      } else if (dim === "host") {
                        form.setFieldsValue({ host: key, page: 1 });
                        void runSearch({ page: 1, host: key });
                      } else if (dim === "pod") {
                        form.setFieldsValue({ pod: key, page: 1 });
                        void runSearch({ page: 1, pod: key });
                      } else if (dim === "namespace") {
                        form.setFieldsValue({ namespace: key, page: 1 });
                        void runSearch({ page: 1, namespace: key });
                      } else if (dim === "level" || dim === "status") {
                        form.setFieldsValue({ level: key, page: 1 });
                        void runSearch({ page: 1, level: key });
                      } else if (dim === "container") {
                        form.setFieldsValue({ container: key, page: 1 });
                        void runSearch({ page: 1, container: key });
                      }
                      setActiveTab("logs");
                    }}
                  />
                ) : (
                  <Typography.Text type="secondary">检索后展示级别分布、排行榜与错误签名</Typography.Text>
                ),
              },
              {
                key: "patterns",
                label: `模板 (${patternTotal})`,
                children: (
                  <Table
                    rowKey="id"
                    size="small"
                    dataSource={patterns}
                    pagination={false}
                    columns={[
                      { title: "级别", dataIndex: "level", width: 80, render: (v: string) => <Tag>{v || "-"}</Tag> },
                      { title: "命中", dataIndex: "hit_count", width: 80 },
                      { title: "服务", dataIndex: "service_name", width: 120 },
                      { title: "签名", dataIndex: "signature", render: (v: string) => <Typography.Text code>{v}</Typography.Text> },
                      { title: "样例", dataIndex: "sample", ellipsis: true },
                    ]}
                  />
                ),
              },
              {
                key: "anomalies",
                label: `错误追踪 (${anomalyTotal})`,
                children: (
                  <Table
                    rowKey="id"
                    size="small"
                    dataSource={anomalies}
                    pagination={false}
                    columns={[
                      {
                        title: "类型",
                        dataIndex: "anomaly_type",
                        width: 90,
                        render: (v: string) => (v === "error_spike" ? "量突增" : "新模板"),
                      },
                      {
                        title: "级别",
                        dataIndex: "severity",
                        width: 80,
                        render: (v: string) => <Tag color={v === "critical" ? "error" : "warning"}>{v}</Tag>,
                      },
                      {
                        title: "状态",
                        dataIndex: "status",
                        width: 100,
                        render: (v: string) => {
                          const color =
                            v === "open" ? " magentared" : v === "resolved" ? "success" : v === "muted" ? "default" : "processing";
                          return <Tag color={color === " magentared" ? "magenta" : color}>{v}</Tag>;
                        },
                      },
                      { title: "标题", dataIndex: "title", ellipsis: true },
                      {
                        title: "负责人",
                        dataIndex: "assignee_name",
                        width: 100,
                        render: (v: string) => v || "-",
                      },
                      {
                        title: "签名",
                        dataIndex: "signature",
                        width: 160,
                        render: (v: string) =>
                          v ? <Typography.Text code>{v.slice(0, 40)}</Typography.Text> : "-",
                      },
                      { title: "时间", dataIndex: "detected_at", width: 160, render: (v: string) => formatDateTime(v) },
                      {
                        title: "操作",
                        width: 220,
                        render: (_: unknown, row: LogAnomalyItem) => {
                          const pid = form.getFieldValue("project_id") as number;
                          const refresh = () => void loadOverviewAndIntel(form.getFieldsValue());
                          return (
                            <Space size={4} wrap>
                              {row.status !== "acknowledged" ? (
                                <Button
                                  type="link"
                                  size="small"
                                  onClick={() =>
                                    void updateProjectLogAnomaly(pid, row.id, { status: "acknowledged" }).then(() => {
                                      message.success("已确认");
                                      refresh();
                                    })
                                  }
                                >
                                  确认
                                </Button>
                              ) : null}
                              {row.status !== "resolved" ? (
                                <Button
                                  type="link"
                                  size="small"
                                  onClick={() =>
                                    void updateProjectLogAnomaly(pid, row.id, { status: "resolved" }).then(() => {
                                      message.success("已解决");
                                      refresh();
                                    })
                                  }
                                >
                                  解决
                                </Button>
                              ) : null}
                              <Button
                                type="link"
                                size="small"
                                onClick={() => {
                                  const name = window.prompt("负责人名称", row.assignee_name || "");
                                  if (name == null) return;
                                  void updateProjectLogAnomaly(pid, row.id, { assignee_name: name.trim() }).then(() => {
                                    message.success("已更新负责人");
                                    refresh();
                                  });
                                }}
                              >
                                指派
                              </Button>
                              <Button
                                type="link"
                                size="small"
                                onClick={() =>
                                  void updateProjectLogAnomaly(pid, row.id, { mute_minutes: 60 }).then(() => {
                                    message.success("已静默 60 分钟");
                                    refresh();
                                  })
                                }
                              >
                                静默
                              </Button>
                              {row.signature ? (
                                <Button
                                  type="link"
                                  size="small"
                                  onClick={() => {
                                    form.setFieldsValue({ keyword: row.signature, level: "ERROR", page: 1 });
                                    setActiveTab("logs");
                                    void runSearch({ page: 1, keyword: row.signature, level: "ERROR" });
                                  }}
                                >
                                  关联日志
                                </Button>
                              ) : null}
                            </Space>
                          );
                        },
                      },
                    ]}
                  />
                ),
              },
            ]}
          />
        </main>
      </div>

      <Drawer title="AI 日志分析" width={560} open={aiOpen} onClose={() => setAiOpen(false)} destroyOnClose>
        {aiLoading ? (
          <Typography.Text>分析中…</Typography.Text>
        ) : aiResult ? (
          <Space direction="vertical" size="middle" style={{ width: "100%" }}>
            <Alert type="info" message={aiResult.ai_summary} />
            {(aiResult.root_causes?.length ?? 0) > 0 ? (
              <List
                size="small"
                header={<Typography.Title level={5}>根因候选</Typography.Title>}
                dataSource={aiResult.root_causes}
                renderItem={(item) => (
                  <List.Item>
                    <List.Item.Meta title={item.title} description={item.evidence} />
                  </List.Item>
                )}
              />
            ) : null}
            {(aiResult.actions?.length ?? 0) > 0 ? (
              <List
                size="small"
                header={<Typography.Title level={5}>排查建议</Typography.Title>}
                dataSource={aiResult.actions}
                renderItem={(item) => (
                  <List.Item>
                    <Typography.Text>
                      {item.priority ? `[P${item.priority}] ` : ""}
                      {item.action}
                    </Typography.Text>
                  </List.Item>
                )}
              />
            ) : null}
          </Space>
        ) : (
          <Typography.Text type="secondary">基于当前筛选条件生成 AI 摘要与排查建议。</Typography.Text>
        )}
      </Drawer>

      <Drawer title="日志上下文" width={520} open={contextOpen} onClose={() => setContextOpen(false)} destroyOnClose>
        {contextLoading ? (
          <Typography.Text>加载中…</Typography.Text>
        ) : contextResult ? (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            <Typography.Text type="secondary">
              窗口 {formatDateTime(contextResult.window_from)} ~ {formatDateTime(contextResult.window_to)}
            </Typography.Text>
            {contextResult.overview ? (
              <Alert type="info" message={`窗口内命中 ${contextResult.overview.total.toLocaleString()} 条`} />
            ) : null}
            {(contextResult.recent_changes?.length ?? 0) > 0 ? (
              <List
                size="small"
                header="近期变更"
                dataSource={contextResult.recent_changes}
                renderItem={(item) => (
                  <List.Item>
                    <List.Item.Meta title={item.summary || item.action} description={formatDateTime(item.started_at)} />
                  </List.Item>
                )}
              />
            ) : null}
            {(contextResult.recent_alerts?.length ?? 0) > 0 ? (
              <List
                size="small"
                header="近期告警"
                dataSource={contextResult.recent_alerts}
                renderItem={(item) => (
                  <List.Item>
                    <List.Item.Meta title={item.title} description={item.severity} />
                  </List.Item>
                )}
              />
            ) : null}
          </Space>
        ) : (
          <Typography.Text type="secondary">暂无上下文</Typography.Text>
        )}
      </Drawer>

      <Modal
        title="解析效果对比"
        open={parseOpen}
        onCancel={() => setParseOpen(false)}
        footer={[
          <Button key="close" onClick={() => setParseOpen(false)}>
            关闭
          </Button>,
          <Button
            key="ai"
            type="primary"
            onClick={() => {
              setParseOpen(false);
              goAdjustPipeline(parseRow?.message ? [parseRow.message] : undefined);
            }}
          >
            AI 调整 Pipeline
          </Button>,
        ]}
        width={720}
        destroyOnClose
      >
        {parseRow ? (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            <div>
              <Typography.Text strong>原始消息</Typography.Text>
              <pre className="log-parse-compare-pre">{parseRow.message || "-"}</pre>
            </div>
            <div>
              <Typography.Text strong>当前可观察字段</Typography.Text>
              <div className="log-parse-compare-fields">
                {Object.entries(parseRow.fields || {}).length ? (
                  Object.entries(parseRow.fields || {}).map(([k, v]) => (
                    <div key={k} className="log-viewer-kv-row">
                      <span className="log-viewer-kv-key">{k}</span>
                      <span className="log-viewer-kv-val">{v}</span>
                    </div>
                  ))
                ) : (
                  <Typography.Text type="secondary">暂无结构化字段，建议用 AI 调整解析规则</Typography.Text>
                )}
              </div>
            </div>
          </Space>
        ) : null}
      </Modal>

      <Drawer
        title="日志黑名单（查询侧过滤）"
        width={560}
        open={dropOpen}
        onClose={() => setDropOpen(false)}
        destroyOnClose
      >
        <Typography.Paragraph type="secondary" style={{ marginTop: 0 }}>
          命中规则的日志在检索/概览中被 must_not 排除，不影响采集写入。可用于过滤噪声服务、主机或消息片段。
        </Typography.Paragraph>
        <Form
          form={dropForm}
          layout="vertical"
          initialValues={{ operator: "eq", enabled: true, field: "service_name" }}
          onFinish={(vals) => {
            const pid = form.getFieldValue("project_id") as number;
            if (!pid) {
              message.warning("请选择项目");
              return;
            }
            setDropLoading(true);
            void createLogDropRule(pid, {
              name: vals.name,
              field: vals.field,
              operator: vals.operator,
              value: vals.value,
              enabled: vals.enabled,
            })
              .then(() => {
                message.success("已添加");
                dropForm.resetFields();
                dropForm.setFieldsValue({ operator: "eq", enabled: true, field: "service_name" });
                return reloadDropRules(pid);
              })
              .catch((e: unknown) => message.error(extractApiErrorMessage(e, "添加失败")))
              .finally(() => setDropLoading(false));
          }}
        >
          <Row gutter={8}>
            <Col span={8}>
              <Form.Item name="name" label="名称" rules={[{ required: true }]}>
                <Input placeholder="规则名" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="field" label="字段" rules={[{ required: true }]}>
                <Select
                  options={[
                    { value: "service_name", label: "service_name" },
                    { value: "host", label: "host" },
                    { value: "level", label: "level" },
                    { value: "pod", label: "pod" },
                    { value: "namespace", label: "namespace" },
                    { value: "message", label: "message" },
                    { value: "signature", label: "signature" },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="operator" label="匹配">
                <Select
                  options={[
                    { value: "eq", label: "等于" },
                    { value: "contains", label: "包含" },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={16}>
              <Form.Item name="value" label="值" rules={[{ required: true }]}>
                <Input placeholder="匹配值" />
              </Form.Item>
            </Col>
            <Col span={4}>
              <Form.Item name="enabled" label="启用" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
            <Col span={4}>
              <Form.Item label=" ">
                <Button type="primary" htmlType="submit" loading={dropLoading} block>
                  添加
                </Button>
              </Form.Item>
            </Col>
          </Row>
        </Form>
        <Table
          rowKey="id"
          size="small"
          dataSource={dropRules}
          pagination={false}
          columns={[
            { title: "名称", dataIndex: "name" },
            {
              title: "条件",
              render: (_: unknown, r: LogDropRuleItem) => (
                <Typography.Text code>
                  {r.field} {r.operator} {r.value}
                </Typography.Text>
              ),
            },
            {
              title: "启用",
              dataIndex: "enabled",
              width: 70,
              render: (v: boolean, r: LogDropRuleItem) => (
                <Switch
                  size="small"
                  checked={v}
                  onChange={(checked) => {
                    const pid = form.getFieldValue("project_id") as number;
                    void updateLogDropRule(pid, r.id, {
                      name: r.name,
                      field: r.field,
                      operator: r.operator,
                      value: r.value,
                      enabled: checked,
                      remark: r.remark,
                    }).then(() => reloadDropRules(pid));
                  }}
                />
              ),
            },
            {
              title: "操作",
              width: 80,
              render: (_: unknown, r: LogDropRuleItem) => (
                <Popconfirm
                  title="删除该规则？"
                  onConfirm={() => {
                    const pid = form.getFieldValue("project_id") as number;
                    void deleteLogDropRule(pid, r.id).then(() => {
                      message.success("已删除");
                      void reloadDropRules(pid);
                    });
                  }}
                >
                  <Button type="link" danger size="small">
                    删除
                  </Button>
                </Popconfirm>
              ),
            },
          ]}
        />
      </Drawer>
    </div>
  );
}

async function exportLogs(form: ReturnType<typeof Form.useForm<SearchForm>>[0]) {
  const v = form.getFieldsValue();
  if (!v.project_id) {
    message.warning("请选择项目");
    return;
  }
  const range = v.time_range;
  try {
    const blob = await exportProjectLogs(v.project_id, {
      server_id: v.server_id,
      service_id: v.service_id,
      log_source_id: v.log_source_id,
      service_name: v.service_name?.trim() || undefined,
      host: v.host?.trim() || undefined,
      collector_mode: v.collector_mode || undefined,
      cluster_id: v.cluster_id,
      namespace: v.namespace?.trim() || undefined,
      pod: v.pod?.trim() || undefined,
      container: v.container?.trim() || undefined,
      keyword: v.keyword?.trim() || undefined,
      level: v.level?.trim() || undefined,
      file_path: v.file_path?.trim() || undefined,
      extra_field: v.extra_field?.trim() || undefined,
      extra_value: v.extra_value?.trim() || undefined,
      from: range?.[0]?.toISOString(),
      to: range?.[1]?.toISOString(),
      page_size: 1000,
    });
    if (!(blob instanceof Blob)) {
      message.error("导出失败");
      return;
    }
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `project-${v.project_id}-logs.txt`;
    a.click();
    window.URL.revokeObjectURL(url);
    message.success("导出完成");
  } catch (e: unknown) {
    message.error(extractApiErrorMessage(e));
  }
}

function LogLineCell({
  row,
  visibleFields,
  stacked,
  onFieldClick,
  onContext,
  onParseCompare,
}: {
  row: LogSearchItem;
  visibleFields: string[];
  stacked: boolean;
  onFieldClick?: (name: string, value: string) => void;
  onContext?: () => void;
  onParseCompare?: () => void;
}) {
  const level = normalizeLogLevel(row.level || extractLogLevel(row.message));
  const color =
    level === "ERROR" || level === "FATAL" ? "#cf1322" : level === "WARN" ? "#d48806" : level === "INFO" ? "#1677ff" : "#595959";
  const fields = row.fields || {};
  const pick = (name: string) => {
    if (fields[name]) return fields[name];
    const anyRow = row as unknown as Record<string, unknown>;
    const v = anyRow[name];
    return v == null ? "" : String(v);
  };
  const tagFields = visibleFields
    .map((name) => {
      let val = "";
      if (name === "level") val = level;
      else if (name === "service_name") val = row.service_name || String(pick(name) || "");
      else if (name === "host") val = row.host || row.server_host || String(pick(name) || "");
      else if (name === "trace_id") val = row.trace_id || String(pick(name) || "");
      else if (name === "namespace") val = row.namespace || String(pick(name) || "");
      else if (name === "pod") val = row.podname || row.pod || String(pick(name) || "");
      else val = String(pick(name) || "");
      return { name, val: String(val || "").trim() };
    })
    .filter((x) => x.val);

  const actions = (
    <Space size={4} className="log-viewer-row-actions">
      <Button type="link" size="small" onClick={onContext}>
        上下文
      </Button>
      <Button type="link" size="small" onClick={onParseCompare}>
        解析对比
      </Button>
      {row.trace_id ? (
        <Button type="link" size="small" onClick={() => onFieldClick?.("trace_id", row.trace_id || "")}>
          Trace
        </Button>
      ) : null}
    </Space>
  );

  if (!stacked) {
    return (
      <div className="log-viewer-line">
        {level ? (
          <span
            className="log-viewer-level"
            style={{ color, cursor: "pointer" }}
            onClick={() => level && onFieldClick?.("level", level)}
          >
            [{level}]
          </span>
        ) : null}
        <LogMessageCell highlight={row.highlight} message={row.message} />
        <span className="log-viewer-meta">
          {[row.service_name, row.host || row.server_host, row.namespace, row.podname || row.pod]
            .filter(Boolean)
            .join(" · ")}
        </span>
        {actions}
      </div>
    );
  }

  return (
    <div className="log-viewer-stacked">
      <div className="log-viewer-stacked-tags">
        {tagFields.map((t) => (
          <Tag
            key={t.name}
            className="log-viewer-clickable-tag"
            color={t.name === "level" ? undefined : "default"}
            style={t.name === "level" ? { color, borderColor: color, cursor: "pointer" } : { cursor: "pointer" }}
            onClick={() => onFieldClick?.(t.name, t.val)}
          >
            <span className="log-viewer-tag-key">{t.name}</span> {t.val}
          </Tag>
        ))}
        {actions}
      </div>
      <div className="log-viewer-stacked-kv">
        {tagFields.slice(0, 8).map((t) => (
          <div key={`kv-${t.name}`} className="log-viewer-kv-row">
            <span className="log-viewer-kv-key">{t.name}</span>
            <span className="log-viewer-kv-val log-viewer-clickable" onClick={() => onFieldClick?.(t.name, t.val)}>
              {t.val}
            </span>
          </div>
        ))}
        <div className="log-viewer-kv-row">
          <span className="log-viewer-kv-key">日志内容</span>
          <span className="log-viewer-kv-val">
            <LogMessageCell highlight={row.highlight} message={row.message} />
          </span>
        </div>
      </div>
    </div>
  );
}

function LogAnalysisPanel({
  overview,
  topn,
  topnDim,
  topnLoading,
  onAi,
  onDimChange,
  onTopNClick,
}: {
  overview: LogOverviewResult;
  topn: LogTopNResult | null;
  topnDim: string;
  topnLoading: boolean;
  onAi: () => void;
  onDimChange: (dim: string) => void;
  onTopNClick: (dim: string, key: string) => void;
}) {
  const levelEntries = Object.entries(overview.level_counts || {}).sort((a, b) => b[1] - a[1]);
  const topMax = Math.max(1, ...(topn?.items || []).map((i) => i.count));
  return (
    <div className="project-logs-analysis-panel">
      <Space style={{ marginBottom: 12 }}>
        <Button type="primary" ghost icon={<RobotOutlined />} onClick={onAi}>
          AI 分析建议
        </Button>
        <Tag>命中 {overview.total.toLocaleString()}</Tag>
      </Space>
      <Row gutter={12}>
        <Col span={12}>
          <LogLevelPieChart levelCounts={overview.level_counts || {}} />
        </Col>
        <Col span={12}>
          <div className="project-logs-level-bars">
            {levelEntries.map(([lv, cnt]) => {
              const max = Math.max(1, ...levelEntries.map(([, c]) => c));
              return (
                <div key={lv} className="project-logs-level-row">
                  <Tag>{lv}</Tag>
                  <div className="project-logs-level-bar-track">
                    <div className="project-logs-level-bar-fill" style={{ width: `${Math.round((cnt / max) * 100)}%` }} />
                  </div>
                  <span className="project-logs-level-count">{cnt.toLocaleString()}</span>
                </div>
              );
            })}
          </div>
        </Col>
      </Row>

      <div style={{ marginTop: 16 }}>
        <Space style={{ marginBottom: 8 }}>
          <Typography.Text strong>排行榜</Typography.Text>
          <Select
            size="small"
            value={topnDim}
            style={{ width: 140 }}
            onChange={onDimChange}
            options={[
              { value: "service", label: "Service" },
              { value: "pod", label: "Pod" },
              { value: "host", label: "Host" },
              { value: "level", label: "Level" },
              { value: "namespace", label: "Namespace" },
              { value: "container", label: "Container" },
            ]}
          />
          {topnLoading ? <Typography.Text type="secondary">加载中…</Typography.Text> : null}
        </Space>
        {(topn?.items?.length ?? 0) > 0 ? (
          <div className="project-logs-level-bars">
            {topn!.items.map((item) => (
              <div
                key={item.key}
                className="project-logs-level-row"
                style={{ cursor: "pointer" }}
                onClick={() => onTopNClick(topnDim, item.key)}
              >
                <Typography.Link style={{ minWidth: 120 }} ellipsis>
                  {item.key}
                </Typography.Link>
                <div className="project-logs-level-bar-track">
                  <div
                    className="project-logs-level-bar-fill"
                    style={{ width: `${Math.round((item.count / topMax) * 100)}%` }}
                  />
                </div>
                <span className="project-logs-level-count">{item.count.toLocaleString()}</span>
              </div>
            ))}
          </div>
        ) : (
          <Typography.Text type="secondary">暂无排行数据</Typography.Text>
        )}
      </div>

      {(overview.top_error_signatures?.length ?? 0) > 0 ? (
        <List
          style={{ marginTop: 12 }}
          size="small"
          header="Top 错误签名"
          dataSource={overview.top_error_signatures.slice(0, 8)}
          renderItem={(item) => (
            <List.Item>
              <Space direction="vertical" size={0}>
                <Space>
                  <Tag>{item.count}</Tag>
                  <Typography.Text code>{item.signature}</Typography.Text>
                </Space>
                <Typography.Text type="secondary" ellipsis>
                  {item.sample}
                </Typography.Text>
              </Space>
            </List.Item>
          )}
        />
      ) : null}
    </div>
  );
}

function LogLevelPieChart({ levelCounts }: { levelCounts: Record<string, number> }) {
  const entries = Object.entries(levelCounts).filter(([, c]) => c > 0);
  const total = entries.reduce((s, [, c]) => s + c, 0);
  if (!total) {
    return <Typography.Text type="secondary">无级别分布数据</Typography.Text>;
  }
  const colors: Record<string, string> = {
    ERROR: "#ff4d4f",
    FATAL: "#a8071a",
    WARN: "#faad14",
    INFO: "#1677ff",
    DEBUG: "#8c8c8c",
  };
  let acc = 0;
  const slices = entries.map(([lv, cnt]) => {
    const start = (acc / total) * 360;
    acc += cnt;
    const end = (acc / total) * 360;
    return { lv, cnt, start, end, color: colors[lv] || "#722ed1" };
  });
  const r = 52;
  const cx = 70;
  const cy = 70;
  function arc(start: number, end: number) {
    const s = ((start - 90) * Math.PI) / 180;
    const e = ((end - 90) * Math.PI) / 180;
    const x1 = cx + r * Math.cos(s);
    const y1 = cy + r * Math.sin(s);
    const x2 = cx + r * Math.cos(e);
    const y2 = cy + r * Math.sin(e);
    const large = end - start > 180 ? 1 : 0;
    return `M ${cx} ${cy} L ${x1} ${y1} A ${r} ${r} 0 ${large} 1 ${x2} ${y2} Z`;
  }
  return (
    <div className="project-logs-pie-wrap">
      <svg width={140} height={140} viewBox="0 0 140 140">
        {slices.map((s) => (
          <path key={s.lv} d={arc(s.start, s.end)} fill={s.color} opacity={0.9}>
            <title>{`${s.lv}: ${s.cnt}`}</title>
          </path>
        ))}
      </svg>
      <div className="project-logs-pie-legend">
        {slices.map((s) => (
          <div key={s.lv} className="project-logs-pie-legend-item">
            <span className="project-logs-pie-dot" style={{ background: s.color }} />
            {s.lv} {s.cnt.toLocaleString()}
          </div>
        ))}
      </div>
    </div>
  );
}

function LogHistogramChart({
  buckets,
  onBucketClick,
}: {
  buckets: LogHistogramBucket[];
  onBucketClick?: (b: LogHistogramBucket) => void;
}) {
  const sorted = [...buckets].sort((a, b) => a.time.localeCompare(b.time));
  const sampled = sorted.length > 60 ? sorted.filter((_, i) => i % Math.ceil(sorted.length / 60) === 0) : sorted;
  const max = Math.max(1, ...sampled.map((b) => b.count));
  if (!sampled.length) {
    return <Typography.Text type="secondary">无时间分布数据（点击柱可缩放时间窗）</Typography.Text>;
  }
  const chartW = Math.max(720, sampled.length * 14);
  const chartH = 120;
  const padL = 32;
  const padB = 22;
  const padT = 6;
  const innerH = chartH - padB - padT;
  const barW = Math.max(4, (chartW - padL - 8) / sampled.length - 2);
  const stackOrder = ["ERROR", "FATAL", "WARN", "WARNING", "INFO", "DEBUG", "TRACE"];

  return (
    <div className="project-logs-histogram-wrap">
      <svg width="100%" height={chartH} viewBox={`0 0 ${chartW} ${chartH}`} preserveAspectRatio="xMinYMid meet">
        {[0, 0.5, 1].map((r) => {
          const y = padT + innerH * (1 - r);
          return (
            <g key={r}>
              <line x1={padL} y1={y} x2={chartW - 4} y2={y} stroke="rgba(0,0,0,0.06)" />
              <text x={padL - 4} y={y + 3} textAnchor="end" fontSize={9} fill="#8c8c8c">
                {Math.round(max * r)}
              </text>
            </g>
          );
        })}
        {sampled.map((b, i) => {
          const x = padL + i * (barW + 2);
          const levels = b.level_counts || {};
          const entries = Object.entries(levels);
          const hasStack = entries.some(([, c]) => c > 0);
          if (!hasStack) {
            const h = Math.max(1, (b.count / max) * innerH);
            const y = padT + innerH - h;
            return (
              <g key={`${b.time}-${i}`} style={{ cursor: "pointer" }} onClick={() => onBucketClick?.(b)}>
                <title>{`${formatDateTime(b.time)}: ${b.count}`}</title>
                <rect x={x} y={y} width={barW} height={h} rx={1} fill="#6366f1" opacity={0.88} />
              </g>
            );
          }
          let yCursor = padT + innerH;
          const ordered = [
            ...stackOrder.filter((lv) => levels[lv]),
            ...Object.keys(levels).filter((lv) => !stackOrder.includes(lv)),
          ];
          return (
            <g key={`${b.time}-${i}`} style={{ cursor: "pointer" }} onClick={() => onBucketClick?.(b)}>
              <title>
                {`${formatDateTime(b.time)}: ${b.count}` +
                  ordered.map((lv) => `\n${lv}=${levels[lv]}`).join("")}
              </title>
              {ordered.map((lv) => {
                const cnt = levels[lv] || 0;
                if (!cnt) return null;
                const h = Math.max(1, (cnt / max) * innerH);
                yCursor -= h;
                return (
                  <rect
                    key={lv}
                    x={x}
                    y={yCursor}
                    width={barW}
                    height={h}
                    fill={LEVEL_STACK_COLORS[lv] || "#722ed1"}
                    opacity={0.9}
                  />
                );
              })}
            </g>
          );
        })}
      </svg>
    </div>
  );
}

function LogMessageCell({ message, highlight }: { message?: string; highlight?: string }) {
  if (highlight) {
    return <span className="log-message-cell" dangerouslySetInnerHTML={{ __html: sanitizeLogHighlight(highlight) }} />;
  }
  return <span className="log-message-cell">{message || "-"}</span>;
}

function sanitizeLogHighlight(input: string): string {
  const escaped = String(input)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
  return escaped.replace(/&lt;mark&gt;/gi, "<mark>").replace(/&lt;\/mark&gt;/gi, "</mark>");
}

function normalizeLogLevel(level?: string) {
  const v = String(level || "").trim().toUpperCase();
  if (!v) return "";
  return v === "WARNING" ? "WARN" : v;
}

function extractLogLevel(message?: string) {
  if (!message) return "";
  const bracket = message.match(/\[(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|PANIC)\s*\]/i);
  if (bracket?.[1]) return normalizeLogLevel(bracket[1]);
  const token = message.match(/\s(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|PANIC)\s/i);
  if (token?.[1]) return normalizeLogLevel(token[1]);
  return "";
}
