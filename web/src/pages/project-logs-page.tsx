import {
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
  Form,
  Input,
  List,
  Row,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
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
  getProjectLogFields,
  getProjectLogOverview,
  getProjectLogPatterns,
  getProjectLogSources,
  getProjectServers,
  getProjectServices,
  getProjects,
  searchProjectLogs,
  type LogAnalyzeResult,
  type LogAnomalyItem,
  type LogFieldStat,
  type LogOverviewResult,
  type LogPatternItem,
  type LogSearchItem,
  type LogSourceItem,
  type ProjectItem,
  type ServerItem,
  type ServiceItem,
} from "../services/projects";
import { getClusters, type ClusterItem } from "../services/clusters";
import { extractApiErrorMessage } from "../services/http";
import { formatDateTime } from "../utils/format";

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
  const [chartMode, setChartMode] = useState<"timeline" | "pie">("timeline");
  const [aiOpen, setAiOpen] = useState(false);
  const [aiLoading, setAiLoading] = useState(false);
  const [aiResult, setAiResult] = useState<LogAnalyzeResult | null>(null);
  const [facetSearch, setFacetSearch] = useState("");
  const [activePreset, setActivePreset] = useState<string>("24h");
  const [fieldStats, setFieldStats] = useState<LogFieldStat[]>([]);
  const [listMode, setListMode] = useState<"stacked" | "compact">("stacked");
  const [visibleFields, setVisibleFields] = useState<string[]>([
    "level",
    "service_name",
    "host",
    "trace_id",
    "namespace",
    "pod",
  ]);

  const [form] = Form.useForm<SearchForm>();
  const watchCollectorMode = Form.useWatch("collector_mode", form);
  const watchLevel = Form.useWatch("level", form);
  const watchServiceName = Form.useWatch("service_name", form);
  const watchHost = Form.useWatch("host", form);

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
      from: range?.[0]?.toISOString(),
      to: range?.[1]?.toISOString(),
      page: page ?? values.page ?? 1,
      page_size: pageSize ?? values.page_size ?? 100,
    };
  }, []);

  const loadOverviewAndIntel = useCallback(
    async (values: SearchForm) => {
      if (!values.project_id) return;
      const params = buildSearchParams(values);
      const [ov, pat, an, fields] = await Promise.all([
        getProjectLogOverview(values.project_id, params).catch(() => null),
        getProjectLogPatterns(values.project_id, { ...params, page: 1, page_size: 20 }).catch(() => ({ list: [], total: 0 })),
        getProjectLogAnomalies(values.project_id, { status: "open", page: 1, page_size: 20 }).catch(() => ({ list: [], total: 0 })),
        getProjectLogFields(values.project_id, params).catch(() => null),
      ]);
      setOverview(ov);
      setPatterns(pat?.list ?? []);
      setPatternTotal(pat?.total ?? 0);
      setAnomalies(an?.list ?? []);
      setAnomalyTotal(an?.total ?? 0);
      setFieldStats(fields?.fields ?? []);
    },
    [buildSearchParams],
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
        if ((res.total ?? 0) === 0 && filePath) {
          setEmptyHint(`按文件名「${filePath}」无命中，可清空文件名或扩大时间范围。`);
        } else if ((res.total ?? 0) === 0 && !range?.[0] && !range?.[1]) {
          setEmptyHint("未选时间范围且无数据，请确认 Agent / DaemonSet 已写入 ES。");
        }
      } catch (e: unknown) {
        message.error(extractApiErrorMessage(e));
      } finally {
        setLoading(false);
      }
    },
    [form, buildSearchParams, loadOverviewAndIntel],
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
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const levelFacets = useMemo<FacetItem[]>(() => {
    const counts = overview?.level_counts || {};
    return Object.entries(counts)
      .map(([key, count]) => ({ key, label: key, count: Number(count) || 0 }))
      .sort((a, b) => b.count - a.count);
  }, [overview]);

  const serviceFacets = useMemo<FacetItem[]>(() => {
    const counts = overview?.service_name_counts || {};
    const fromOverview = Object.entries(counts)
      .map(([key, count]) => ({ key, label: key, count: Number(count) || 0 }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 15);
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
    const counts = overview?.host_counts || {};
    const fromOverview = Object.entries(counts)
      .map(([key, count]) => ({ key, label: key, count: Number(count) || 0 }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 15);
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

  const filteredFacetGroups = useMemo(() => {
    const q = facetSearch.trim().toLowerCase();
    const filter = (items: FacetItem[]) => (!q ? items : items.filter((i) => i.label.toLowerCase().includes(q)));
    return [
      { title: "日志级别", field: "level" as const, items: filter(levelFacets) },
      { title: "服务", field: "service" as const, items: filter(serviceFacets) },
      { title: "主机", field: "host" as const, items: filter(hostFacets) },
    ];
  }, [facetSearch, levelFacets, serviceFacets, hostFacets]);

  function applyFacet(field: "level" | "service" | "host", value: string) {
    if (field === "level") {
      const next = watchLevel === value ? undefined : value;
      form.setFieldsValue({ level: next });
      void runSearch({ page: 1, level: next });
      return;
    }
    if (field === "service") {
      const next = watchServiceName === value ? undefined : value;
      form.setFieldsValue({ service_name: next });
      void runSearch({ page: 1, service_name: next });
      return;
    }
    const next = watchHost === value ? undefined : value;
    form.setFieldsValue({ host: next });
    void runSearch({ page: 1, host: next });
  }

  function applyTimePreset(label: string, minutes: number) {
    const range: [Dayjs, Dayjs] = [dayjs().subtract(minutes, "minute"), dayjs()];
    setActivePreset(label);
    form.setFieldsValue({ time_range: range });
    void runSearch({ page: 1, time_range: range });
  }

  function goAdjustPipeline() {
    const pid = form.getFieldValue("project_id") as number | undefined;
    const samples = rows
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
      render: (_: string, r) => <LogLineCell row={r} visibleFields={visibleFields} stacked={listMode === "stacked"} />,
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
              <Input allowClear placeholder="关键词检索" style={{ width: 200 }} />
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
          <Button type="primary" ghost icon={<RobotOutlined />} onClick={() => void runAiAnalyze()}>
            AI 分析建议
          </Button>
          <Button onClick={goAdjustPipeline}>Pipeline 仓库</Button>
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
                        (group.field === "host" && watchHost === item.key)
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
                              : watchHost === item.key
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
              <LogHistogramChart buckets={overview?.histogram || []} />
            ) : (
              <LogLevelPieChart levelCounts={overview?.level_counts || {}} />
            )}
          </div>

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
                          <Button type="link" size="small" onClick={goAdjustPipeline}>
                            调整解析
                          </Button>
                        </Space>
                      </div>
                    ) : null}
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
                  </>
                ),
              },
              {
                key: "analysis",
                label: "智能分析",
                children: overview ? (
                  <LogAnalysisPanel overview={overview} onAi={() => void runAiAnalyze()} />
                ) : (
                  <Typography.Text type="secondary">检索后展示级别分布与错误签名</Typography.Text>
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
                label: `异常 (${anomalyTotal})`,
                children: (
                  <Table
                    rowKey="id"
                    size="small"
                    dataSource={anomalies}
                    pagination={false}
                    columns={[
                      { title: "类型", dataIndex: "anomaly_type", width: 100, render: (v: string) => (v === "error_spike" ? "量突增" : "新模板") },
                      { title: "级别", dataIndex: "severity", width: 90, render: (v: string) => <Tag color={v === "critical" ? "error" : "warning"}>{v}</Tag> },
                      { title: "标题", dataIndex: "title" },
                      { title: "详情", dataIndex: "detail", ellipsis: true },
                      { title: "时间", dataIndex: "detected_at", width: 170, render: (v: string) => formatDateTime(v) },
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
}: {
  row: LogSearchItem;
  visibleFields: string[];
  stacked: boolean;
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

  if (!stacked) {
    return (
      <div className="log-viewer-line">
        {level ? (
          <span className="log-viewer-level" style={{ color }}>
            [{level}]
          </span>
        ) : null}
        <LogMessageCell highlight={row.highlight} message={row.message} />
        <span className="log-viewer-meta">
          {[row.service_name, row.host || row.server_host, row.namespace, row.podname || row.pod]
            .filter(Boolean)
            .join(" · ")}
        </span>
      </div>
    );
  }

  return (
    <div className="log-viewer-stacked">
      <div className="log-viewer-stacked-tags">
        {tagFields.map((t) => (
          <Tag key={t.name} color={t.name === "level" ? undefined : "default"} style={t.name === "level" ? { color, borderColor: color } : undefined}>
            <span className="log-viewer-tag-key">{t.name}</span> {t.val}
          </Tag>
        ))}
      </div>
      <div className="log-viewer-stacked-kv">
        {tagFields.slice(0, 8).map((t) => (
          <div key={`kv-${t.name}`} className="log-viewer-kv-row">
            <span className="log-viewer-kv-key">{t.name}</span>
            <span className="log-viewer-kv-val">{t.val}</span>
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

function LogAnalysisPanel({ overview, onAi }: { overview: LogOverviewResult; onAi: () => void }) {
  const levelEntries = Object.entries(overview.level_counts || {}).sort((a, b) => b[1] - a[1]);
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

function LogHistogramChart({ buckets }: { buckets: Array<{ time: string; count: number }> }) {
  const sorted = [...buckets].sort((a, b) => a.time.localeCompare(b.time));
  const sampled = sorted.length > 60 ? sorted.filter((_, i) => i % Math.ceil(sorted.length / 60) === 0) : sorted;
  const max = Math.max(1, ...sampled.map((b) => b.count));
  if (!sampled.length) {
    return <Typography.Text type="secondary">无时间分布数据</Typography.Text>;
  }
  const chartW = Math.max(720, sampled.length * 14);
  const chartH = 120;
  const padL = 32;
  const padB = 22;
  const padT = 6;
  const innerH = chartH - padB - padT;
  const barW = Math.max(4, (chartW - padL - 8) / sampled.length - 2);

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
          const h = Math.max(1, (b.count / max) * innerH);
          const x = padL + i * (barW + 2);
          const y = padT + innerH - h;
          return (
            <g key={`${b.time}-${i}`}>
              <title>{`${formatDateTime(b.time)}: ${b.count}`}</title>
              <rect x={x} y={y} width={barW} height={h} rx={1} fill="#6366f1" opacity={0.88} />
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
