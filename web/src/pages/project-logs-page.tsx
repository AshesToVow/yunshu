import { DownloadOutlined, ReloadOutlined, SearchOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Col, DatePicker, Form, Input, Row, Select, Space, Table, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs, { type Dayjs } from "dayjs";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  exportProjectLogs,
  getProjectLogSources,
  getProjectServers,
  getProjectServices,
  getProjects,
  searchProjectLogs,
  type LogSearchItem,
  type LogSourceItem,
  type ProjectItem,
  type ServerItem,
  type ServiceItem,
} from "../services/projects";
import { formatDateTime } from "../utils/format";

type SearchForm = {
  project_id?: number;
  server_id?: number;
  service_id?: number;
  log_source_id?: number;
  keyword?: string;
  level?: string;
  file_path?: string;
  time_range?: [Dayjs, Dayjs];
  page?: number;
  page_size?: number;
};

export function ProjectLogsPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [servers, setServers] = useState<ServerItem[]>([]);
  const [services, setServices] = useState<ServiceItem[]>([]);
  const [sources, setSources] = useState<LogSourceItem[]>([]);
  const [rows, setRows] = useState<LogSearchItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  const [form] = Form.useForm<SearchForm>();
  const watchProjectId = Form.useWatch("project_id", form);
  const watchServerId = Form.useWatch("server_id", form);

  const projectOptions = useMemo(() => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })), [projects]);
  const serverOptions = useMemo(
    () => servers.map((s) => ({ value: s.id, label: `${s.name} ${s.host}:${s.port}` })),
    [servers],
  );
  const serviceOptions = useMemo(() => services.map((s) => ({ value: s.id, label: s.name })), [services]);
  const sourceOptions = useMemo(() => sources.map((s) => ({ value: s.id, label: `${s.log_type}:${s.path}` })), [sources]);

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
      setLoading(true);
      setEmptyHint("");
      try {
        const res = await searchProjectLogs(values.project_id, {
          server_id: values.server_id,
          service_id: values.service_id,
          log_source_id: values.log_source_id,
          keyword: values.keyword?.trim() || undefined,
          level: values.level?.trim() || undefined,
          file_path: filePath,
          from: range?.[0]?.toISOString(),
          to: range?.[1]?.toISOString(),
          page,
          page_size: pageSize,
        });
        setRows(res.list);
        setTotal(res.total);
        form.setFieldsValue({ page, page_size: pageSize });
        if ((res.total ?? 0) === 0 && filePath) {
          setEmptyHint(
            `按文件名「${filePath}」无命中。常见原因：① ES 里历史文档还没有 file_path（需在 Loggie 状态页「同步下发」后才会写入）；② ${filePath} 已轮转停写，活跃文件可能是更大编号（如 748.log）。请先清空文件名筛选项再查，或把时间范围扩到该文件最后写入时间。`,
          );
        } else if ((res.total ?? 0) === 0 && !range?.[0] && !range?.[1]) {
          setEmptyHint("未选时间范围且无数据。可先设近 24 小时，或确认 Loggie 已在采集并写入 ES。");
        }
      } catch (e: unknown) {
        message.error(String((e as Error)?.message ?? e));
      } finally {
        setLoading(false);
      }
    },
    [form],
  );

  useEffect(() => {
    void (async () => {
      const data = await getProjects({ page: 1, page_size: 1000 });
      setProjects(data.list);
      const defaultProject = data.list[0]?.id;
      if (defaultProject) {
        form.setFieldsValue({
          project_id: defaultProject,
          page: 1,
          page_size: 100,
          time_range: [dayjs().subtract(24, "hour"), dayjs()],
        });
        await reloadServers(defaultProject);
        await runSearch({ project_id: defaultProject, page: 1, page_size: 100, time_range: [dayjs().subtract(24, "hour"), dayjs()] });
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
      title: "文件",
      dataIndex: "file_path",
      width: 200,
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
      title: "Pod",
      dataIndex: "pod",
      width: 140,
      render: (v?: string) => <span className="log-meta-cell">{v || "-"}</span>,
    },
    {
      title: "容器",
      dataIndex: "container",
      width: 100,
      render: (v?: string) => <span className="log-meta-cell">{v || "-"}</span>,
    },
  ];

  return (
    <div className="project-logs-page">
      <Card
        className="table-card project-logs-card"
        title="日志检索"
        extra={
          <Space>
            <Tag color="blue">Loggie → Elasticsearch</Tag>
            <Button icon={<ReloadOutlined />} onClick={() => void runSearch()}>
              刷新
            </Button>
            <Button
              icon={<DownloadOutlined />}
              onClick={() => {
                const v = form.getFieldsValue();
                if (!v.project_id) return;
                const range = v.time_range;
                void (async () => {
                  const blob = await exportProjectLogs(v.project_id!, {
                    server_id: v.server_id,
                    service_id: v.service_id,
                    log_source_id: v.log_source_id,
                    keyword: v.keyword,
                    from: range?.[0]?.toISOString(),
                    to: range?.[1]?.toISOString(),
                    page_size: 1000,
                  });
                  const url = window.URL.createObjectURL(blob);
                  const a = document.createElement("a");
                  a.href = url;
                  a.download = `project-${v.project_id}-logs.txt`;
                  a.click();
                  window.URL.revokeObjectURL(url);
                })();
              }}
            >
              导出
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
            <Col span={4}>
              <Form.Item label="服务器" name="server_id">
                <Select
                  allowClear
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
                <Select allowClear options={sourceOptions} placeholder="全部" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="时间范围" name="time_range">
                <DatePicker.RangePicker showTime style={{ width: "100%" }} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item label="关键词" name="keyword">
                <Input placeholder="支持 simple_query_string，高亮匹配内容" allowClear />
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
      </Card>
    </div>
  );
}

function LogMessageCell({ message, highlight }: { message?: string; highlight?: string }) {
  if (highlight) {
    return <div className="log-message-cell" dangerouslySetInnerHTML={{ __html: highlight }} />;
  }
  return <div className="log-message-cell">{message || "-"}</div>;
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
