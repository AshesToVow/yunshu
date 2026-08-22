import {
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  FileTextOutlined,
  MailOutlined,
  PlusOutlined,
  ReloadOutlined,
  ScheduleOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Progress,
  Row,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { DashboardStatCard } from "../components/ops/dashboard-stat-card";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import { CHART_BRAND, CHART_ERROR, CHART_SUCCESS, CHART_WARNING } from "../constants/chart-colors";
import { listAlertDatasources, type AlertDatasourceItem } from "../services/alert-platform";
import {
  createInspectItem,
  copyInspectReportTemplate,
  deleteInspectItem,
  deleteInspectReportTemplate,
  getInspectPlan,
  getInspectStorageInfo,
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
  type InspectStorageInfo,
} from "../services/inspect";
import { extractApiErrorMessage, http } from "../services/http";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";

const CRON_PRESETS = [
  { label: "每天 09:00", value: "0 0 9 * * *" },
  { label: "每天 02:00", value: "0 0 2 * * *" },
  { label: "每天 18:00", value: "0 0 18 * * *" },
  { label: "每 6 小时", value: "0 0 */6 * * *" },
  { label: "每周一 09:00", value: "0 0 9 * * 1" },
  { label: "每周一 02:00", value: "0 0 2 * * 1" },
];

const THRESHOLD_TYPE_OPTIONS = [
  { label: "大于 (>)", value: "greater" },
  { label: "大于等于 (≥)", value: "greater_equal" },
  { label: "小于 (<)", value: "less" },
  { label: "小于等于 (≤)", value: "less_equal" },
  { label: "等于 (=)", value: "equal" },
  { label: "不等于 (≠)", value: "not_equal" },
];

const THRESHOLD_TYPE_LABEL: Record<string, string> = Object.fromEntries(
  THRESHOLD_TYPE_OPTIONS.map((o) => [o.value, o.label]),
);

function statusMeta(status?: string): { color: string; label: string } {
  switch (status) {
    case "success":
      return { color: "success", label: "成功" };
    case "failed":
      return { color: "error", label: "失败" };
    case "running":
      return { color: "processing", label: "执行中" };
    case "pending":
      return { color: "default", label: "排队中" };
    default:
      return { color: "default", label: status || "-" };
  }
}

function triggerLabel(trigger?: string) {
  if (trigger === "cron") return "定时";
  if (trigger === "manual") return "手动";
  return trigger || "-";
}

function gradeColor(grade?: string) {
  switch (grade) {
    case "A":
      return CHART_SUCCESS;
    case "B":
      return CHART_BRAND;
    case "C":
      return CHART_WARNING;
    case "D":
      return CHART_ERROR;
    default:
      return CHART_BRAND;
  }
}

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
      const type = url.endsWith(".pdf")
        ? "application/pdf"
        : url.endsWith(".xlsx")
          ? "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
          : "text/html;charset=utf-8";
      const blob = toReportBlob(raw, type);
      const obj = URL.createObjectURL(blob);
      window.open(obj, "_blank");
      setTimeout(() => URL.revokeObjectURL(obj), 60_000);
    })
    .catch((e) => message.error(extractApiErrorMessage(e, "打开报告失败")));
}

function storageLabel(storage?: string) {
  const s = (storage || "local").toLowerCase();
  if (s === "minio") return "MinIO";
  return "本地";
}

function storageColor(storage?: string) {
  return (storage || "local").toLowerCase() === "minio" ? "blue" : "default";
}

