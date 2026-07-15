import {
  CloudServerOutlined,
  CloudUploadOutlined,
  DeleteOutlined,
  DownloadOutlined,
  PlusOutlined,
  PoweroffOutlined,
  ReloadOutlined,
  SyncOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";
import { Alert, Button, Card, Checkbox, Dropdown, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag, Tooltip, message } from "antd";
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
  installLoggie,
  restartLoggie,
  startLoggie,
  stopLoggie,
  syncLoggieFromLogSources,
  uninstallLoggie,
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
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [addOpen, setAddOpen] = useState(false);
  const [addServerId, setAddServerId] = useState<number>();
  const [form] = Form.useForm();

  const projectOptions = useMemo(() => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })), [projects]);
  const addServerOptions = useMemo(
    () =>
      rows.map((r) => ({
        value: r.server_id,
        label: `${r.server_name} ${r.server_host}${r.registered ? "（已登记）" : "（未登记）"}`,
      })),
    [rows],
  );

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
        monitor_port: values.monitor_port,
        yunshu_url: values.yunshu_url,
        deploy_dir: values.deploy_dir,
        auto_from_log_sources: autoFrom,
        deploy_after_bootstrap: values.deploy_after_bootstrap,
      });
      setBootstrapResult(res);
      if (res.deployed) {
        message.success(`引导完成，索引 ${res.es_index_pattern}，pipeline ${res.pipeline_count ?? 0} 条`);
      } else if (res.deploy_message) {
        const hint = res.deploy_message.length > 120 ? `${res.deploy_message.slice(0, 120)}…` : res.deploy_message;
        message.warning(`引导完成（配置未完全生效）：${hint}`);
      } else {
        message.success(`引导完成，索引 ${res.es_index_pattern}，pipeline ${res.pipeline_count ?? 0} 条`);
      }
      await reload(projectId);
    } catch (e: unknown) {
      message.error(String((e as Error)?.message ?? e));
    } finally {
      setBootstrapLoading(false);
    }
  };

  type AgentAction = "sync" | "deploy" | "restart" | "start" | "stop" | "install" | "uninstall";

  const runDeployAction = async (row: LoggieStatusItem, action: AgentAction) => {
    if (!projectId || !row.server_id) return;
    const key = `${action}-${row.server_id}`;
    setActionLoading(key);
    try {
      const payload = { server_id: row.server_id };
      let res;
      if (action === "install") {
        const values = form.getFieldsValue();
        res = await installLoggie(projectId, {
          ...payload,
          deploy_dir: values.deploy_dir || "/export/loggie",
          yunshu_url: values.yunshu_url || DEFAULT_YUNSHU_URL,
          monitor_port: values.monitor_port || row.monitor_port || 9196,
        });
      } else if (action === "uninstall") {
        res = await uninstallLoggie(projectId, payload);
      } else if (action === "sync") {
        res = await syncLoggieFromLogSources(projectId, payload);
      } else if (action === "deploy") {
        res = await deployLoggieConfig(projectId, { ...payload, sync_from_db: true });
      } else if (action === "start") {
        res = await startLoggie(projectId, payload);
      } else if (action === "stop") {
        res = await stopLoggie(projectId, payload);
      } else {
        res = await restartLoggie(projectId, payload);
      }
      if (res.success) message.success(res.message || "操作成功");
      else message.error(res.message || "操作失败");
      await reload(projectId);
    } catch (e: unknown) {
      message.error(String((e as Error)?.message ?? e));
    } finally {
      setActionLoading(null);
    }
  };

  const forceUnregister = async (row: LoggieStatusItem) => {
    if (!projectId || !row.server_id) return;
    Modal.confirm({
      title: "仅清除平台登记？",
      content: "不会对远端服务器做任何操作，仅删除本平台的 Agent 登记记录。",
      okText: "清除登记",
      okButtonProps: { danger: true },
      onOk: async () => {
        const key = `uninstall-${row.server_id}`;
        setActionLoading(key);
        try {
          const res = await uninstallLoggie(projectId, {
            server_id: row.server_id,
            skip_remote: true,
          });
          if (res.success) message.success(res.message || "已清除登记");
          else message.error(res.message || "操作失败");
          await reload(projectId);
        } catch (e: unknown) {
          message.error(String((e as Error)?.message ?? e));
        } finally {
          setActionLoading(null);
        }
      },
    });
  };

  const columns: ColumnsType<LoggieStatusItem> = [
    { title: "服务器", dataIndex: "server_name", width: 120, ellipsis: true },
    { title: "地址", dataIndex: "server_host", width: 140, ellipsis: true },
    {
      title: "状态",
      width: 150,
      render: (_, r) => (
        <Space size={4} wrap>
          {r.registered ? <Tag color="blue">已登记</Tag> : <Tag>未登记</Tag>}
          {r.online ? <Tag color="success">在线</Tag> : <Tag color="error">离线</Tag>}
          {r.recent_ingest ? <Tag color="processing">采集</Tag> : <Tag color="warning">无上报</Tag>}
        </Space>
      ),
    },
    {
      title: "监控/FD",
      width: 110,
      render: (_, r) => {
        const fd = r.live_probe?.reachable ? r.live_probe.active_fd_count : (r.active_fd_count ?? 0);
        return (
          <Tooltip title={r.live_probe?.error || r.monitor_detail || `索引 yunshu-agent-${r.server_id}-*`}>
            <span>
              {r.monitor_reachable ? `:${r.monitor_port ?? 9196}` : "-"} / {fd}
            </span>
          </Tooltip>
        );
      },
    },
    {
      title: "ES",
      dataIndex: "es_sink_ok",
      width: 70,
      render: (v: boolean) => (v ? <Tag color="success">OK</Tag> : <Tag>-</Tag>),
    },
    { title: "心跳", dataIndex: "last_seen_at", width: 150, render: (v?: string) => (v ? formatDateTime(v) : "-") },
    { title: "错误", dataIndex: "last_error", ellipsis: true, width: 160 },
    {
      title: "操作",
      width: 260,
      fixed: "right",
      render: (_, r) => {
        const busy = (a: AgentAction) => actionLoading === `${a}-${r.server_id}`;
        const more = [
          { key: "start", label: "启动", disabled: !r.registered, onClick: () => void runDeployAction(r, "start") },
          { key: "stop", label: "停止", disabled: !r.registered, onClick: () => void runDeployAction(r, "stop") },
          { key: "restart", label: "重启", disabled: !r.registered, onClick: () => void runDeployAction(r, "restart") },
          {
            key: "pipelines",
            label: "下载 pipelines",
            disabled: !r.registered,
            onClick: () =>
              void downloadLoggieFile(projectId!, r.server_id, "pipelines").catch((e: unknown) =>
                message.error(String((e as Error)?.message ?? e)),
              ),
          },
          {
            key: "force-unregister",
            label: "仅清登记",
            disabled: !r.registered,
            danger: true,
            onClick: () => void forceUnregister(r),
          },
        ];
        return (
          <Space size={4} wrap={false}>
            <Button size="small" type="link" icon={<ThunderboltOutlined />} onClick={() => void openBootstrap(r)}>
              引导
            </Button>
            <Button size="small" type="link" icon={<PoweroffOutlined />} loading={busy("install")} onClick={() => void runDeployAction(r, "install")}>
              安装
            </Button>
            <Button size="small" type="link" icon={<SyncOutlined />} disabled={!r.registered} loading={busy("sync")} onClick={() => void runDeployAction(r, "sync")}>
              热更
            </Button>
            <Popconfirm
              title="删除 Agent"
              description="将停止并卸载远端 Loggie（含部署目录），并清除平台登记。是否继续？"
              okText="删除"
              okButtonProps={{ danger: true }}
              cancelText="取消"
              disabled={!r.registered}
              onConfirm={() => void runDeployAction(r, "uninstall")}
            >
              <Button size="small" type="link" danger icon={<DeleteOutlined />} disabled={!r.registered} loading={busy("uninstall")}>
                删除
              </Button>
            </Popconfirm>
            <Dropdown
              menu={{
                items: more.map((m) => ({
                  key: m.key,
                  label: m.label,
                  disabled: m.disabled,
                  danger: m.danger,
                  onClick: m.onClick,
                })),
              }}
            >
              <Button size="small" type="link">
                更多
              </Button>
            </Dropdown>
          </Space>
        );
      },
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
        className="table-card"
        title="Agent 管理"
        extra={
          <Space wrap>
            <Select
              style={{ minWidth: 180, maxWidth: 260 }}
              options={projectOptions}
              value={projectId}
              onChange={(v) => {
                setProjectId(v);
                void reload(v);
              }}
            />
            <Button
              type="primary"
              icon={<PlusOutlined />}
              disabled={!projectId || rows.length === 0}
              onClick={() => {
                const prefer = rows.find((r) => !r.registered) ?? rows[0];
                setAddServerId(prefer?.server_id);
                setAddOpen(true);
              }}
            >
              添加 Agent
            </Button>
            <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void reload(projectId)}>
              刷新
            </Button>
          </Space>
        }
      >
        {esCfg ? (
          <Space wrap size={[8, 8]} style={{ marginBottom: 12, maxWidth: "100%" }}>
            <Tag icon={<CloudServerOutlined />} color={esCfg.enabled ? "blue" : "default"}>
              ES {esCfg.enabled ? "已启用" : "未启用"}
            </Tag>
            <Tag>离线包 deploy/loggie/binary/loggie → /export/loggie</Tag>
            <Tag>检索 {esCfg.index_pattern}</Tag>
          </Space>
        ) : null}
        <Table
          rowKey="server_id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          size="small"
          scroll={{ x: 1100 }}
          pagination={false}
        />
      </Card>

      <Modal
        title="添加 Agent"
        open={addOpen}
        onCancel={() => setAddOpen(false)}
        okText="下一步：引导配置"
        onOk={() => {
          const row = rows.find((r) => r.server_id === addServerId);
          if (!row) {
            message.warning("请选择服务器");
            return;
          }
          setAddOpen(false);
          void openBootstrap(row);
        }}
      >
        <Form layout="vertical">
          <Form.Item label="目标服务器" required extra="服务器需已在「项目管理 → 服务器管理」登记并可 SSH">
            <Select
              style={{ width: "100%" }}
              options={addServerOptions}
              value={addServerId}
              onChange={setAddServerId}
              placeholder="选择服务器"
            />
          </Form.Item>
          <Alert type="info" showIcon message="流程：引导生成 Token/配置 → 一键安装（上传离线二进制 + 启用 Loggie 与心跳）" />
        </Form>
      </Modal>

      <Modal
        className="loggie-bootstrap-modal"
        title={bootstrapServer ? `Agent 引导 — ${bootstrapServer.server_name}` : "Agent 引导"}
        open={bootstrapOpen}
        onCancel={() => setBootstrapOpen(false)}
        width={820}
        styles={{ body: { maxWidth: "100%", overflowX: "hidden" } }}
        destroyOnHidden
        footer={
          bootstrapResult ? (
            <Space>
              <Button onClick={() => bootstrapResult && downloadLoggieBundle(bootstrapResult)} type="primary" icon={<DownloadOutlined />}>
                下载全部文件
              </Button>
              <Button
                icon={<PoweroffOutlined />}
                loading={actionLoading?.startsWith("install-")}
                onClick={() => bootstrapServer && void runDeployAction(bootstrapServer, "install")}
              >
                一键安装
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
              <Checkbox>从 CMDB 日志源自动生成 pipeline（字段含 service_name / project / server）</Checkbox>
            </Form.Item>
            {bootstrapSources.length > 0 ? (
              <Table size="small" rowKey="log_source_id" columns={sourceColumns} dataSource={bootstrapSources} pagination={false} style={{ marginBottom: 16 }} />
            ) : (
              <Tag color="warning" style={{ marginBottom: 12 }}>
                该服务器暂无已启用的文件类日志源
              </Tag>
            )}
            <Form.Item noStyle shouldUpdate={(prev, cur) => prev.auto_from_log_sources !== cur.auto_from_log_sources}>
              {({ getFieldValue }) =>
                getFieldValue("auto_from_log_sources") === false ? (
                  <Form.Item name="log_paths" label="手动日志路径（每行一条 glob）">
                    <Input.TextArea rows={3} placeholder="/var/log/messages" />
                  </Form.Item>
                ) : null
              }
            </Form.Item>
            <Space wrap style={{ width: "100%" }}>
              <Form.Item name="monitor_port" label="监控端口">
                <InputNumber min={1} max={65535} style={{ width: 140 }} />
              </Form.Item>
              <Form.Item name="deploy_dir" label="远端部署目录">
                <Input style={{ width: 200 }} placeholder="/export/loggie" />
              </Form.Item>
            </Space>
            <Form.Item
              name="yunshu_url"
              label="Yunshu 后端 API 地址"
              rules={[{ required: true }]}
              extra="填后端地址（如 http://IP:8080），不是前端 Vite 端口；供目标机心跳上报使用"
            >
              <Input placeholder="http://10.10.10.103:8080" />
            </Form.Item>
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 12 }}
              message="安装使用离线包"
              description="平台从 deploy/loggie/binary/loggie 上传到目标机 /export/loggie/loggie，不再在线下载。"
            />
            <Form.Item name="deploy_after_bootstrap" valuePropName="checked">
              <Checkbox>
                <CloudUploadOutlined /> 引导后仅热更配置（完整安装请用「安装」；会同步重启心跳）
              </Checkbox>
            </Form.Item>
          </Form>
        ) : (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            <Space wrap>
              <Tag color="blue">Token 已生成</Tag>
              <Tag>索引 {bootstrapResult.es_index_pattern}</Tag>
              <Tag>Pipeline {bootstrapResult.pipeline_count ?? 0}</Tag>
              {bootstrapResult.deployed ? <Tag color="success">已下发</Tag> : null}
            </Space>
            {bootstrapResult.deploy_message ? (
              <Alert
                type={bootstrapResult.deployed ? "success" : "warning"}
                showIcon
                style={{ maxWidth: "100%" }}
                message={bootstrapResult.deployed ? "配置已下发" : "配置已生成；systemd 未安装时请点「一键安装」"}
                description={
                  <div className="loggie-deploy-msg" title={bootstrapResult.deploy_message}>
                    {bootstrapResult.deploy_message}
                  </div>
                }
              />
            ) : null}
            <Input.TextArea
              value={bootstrapResult.pipelines_only_yaml || bootstrapResult.pipeline_yaml}
              rows={14}
              readOnly
              style={{ maxWidth: "100%", fontFamily: "monospace", fontSize: 12 }}
            />
          </Space>
        )}
      </Modal>
    </div>
  );
}
