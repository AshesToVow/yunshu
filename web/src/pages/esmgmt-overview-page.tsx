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

export function EsmgmtOverviewPage() {
  const [connections, setConnections] = useState<EsmgmtConnection[]>([]);
  const [connectionId, setConnectionId] = useState<number>();
  const [health, setHealth] = useState<Record<string, unknown> | null>(null);
  const [indices, setIndices] = useState<Array<{ name: string; store_bytes?: number; docs_count?: number }>>([]);
  const [nodes, setNodes] = useState<Record<string, unknown>[]>([]);
  const [backups, setBackups] = useState<EsmgmtBackupJob[]>([]);
  const [restores, setRestores] = useState<EsmgmtRestoreJob[]>([]);
  const [schedules, setSchedules] = useState<EsmgmtBackupSchedule[]>([]);
  const [loading, setLoading] = useState(false);
  const [backingUp, setBackingUp] = useState<string | null>(null);
  const [scheduleOpen, setScheduleOpen] = useState(false);
  const [scheduleForm] = Form.useForm();

  useEffect(() => {
    void listEsmgmtConnections()
      .then((list) => {
        setConnections(list || []);
        const def = list?.find((c) => c.is_default) || list?.[0];
        if (def) setConnectionId(def.id);
      })
      .catch(() => undefined);
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
    } catch {
      /* ignore */
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

  async function onRestore(job: EsmgmtBackupJob, deleteExisting: boolean) {
    try {
      const r = await createEsmgmtRestore({
        backup_job_id: job.id,
        connection_id: connectionId,
        target_index: job.index_name,
        delete_existing: deleteExisting,
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

  function jobStatusColor(s: string) {
    if (s === "success") return "success";
    if (s === "failed") return "error";
    if (s === "running" || s === "pending") return "processing";
    return "default";
  }

  return (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      <Card size="small">
        <Space wrap>
          <Select
            style={{ minWidth: 240 }}
            placeholder="选择连接"
            value={connectionId}
            options={connections.map((c) => ({ value: c.id, label: c.name }))}
            onChange={setConnectionId}
            allowClear
          />
          <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>
            刷新
          </Button>
          {status ? <Tag color={statusColor}>{status}</Tag> : null}
        </Space>
      </Card>
      {health ? (
        <Alert
          type={statusColor === "success" ? "success" : statusColor === "warning" ? "warning" : "info"}
          showIcon
          message={`集群 ${String(health.cluster_name || "-")} · 节点 ${String(health.number_of_nodes ?? "-")} · 分片 ${String(health.active_shards ?? "-")}`}
        />
      ) : null}
      <Card title="索引" size="small">
        <Table
          size="small"
          rowKey="name"
          loading={loading}
          dataSource={indices}
          pagination={{ pageSize: 10 }}
          columns={[
            { title: "索引", dataIndex: "name", ellipsis: true },
            { title: "文档数", dataIndex: "docs_count", width: 100 },
            { title: "存储字节", dataIndex: "store_bytes", width: 120 },
            {
              title: "操作",
              width: 280,
              render: (_: unknown, row: { name: string }) => (
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
                    title={row.name.includes("yunshu-agent") ? "日志索引，需强制删除确认" : "确认删除索引？"}
                    onConfirm={() =>
                      void deleteEsmgmtIndex(row.name, row.name.includes("yunshu-agent"), connectionId).then(load)
                    }
                  >
                    <Button type="link" size="small" danger>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              ),
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
            { title: "触发", dataIndex: "trigger", width: 90 },
            {
              title: "状态",
              dataIndex: "status",
              width: 100,
              render: (v: string) => <Tag color={jobStatusColor(v)}>{v}</Tag>,
            },
            { title: "阶段", dataIndex: "phase", width: 90 },
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
                  <Popconfirm
                    title="覆盖同名索引并恢复？"
                    disabled={row.status !== "success"}
                    onConfirm={() => void onRestore(row, true)}
                  >
                    <Button type="link" size="small" disabled={row.status !== "success"}>
                      恢复
                    </Button>
                  </Popconfirm>
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
              render: (v: string) => <Tag color={jobStatusColor(v)}>{v}</Tag>,
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
    </Space>
  );
}

export default EsmgmtOverviewPage;