export function ProjectInspectPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>(0);
  const [plan, setPlan] = useState<InspectPlan | null>(null);
  const [items, setItems] = useState<InspectItem[]>([]);
  const [runs, setRuns] = useState<InspectRun[]>([]);
  const [runTotal, setRunTotal] = useState(0);
  const [runPage, setRunPage] = useState(1);
  const [runPageSize, setRunPageSize] = useState(10);
  const [dsList, setDsList] = useState<AlertDatasourceItem[]>([]);
  const [reportTemplates, setReportTemplates] = useState<InspectReportTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState(false);
  const [itemTypeFilter, setItemTypeFilter] = useState<string | undefined>();
  const [itemModalOpen, setItemModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<InspectItem | null>(null);
  const [tplModalOpen, setTplModalOpen] = useState(false);
  const [editingTpl, setEditingTpl] = useState<InspectReportTemplate | null>(null);
  const [runDetail, setRunDetail] = useState<InspectRun | null>(null);
  const [storageInfo, setStorageInfo] = useState<InspectStorageInfo | null>(null);
  const [activeTab, setActiveTab] = useState("plan");
  const [planForm] = Form.useForm();
  const [itemForm] = Form.useForm();
  const [tplForm] = Form.useForm();

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 })
      .then((r) => {
        const list = r.list || [];
        setProjects(list);
        const fromQuery = Number(searchParams.get("project_id") || 0);
        if (fromQuery > 0 && list.some((p) => p.id === fromQuery)) {
          setProjectId(fromQuery);
        } else if (list.length > 0) {
          setProjectId(list[0].id);
        }
      })
      .catch((e) => message.error(extractApiErrorMessage(e, "加载项目失败")));
  }, [searchParams]);

  const refresh = useCallback(
    async (pid: number, page = runPage, pageSize = runPageSize) => {
      if (!pid) return;
      setLoading(true);
      try {
        const [p, its, rs, ds, tpls, storage] = await Promise.all([
          getInspectPlan(pid),
          listInspectItems(pid),
          listInspectRuns(pid, { page, page_size: pageSize }),
          listAlertDatasources({ project_id: pid, page: 1, page_size: 200 }),
          listInspectReportTemplates(pid),
          getInspectStorageInfo(pid),
        ]);
        setPlan(p);
        setItems(its || []);
        setRuns(rs.list || []);
        setRunTotal(rs.total || 0);
        setStorageInfo(storage);
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
        message.error(extractApiErrorMessage(e, "加载失败"));
      } finally {
        setLoading(false);
      }
    },
    [planForm, runPage, runPageSize],
  );

  useEffect(() => {
    if (projectId > 0) {
      setSearchParams({ project_id: String(projectId) }, { replace: true });
      setRunPage(1);
      void refresh(projectId, 1, runPageSize);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only reset when project changes
  }, [projectId]);

  useEffect(() => {
    if (projectId > 0) {
      void refresh(projectId, runPage, runPageSize);
    }
  }, [runPage, runPageSize]); // eslint-disable-line react-hooks/exhaustive-deps

  const latestRun = runs[0];
  const enabledItems = useMemo(() => items.filter((i) => i.enabled).length, [items]);
  const itemTypes = useMemo(() => {
    const set = new Set(items.map((i) => i.type).filter(Boolean));
    return Array.from(set);
  }, [items]);
  const filteredItems = useMemo(
    () => (itemTypeFilter ? items.filter((i) => i.type === itemTypeFilter) : items),
    [items, itemTypeFilter],
  );
  const projectName = projects.find((p) => p.id === projectId)?.name;
  const recipients = parseRecipients(plan?.recipients_json);
  const dsName =
    dsList.find((d) => d.id === plan?.datasource_id)?.name ||
    (plan?.datasource_id ? `#${plan.datasource_id}` : "未配置");

  const itemColumns: ColumnsType<InspectItem> = useMemo(
    () => [
      { title: "分类", dataIndex: "type", width: 120, ellipsis: true },
      {
        title: "名称",
        dataIndex: "name",
        width: 160,
        render: (name: string, r) => (
          <Space direction="vertical" size={0}>
            <span>{name}</span>
            {r.description ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }} ellipsis>
                {r.description}
              </Typography.Text>
            ) : null}
          </Space>
        ),
      },
      {
        title: "来源",
        width: 96,
        render: (_, r) =>
          r.project_id === 0 ? <Tag>全局模板</Tag> : <Tag color="blue">项目定制</Tag>,
      },
      {
        title: "阈值",
        width: 150,
        render: (_, r) => (
          <Tooltip title={THRESHOLD_TYPE_LABEL[r.threshold_type] || r.threshold_type}>
            <span>
              {THRESHOLD_TYPE_LABEL[r.threshold_type]?.split(" ")[0] || r.threshold_type}{" "}
              {r.threshold}
              {r.unit || ""}
            </span>
          </Tooltip>
        ),
      },
      {
        title: "PromQL",
        dataIndex: "query",
        ellipsis: true,
        render: (q: string) => (
          <Tooltip title={q}>
            <Typography.Text code style={{ fontSize: 12 }}>
              {q}
            </Typography.Text>
          </Tooltip>
        ),
      },
      {
        title: "启用",
        width: 88,
        render: (_, r) =>
          r.project_id === 0 ? (
            r.enabled ? <Tag color="success">是</Tag> : <Tag>否</Tag>
          ) : (
            <Switch
              size="small"
              checked={r.enabled}
              onChange={async (checked) => {
                try {
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
                } catch (e) {
                  message.error(extractApiErrorMessage(e, "更新失败"));
                }
              }}
            />
          ),
      },
      {
        title: "操作",
        width: 130,
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
                  try {
                    await deleteInspectItem(projectId, r.id);
                    message.success("已删除");
                    void refresh(projectId);
                  } catch (e) {
                    message.error(extractApiErrorMessage(e, "删除失败"));
                  }
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
        const m = statusMeta(s);
        return <Tag color={m.color}>{m.label}</Tag>;
      },
    },
    {
      title: "触发",
      dataIndex: "trigger",
      width: 72,
      render: (t: string) => triggerLabel(t),
    },
    {
      title: "健康分",
      width: 110,
      render: (_, r) => (
        <Space size={4}>
          <Typography.Text strong style={{ color: gradeColor(r.grade) }}>
            {r.score}
          </Typography.Text>
          {r.grade ? <Tag color={r.grade === "A" ? "success" : r.grade === "D" ? "error" : "default"}>{r.grade}</Tag> : null}
        </Space>
      ),
    },
    {
      title: "样本",
      width: 150,
      render: (_, r) => (
        <span className="project-inspect-page__counts">
          <Typography.Text type="danger">{r.critical_count} 严重</Typography.Text>
          <span>·</span>
          <Typography.Text type="warning">{r.warning_count} 警告</Typography.Text>
          <span>·</span>
          <Typography.Text type="secondary">{r.normal_count} 正常</Typography.Text>
        </span>
      ),
    },
    {
      title: "摘要",
      dataIndex: "summary",
      ellipsis: true,
      render: (s: string, r) => (
        <Tooltip title={r.error_message || s}>
          <Typography.Text type={r.error_message ? "danger" : undefined} ellipsis style={{ maxWidth: 240 }}>
            {r.error_message || s || "-"}
          </Typography.Text>
        </Tooltip>
      ),
    },
    {
      title: "数据源",
      width: 120,
      ellipsis: true,
      render: (_, r) => r.datasource_name || (r.datasource_id ? `#${r.datasource_id}` : "-"),
    },
    {
      title: "存储",
      width: 72,
      render: (_, r) => (
        <Tag color={storageColor(r.storage)}>{storageLabel(r.storage)}</Tag>
      ),
    },
    {
      title: "时间",
      width: 168,
      render: (_, r) => formatDateTime(r.finished_at || r.started_at || r.created_at),
    },
    {
      title: "报告",
      width: 300,
      render: (_, r) => (
        <Space wrap size={[0, 0]}>
          <Button type="link" size="small" onClick={() => setRunDetail(r)}>
            详情
          </Button>
          <Button
            type="link"
            size="small"
            icon={<FileTextOutlined />}
            disabled={r.status !== "success"}
            onClick={() => openAuthorized(inspectReportHtmlUrl(projectId, r.id))}
          >
            HTML
          </Button>
          <Button
            type="link"
            size="small"
            disabled={r.status !== "success"}
            onClick={() => openAuthorized(inspectReportPrintUrl(projectId, r.id))}
          >
            打印
          </Button>
          <Button
            type="link"
            size="small"
            disabled={r.status !== "success"}
            onClick={() => openAuthorized(inspectReportPdfUrl(projectId, r.id))}
          >
            PDF
          </Button>
          <Button
            type="link"
            size="small"
            disabled={r.status !== "success"}
            onClick={() => openAuthorized(inspectReportExcelUrl(projectId, r.id))}
          >
            Excel
          </Button>
          <Button
            type="link"
            size="small"
            icon={<MailOutlined />}
            disabled={!recipients.length || r.status !== "success"}
            onClick={async () => {
              try {
                await resendInspectEmail(projectId, r.id);
                message.success("已触发重发");
                void refresh(projectId);
              } catch (e) {
                message.error(extractApiErrorMessage(e, "重发失败"));
              }
            }}
          >
            {r.email_sent_at ? "重发" : "发邮件"}
          </Button>
        </Space>
      ),
    },
  ];

  // 有排队/执行中任务时轮询历史，完成后自动刷新分数
  useEffect(() => {
    if (!projectId) return;
    const inflight = runs.some((r) => r.status === "pending" || r.status === "running");
    if (!inflight) return;
    const timer = window.setInterval(() => {
      void listInspectRuns(projectId, { page: runPage, page_size: runPageSize })
        .then((rs) => {
          setRuns(rs.list || []);
          setRunTotal(rs.total || 0);
          const still = (rs.list || []).some((r) => r.status === "pending" || r.status === "running");
          if (!still) {
            void refresh(projectId, runPage, runPageSize);
          }
        })
        .catch(() => undefined);
    }, 3000);
    return () => window.clearInterval(timer);
  }, [projectId, runs, runPage, runPageSize, refresh]);

  async function handleImmediateRun() {
    setRunning(true);
    try {
      const dsId = planForm.getFieldValue("datasource_id") || plan?.datasource_id;
      const run = await startInspectRun(projectId, dsId || undefined);
      message.success(`巡检已加入队列（#${run.id}），可继续操作，完成后自动刷新`);
      setActiveTab("runs");
      setRunPage(1);
      void refresh(projectId, 1, runPageSize);
    } catch (e: unknown) {
      message.error(extractApiErrorMessage(e, "提交巡检失败"));
    } finally {
      setRunning(false);
    }
  }

  return (
    <div className="page-stack project-inspect-page">
      <OpsPageHeader
        title="项目巡检"
        description="基于 Prometheus（Telegraf / Blackbox / kube-state-metrics 等）采集指标，定时或手动巡检项目健康，并生成 HTML / PDF / Excel 报告与邮件通知。"
        breadcrumbs={[{ title: "项目运维" }, { title: "项目巡检" }]}
        meta={
          projectId ? (
            <Space wrap size="small">
              <Tag>{projectName || `项目 #${projectId}`}</Tag>
              <Tag color={plan?.enabled ? "success" : "default"}>
                {plan?.enabled ? "定时已启用" : "定时未启用"}
              </Tag>
              <Tag>数据源 · {dsName}</Tag>
              {plan?.last_run_at ? <Tag>最近执行 · {formatDateTime(plan.last_run_at)}</Tag> : <Tag>尚未执行</Tag>}
              {recipients.length ? <Tag color="blue">邮件 · {recipients.length} 人</Tag> : <Tag>未配置收件人</Tag>}
            </Space>
          ) : null
        }
        extra={
          <>
            <Select
              style={{ width: 220 }}
              value={projectId || undefined}
              placeholder="选择项目"
              options={projects.map((p) => ({ label: p.name, value: p.id }))}
              onChange={(v) => setProjectId(v)}
              showSearch
              optionFilterProp="label"
            />
            <Button icon={<ReloadOutlined />} onClick={() => void refresh(projectId)} loading={loading}>
              刷新
            </Button>
            <Button
              type="primary"
              icon={<ThunderboltOutlined />}
              loading={running}
              disabled={!projectId}
              onClick={() => void handleImmediateRun()}
            >
              立即巡检
            </Button>
          </>
        }
      />

      {!projectId ? (
        <Card className="table-card">
          <Empty description="暂无可用项目，请先创建项目或确认成员权限" />
        </Card>
      ) : (
        <>
          {storageInfo && !storageInfo.minio_ready ? (
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
              message="巡检报告当前写入本地目录，容器重启后可能丢失"
              description={
                <span>
                  请在数据字典启用并填写 MinIO 配置（minio_endpoint、minio_access_key、minio_secret_key、minio_bucket），
                  与 MySQL 备份共用同一套连接。当前路径：
                  <Typography.Text code copyable>
                    {storageInfo.local_root || "logs/inspect-reports"}
                  </Typography.Text>
                  {storageInfo.minio_reason ? (
                    <>
                      {" "}
                      原因：{storageInfo.minio_reason}
                    </>
                  ) : null}
                </span>
              }
            />
          ) : null}

          <Row gutter={[16, 16]} className="project-inspect-page__kpis">
            <Col xs={24} sm={12} lg={6}>
              <Card className="project-inspect-page__score-card" loading={loading} bordered>
                <div className="project-inspect-page__score-head">
                  <Typography.Text type="secondary">最近健康分</Typography.Text>
                  {latestRun?.grade ? (
                    <Tag color={latestRun.grade === "A" ? "success" : latestRun.grade === "D" ? "error" : "processing"}>
                      等级 {latestRun.grade}
                    </Tag>
                  ) : null}
                </div>
                <div className="project-inspect-page__score-value" style={{ color: gradeColor(latestRun?.grade) }}>
                  {latestRun ? latestRun.score : "—"}
                </div>
                <Progress
                  percent={latestRun ? Math.min(100, Math.max(0, latestRun.score)) : 0}
                  showInfo={false}
                  strokeColor={gradeColor(latestRun?.grade)}
                  size="small"
                />
                <Typography.Paragraph type="secondary" className="project-inspect-page__score-hint">
                  {latestRun
                    ? `${statusMeta(latestRun.status).label} · ${triggerLabel(latestRun.trigger)} · ${formatDateTime(latestRun.finished_at || latestRun.started_at)}`
                    : "执行一次巡检后显示评分"}
                </Typography.Paragraph>
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <DashboardStatCard
                title="严重项"
                value={latestRun?.critical_count ?? 0}
                hint={latestRun ? `警告 ${latestRun.warning_count} · 正常 ${latestRun.normal_count}` : "暂无最近结果"}
                icon={<ExclamationCircleOutlined />}
                accent={CHART_ERROR}
                loading={loading}
              />
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <DashboardStatCard
                title="启用巡检项"
                value={enabledItems}
                hint={`共 ${items.length} 项 · 可按分类筛选`}
                icon={<CheckCircleOutlined />}
                accent={CHART_SUCCESS}
                loading={loading}
              />
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <DashboardStatCard
                title="历史记录"
                value={runTotal}
                hint={plan?.enabled ? `Cron ${plan.cron_spec || "-"}` : "定时巡检未开启"}
                icon={<ScheduleOutlined />}
                accent={CHART_BRAND}
                loading={loading}
              />
            </Col>
          </Row>

          {!dsList.length ? (
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
              message="当前项目尚未配置可用的 Prometheus 数据源"
              description={
                <span>
                  请先到{" "}
                  <Link to={`/alert-monitor-platform/datasources?project_id=${projectId}`}>告警监控平台 · 数据源</Link>{" "}
                  为本项目添加并启用 Prometheus，再保存巡检计划或执行立即巡检。
                </span>
              }
            />
          ) : null}

          <Tabs
            activeKey={activeTab}
            onChange={setActiveTab}
            items={[
              {
                key: "plan",
                label: "计划配置",
                children: (
                  <Card className="table-card" loading={loading}>
                    <Alert
                      type="info"
                      showIcon
                      className="project-inspect-page__guide"
                      message="适配 Prometheus + Telegraf + Blackbox + Pushgateway"
                      description={
                        <ol className="project-inspect-page__guide-list">
                          <li>
                            在「告警监控平台」为本项目配置 Prometheus 数据源（指向你们的 Prometheus）。
                          </li>
                          <li>
                            主机/中间件指标：在对应服务器 <code>telegraf.conf</code> 配置 <code>inputs.*</code>
                            ，由 Prometheus 拉取后再启用巡检项。
                          </li>
                          <li>
                            连通性/端口：使用 Blackbox 的 <code>probe_success</code>
                            （ICMP/TCP/HTTP 的 job 名按 scrape 配置调整 PromQL）。
                          </li>
                          <li>批次任务：Pushgateway 推送后，按 job 名改「Pushgateway」相关巡检项。</li>
                          <li>已有旧模板时，可在「巡检项」页签点击「重置为 Telegraf 模板」一键重建。</li>
                        </ol>
                      }
                    />
                    <Form
                      form={planForm}
                      layout="vertical"
                      className="project-inspect-page__plan-form"
                      onFinish={async (values) => {
                        try {
                          const list = String(values.recipients || "")
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
                            recipients: list,
                          });
                          message.success("计划已保存");
                          void refresh(projectId);
                        } catch (e) {
                          message.error(extractApiErrorMessage(e, "保存失败"));
                        }
                      }}
                    >
                      <Row gutter={16}>
                        <Col xs={24} md={12}>
                          <Form.Item name="enabled" label="启用定时巡检" valuePropName="checked">
                            <Switch checkedChildren="开" unCheckedChildren="关" />
                          </Form.Item>
                          <Form.Item
                            name="datasource_id"
                            label="Prometheus 数据源"
                            rules={[{ required: true, message: "请选择数据源" }]}
                            extra={
                              <Link to={`/alert-monitor-platform/datasources?project_id=${projectId}`}>管理数据源</Link>
                            }
                          >
                            <Select
                              allowClear
                              options={dsList.map((d) => ({
                                label: `${d.name} (#${d.id})`,
                                value: d.id,
                              }))}
                              placeholder="选择项目内数据源"
                            />
                          </Form.Item>
                          <Form.Item
                            name="cron_spec"
                            label="Cron 表达式"
                            extra="秒 分 时 日 月 周；可选手动输入自定义表达式"
                          >
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
                                      placeholder="自定义 Cron，回车填入"
                                      onPressEnter={(e) =>
                                        planForm.setFieldValue(
                                          "cron_spec",
                                          (e.target as HTMLInputElement).value,
                                        )
                                      }
                                    />
                                  </div>
                                </>
                              )}
                            />
                          </Form.Item>
                        </Col>
                        <Col xs={24} md={12}>
                          <Form.Item name="report_list_mode" label="报告明细模式">
                            <Select
                              options={[
                                { label: "仅异常（推荐日常）", value: "abnormal_only" },
                                { label: "摘要（按项汇总）", value: "summary" },
                                { label: "全部样本", value: "all" },
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
                          <Form.Item
                            name="recipients"
                            label="邮件收件人"
                            extra="多个地址用逗号或空格分隔；需平台已配置发信"
                          >
                            <Input.TextArea rows={2} placeholder="ops@example.com, oncall@example.com" />
                          </Form.Item>
                        </Col>
                      </Row>
                      <Space wrap>
                        <Button type="primary" htmlType="submit">
                          保存计划
                        </Button>
                        <Button onClick={() => setActiveTab("items")}>去配置巡检项</Button>
                        <Button onClick={() => setActiveTab("runs")}>查看历史</Button>
                      </Space>
                    </Form>
                  </Card>
                ),
              },
              {
                key: "items",
                label: `巡检项 (${enabledItems}/${items.length})`,
                children: (
                  <Card
                    className="table-card"
                    loading={loading}
                    extra={
                      <Space wrap>
                        <Select
                          allowClear
                          placeholder="按分类筛选"
                          style={{ width: 160 }}
                          value={itemTypeFilter}
                          options={itemTypes.map((t) => ({ label: t, value: t }))}
                          onChange={(v) => setItemTypeFilter(v)}
                        />
                        <Button
                          onClick={async () => {
                            try {
                              const r = await syncInspectItems(projectId);
                              message.success(`已补充 ${r.created} 项（同名跳过）`);
                              void refresh(projectId);
                            } catch (e) {
                              message.error(extractApiErrorMessage(e, "同步失败"));
                            }
                          }}
                        >
                          从模板同步
                        </Button>
                        <Popconfirm
                          title="将删除本项目全部巡检项，并按 Telegraf/Blackbox 全局模板重建？"
                          okText="重置"
                          okButtonProps={{ danger: true }}
                          onConfirm={async () => {
                            try {
                              const r = await resetInspectItems(projectId);
                              message.success(`已重置为 ${r.created} 项`);
                              void refresh(projectId);
                            } catch (e) {
                              message.error(extractApiErrorMessage(e, "重置失败"));
                            }
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
                    <Table
                      rowKey="id"
                      size="small"
                      columns={itemColumns}
                      dataSource={filteredItems}
                      pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 项` }}
                      locale={{
                        emptyText: (
                          <Empty
                            image={Empty.PRESENTED_IMAGE_SIMPLE}
                            description="暂无巡检项，可从模板同步或重置为 Telegraf 模板"
                          />
                        ),
                      }}
                    />
                  </Card>
                ),
              },
              {
                key: "runs",
                label: `历史 (${runTotal})`,
                children: (
                  <Card className="table-card" loading={loading}>
                    <Table
                      rowKey="id"
                      size="small"
                      columns={runColumns}
                      dataSource={runs}
                      onRow={(r) => ({
                        onClick: (e) => {
                          const target = e.target as HTMLElement;
                          if (target.closest("button, a, .ant-btn")) return;
                          setRunDetail(r);
                        },
                        style: { cursor: "pointer" },
                      })}
                      pagination={{
                        current: runPage,
                        pageSize: runPageSize,
                        total: runTotal,
                        showSizeChanger: true,
                        showTotal: (t) => `共 ${t} 次`,
                        onChange: (page, size) => {
                          setRunPage(page);
                          setRunPageSize(size);
                        },
                      }}
                      locale={{
                        emptyText: (
                          <Empty
                            image={Empty.PRESENTED_IMAGE_SIMPLE}
                            description="暂无巡检记录，点击右上角「立即巡检」开始"
                          >
                            <Button type="primary" icon={<ThunderboltOutlined />} onClick={() => void handleImmediateRun()}>
                              立即巡检
                            </Button>
                          </Empty>
                        ),
                      }}
                    />
                  </Card>
                ),
              },
              {
                key: "templates",
                label: "报告版式",
                children: (
                  <Card
                    className="table-card"
                    loading={loading}
                    extra={
                      <Typography.Text type="secondary">
                        内置版式可复制为项目模板后编辑；保存前可用预览校验语法。
                      </Typography.Text>
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
                          render: (_, r) =>
                            r.project_id === 0 ? <Tag>全局</Tag> : <Tag color="blue">项目</Tag>,
                        },
                        {
                          title: "说明",
                          dataIndex: "remark",
                          ellipsis: true,
                          render: (v: string) => v || "—",
                        },
                        {
                          title: "操作",
                          width: 260,
                          render: (_, r) => (
                            <Space wrap>
                              <Button
                                type="link"
                                size="small"
                                onClick={async () => {
                                  try {
                                    const resp = await previewInspectReportTemplate(projectId, {
                                      template_id: r.id,
                                    });
                                    const blob = toReportBlob(resp, "text/html;charset=utf-8");
                                    const url = URL.createObjectURL(blob);
                                    window.open(url, "_blank", "noopener,noreferrer");
                                    setTimeout(() => URL.revokeObjectURL(url), 60_000);
                                  } catch (e) {
                                    message.error(extractApiErrorMessage(e, "预览失败"));
                                  }
                                }}
                              >
                                预览
                              </Button>
                              {r.project_id === 0 ? (
                                <Button
                                  type="link"
                                  size="small"
                                  onClick={async () => {
                                    try {
                                      await copyInspectReportTemplate(projectId, { source_id: r.id });
                                      message.success("已复制到本项目，可在列表中编辑");
                                      void refresh(projectId);
                                    } catch (e) {
                                      message.error(extractApiErrorMessage(e, "复制失败"));
                                    }
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
                                      try {
                                        await deleteInspectReportTemplate(projectId, r.id);
                                        message.success("已删除");
                                        void refresh(projectId);
                                      } catch (e) {
                                        message.error(extractApiErrorMessage(e, "删除失败"));
                                      }
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
        </>
      )}

      <Drawer
        title={runDetail ? `巡检详情 #${runDetail.id}` : "巡检详情"}
        open={Boolean(runDetail)}
        onClose={() => setRunDetail(null)}
        width={520}
        destroyOnClose
        extra={
          runDetail ? (
            <Space>
              <Button
                size="small"
                icon={<FileTextOutlined />}
                onClick={() => openAuthorized(inspectReportHtmlUrl(projectId, runDetail.id))}
              >
                打开报告
              </Button>
            </Space>
          ) : null
        }
      >
        {runDetail ? (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            <div className="project-inspect-page__drawer-score">
              <div>
                <Typography.Text type="secondary">健康分</Typography.Text>
                <div
                  className="project-inspect-page__drawer-score-num"
                  style={{ color: gradeColor(runDetail.grade) }}
                >
                  {runDetail.score}
                  {runDetail.grade ? <span> / {runDetail.grade}</span> : null}
                </div>
              </div>
              <Progress
                type="circle"
                width={72}
                percent={Math.min(100, Math.max(0, runDetail.score))}
                strokeColor={gradeColor(runDetail.grade)}
              />
            </div>
            <Descriptions column={1} size="small" bordered>
              <Descriptions.Item label="状态">
                <Tag color={statusMeta(runDetail.status).color}>{statusMeta(runDetail.status).label}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="触发方式">{triggerLabel(runDetail.trigger)}</Descriptions.Item>
              <Descriptions.Item label="数据源">
                {runDetail.datasource_name || `#${runDetail.datasource_id}`}
              </Descriptions.Item>
              <Descriptions.Item label="样本统计">
                严重 {runDetail.critical_count} · 警告 {runDetail.warning_count} · 正常{" "}
                {runDetail.normal_count} · 合计 {runDetail.total_count}
              </Descriptions.Item>
              <Descriptions.Item label="版式">
                {runDetail.report_template_code || "default"}
              </Descriptions.Item>
              <Descriptions.Item label="存储">
                <Tag color={storageColor(runDetail.storage)}>{storageLabel(runDetail.storage)}</Tag>
              </Descriptions.Item>
              {runDetail.report_html_path ? (
                <Descriptions.Item label="HTML 路径">
                  <Typography.Text code copyable style={{ wordBreak: "break-all" }}>
                    {runDetail.report_html_path}
                  </Typography.Text>
                </Descriptions.Item>
              ) : null}
              {runDetail.report_pdf_path ? (
                <Descriptions.Item label="PDF 路径">
                  <Typography.Text code copyable style={{ wordBreak: "break-all" }}>
                    {runDetail.report_pdf_path}
                  </Typography.Text>
                </Descriptions.Item>
              ) : null}
              {runDetail.report_excel_path ? (
                <Descriptions.Item label="Excel 路径">
                  <Typography.Text code copyable style={{ wordBreak: "break-all" }}>
                    {runDetail.report_excel_path}
                  </Typography.Text>
                </Descriptions.Item>
              ) : null}
              <Descriptions.Item label="开始时间">
                {formatDateTime(runDetail.started_at || runDetail.created_at)}
              </Descriptions.Item>
              <Descriptions.Item label="结束时间">
                {runDetail.finished_at ? formatDateTime(runDetail.finished_at) : "—"}
              </Descriptions.Item>
              <Descriptions.Item label="邮件">
                {runDetail.email_sent_at ? `已发送 · ${formatDateTime(runDetail.email_sent_at)}` : "未发送"}
              </Descriptions.Item>
              <Descriptions.Item label="摘要">{runDetail.summary || "—"}</Descriptions.Item>
              {runDetail.error_message ? (
                <Descriptions.Item label="错误">
                  <Typography.Text type="danger">{runDetail.error_message}</Typography.Text>
                </Descriptions.Item>
              ) : null}
            </Descriptions>
            <Space wrap>
              <Button onClick={() => openAuthorized(inspectReportHtmlUrl(projectId, runDetail.id))}>
                HTML
              </Button>
              <Button onClick={() => openAuthorized(inspectReportPrintUrl(projectId, runDetail.id))}>
                打印版
              </Button>
              <Button onClick={() => openAuthorized(inspectReportPdfUrl(projectId, runDetail.id))}>
                PDF
              </Button>
              <Button onClick={() => openAuthorized(inspectReportExcelUrl(projectId, runDetail.id))}>
                Excel
              </Button>
              <Button
                icon={<MailOutlined />}
                disabled={!recipients.length}
                onClick={async () => {
                  try {
                    await resendInspectEmail(projectId, runDetail.id);
                    message.success("已触发重发");
                    void refresh(projectId);
                  } catch (e) {
                    message.error(extractApiErrorMessage(e, "重发失败"));
                  }
                }}
              >
                重发邮件
              </Button>
            </Space>
          </Space>
        ) : null}
      </Drawer>

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
              try {
                const values = await tplForm.validateFields();
                const resp = await previewInspectReportTemplate(projectId, {
                  code: editingTpl?.code,
                  body: values.body,
                });
                const blob = toReportBlob(resp, "text/html;charset=utf-8");
                const url = URL.createObjectURL(blob);
                window.open(url, "_blank", "noopener,noreferrer");
                setTimeout(() => URL.revokeObjectURL(url), 60_000);
              } catch (e) {
                message.error(extractApiErrorMessage(e, "预览失败"));
              }
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
              try {
                const values = await tplForm.validateFields();
                await updateInspectReportTemplate(projectId, editingTpl.id, values);
                message.success("已保存");
                setTplModalOpen(false);
                void refresh(projectId);
              } catch (e) {
                message.error(extractApiErrorMessage(e, "保存失败"));
              }
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
            extra="Go html/template 语法；可用字段：Project、Score、Grade、Summary、Groups、Findings 等。"
          >
            <Input.TextArea
              rows={18}
              style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace" }}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingItem ? "编辑巡检项" : "新增巡检项"}
        open={itemModalOpen}
        onCancel={() => setItemModalOpen(false)}
        onOk={async () => {
          try {
            const values = await itemForm.validateFields();
            if (editingItem) {
              await updateInspectItem(projectId, editingItem.id, values);
            } else {
              await createInspectItem(projectId, values);
            }
            message.success("已保存");
            setItemModalOpen(false);
            void refresh(projectId);
          } catch (e) {
            if (e && typeof e === "object" && "errorFields" in e) return;
            message.error(extractApiErrorMessage(e, "保存失败"));
          }
        }}
        destroyOnClose
        width={720}
      >
        <Form form={itemForm} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="type" label="分类" rules={[{ required: true }]}>
                <Input placeholder="如：基础设施层 / 数据库监控" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="name" label="名称" rules={[{ required: true }]}>
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="description" label="说明">
            <Input.TextArea rows={2} placeholder="指标来源、标签约定、何时启用等" />
          </Form.Item>
          <Form.Item
            name="query"
            label="PromQL"
            rules={[{ required: true }]}
            extra="即时向量查询；无数据时该项会记为异常。"
          >
            <Input.TextArea rows={3} style={{ fontFamily: "ui-monospace, Menlo, Consolas, monospace" }} />
          </Form.Item>
          <Row gutter={16}>
            <Col xs={24} sm={8}>
              <Form.Item name="threshold_type" label="比较方式" rules={[{ required: true }]}>
                <Select options={THRESHOLD_TYPE_OPTIONS} />
              </Form.Item>
            </Col>
            <Col xs={12} sm={6}>
              <Form.Item name="threshold" label="阈值" rules={[{ required: true }]}>
                <InputNumber style={{ width: "100%" }} />
              </Form.Item>
            </Col>
            <Col xs={12} sm={4}>
              <Form.Item name="unit" label="单位">
                <Input placeholder="%" />
              </Form.Item>
            </Col>
            <Col xs={24} sm={6}>
              <Form.Item name="enabled" label="启用" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  );
}

export default ProjectInspectPage;
