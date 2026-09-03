import { DownloadOutlined, ReloadOutlined, RobotOutlined, SearchOutlined } from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Col,
  DatePicker,
  Drawer,
  Form,
  Input,
  List,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs, { type Dayjs } from "dayjs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import {
  analyzeProjectLogs,
  exportProjectLogs,
  getProjectLogAnomalies,
  getProjectLogOverview,
  getProjectLogPatterns,
  getProjectLogSources,
  getProjectServers,
  getProjectServices,
  getProjects,
  searchProjectLogs,
  type LogAnalyzeResult,
  type LogAnomalyItem,
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

export function ProjectLogsPage() {
  const [searchParams] = useSearchParams();
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
  const [aiOpen, setAiOpen] = useState(false);
  const [aiLoading, setAiLoading] = useState(false);
  const [aiResult, setAiResult] = useState<LogAnalyzeResult | null>(null);

  const [form] = Form.useForm<SearchForm>();
  const watchProjectId = Form.useWatch("project_id", form);
  const watchServerId = Form.useWatch("server_id", form);
  const watchCollectorMode = Form.useWatch("collector_mode", form);

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

  const buildSearchParams = useCallback((values: SearchForm, page?: number, pageSize?: number) => {
    const range = values.time_range;
    return {
      server_id: values.server_id,
      service_id: values.service_id,
      log_source_id: values.log_source_id,
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
      const [ov, pat, an] = await Promise.all([
        getProjectLogOverview(values.project_id, params).catch(() => null),
        getProjectLogPatterns(values.project_id, { ...params, page: 1, page_size: 20 }).catch(() => ({ list: [], total: 0 })),
        getProjectLogAnomalies(values.project_id, { status: "open", page: 1, page_size: 20 }).catch(() => ({ list: [], total: 0 })),
      ]);
      setOverview(ov);
      setPatterns(pat?.list ?? []);
      setPatternTotal(pat?.total ?? 0);
      setAnomalies(an?.list ?? []);
      setAnomalyTotal(an?.total ?? 0);
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
        form.setFieldsValue({ page, page_size: pageSize });
        void loadOverviewAndIntel({ ...values, page, page_size: pageSize });
        if ((res.total ?? 0) === 0 && filePath) {
          setEmptyHint(
            `按文件名「${filePath}」无命中。请确认 Agent「热更」后已写入 file_path，或清空文件名筛选项/扩大时间范围。`,
          );
        } else if ((res.total ?? 0) === 0 && values.cluster_id) {
          message.warning(`无结果：将检索索引 yunshu-k8s-${values.cluster_id}-*，可确认 DaemonSet 已写入或清空集群筛选`);
        } else if ((res.total ?? 0) === 0 && values.server_id) {
          message.warning(`无结果：将检索索引 yunshu-agent-${values.server_id}-*，可确认 Agent 已写入或清空服务器筛选`);
        } else if ((res.total ?? 0) === 0 && range?.[0] && range?.[1]) {
          message.warning("无结果：若 ES 文档缺少 @timestamp，请先清空时间范围后再查");
        } else if ((res.total ?? 0) === 0 && !range?.[0] && !range?.[1]) {
          setEmptyHint("未选时间范围且无数据。确认主机 Agent 或集群 DaemonSet 已写入对应索引。");
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
        setActiveTab(tab);
      }
      const anchorTime = searchParams.get("anchor_time");
      const windowMinutes = Number(searchParams.get("window_minutes") || 5) || 5;
      let timeRange: [Dayjs, Dayjs] = [dayjs().subtract(24, "hour"), dayjs()];
      if (anchorTime) {
        const anchor = dayjs(anchorTime);
        if (anchor.isValid()) {
          timeRange = [anchor.subtract(windowMinutes, "minute"), anchor.add(windowMinutes, "minute")];
        }
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
        if (initial.service_id) {
          await reloadServices(defaultProject, initial.server_id);
          await reloadSources(defaultProject, initial.service_id);
        }
        await runSearch(initial);
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const columns: ColumnsType<LogSearchItem> = [
    {
      title: "时间",
      dataIndex: "timestamp",
      width: 170,
      fixed: "left",
      render: (v: string) => <span className="log-meta-cell">{formatDateTime(v)}</span>,
    },
    {
      title: "级别",
      dataIndex: "level",
      width: 76,
      fixed: "left",
      render: (v?: string, r?: LogSearchItem) => {
        const level = normalizeLogLevel(v || extractLogLevel(r?.message));
        if (!level) return "-";
        const color = level === "ERROR" || level === "FATAL" ? "error" : level === "WARN" ? "warning" : level === "INFO" ? "processing" : "default";
        return <Tag color={color}>{level}</Tag>;
      },
    },
    {
      title: "内容",
      dataIndex: "message",
      render: (_: string, r) => <LogMessageCell highlight={r.highlight} message={r.message} />,
    },
    {
      title: "来源",
      dataIndex: "collector_mode",
      width: 72,
      render: (v?: string, r?: LogSearchItem) => {
        const mode = v || (r?.cluster_id ? "k8s" : "host");
        return <Tag>{mode === "k8s" ? "K8s" : "主机"}</Tag>;
      },
    },
    {
      title: "服务",
      dataIndex: "service_name",
      width: 120,
      render: (v?: string) => <span className="log-meta-cell">{v || "-"}</span>,
    },
    {
      title: "主机",
      dataIndex: "host",
      width: 120,
      render: (v?: string, r?: LogSearchItem) => <span className="log-meta-cell">{v || r?.server_host || "-"}</span>,
    },
    {
      title: "文件",
      dataIndex: "file_path",
      width: 180,
      render: (v?: string) => {
        if (!v) return "-";
        const base = v.split(/[/\\]/).pop() || v;
        return (
          <Typography.Text className="log-meta-cell log-file-cell" title={v}>
            {base}
          </Typography.Text>
        );
      },
    },
    {
      title: "Namespace",
      dataIndex: "namespace",
      width: 110,
      render: (v?: string) => <span className="log-meta-cell">{v || "-"}</span>,
    },
    {
      title: "Pod",
      dataIndex: "podname",
      width: 140,
      render: (v?: string, r?: LogSearchItem) => <span className="log-meta-cell">{v || r?.pod || "-"}</span>,
    },
    {
      title: "容器",
      dataIndex: "containername",
      width: 100,
      render: (v?: string, r?: LogSearchItem) => <span className="log-meta-cell">{v || r?.container || "-"}</span>,
    },
  ];

  return (
    <div className="project-logs-page">
      <Card
        className="table-card project-logs-card"
        title="日志检索"
        extra={
          <Space>
            <Tag color="blue">主机 yunshu-agent-* / 集群 yunshu-k8s-*</Tag>
            <Button icon={<ReloadOutlined />} onClick={() => void runSearch()}>
              刷新
            </Button>
            <Button
              icon={<DownloadOutlined />}
              onClick={() => {
                const v = form.getFieldsValue();
                if (!v.project_id) {
                  message.warning("请选择项目");
                  return;
                }
                const range = v.time_range;
                void (async () => {
                  try {
                    const blob = await exportProjectLogs(v.project_id!, {
                      server_id: v.server_id,
                      service_id: v.service_id,
                      log_source_id: v.log_source_id,
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
                      message.error("导出失败：响应格式异常");
                      return;
                    }
                    if (blob.type && blob.type.includes("application/json")) {
                      const text = await blob.text();
                      try {
                        const err = JSON.parse(text) as { message?: string };
                        message.error(err.message || "导出失败");
                      } catch {
                        message.error("导出失败");
                      }
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
                })();
              }}
            >
              导出
            </Button>
            <Button icon={<RobotOutlined />} onClick={() => void runAiAnalyze()}>
              AI 分析
            </Button>
            <Button type="primary" icon={<SearchOutlined />} loading={loading} onClick={() => void runSearch({ page: 1 })}>
              检索
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical">
          <Row gutter={12}>
            <Col span={4}>
              <Form.Item label="项目" name="project_id" rules={[{ required: true, message: "请选择项目" }]}>
                <Select
                  options={projectOptions}
                  onChange={(pid) => {
                    form.setFieldsValue({ server_id: undefined, service_id: undefined, log_source_id: undefined });
                    setServers([]);
                    setServices([]);
                    setSources([]);
                    void reloadServers(pid);
                  }}
                />
              </Form.Item>
            </Col>
            <Col span={3}>
              <Form.Item label="采集来源" name="collector_mode">
                <Select
                  allowClear
                  placeholder="全部"
                  options={[
                    { value: "host", label: "主机 Agent" },
                    { value: "k8s", label: "集群 K8s" },
                  ]}
                  onChange={(mode) => {
                    if (mode === "k8s") {
                      form.setFieldsValue({ server_id: undefined, service_id: undefined, log_source_id: undefined });
                    }
                    if (mode === "host") {
                      form.setFieldsValue({ cluster_id: undefined, namespace: undefined, pod: undefined, container: undefined });
                    }
                  }}
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
            <Col span={5}>
              <Form.Item label="时间范围" name="time_range">
                <DatePicker.RangePicker showTime style={{ width: "100%" }} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={4}>
              <Form.Item label="K8s 集群" name="cluster_id">
                <Select allowClear disabled={watchCollectorMode === "host"} options={clusterOptions} placeholder="全部" />
              </Form.Item>
            </Col>
            <Col span={4}>
              <Form.Item label="Namespace" name="namespace">
                <Input allowClear disabled={watchCollectorMode === "host"} placeholder="default" />
              </Form.Item>
            </Col>
            <Col span={4}>
              <Form.Item label="Pod" name="pod">
                <Input allowClear disabled={watchCollectorMode === "host"} placeholder="pod 名" />
              </Form.Item>
            </Col>
            <Col span={4}>
              <Form.Item label="容器" name="container">
                <Input allowClear disabled={watchCollectorMode === "host"} placeholder="container" />
              </Form.Item>
            </Col>
            <Col span={4}>
              <Form.Item label="关键词" name="keyword">
                <Input placeholder="simple_query_string" allowClear />
              </Form.Item>
            </Col>
            <Col span={4}>
              <Form.Item label="日志级别" name="level">
                <Select
                  allowClear
                  placeholder="全部"
                  options={[
                    { value: "ERROR", label: "ERROR" },
                    { value: "WARN", label: "WARN" },
                    { value: "INFO", label: "INFO" },
                    { value: "DEBUG", label: "DEBUG" },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item label="文件名" name="file_path" tooltip="需先「同步下发」Loggie 后新日志才有 file_path。清空可查看全部文件。示例：748.log / info.log">
                <Input allowClear placeholder="748.log / info.log（留空=不限文件）" />
              </Form.Item>
            </Col>
          </Row>
        </Form>

        {emptyHint ? (
          <Alert type="warning" showIcon style={{ marginBottom: 12 }} message={emptyHint} />
        ) : null}

        {searchParams.get("alert_id") || searchParams.get("change_id") || searchParams.get("fingerprint") ? (
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 12 }}
            message="关联上下文检索"
            description={`已按${searchParams.get("alert_id") ? "告警" : searchParams.get("change_id") ? "变更" : "告警指纹"}锚点设置时间窗口（±${searchParams.get("window_minutes") || 5} 分钟）`}
          />
        ) : null}

        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: "overview",
              label: `统计概览 (${overview?.total ?? 0})`,
              children: overview ? (
                <LogOverviewPanel overview={overview} />
              ) : (
                <Typography.Text type="secondary">检索后将展示时间分布与错误签名</Typography.Text>
              ),
            },
            {
              key: "logs",
              label: `原始日志 (${total})`,
              children: (
                <div className="project-logs-table-wrap">
                  <Table
                    rowKey={(r, i) => `${r.timestamp}-${i}`}
                    loading={loading}
                    columns={columns}
                    dataSource={rows}
                    size="small"
                    className="project-logs-table"
                    tableLayout="fixed"
                    pagination={{
                      current: form.getFieldValue("page") ?? 1,
                      pageSize: form.getFieldValue("page_size") ?? 100,
                      total,
                      showSizeChanger: true,
                      pageSizeOptions: ["50", "100", "200", "500"],
                      showTotal: (t) => `共 ${t} 条`,
                      onChange: (page, pageSize) => void runSearch({ page, page_size: pageSize }),
                    }}
                    scroll={{ x: 1100 }}
                  />
                </div>
              ),
            },
            {
              key: "patterns",
              label: `模板聚类 (${patternTotal})`,
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
                    { title: "最近", dataIndex: "last_seen_at", width: 170, render: (v: string) => formatDateTime(v) },
                  ]}
                />
              ),
            },
            {
              key: "anomalies",
              label: `异常事件 (${anomalyTotal})`,
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
                      width: 110,
                      render: (v: string) => (v === "error_spike" ? "量突增" : "新模板"),
                    },
                    {
                      title: "级别",
                      dataIndex: "severity",
                      width: 90,
                      render: (v: string) => <Tag color={v === "critical" ? "error" : "warning"}>{v}</Tag>,
                    },
                    { title: "标题", dataIndex: "title" },
                    { title: "详情", dataIndex: "detail", ellipsis: true },
                    { title: "时间", dataIndex: "detected_at", width: 170, render: (v: string) => formatDateTime(v) },
                  ]}
                />
              ),
            },
          ]}
        />
      </Card>

      <Drawer title="AI 日志分析" width={560} open={aiOpen} onClose={() => setAiOpen(false)} destroyOnClose>
        {aiLoading ? (
          <Typography.Text>分析中…</Typography.Text>
        ) : aiResult ? (
          <Space direction="vertical" size="middle" style={{ width: "100%" }}>
            <Alert type="info" message={aiResult.ai_summary} />
            {(aiResult.root_causes?.length ?? 0) > 0 ? (
              <div>
                <Typography.Title level={5}>根因候选</Typography.Title>
                <List
                  size="small"
                  dataSource={aiResult.root_causes}
                  renderItem={(item) => (
                    <List.Item>
                      <List.Item.Meta
                        title={
                          <Space>
                            <span>{item.title || "—"}</span>
                            {item.confidence ? <Tag>{item.confidence}</Tag> : null}
                          </Space>
                        }
                        description={item.evidence}
                      />
                    </List.Item>
                  )}
                />
              </div>
            ) : null}
            {(aiResult.actions?.length ?? 0) > 0 ? (
              <div>
                <Typography.Title level={5}>排查建议</Typography.Title>
                <List
                  size="small"
                  dataSource={aiResult.actions}
                  renderItem={(item) => (
                    <List.Item>
                      <Typography.Text>
                        {item.priority ? `[P${item.priority}] ` : ""}
                        {item.action}
                        {item.command_hint ? ` — ${item.command_hint}` : ""}
                      </Typography.Text>
                    </List.Item>
                  )}
                />
              </div>
            ) : null}
          </Space>
        ) : (
          <Typography.Text type="secondary">点击「AI 分析」基于当前筛选条件生成摘要与排查建议。</Typography.Text>
        )}
      </Drawer>
    </div>
  );
}

