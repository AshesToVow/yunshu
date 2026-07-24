import {
  FileTextOutlined,
  MailOutlined,
  PlusOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";
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
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { listAlertDatasources, type AlertDatasourceItem } from "../services/alert-platform";
import {
  createInspectItem,
  deleteInspectItem,
  getInspectPlan,
  inspectReportHtmlUrl,
  inspectReportPdfUrl,
  listInspectItems,
  listInspectRuns,
  resendInspectEmail,
  startInspectRun,
  syncInspectItems,
  updateInspectItem,
  updateInspectPlan,
  type InspectItem,
  type InspectPlan,
  type InspectRun,
} from "../services/inspect";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";
import { http } from "../services/http";

const CRON_PRESETS = [
  { label: "每天 09:00", value: "0 0 9 * * *" },
  { label: "每天 02:00", value: "0 0 2 * * *" },
  { label: "每周一 09:00", value: "0 0 9 * * 1" },
];

function parseRecipients(raw?: string): string[] {
  if (!raw) return [];
  try {
    const v = JSON.parse(raw);
    return Array.isArray(v) ? v.map(String) : [];
  } catch {
    return raw.split(",").map((s) => s.trim()).filter(Boolean);
  }
}

function openAuthorized(url: string) {
  void http
    .get(url, { responseType: "blob" })
    .then((data) => {
      const type = url.endsWith(".pdf") ? "application/pdf" : "text/html;charset=utf-8";
      const blob = data instanceof Blob ? data : new Blob([data as BlobPart], { type });
      const obj = URL.createObjectURL(blob.type ? blob : new Blob([blob], { type }));
      window.open(obj, "_blank");
      setTimeout(() => URL.revokeObjectURL(obj), 60_000);
    })
    .catch(() => message.error("打开报告失败"));
}

export function ProjectInspectPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>(0);
  const [plan, setPlan] = useState<InspectPlan | null>(null);
  const [items, setItems] = useState<InspectItem[]>([]);
  const [runs, setRuns] = useState<InspectRun[]>([]);
  const [runTotal, setRunTotal] = useState(0);
  const [dsList, setDsList] = useState<AlertDatasourceItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState(false);
  const [itemModalOpen, setItemModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<InspectItem | null>(null);
  const [planForm] = Form.useForm();
  const [itemForm] = Form.useForm();

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((r) => {
      const list = r.list || [];
      setProjects(list);
      const fromQuery = Number(searchParams.get("project_id") || 0);
      if (fromQuery > 0 && list.some((p) => p.id === fromQuery)) {
        setProjectId(fromQuery);
      } else if (list.length > 0) {
        setProjectId(list[0].id);
      }
    });
  }, [searchParams]);

  const refresh = useCallback(async (pid: number) => {
    if (!pid) return;
    setLoading(true);
    try {
      const [p, its, rs, ds] = await Promise.all([
        getInspectPlan(pid),
        listInspectItems(pid),
        listInspectRuns(pid, { page: 1, page_size: 20 }),
        listAlertDatasources({ project_id: pid, page: 1, page_size: 200 }),
      ]);
      setPlan(p);
      setItems(its || []);
      setRuns(rs.list || []);
      setRunTotal(rs.total || 0);
      setDsList((ds.list || []).filter((d) => d.enabled !== false));
      planForm.setFieldsValue({
        enabled: p.enabled,
        cron_spec: p.cron_spec,
        datasource_id: p.datasource_id || undefined,
        report_list_mode: p.report_list_mode || "abnormal_only",
        recipients: parseRecipients(p.recipients_json).join(", "),
      });
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, [planForm]);

  useEffect(() => {
    if (projectId > 0) {
      setSearchParams({ project_id: String(projectId) }, { replace: true });
      void refresh(projectId);
    }
  }, [projectId, refresh, setSearchParams]);

  const itemColumns: ColumnsType<InspectItem> = useMemo(
    () => [
      { title: "分类", dataIndex: "type", width: 140 },
      { title: "名称", dataIndex: "name", width: 160 },
      {
        title: "来源",
        width: 90,
        render: (_, r) => (r.project_id === 0 ? <Tag>全局模板</Tag> : <Tag color="blue">项目</Tag>),
      },
      { title: "阈值", width: 100, render: (_, r) => `${r.threshold_type} ${r.threshold}${r.unit || ""}` },
      {
        title: "启用",
        width: 70,
        render: (_, r) => (r.enabled ? <Tag color="success">是</Tag> : <Tag>否</Tag>),
      },
      {
        title: "操作",
        width: 140,
        render: (_, r) =>
          r.project_id === 0 ? (
            <Typography.Text type="secondary">只读</Typography.Text>
          ) : (
            <Space>
              <Button
                type="link"
                size="small"
                onClick={() => {
                  setEditingItem(r);
                  itemForm.setFieldsValue(r);
                  setItemModalOpen(true);
                }}
              >
                编辑
              </Button>
              <Popconfirm
                title="删除该巡检项？"
                onConfirm={async () => {
                  await deleteInspectItem(projectId, r.id);
                  message.success("已删除");
                  void refresh(projectId);
                }}
              >
                <Button type="link" danger size="small">
                  删除
                </Button>
              </Popconfirm>
            </Space>
          ),
      },
    ],
    [itemForm, projectId, refresh],
  );

  const runColumns: ColumnsType<InspectRun> = [
    { title: "ID", dataIndex: "id", width: 70 },
    {
      title: "状态",
      dataIndex: "status",
      width: 90,
      render: (s: string) => {
        const color = s === "success" ? "success" : s === "failed" ? "error" : "processing";
        return <Tag color={color}>{s}</Tag>;
      },
    },
    { title: "触发", dataIndex: "trigger", width: 80 },
    { title: "分数", width: 90, render: (_, r) => `${r.score}${r.grade ? ` (${r.grade})` : ""}` },
    {
      title: "异常",
      width: 120,
      render: (_, r) => (
        <span>
          <Typography.Text type="danger">{r.critical_count}</Typography.Text> / {r.warning_count}
        </span>
      ),
    },
    { title: "时间", width: 170, render: (_, r) => formatDateTime(r.finished_at || r.started_at || r.created_at) },
    {
      title: "报告",
      width: 220,
      render: (_, r) => (
        <Space wrap>
          <Button
            type="link"
            size="small"
            icon={<FileTextOutlined />}
            onClick={() => openAuthorized(inspectReportHtmlUrl(projectId, r.id))}
          >
            HTML
          </Button>
          <Button type="link" size="small" onClick={() => openAuthorized(inspectReportPdfUrl(projectId, r.id))}>
            PDF
          </Button>
          <Button
            type="link"
            size="small"
            icon={<MailOutlined />}
            onClick={async () => {
              await resendInspectEmail(projectId, r.id);
              message.success("已触发重发");
            }}
          >
            重发邮件
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 16 }}>
      <Space style={{ marginBottom: 16 }} wrap>
        <Typography.Title level={4} style={{ margin: 0 }}>
          项目巡检
        </Typography.Title>
        <Select
          style={{ width: 240 }}
          value={projectId || undefined}
          placeholder="选择项目"
          options={projects.map((p) => ({ label: p.name, value: p.id }))}
          onChange={(v) => setProjectId(v)}
        />
        <Button icon={<ReloadOutlined />} onClick={() => void refresh(projectId)} loading={loading}>
          刷新
        </Button>
        <Button
          type="primary"
          icon={<ThunderboltOutlined />}
          loading={running}
          disabled={!projectId}
          onClick={async () => {
            setRunning(true);
            try {
              const dsId = planForm.getFieldValue("datasource_id") || plan?.datasource_id;
              await startInspectRun(projectId, dsId || undefined);
              message.success("巡检完成");
              void refresh(projectId);
            } catch (e: unknown) {
              message.error(e instanceof Error ? e.message : "执行失败");
            } finally {
              setRunning(false);
            }
          }}
        >
          立即巡检
        </Button>
      </Space>

      <Tabs
        items={[
          {
            key: "plan",
            label: "计划配置",
            children: (
              <Card loading={loading}>
                <Form
                  form={planForm}
                  layout="vertical"
                  onFinish={async (values) => {
                    const recipients = String(values.recipients || "")
                      .split(/[,;\s]+/)
                      .map((s: string) => s.trim())
                      .filter(Boolean);
                    await updateInspectPlan(projectId, {
                      enabled: values.enabled,
                      cron_spec: values.cron_spec,
                      datasource_id: values.datasource_id,
                      report_list_mode: values.report_list_mode,
                      recipients,
                    });
                    message.success("已保存");
                    void refresh(projectId);
                  }}
                >
                  <Form.Item name="enabled" label="启用定时巡检" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                  <Form.Item name="datasource_id" label="Prometheus 数据源" rules={[{ required: true, message: "请选择数据源" }]}>
                    <Select
                      allowClear
                      options={dsList.map((d) => ({ label: `${d.name} (#${d.id})`, value: d.id }))}
                      placeholder="选择项目内数据源"
                    />
                  </Form.Item>
                  <Form.Item name="cron_spec" label="Cron">
                    <Select
                      showSearch
                      allowClear
                      options={CRON_PRESETS}
                      placeholder="选择或输入 Cron"
                      dropdownRender={(menu) => (
                        <>
                          {menu}
                          <div style={{ padding: 8 }}>
                            <Input
                              placeholder="自定义 Cron"
                              onPressEnter={(e) => planForm.setFieldValue("cron_spec", (e.target as HTMLInputElement).value)}
                            />
                          </div>
                        </>
                      )}
                    />
                  </Form.Item>
                  <Form.Item name="report_list_mode" label="报告明细模式">
                    <Select
                      options={[
                        { label: "仅异常 (abnormal_only)", value: "abnormal_only" },
                        { label: "摘要 (summary)", value: "summary" },
                        { label: "全部 (all)", value: "all" },
                      ]}
                    />
                  </Form.Item>
                  <Form.Item name="recipients" label="邮件收件人（逗号分隔）">
                    <Input.TextArea rows={2} placeholder="ops@example.com, oncall@example.com" />
                  </Form.Item>
                  {plan?.last_run_at ? (
                    <Typography.Paragraph type="secondary">最近执行：{formatDateTime(plan.last_run_at)}</Typography.Paragraph>
                  ) : null}
                  <Button type="primary" htmlType="submit">
                    保存计划
                  </Button>
                </Form>
              </Card>
            ),
          },
          {
            key: "items",
            label: "巡检项",
            children: (
              <Card
                loading={loading}
                extra={
                  <Space>
                    <Button
                      onClick={async () => {
                        const r = await syncInspectItems(projectId);
                        message.success(`已同步 ${r.created} 项`);
                        void refresh(projectId);
                      }}
                    >
                      从模板同步
                    </Button>
                    <Button
                      type="primary"
                      icon={<PlusOutlined />}
                      onClick={() => {
                        setEditingItem(null);
                        itemForm.resetFields();
                        itemForm.setFieldsValue({
                          type: "自定义",
                          threshold: 80,
                          threshold_type: "greater",
                          enabled: true,
                        });
                        setItemModalOpen(true);
                      }}
                    >
                      新增
                    </Button>
                  </Space>
                }
              >
                <Table rowKey="id" size="small" columns={itemColumns} dataSource={items} pagination={{ pageSize: 20 }} />
              </Card>
            ),
          },
          {
            key: "runs",
            label: `历史 (${runTotal})`,
            children: (
              <Card loading={loading}>
                <Table rowKey="id" size="small" columns={runColumns} dataSource={runs} pagination={false} />
              </Card>
            ),
          },
        ]}
      />

      <Modal
        title={editingItem ? "编辑巡检项" : "新增巡检项"}
        open={itemModalOpen}
        onCancel={() => setItemModalOpen(false)}
        onOk={async () => {
          const values = await itemForm.validateFields();
          if (editingItem) {
            await updateInspectItem(projectId, editingItem.id, values);
          } else {
            await createInspectItem(projectId, values);
          }
          message.success("已保存");
          setItemModalOpen(false);
          void refresh(projectId);
        }}
        destroyOnClose
        width={720}
      >
        <Form form={itemForm} layout="vertical">
          <Form.Item name="type" label="分类" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="说明">
            <Input />
          </Form.Item>
          <Form.Item name="query" label="PromQL" rules={[{ required: true }]}>
            <Input.TextArea rows={3} />
          </Form.Item>
          <Space style={{ width: "100%" }} size="large">
            <Form.Item name="threshold_type" label="比较" rules={[{ required: true }]}>
              <Select
                style={{ width: 140 }}
                options={["greater", "greater_equal", "less", "less_equal", "equal", "not_equal"].map((v) => ({
                  label: v,
                  value: v,
                }))}
              />
            </Form.Item>
            <Form.Item name="threshold" label="阈值" rules={[{ required: true }]}>
              <InputNumber style={{ width: 120 }} />
            </Form.Item>
            <Form.Item name="unit" label="单位">
              <Input style={{ width: 80 }} />
            </Form.Item>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </div>
  );
}
