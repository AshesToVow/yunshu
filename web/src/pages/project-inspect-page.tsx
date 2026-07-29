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
  Alert,
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
  copyInspectReportTemplate,
  deleteInspectItem,
  deleteInspectReportTemplate,
  getInspectPlan,
  inspectReportExcelUrl,
  inspectReportHtmlUrl,
  inspectReportPdfUrl,
  inspectReportPrintUrl,
  listInspectItems,
  listInspectReportTemplates,
  listInspectRuns,
  previewInspectReportTemplate,
  resendInspectEmail,
  resetInspectItems,
  startInspectRun,
  syncInspectItems,
  updateInspectItem,
  updateInspectPlan,
  updateInspectReportTemplate,
  type InspectItem,
  type InspectPlan,
  type InspectReportTemplate,
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

function toReportBlob(raw: unknown, type: string): Blob {
  if (raw instanceof Blob) return raw;
  // http 拦截器会解包为 response.data；TS 仍可能标成 AxiosResponse
  if (raw && typeof raw === "object" && "data" in raw) {
    const inner = (raw as { data: unknown }).data;
    if (inner instanceof Blob) return inner;
    if (typeof inner === "string" || inner instanceof ArrayBuffer || ArrayBuffer.isView(inner)) {
      return new Blob([inner as BlobPart], { type });
    }
  }
  if (typeof raw === "string" || raw instanceof ArrayBuffer || ArrayBuffer.isView(raw)) {
    return new Blob([raw as BlobPart], { type });
  }
  return new Blob([], { type });
}

