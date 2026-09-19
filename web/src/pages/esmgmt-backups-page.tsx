import { CloudDownloadOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import {
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
  Tabs,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  createEsmgmtRestore,
  createEsmgmtSchedule,
  deleteEsmgmtSchedule,
  downloadEsmgmtBackup,
  listEsmgmtBackups,
  listEsmgmtConnections,
  listEsmgmtIndices,
  listEsmgmtRestores,
  listEsmgmtSchedules,
  updateEsmgmtSchedule,
  type EsmgmtBackupJob,
  type EsmgmtBackupSchedule,
  type EsmgmtConnection,
  type EsmgmtRestoreJob,
} from "../services/esmgmt";
import { extractApiErrorMessage } from "../services/http";

const TABS = ["backups", "restores", "schedules"] as const;

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

export function EsmgmtBackupsPage() {
  const [params, setParams] = useSearchParams();
  const tabParam = params.get("tab") || "backups";
  const tab = TABS.includes(tabParam as (typeof TABS)[number]) ? tabParam : "backups";

  const [connections, setConnections] = useState<EsmgmtConnection[]>([]);
  const [connectionId, setConnectionId] = useState<number>();
  const [indices, setIndices] = useState<Array<{ name: string }>>([]);
  const [backups, setBackups] = useState<EsmgmtBackupJob[]>([]);
  const [restores, setRestores] = useState<EsmgmtRestoreJob[]>([]);
  const [schedules, setSchedules] = useState<EsmgmtBackupSchedule[]>([]);
  const [loading, setLoading] = useState(false);
  const [scheduleOpen, setScheduleOpen] = useState(false);
  const [restoreOpen, setRestoreOpen] = useState(false);
  const [restoreJob, setRestoreJob] = useState<EsmgmtBackupJob | null>(null);
  const [restoreConfirm, setRestoreConfirm] = useState("");
  const [restoreSubmitting, setRestoreSubmitting] = useState(false);
  const [scheduleForm] = Form.useForm();

  useEffect(() => {
    void listEsmgmtConnections()
      .then((list) => {
        setConnections(list || []);
        const def = list?.find((c) => c.is_default) || list?.[0];
        if (def) setConnectionId(def.id);
      })
      .catch((e) => message.error(extractApiErrorMessage(e, "加载连接失败")));
  }, []);

  async function load() {
    setLoading(true);
    try {
      const [b, r, s, idx] = await Promise.all([
        listEsmgmtBackups({ connection_id: connectionId, limit: 50 }),
        listEsmgmtRestores({ connection_id: connectionId, limit: 30 }),
        listEsmgmtSchedules(connectionId),
        listEsmgmtIndices({ connection_id: connectionId }),
      ]);
      setBackups(b || []);
      setRestores(r || []);
      setSchedules(s || []);
      setIndices(idx || []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载备份数据失败"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [connectionId]);

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
    if (!confirm || confirm !== targetIndex) {
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
      setParams({ tab: "restores" });
      await load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建恢复失败"));
    } finally {
      setRestoreSubmitting(false);
    }
  }

  async function onRestore(job: EsmgmtBackupJob) {
    try {
      const r = await createEsmgmtRestore({
        backup_job_id: job.id,
        connection_id: connectionId,
        target_index: job.index_name,
        delete_existing: false,
      });
      message.success(`恢复任务已创建 #${r.id}`);
      setParams({ tab: "restores" });
      await load();
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
      await load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建调度失败"));
    }
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
          <Link to="/esmgmt/overview">返回集群概览</Link>
        </Space>
      </Card>
      <Tabs
        activeKey={tab}
        onChange={(key) => setParams({ tab: key })}
        items={[
          {
            key: "backups",
            label: "备份任务",
            children: (
              <Table
                size="small"
                rowKey="id"
                loading={loading}
                dataSource={backups}
                pagination={{ pageSize: 10, showTotal: (t) => `共 ${t} 条` }}
                locale={{ emptyText: "暂无备份记录。可在集群概览的索引行发起备份。" }}
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
                        <Button type="link" size="small" disabled={row.status !== "success"} onClick={() => void onRestore(row)}>
                          恢复
                        </Button>
                        <Button type="link" size="small" danger disabled={row.status !== "success"} onClick={() => openRestoreModal(row)}>
                          覆盖恢复
                        </Button>
                      </Space>
                    ),
                  },
                  { title: "错误", dataIndex: "error_message", ellipsis: true, render: (v: string) => v || "—" },
                ]}
              />
            ),
          },
          {
            key: "restores",
            label: "恢复任务",
            children: (
              <Table
                size="small"
                rowKey="id"
                loading={loading}
                dataSource={restores}
                pagination={{ pageSize: 10, showTotal: (t) => `共 ${t} 条` }}
                locale={{ emptyText: "暂无恢复记录" }}
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
                  { title: "错误", dataIndex: "error_message", ellipsis: true, render: (v: string) => v || "—" },
                ]}
              />
            ),
          },
          {
            key: "schedules",
            label: "定时规则",
            children: (
              <Space direction="vertical" style={{ width: "100%" }}>
                <Button
                  type="primary"
                  size="small"
                  icon={<PlusOutlined />}
                  disabled={!connectionId}
                  onClick={() => {
                    scheduleForm.resetFields();
                    scheduleForm.setFieldsValue({ cron_spec: "0 2 * * *", enabled: true, max_docs: 0 });
                    setScheduleOpen(true);
                  }}
                >
                  新建定时备份
                </Button>
                <Table
                  size="small"
                  rowKey="id"
                  loading={loading}
                  dataSource={schedules}
                  pagination={{ pageSize: 10 }}
                  locale={{ emptyText: "暂无定时规则" }}
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
                            void updateEsmgmtSchedule(row.id, { enabled: checked })
                              .then(load)
                              .catch((e) => message.error(extractApiErrorMessage(e, "更新失败")))
                          }
                        />
                      ),
                    },
                    { title: "上次调度", dataIndex: "last_scheduled_at", width: 170, render: (v: string) => v || "—" },
                    {
                      title: "操作",
                      width: 80,
                      render: (_: unknown, row: EsmgmtBackupSchedule) => (
                        <Popconfirm title="删除该调度？" onConfirm={() => void deleteEsmgmtSchedule(row.id).then(load)}>
                          <Button type="link" size="small" danger>
                            删除
                          </Button>
                        </Popconfirm>
                      ),
                    },
                  ]}
                />
              </Space>
            ),
          },
        ]}
      />
      <Modal title="新建定时备份" open={scheduleOpen} onCancel={() => setScheduleOpen(false)} onOk={() => void onCreateSchedule()} destroyOnClose>
        <Form form={scheduleForm} layout="vertical">
          <Form.Item name="index_name" label="索引名" rules={[{ required: true }]}>
            <Select showSearch options={indices.map((i) => ({ value: i.name, label: i.name }))} placeholder="选择索引" />
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
        title="覆盖恢复"
        open={restoreOpen}
        onCancel={() => {
          setRestoreOpen(false);
          setRestoreJob(null);
          setRestoreConfirm("");
        }}
        onOk={() => void submitRestore()}
        okText="确认覆盖恢复"
        okButtonProps={{ danger: true, disabled: !restoreJob || restoreConfirm.trim() !== restoreJob.index_name }}
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
              style={{ marginTop: 8 }}
            />
          </div>
        </Space>
      </Modal>
    </Space>
  );
}

export default EsmgmtBackupsPage;
