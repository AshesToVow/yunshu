// @ts-nocheck
import { CloudDownloadOutlined, CloudUploadOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useState } from "react";
import {
  closeEsmgmtIndex,
  createEsmgmtBackup,
  createEsmgmtIndex,
  createEsmgmtRestore,
  createEsmgmtSchedule,
  deleteEsmgmtIndex,
  deleteEsmgmtSchedule,
  downloadEsmgmtBackup,
  getEsmgmtClusterHealth,
  listEsmgmtBackups,
  listEsmgmtConnections,
  listEsmgmtIndices,
  listEsmgmtNodes,
  listEsmgmtRestores,
  listEsmgmtSchedules,
  openEsmgmtIndex,
  updateEsmgmtSchedule,
  type EsmgmtBackupJob,
  type EsmgmtBackupSchedule,
  type EsmgmtConnection,
  type EsmgmtRestoreJob,
} from "../services/esmgmt";
import { extractApiErrorMessage } from "../services/http";

const DEFAULT_CREATE_BODY = `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 1
  },
  "mappings": {
    "properties": {
      "message": { "type": "text" },
      "@timestamp": { "type": "date" }
    }
  }
}`;

export function EsmgmtOverviewPage() {
  const [connections, setConnections] = useState<EsmgmtConnection[]>([]);
  const [connectionId, setConnectionId] = useState<number>();
  const [health, setHealth] = useState<Record<string, unknown> | null>(null);
  const [indices, setIndices] = useState<Array<{ name: string; store_bytes?: number; docs_count?: number }>>([]);
  const [nodes, setNodes] = useState<Record<string, unknown>[]>([]);
  const [backups, setBackups] = useState<EsmgmtBackupJob[]>([]);
  const [restores, setRestores] = useState<EsmgmtRestoreJob[]>([]);
  const [schedules, setSchedules] = useState<EsmgmtBackupSchedule[]>([]);
  const [indexFilter, setIndexFilter] = useState("");
  const [loading, setLoading] = useState(false);
  const [backingUp, setBackingUp] = useState<string | null>(null);
  const [scheduleOpen, setScheduleOpen] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [restoreOpen, setRestoreOpen] = useState(false);
  const [restoreJob, setRestoreJob] = useState<EsmgmtBackupJob | null>(null);
  const [restoreConfirm, setRestoreConfirm] = useState("");
  const [restoreSubmitting, setRestoreSubmitting] = useState(false);
  const [scheduleForm] = Form.useForm();
  const [createForm] = Form.useForm();

  useEffect(() => {
    void listEsmgmtConnections()
      .then((list) => {
        setConnections(list || []);
        const def = list?.find((c) => c.is_default) || list?.[0];
        if (def) setConnectionId(def.id);
      })
      .catch((e) => message.error(extractApiErrorMessage(e, "加载连接失败")));
  }, []);

  async function loadJobs() {
    try {
      const [b, r, s] = await Promise.all([
        listEsmgmtBackups({ connection_id: connectionId, limit: 30 }),
        listEsmgmtRestores({ connection_id: connectionId, limit: 20 }),
        listEsmgmtSchedules(connectionId),
      ]);
      setBackups(b || []);
      setRestores(r || []);
      setSchedules(s || []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载任务列表失败"));
    }
  }

  async function load() {
    setLoading(true);
    try {
      const [h, idx, nd] = await Promise.all([
        getEsmgmtClusterHealth(connectionId),
        listEsmgmtIndices({ connection_id: connectionId }),
        listEsmgmtNodes(connectionId),
      ]);
      setHealth(h);
      setIndices(idx || []);
      setNodes(nd || []);
      await loadJobs();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载集群信息失败"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [connectionId]);

  async function onBackup(indexName: string) {
    setBackingUp(indexName);
    try {
      const job = await createEsmgmtBackup({ connection_id: connectionId, index: indexName });
      message.success(`备份任务已创建 #${job.id}`);
      await loadJobs();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建备份失败"));
    } finally {
      setBackingUp(null);
    }
  }

  async function onCreateIndex() {
    const values = await createForm.validateFields();
    let settings: Record<string, unknown> | undefined;
    let mappings: Record<string, unknown> | undefined;
    const raw = String(values.body_json || "").trim();
    if (raw) {
      try {
        const parsed = JSON.parse(raw) as Record<string, unknown>;
        if (parsed.settings && typeof parsed.settings === "object") {
          settings = parsed.settings as Record<string, unknown>;
        }
        if (parsed.mappings && typeof parsed.mappings === "object") {
          mappings = parsed.mappings as Record<string, unknown>;
        }
        // 允许直接把整段当 settings（无 settings/mappings 键时忽略）
        if (!settings && !mappings && (parsed.number_of_shards != null || parsed.index != null)) {
          settings = parsed;
        }
      } catch {
        message.error("Body JSON 无法解析");
        return;
      }
    }
    setCreating(true);
    try {
      await createEsmgmtIndex({
        connection_id: connectionId,
        name: String(values.name).trim(),
        settings,
        mappings,
      });
      message.success("索引已创建");
      setCreateOpen(false);
      void load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建索引失败"));
    } finally {
      setCreating(false);
    }
  }

  async function onDownload(job: EsmgmtBackupJob) {
    try {
      const res = await downloadEsmgmtBackup(job.id, "zip");
      if (res?.url) {
        window.open(res.url, "_blank");
      } else {
        message.error("未取得下载链接");
      }
    } catch (e) {
      message.error(extractApiErrorMessage(e, "获取下载链接失败"));
    }
  }

  function openRestoreModal(job: EsmgmtBackupJob) {
    setRestoreJob(job);
    setRestoreConfirm("");
    setRestoreOpen(true);
  }

  async function submitRestore() {
    if (!restoreJob) return;
    const targetIndex = restoreJob.index_name;
    const confirm = restoreConfirm.trim();
    if (!confirm) {
      message.error("请输入目标索引名以确认");
      return;
    }
    if (confirm !== targetIndex) {
      message.error("输入的索引名与目标不一致");
      return;
    }
    setRestoreSubmitting(true);
    try {
      const r = await createEsmgmtRestore({
        backup_job_id: restoreJob.id,
        connection_id: connectionId,
        target_index: targetIndex,
        delete_existing: true,
        confirm_target_index: confirm,
      });
      message.success(`恢复任务已创建 #${r.id}`);
      setRestoreOpen(false);
      setRestoreJob(null);
      setRestoreConfirm("");
      await loadJobs();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建恢复失败"));
    } finally {
      setRestoreSubmitting(false);
    }
  }

  async function onRestore(job: EsmgmtBackupJob, deleteExisting: boolean) {
    if (deleteExisting) {
      openRestoreModal(job);
      return;
    }
    const targetIndex = job.index_name;
    try {
      const r = await createEsmgmtRestore({
        backup_job_id: job.id,
        connection_id: connectionId,
        target_index: targetIndex,
        delete_existing: false,
      });
      message.success(`恢复任务已创建 #${r.id}`);
      await loadJobs();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建恢复失败"));
    }
  }

  async function onCreateSchedule() {
    const values = await scheduleForm.validateFields();
    try {
      await createEsmgmtSchedule({
        connection_id: connectionId,
        index_name: values.index_name,
        cron_spec: values.cron_spec,
        max_docs: values.max_docs || 0,
        enabled: values.enabled ?? true,
        remark: values.remark,
      });
      message.success("调度已创建");
      setScheduleOpen(false);
      await loadJobs();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建调度失败"));
    }
  }

  const status = String(health?.status || "");
  const statusColor = status === "green" ? "success" : status === "yellow" ? "warning" : status ? "error" : "default";
  const statusLabel =
    status === "green" ? "健康" : status === "yellow" ? "降级" : status === "red" ? "异常" : status || "-";

  function jobStatusColor(s: string) {
    if (s === "success") return "success";
    if (s === "failed") return "error";
    if (s === "running" || s === "pending") return "processing";
    return "default";
  }
  function jobStatusLabel(s: string) {
    const map: Record<string, string> = {
      success: "成功",
      failed: "失败",
      running: "执行中",
      pending: "排队中",
      manual: "手动",
      cron: "定时",
    };
    return map[s] || s || "-";
  }

  return (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      <Card size="small">
        <Space wrap>
          <Select
            style={{ minWidth: 360 }}
            placeholder="选择连接"
            value={connectionId}
            options={connections.map((c) => ({
              value: c.id,
              label: `${c.name}${c.is_default ? "（默认）" : ""}（${c.addresses || "-"}）`,
            }))}
            onChange={setConnectionId}
          />
          <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>
            刷新
          </Button>
          {status ? <Tag color={statusColor}>{statusLabel}</Tag> : null}
        </Space>
      </Card>
      {health ? (
        <Alert
          type={statusColor === "success" ? "success" : statusColor === "warning" ? "warning" : "info"}
          showIcon
          message={`集群 ${String(health.cluster_name || "-")} · 节点 ${String(health.number_of_nodes ?? "-")} · 分片 ${String(health.active_shards ?? "-")}`}
        />
      ) : null}
      <Card
        title="索引"
        size="small"
        extra={
          <Space wrap>
            <Input
              allowClear
              size="small"
              placeholder="过滤索引名，如 yunshu-k8s"
              style={{ width: 220 }}
              value={indexFilter}
              onChange={(e) => setIndexFilter(e.target.value)}
            />
            <Button
              size="small"
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                createForm.resetFields();
                createForm.setFieldsValue({ body_json: DEFAULT_CREATE_BODY });
                setCreateOpen(true);
              }}
            >
              新建索引
            </Button>
          </Space>
        }
      >
        <Table
          size="small"
          rowKey="name"
          loading={loading}
          dataSource={indices.filter((i) =>
            !indexFilter.trim() ? true : i.name.toLowerCase().includes(indexFilter.trim().toLowerCase()),
          )}
          pagination={{ pageSize: 20, showSizeChanger: true }}
          columns={[
            { title: "索引", dataIndex: "name", ellipsis: true },
            { title: "文档数", dataIndex: "docs_count", width: 100 },
            { title: "存储字节", dataIndex: "store_bytes", width: 120 },
            {
              title: "操作",
              width: 280,
              render: (_: unknown, row?: { name: string }) =>
                row ? (
                <Space wrap>
                  <Button
                    type="link"
                    size="small"
                    icon={<CloudUploadOutlined />}
                    loading={backingUp === row.name}
                    onClick={() => void onBackup(row.name)}
                  >
                    备份
                  </Button>
                  <Button type="link" size="small" onClick={() => void openEsmgmtIndex(row.name, connectionId).then(load)}>
                    打开
                  </Button>
                  <Button type="link" size="small" onClick={() => void closeEsmgmtIndex(row.name, connectionId).then(load)}>
                    关闭
                  </Button>
                  <Popconfirm
                    title={
                      row.name.includes("yunshu-agent") || row.name.includes("yunshu-k8s")
                        ? "日志索引，需强制删除确认"
                        : "确认删除索引？"
                    }
                    onConfirm={() =>
                      void deleteEsmgmtIndex(
                        row.name,
                        row.name.includes("yunshu-agent") || row.name.includes("yunshu-k8s"),
                        connectionId,
                      ).then(load)
                    }
                  >
                    <Button type="link" size="small" danger>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
                ) : null,
            },
          ]}
        />
      </Card>
      <Card
        title="备份记录"
        size="small"
        extra={
          <Button size="small" icon={<ReloadOutlined />} onClick={() => void loadJobs()}>
            刷新
          </Button>
        }
      >
        <Table
          size="small"
          rowKey="id"
          dataSource={backups}
          pagination={{ pageSize: 8 }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "索引", dataIndex: "index_name", ellipsis: true },
            { title: "触发", dataIndex: "trigger", width: 90, render: (v: string) => jobStatusLabel(v) },
            {
              title: "状态",
              dataIndex: "status",
              width: 100,
              render: (v: string) => <Tag color={jobStatusColor(v)}>{jobStatusLabel(v)}</Tag>,
            },
            { title: "阶段", dataIndex: "phase", width: 90, render: (v: string) => jobStatusLabel(v) },
            { title: "文档数", dataIndex: "doc_count", width: 80 },
            {
              title: "操作",
              width: 200,
              render: (_: unknown, row: EsmgmtBackupJob) => (
                <Space wrap>
                  <Button
                    type="link"
                    size="small"
                    icon={<CloudDownloadOutlined />}
                    disabled={row.status !== "success"}
                    onClick={() => void onDownload(row)}
                  >
                    下载
                  </Button>
                  <Button
                    type="link"
                    size="small"
                    disabled={row.status !== "success"}
                    onClick={() => void onRestore(row, false)}
                  >
                    恢复
                  </Button>
                  <Button
                    type="link"
                    size="small"
                    danger
                    disabled={row.status !== "success"}
                    onClick={() => openRestoreModal(row)}
                  >
                    覆盖恢复
                  </Button>
                </Space>
              ),
            },
            {
              title: "错误",
              dataIndex: "error_message",
              ellipsis: true,
              render: (v: string) => v || "—",
            },
          ]}
        />
        {!backups.length ? <Typography.Text type="secondary">暂无备份记录</Typography.Text> : null}
      </Card>
      <Card title="恢复记录" size="small">
        <Table
          size="small"
          rowKey="id"
          dataSource={restores}
          pagination={{ pageSize: 6 }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "备份ID", dataIndex: "backup_job_id", width: 80 },
            { title: "目标索引", dataIndex: "target_index", ellipsis: true },
            {
              title: "状态",
              dataIndex: "status",
              width: 100,
              render: (v: string) => <Tag color={jobStatusColor(v)}>{jobStatusLabel(v)}</Tag>,
            },
            { title: "阶段", dataIndex: "phase", width: 100 },
            { title: "文档数", dataIndex: "doc_count", width: 80 },
            {
              title: "错误",
              dataIndex: "error_message",
              ellipsis: true,
              render: (v: string) => v || "—",
            },
          ]}
        />
        {!restores.length ? <Typography.Text type="secondary">暂无恢复记录</Typography.Text> : null}
      </Card>
      <Card
        title="定时备份"
        size="small"
        extra={
          <Button
            size="small"
            type="primary"
            icon={<PlusOutlined />}
            disabled={!connectionId}
            onClick={() => {
              scheduleForm.resetFields();
              scheduleForm.setFieldsValue({ cron_spec: "0 2 * * *", enabled: true, max_docs: 0 });
              setScheduleOpen(true);
            }}
          >
            新建
          </Button>
        }
      >
        <Table
          size="small"
          rowKey="id"
          dataSource={schedules}
          pagination={{ pageSize: 6 }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "索引", dataIndex: "index_name", ellipsis: true },
            { title: "Cron", dataIndex: "cron_spec", width: 140 },
            {
              title: "启用",
              dataIndex: "enabled",
              width: 80,
              render: (v: boolean, row: EsmgmtBackupSchedule) => (
                <Switch
                  size="small"
                  checked={v}
                  onChange={(checked) =>
                    void updateEsmgmtSchedule(row.id, { enabled: checked }).then(loadJobs).catch((e) => {
                      message.error(extractApiErrorMessage(e, "更新失败"));
                    })
                  }
                />
              ),
            },
            { title: "上次调度", dataIndex: "last_scheduled_at", width: 170, render: (v: string) => v || "—" },
            {
              title: "操作",
              width: 80,
              render: (_: unknown, row: EsmgmtBackupSchedule) => (
                <Popconfirm title="删除该调度？" onConfirm={() => void deleteEsmgmtSchedule(row.id).then(loadJobs)}>
                  <Button type="link" size="small" danger>
                    删除
                  </Button>
                </Popconfirm>
              ),
            },
          ]}
        />
        {!schedules.length ? <Typography.Text type="secondary">暂无定时规则</Typography.Text> : null}
      </Card>
      <Card title="节点" size="small">
        <Table
          size="small"
          rowKey={(r) => String(r.name || r.ip)}
          loading={loading}
          dataSource={nodes}
          pagination={{ pageSize: 8 }}
          columns={[
            { title: "名称", dataIndex: "name" },
            { title: "IP", dataIndex: "ip", width: 140 },
            { title: "角色", dataIndex: "node.role", width: 100 },
            { title: "CPU", dataIndex: "cpu", width: 80 },
            { title: "Heap%", dataIndex: "heap.percent", width: 90 },
          ]}
        />
        {!nodes.length && !loading ? <Typography.Text type="secondary">暂无节点数据</Typography.Text> : null}
      </Card>
      <Modal
        title="新建定时备份"
        open={scheduleOpen}
        onCancel={() => setScheduleOpen(false)}
        onOk={() => void onCreateSchedule()}
        destroyOnClose
      >
        <Form form={scheduleForm} layout="vertical">
          <Form.Item name="index_name" label="索引名" rules={[{ required: true }]}>
            <Select
              showSearch
              options={indices.map((i) => ({ value: i.name, label: i.name }))}
              placeholder="选择或输入索引"
            />
          </Form.Item>
          <Form.Item name="cron_spec" label="Cron（支持五/六段）" rules={[{ required: true }]}>
            <Input placeholder="0 2 * * *" />
          </Form.Item>
          <Form.Item name="max_docs" label="最大文档数（0=默认）">
            <InputNumber min={0} max={200000} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title="新建索引"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => void onCreateIndex()}
        confirmLoading={creating}
        width={640}
        destroyOnClose
      >
        <Form form={createForm} layout="vertical">
          <Form.Item
            name="name"
            label="索引名"
            rules={[
              { required: true, message: "请输入索引名" },
              {
                validator: async (_, v) => {
                  const name = String(v || "").trim();
                  if (!name) return;
                  if (name.startsWith(".")) throw new Error("禁止系统索引名");
                  if (/[\\/?*"<>|,#\s]/.test(name)) throw new Error("索引名含非法字符");
                },
              },
            ]}
          >
            <Input placeholder="例如 my-app-logs" />
          </Form.Item>
          <Form.Item
            name="body_json"
            label="settings / mappings（JSON，可选）"
            extra="可填 { settings, mappings }；留空则按 ES 默认创建空索引"
          >
            <Input.TextArea rows={14} style={{ fontFamily: "monospace" }} />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title="覆盖恢复"
        open={restoreOpen}
        onCancel={() => {
          setRestoreOpen(false);
          setRestoreJob(null);
          setRestoreConfirm("");
        }}
        onOk={() => void submitRestore()}
        okText="确认覆盖恢复"
        okButtonProps={{
          danger: true,
          disabled: !restoreJob || restoreConfirm.trim() !== restoreJob.index_name,
        }}
        confirmLoading={restoreSubmitting}
        destroyOnClose
      >
        <Space direction="vertical" style={{ width: "100%" }} size="middle">
          <Typography.Text>
            将删除索引「{restoreJob?.index_name}」现有数据并从备份 #{restoreJob?.id} 恢复，此操作不可撤销。
          </Typography.Text>
          <div>
            <Typography.Text type="secondary">请输入目标索引名「{restoreJob?.index_name}」以确认：</Typography.Text>
            <Input
              value={restoreConfirm}
              placeholder={restoreJob?.index_name}
              onChange={(e) => setRestoreConfirm(e.target.value)}
              onPressEnter={() => void submitRestore()}
              style={{ marginTop: 8 }}
            />
          </div>
        </Space>
      </Modal>
    </Space>
  );
}

export default EsmgmtOverviewPage;