function openAuthorized(url: string) {
  void http
    .get(url, { responseType: "blob" })
    .then((raw: unknown) => {
      const type = url.endsWith(".pdf") ? "application/pdf" : "text/html;charset=utf-8";
      const blob = toReportBlob(raw, type);
      const obj = URL.createObjectURL(blob);
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
  const [reportTemplates, setReportTemplates] = useState<InspectReportTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState(false);
  const [itemModalOpen, setItemModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<InspectItem | null>(null);
  const [tplModalOpen, setTplModalOpen] = useState(false);
  const [editingTpl, setEditingTpl] = useState<InspectReportTemplate | null>(null);
  const [planForm] = Form.useForm();
  const [itemForm] = Form.useForm();
  const [tplForm] = Form.useForm();

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
      const [p, its, rs, ds, tpls] = await Promise.all([
        getInspectPlan(pid),
        listInspectItems(pid),
        listInspectRuns(pid, { page: 1, page_size: 20 }),
        listAlertDatasources({ project_id: pid, page: 1, page_size: 200 }),
        listInspectReportTemplates(pid),
      ]);
      setPlan(p);
      setItems(its || []);
      setRuns(rs.list || []);
      setRunTotal(rs.total || 0);
      setDsList((ds.list || []).filter((d) => d.enabled !== false));
      setReportTemplates(tpls || []);
      planForm.setFieldsValue({
        enabled: p.enabled,
        cron_spec: p.cron_spec,
        datasource_id: p.datasource_id || undefined,
        report_list_mode: p.report_list_mode || "abnormal_only",
        report_template_id: p.report_template_id || undefined,
        retain_days: p.retain_days ?? 90,
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
        width: 90,
        render: (_, r) =>
          r.project_id === 0 ? (
            r.enabled ? <Tag color="success">是</Tag> : <Tag>否</Tag>
          ) : (
            <Switch
              size="small"
              checked={r.enabled}
              onChange={async (checked) => {
                await updateInspectItem(projectId, r.id, {
                  name: r.name,
                  query: r.query,
                  type: r.type,
                  description: r.description,
                  threshold: r.threshold,
                  threshold_type: r.threshold_type,
                  unit: r.unit,
                  enabled: checked,
                });
                void refresh(projectId);
              }}
            />
          ),
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
      width: 280,
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
          <Button type="link" size="small" onClick={() => openAuthorized(inspectReportPrintUrl(projectId, r.id))}>
            打印
          </Button>
          <Button type="link" size="small" onClick={() => openAuthorized(inspectReportPdfUrl(projectId, r.id))}>
            PDF
          </Button>
          <Button type="link" size="small" onClick={() => openAuthorized(inspectReportExcelUrl(projectId, r.id))}>
            Excel
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
                <Alert
                  type="info"
                  showIcon
                  style={{ marginBottom: 16 }}
                  message="适配 Prometheus + Telegraf + Blackbox + Pushgateway"
                  description={
                    <div style={{ fontSize: 13, lineHeight: 1.7 }}>
                      <div>1. 在「告警监控平台」为本项目配置 Prometheus 数据源（指向你们的 Prometheus）。</div>
                      <div>2. 主机/中间件指标：在对应服务器 <code>telegraf.conf</code> 配 <code>inputs.*</code>，由 Prometheus 拉取后再启用巡检项。</div>
                      <div>3. 连通性/端口：用 Blackbox 的 <code>probe_success</code>（ICMP/TCP/HTTP job 名按 scrape 配置改 PromQL）。</div>
                      <div>4. 批次任务：Pushgateway 推送后，按 job 名改「Pushgateway」相关巡检项。</div>
                      <div>5. 已有旧模板时点「重置为 Telegraf 模板」可一键替换项目巡检项。</div>
                    </div>
                  }
                />
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
                      report_template_id: values.report_template_id || 0,
                      retain_days: values.retain_days,
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
                  <Form.Item name="report_template_id" label="报告版式">
                    <Select
                      allowClear
                      placeholder="默认标准版"
                      options={reportTemplates.map((t) => ({
                        label: `${t.name}${t.project_id === 0 ? "（全局）" : ""}`,
                        value: t.id,
                      }))}
                    />
                  </Form.Item>
                  <Form.Item name="retain_days" label="报告保留天数（0=不清理）">
                    <InputNumber min={0} max={3650} style={{ width: "100%" }} />
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
                        message.success(`已补充 ${r.created} 项（同名跳过）`);
                        void refresh(projectId);
                      }}
                    >
                      从模板同步
                    </Button>
                    <Popconfirm
                      title="将删除本项目全部巡检项，并按 Telegraf/Blackbox 全局模板重建？"
                      okText="重置"
                      okButtonProps={{ danger: true }}
                      onConfirm={async () => {
                        const r = await resetInspectItems(projectId);
                        message.success(`已重置为 ${r.created} 项`);
                        void refresh(projectId);
                      }}
                    >
                      <Button danger>重置为 Telegraf 模板</Button>
                    </Popconfirm>
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
          {
            key: "templates",
            label: "报告版式",
            children: (
              <Card
                loading={loading}
                extra={
                  <Typography.Text type="secondary">内置版式可复制为项目模板后编辑；布局保持简洁。</Typography.Text>
                }
              >
                <Table
                  rowKey="id"
                  size="small"
                  dataSource={reportTemplates}
                  pagination={false}
                  columns={[
                    { title: "名称", dataIndex: "name", width: 160 },
                    { title: "编码", dataIndex: "code", width: 120 },
                    {
                      title: "范围",
                      width: 100,
                      render: (_, r) => (r.project_id === 0 ? <Tag>全局</Tag> : <Tag color="blue">项目</Tag>),
                    },
                    { title: "说明", dataIndex: "remark", ellipsis: true },
                    {
                      title: "操作",
                      width: 260,
                      render: (_, r) => (
                        <Space wrap>
                          <Button
                            type="link"
                            size="small"
                            onClick={async () => {
                              const resp = await previewInspectReportTemplate(projectId, { template_id: r.id });
                              const blob = toReportBlob(resp, "text/html;charset=utf-8");
                              const url = URL.createObjectURL(blob);
                              window.open(url, "_blank", "noopener,noreferrer");
                              setTimeout(() => URL.revokeObjectURL(url), 60_000);
                            }}
                          >
                            预览
                          </Button>
                          {r.project_id === 0 ? (
                            <Button
                              type="link"
                              size="small"
                              onClick={async () => {
                                await copyInspectReportTemplate(projectId, { source_id: r.id });
                                message.success("已复制到本项目，可在列表中编辑");
                                void refresh(projectId);
                              }}
                            >
                              复制到项目
                            </Button>
                          ) : (
                            <>
                              <Button
                                type="link"
                                size="small"
                                onClick={() => {
                                  setEditingTpl(r);
                                  tplForm.setFieldsValue({
                                    name: r.name,
                                    remark: r.remark,
                                    body: r.body || "",
                                  });
                                  setTplModalOpen(true);
                                }}
                              >
                                编辑
                              </Button>
                              <Popconfirm
                                title="删除该项目版式？"
                                onConfirm={async () => {
                                  await deleteInspectReportTemplate(projectId, r.id);
                                  message.success("已删除");
                                  void refresh(projectId);
                                }}
                              >
                                <Button type="link" size="small" danger>
                                  删除
                                </Button>
                              </Popconfirm>
                            </>
                          )}
                        </Space>
                      ),
                    },
                  ]}
                />
              </Card>
            ),
          },
        ]}
      />

      <Modal
        title="编辑项目报告版式"
        open={tplModalOpen}
        onCancel={() => setTplModalOpen(false)}
        width={880}
        destroyOnClose
        footer={[
          <Button
            key="preview"
            onClick={async () => {
              const values = await tplForm.validateFields();
              const resp = await previewInspectReportTemplate(projectId, {
                code: editingTpl?.code,
                body: values.body,
              });
              const blob = toReportBlob(resp, "text/html;charset=utf-8");
              const url = URL.createObjectURL(blob);
              window.open(url, "_blank", "noopener,noreferrer");
              setTimeout(() => URL.revokeObjectURL(url), 60_000);
            }}
          >
            预览
          </Button>,
          <Button key="cancel" onClick={() => setTplModalOpen(false)}>
            取消
          </Button>,
          <Button
            key="ok"
            type="primary"
            onClick={async () => {
              if (!editingTpl) return;
              const values = await tplForm.validateFields();
              await updateInspectReportTemplate(projectId, editingTpl.id, values);
              message.success("已保存");
              setTplModalOpen(false);
              void refresh(projectId);
            }}
          >
            保存
          </Button>,
        ]}
      >
        <Form form={tplForm} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="remark" label="说明">
            <Input />
          </Form.Item>
          <Form.Item
            name="body"
            label="HTML 模板"
            rules={[{ required: true, message: "请填写模板正文" }]}
            extra="Go html/template 语法；可用字段：Project、Score、Grade、Summary、Groups、Findings 等。保持简洁即可。"
          >
            <Input.TextArea rows={18} style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace" }} />
          </Form.Item>
        </Form>
      </Modal>

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