function LogOverviewPanel({ overview }: { overview: LogOverviewResult }) {
  const levelEntries = Object.entries(overview.level_counts || {}).sort((a, b) => b[1] - a[1]);
  const errorCount = (overview.level_counts?.ERROR ?? 0) + (overview.level_counts?.FATAL ?? 0);
  const warnCount = overview.level_counts?.WARN ?? 0;

  return (
    <div className="project-logs-overview-panel">
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        <Col xs={12} sm={6}>
          <Card size="small" className="project-logs-stat-card">
            <Statistic title="命中总数" value={overview.total} />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" className="project-logs-stat-card">
            <Statistic title="ERROR / FATAL" value={errorCount} valueStyle={{ color: errorCount > 0 ? "#cf1322" : undefined }} />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" className="project-logs-stat-card">
            <Statistic title="WARN" value={warnCount} valueStyle={{ color: warnCount > 0 ? "#d48806" : undefined }} />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" className="project-logs-stat-card">
            <Statistic title="错误签名" value={overview.top_error_signatures?.length ?? 0} />
          </Card>
        </Col>
      </Row>

      <Card size="small" title="时间分布" className="project-logs-histogram-card" style={{ marginBottom: 12 }}>
        <LogHistogramChart buckets={overview.histogram || []} />
      </Card>

      <Row gutter={[12, 12]}>
        <Col xs={24} md={10}>
          <Card size="small" title="级别分布">
            {levelEntries.length ? (
              <div className="project-logs-level-bars">
                {levelEntries.map(([lv, cnt]) => {
                  const max = Math.max(1, ...levelEntries.map(([, c]) => c));
                  const pct = Math.round((cnt / max) * 100);
                  return (
                    <div key={lv} className="project-logs-level-row">
                      <Tag color={lv === "ERROR" || lv === "FATAL" ? "error" : lv === "WARN" ? "warning" : "default"}>{lv}</Tag>
                      <div className="project-logs-level-bar-track">
                        <div className="project-logs-level-bar-fill" style={{ width: `${pct}%` }} />
                      </div>
                      <span className="project-logs-level-count">{cnt.toLocaleString()}</span>
                    </div>
                  );
                })}
              </div>
            ) : (
              <Typography.Text type="secondary">无级别统计</Typography.Text>
            )}
          </Card>
        </Col>
        <Col xs={24} md={14}>
          <Card size="small" title="Top 错误签名">
            {(overview.top_error_signatures?.length ?? 0) > 0 ? (
              <List
                size="small"
                dataSource={overview.top_error_signatures.slice(0, 8)}
                renderItem={(item) => (
                  <List.Item>
                    <List.Item.Meta
                      title={
                        <Space wrap>
                          <Tag>{item.count}</Tag>
                          {item.level ? <Tag color="error">{item.level}</Tag> : null}
                          <Typography.Text code style={{ fontSize: 12 }}>{item.signature}</Typography.Text>
                        </Space>
                      }
                      description={<Typography.Text type="secondary" ellipsis>{item.sample}</Typography.Text>}
                    />
                  </List.Item>
                )}
              />
            ) : (
              <Typography.Text type="secondary">当前筛选下无 ERROR/WARN 签名</Typography.Text>
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
}

function LogHistogramChart({ buckets }: { buckets: Array<{ time: string; count: number }> }) {
  const sorted = [...buckets].sort((a, b) => a.time.localeCompare(b.time));
  const sampled =
    sorted.length > 48
      ? sorted.filter((_, i) => i % Math.ceil(sorted.length / 48) === 0)
      : sorted;
  const max = Math.max(1, ...sampled.map((b) => b.count));
  if (!sampled.length) {
    return <Typography.Text type="secondary">无时间分布数据（可缩小时间范围或确认 @timestamp 字段）</Typography.Text>;
  }

  const chartW = Math.max(640, sampled.length * 18);
  const chartH = 160;
  const padL = 36;
  const padB = 28;
  const padT = 8;
  const innerH = chartH - padB - padT;
  const barW = Math.max(6, (chartW - padL - 8) / sampled.length - 4);

  return (
    <div className="project-logs-histogram-wrap">
      <svg width="100%" height={chartH} viewBox={`0 0 ${chartW} ${chartH}`} preserveAspectRatio="xMinYMid meet">
        {[0, 0.25, 0.5, 0.75, 1].map((r) => {
          const y = padT + innerH * (1 - r);
          const val = Math.round(max * r);
          return (
            <g key={r}>
              <line x1={padL} y1={y} x2={chartW - 4} y2={y} stroke="rgba(0,0,0,0.06)" />
              <text x={padL - 4} y={y + 4} textAnchor="end" fontSize={10} fill="#8c8c8c">
                {val}
              </text>
            </g>
          );
        })}
        {sampled.map((b, i) => {
          const h = Math.max(2, (b.count / max) * innerH);
          const x = padL + i * (barW + 4);
          const y = padT + innerH - h;
          const label = formatDateTime(b.time);
          return (
            <g key={`${b.time}-${i}`}>
              <title>{`${label}: ${b.count}`}</title>
              <rect x={x} y={y} width={barW} height={h} rx={2} fill="#1677ff" opacity={0.85} />
              {(i === 0 || i === sampled.length - 1 || sampled.length <= 12) && (
                <text x={x + barW / 2} y={chartH - 6} textAnchor="middle" fontSize={9} fill="#8c8c8c">
                  {label.slice(5, 16)}
                </text>
              )}
            </g>
          );
        })}
      </svg>
    </div>
  );
}

function LogMessageCell({ message, highlight }: { message?: string; highlight?: string }) {
  if (highlight) {
    return <div className="log-message-cell" dangerouslySetInnerHTML={{ __html: sanitizeLogHighlight(highlight) }} />;
  }
  return <div className="log-message-cell">{message || "-"}</div>;
}

/** 仅允许 <mark> 高亮标签，其余 HTML 一律转义，降低存储型 XSS 风险。 */
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
  const pipe = message.match(/\|\s*(TRACE|DEBUG|INFO|WARN|ERROR|FATAL)\s*\|/i);
  if (pipe?.[1]) return normalizeLogLevel(pipe[1]);
  const klog = message.match(/(?:^|[\s>])([IWEF])\d{4}\s+\d{2}:\d{2}:\d{2}/);
  if (klog?.[1]) {
    const map: Record<string, string> = { I: "INFO", W: "WARN", E: "ERROR", F: "FATAL" };
    return map[klog[1]] || "";
  }
  return "";
}
