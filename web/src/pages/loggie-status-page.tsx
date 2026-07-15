import { CloudServerOutlined, CloudUploadOutlined, DownloadOutlined, ReloadOutlined, SyncOutlined, ThunderboltOutlined } from "@ant-design/icons";
import { Button, Card, Checkbox, Form, Input, InputNumber, Modal, Select, Space, Table, Tag, Tooltip, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  bootstrapLoggie,
  deployLoggieConfig,
  downloadLoggieBundle,
  downloadLoggieFile,
  getESConfigPreview,
  getLoggieBootstrapSources,
  getLoggieStatus,
  getProjects,
  restartLoggie,
  syncLoggieFromLogSources,
  type ESConfigPreview,
  type LoggieBootstrapResult,
  type LoggieBootstrapSourcePreview,
  type LoggieStatusItem,
  type ProjectItem,
} from "../services/log-platform";
import { formatDateTime } from "../utils/format";

const DEFAULT_YUNSHU_URL =
  typeof window !== "undefined" && window.location.port === "5173"
    ? `${window.location.protocol}//${window.location.hostname}:8080`
    : typeof window !== "undefined"
      ? window.location.origin
      : "http://127.0.0.1:8080";

export function LoggieStatusPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [rows, setRows] = useState<LoggieStatusItem[]>([]);
  const [esCfg, setEsCfg] = useState<ESConfigPreview | null>(null);
  const [loading, setLoading] = useState(false);
  const [bootstrapOpen, setBootstrapOpen] = useState(false);
  const [bootstrapServer, setBootstrapServer] = useState<LoggieStatusItem | null>(null);
  const [bootstrapSources, setBootstrapSources] = useState<LoggieBootstrapSourcePreview[]>([]);
  const [bootstrapResult, setBootstrapResult] = useState<LoggieBootstrapResult | null>(null);
  const [bootstrapLoading, setBootstrapLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const [form] = Form.useForm();

  const projectOptions = useMemo(() => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })), [projects]);

  const reload = useCallback(async (pid?: number) => {
    if (!pid) return;
    setLoading(true);
    try {
      const [status, cfg] = await Promise.all([getLoggieStatus(pid), getESConfigPreview().catch(() => null)]);
      setRows(status.list);
      setEsCfg(cfg);
    } catch (e: unknown) {
      message.error(String((e as Error)?.message ?? e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void (async () => {
      const data = await getProjects({ page: 1, page_size: 1000 });
      setProjects(data.list);
      const pid = data.list[0]?.id;
      setProjectId(pid);
      if (pid) await reload(pid);
    })();
  }, [reload]);

  const openBootstrap = async (row: LoggieStatusItem) => {
    setBootstrapServer(row);
    setBootstrapResult(null);
    form.setFieldsValue({
      auto_from_log_sources: true,
      deploy_dir: "/export/loggie",
      monitor_port: row.monitor_port ?? 9196,
      yunshu_url: DEFAULT_YUNSHU_URL,
      deploy_after_bootstrap: false,
      log_paths: "",
    });
    setBootstrapOpen(true);
    if (projectId && row.server_id > 0) {
      try {
        const res = await getLoggieBootstrapSources(projectId, row.server_id);
        setBootstrapSources(res.list ?? []);
      } catch {
        setBootstrapSources([]);
      }
    } else {
      setBootstrapSources([]);
    }
  };

  const submitBootstrap = async () => {
    if (!projectId || !bootstrapServer?.server_id) return;
    const values = await form.validateFields();
    const autoFrom = values.auto_from_log_sources !== false;
    const paths = String(values.log_paths ?? "")
      .split("\n")
      .map((s: string) => s.trim())
      .filter(Boolean);
    if (!autoFrom && paths.length === 0) {
      message.error("未启用自动读取日志源时，请填写日志路径");
      return;
    }
    setBootstrapLoading(true);
    try {
      const res = await bootstrapLoggie(projectId, {
        server_id: bootstrapServer.server_id,
        log_paths: autoFrom ? undefined : paths,
        service_id: values.service_id,
        log_source_id: values.log_source_id,
        monitor_port: values.monitor_port,
        yunshu_url: values.yunshu_url,
        deploy_dir: values.deploy_dir,
        auto_from_log_sources: autoFrom,
        deploy_after_bootstrap: values.deploy_after_bootstrap,
      });
      setBootstrapResult(res);
      if (res.deployed) {
        message.success(`引导完成，已下发 ${res.pipeline_count ?? 0} 条 pipeline`);
      } else if (res.deploy_message) {
        message.warning(`引导完成，但下发失败：${res.deploy_message}`);
      } else {
        message.success(`引导完成，共 ${res.pipeline_count ?? res.source_count ?? 0} 条 pipeline`);
      }
      await reload(projectId);
    } catch (e: unknown) {
      message.error(String((e as Error)?.message ?? e));
    } finally {
      setBootstrapLoading(false);
    }
  };

  const runDeployAction = async (row: LoggieStatusItem, action: "sync" | "deploy" | "restart") => {
    if (!projectId || !row.server_id) return;
    setActionLoading(row.server_id);
    try {
      const payload = { server_id: row.server_id };
      const res =
        action === "sync"
          ? await syncLoggieFromLogSources(projectId, payload)
          : action === "deploy"
            ? await deployLoggieConfig(projectId, { ...payload, sync_from_db: true })
            : await restartLoggie(projectId, payload);
      if (res.success) {
        message.success(res.message || "操作成功");
      } else {
        message.error(res.message || "操作失败");
      }
      await reload(projectId);
    } catch (e: unknown) {
      message.error(String((e as Error)?.message ?? e));
    } finally {
      setActionLoading(null);
    }
  };

  const columns: ColumnsType<LoggieStatusItem> = [
    { title: "服务器", dataIndex: "server_name", width: 140 },
    { title: "地址", dataIndex: "server_host", width: 180 },
    {
      title: "Agent",
      dataIndex: "registered",
      width: 90,
      render: (v: boolean) => (v ? <Tag color="blue">已登记</Tag> : <Tag>未登记</Tag>),
    },
    {
      title: "在线",
      dataIndex: "online",
      width: 80,
      render: (v: boolean) => (v ? <Tag color="success">在线</Tag> : <Tag color="error">离线</Tag>),
    },
    {
      title: "日志上报",
      dataIndex: "recent_ingest",
      width: 120,
      render: (v: boolean, r) =>
        v ? <Tag color="processing">近5分钟 {r.recent_doc_count} 条</Tag> : <Tag color="warning">无上报</Tag>,
    },
    {
      title: "监控端口",
      width: 110,
      render: (_, r) => (
        <Space direction="vertical" size={0}>
          <span>:{r.monitor_port ?? 9196}</span>
          {r.monitor_reachable ? <Tag color="success">心跳可达</Tag> : <Tag>未上报</Tag>}
        </Space>
      ),
    },
    {
      title: "采集 FD",
      width: 100,
      render: (_, r) => {
        const live = r.live_probe?.reachable ? r.live_probe.active_fd_count : undefined;
        const reported = r.active_fd_count ?? 0;
        const text = live != null ? `${reported} / 探测 ${live}` : String(reported);
        return (
          <Tooltip title={r.live_probe?.error || r.monitor_detail || "活跃文件句柄数"}>
            <span>{text}</span>
          </Tooltip>
        );
      },
    },
    {
      title: "Pipeline",
      width: 100,
      render: (_, r) => {
        const cnt = r.active_pipeline_count ?? 0;
        const liveCnt = r.live_probe?.active_pipeline_count;
        return liveCnt != null && r.live_probe?.reachable ? `${cnt} / 探测 ${liveCnt}` : String(cnt || r.pipeline_status || "-");
      },
    },
    { title: "版本", dataIndex: "version", width: 90, render: (v?: string) => v || "-" },
    {
      title: "ES Sink",
      dataIndex: "es_sink_ok",
      width: 90,
      render: (v: boolean) => (v ? <Tag color="success">正常</Tag> : <Tag>未知</Tag>),
    },
    { title: "最后心跳", dataIndex: "last_seen_at", width: 170, render: (v?: string) => (v ? formatDateTime(v) : "-") },
    { title: "最近错误", dataIndex: "last_error", ellipsis: true },
    {
      title: "操作",
      width: 320,
      fixed: "right",
      render: (_, r) => (
        <Space size="small" wrap>
          <Button size="small" icon={<ThunderboltOutlined />} disabled={!projectId} onClick={() => void openBootstrap(r)}>
            引导
          </Button>
          <Button
            size="small"
            icon={<SyncOutlined />}
            disabled={!projectId || !r.registered}
            loading={actionLoading === r.server_id}
            onClick={() => void runDeployAction(r, "sync")}
          >
            同步下发
          </Button>
          <Button
            size="small"
            icon={<ReloadOutlined />}
            disabled={!projectId || !r.registered}
            loading={actionLoading === r.server_id}
            onClick={() => void runDeployAction(r, "restart")}
          >
            重启
          </Button>
          <Button
            size="small"
            icon={<DownloadOutlined />}
            disabled={!projectId || !r.registered}
            onClick={() =>
              void downloadLoggieFile(projectId!, r.server_id, "pipelines").catch((e: unknown) =>
                message.error(String((e as Error)?.message ?? e)),
              )
            }
          >
            pipelines
          </Button>
        </Space>
      ),
    },
  ];

  const sourceColumns: ColumnsType<LoggieBootstrapSourcePreview> = [
    { title: "日志源 ID", dataIndex: "log_source_id", width: 100 },
    { title: "服务 ID", dataIndex: "service_id", width: 90 },
    { title: "类型", dataIndex: "log_type", width: 80 },
    { title: "路径", dataIndex: "path", ellipsis: true },
    { title: "Glob", dataIndex: "glob_path", ellipsis: true },
  ];

  return (
    <div className="loggie-status-page">
      <Card
        title="Loggie 采集状态"
        extra={
          <Space>
            <Select
              style={{ width: 260 }}
              options={projectOptions}
              value={projectId}
              onChange={(v) => {
                setProjectId(v);
                void reload(v);
              }}
            />
            <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void reload(projectId)}>
              刷新
            </Button>
          </Space>
        }
      >
        {esCfg ? (
          <Space wrap style={{ marginBottom: 12 }}>
            <Tag icon={<CloudServerOutlined />} color={esCfg.enabled ? "blue" : "default"}>
              ES {esCfg.enabled ? "已启用" : "未启用"}
            </Tag>
            {esCfg.addresses?.map((a) => (
              <Tag key={a}>{a}</Tag>
            ))}
            <Tag>索引 {esCfg.index_pattern}</Tag>
            {esCfg.username ? <Tag>用户 {esCfg.username}</Tag> : null}
          </Space>
        ) : null}
        <Table rowKey="server_id" loading={loading} columns={columns} dataSource={rows} size="small" scroll={{ x: 1800 }} pagination={false} />
      </Card>

      <Modal
        title={bootstrapServer ? `Loggie 引导 — ${bootstrapServer.server_name}` : "Loggie 引导"}
        open={bootstrapOpen}
        onCancel={() => setBootstrapOpen(false)}
        width={820}
        footer={
          bootstrapResult ? (
            <Space>
              <Button onClick={() => bootstrapResult && downloadLoggieBundle(bootstrapResult)} type="primary" icon={<DownloadOutlined />}>
                下载全部文件
              </Button>
              <Button onClick={() => setBootstrapOpen(false)}>关闭</Button>
            </Space>
          ) : (
            <Space>
              <Button onClick={() => setBootstrapOpen(false)}>取消</Button>
              <Button type="primary" loading={bootstrapLoading} onClick={() => void submitBootstrap()}>
                生成并保存配置
              </Button>
            </Space>
          )
        }
      >
        {!bootstrapResult ? (
          <Form form={form} layout="vertical" initialValues={{ auto_from_log_sources: true, deploy_dir: "/export/loggie" }}>
            <Form.Item name="auto_from_log_sources" valuePropName="checked">
              <Checkbox>从 CMDB 日志源自动生成 pipeline（每个日志源独立 service_id / log_source_id）</Checkbox>
            </Form.Item>
            {bootstrapSources.length > 0 ? (
              <Table
                size="small"
                rowKey="log_source_id"
                columns={sourceColumns}
                dataSource={bootstrapSources}
                pagination={false}
                style={{ marginBottom: 16 }}
              />
            ) : (
              <Tag color="warning" style={{ marginBottom: 12 }}>
                该服务器暂无已启用的文件类日志源，将使用默认路径或手动填写
              </Tag>
            )}
            <Form.Item noStyle shouldUpdate={(prev, cur) => prev.auto_from_log_sources !== cur.auto_from_log_sources}>
              {({ getFieldValue }) =>
                getFieldValue("auto_from_log_sources") === false ? (
                  <Form.Item name="log_paths" label="手动日志路径（每行一条 glob）">
                    <Input.TextArea rows={3} placeholder="/var/log/messages&#10;/var/log/kube-apiserver/*.log" />
                  </Form.Item>
                ) : null
              }
            </Form.Item>
            <Space wrap style={{ width: "100%" }}>
              <Form.Item name="monitor_port" label="Loggie HTTP 监控端口">
                <InputNumber min={1} max={65535} style={{ width: 160 }} />
              </Form.Item>
              <Form.Item name="deploy_dir" label="远端部署目录">
                <Input style={{ width: 220 }} placeholder="/export/loggie" />
              </Form.Item>
            </Space>
            <Form.Item name="yunshu_url" label="Yunshu 后端地址（写入心跳 env，须为 API :8080）" rules={[{ required: true }]}>
              <Input placeholder="http://10.10.10.103:8080" />
            </Form.Item>
            <Form.Item name="deploy_after_bootstrap" valuePropName="checked">
              <Checkbox>
                <CloudUploadOutlined /> 引导后立即 SSH 下发 pipelines.yml 并重启 Loggie（需服务器 SSH 凭证）
              </Checkbox>
            </Form.Item>
          </Form>
        ) : (
          <Space direction="vertical" style={{ width: "100%" }}>
            <Space wrap>
              <Tag color="blue">Token 已生成（保存在 env 文件中）</Tag>
              <Tag>监控端口 {bootstrapResult.monitor_port}</Tag>
              <Tag>Pipeline {bootstrapResult.pipeline_count ?? 0} 条</Tag>
              <Tag>日志源 {bootstrapResult.source_count ?? 0} 个</Tag>
              {bootstrapResult.deployed ? <Tag color="success">已下发</Tag> : bootstrapResult.deploy_message ? <Tag color="warning">未下发</Tag> : null}
            </Space>
            {bootstrapResult.deploy_message ? <Tag color="warning">{bootstrapResult.deploy_message}</Tag> : null}
            <Tag>ES {bootstrapResult.es_addresses?.join(", ")}</Tag>
            <Space wrap>
              <Button icon={<DownloadOutlined />} onClick={() => downloadLoggieBundle(bootstrapResult)}>
                下载全部
              </Button>
              <Button onClick={() => downloadTextFile(bootstrapResult.pipelines_only_yaml || bootstrapResult.pipeline_yaml, "pipelines.yml")}>
                仅 pipelines.yml
              </Button>
            </Space>
            <Input.TextArea value={bootstrapResult.pipelines_only_yaml || bootstrapResult.pipeline_yaml} rows={14} readOnly />
          </Space>
        )}
      </Modal>
    </div>
  );
}

function downloadTextFile(content: string, filename: string) {
  const blob = new Blob([content], { type: "application/x-yaml;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename || "pipelines.yml";
  a.click();
  URL.revokeObjectURL(url);
}
