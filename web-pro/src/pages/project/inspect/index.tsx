// @ts-nocheck
import { PageContainer } from "@ant-design/pro-components";
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
import { Link, useSearchParams } from '@umijs/max';
import { DashboardStatCard } from "@/components/ops/dashboard-stat-card";
import { LineChart } from "@/components/line-chart";
import { CHART_BRAND, CHART_ERROR, CHART_SUCCESS } from "@/constants/chart-colors";
import { listAlertDatasources, type AlertDatasourceItem } from "@/services/alert-platform";
import {
  copyInspectReportTemplate,
  deleteInspectItem,
  deleteInspectReportTemplate,
  getInspectPlan,
  getInspectStorageInfo,
  inspectReportExcelUrl,
  inspectReportHtmlUrl,
  inspectReportPrintUrl,
  listInspectItems,
  listInspectReportTemplates,
  listInspectRunTrends,
  listInspectRuns,
  migrateInspectReportsToMinio,
  previewInspectReportTemplate,
  resendInspectEmail,
  resetInspectItems,
  startInspectRun,
  syncInspectItems,
  updateInspectItem,
  type InspectItem,
  type InspectPlan,
  type InspectReportTemplate,
  type InspectRun,
  type InspectRunTrendItem,
  type InspectStorageInfo,
} from "@/services/inspect";
import { extractApiErrorMessage } from "@/services/http";
import { getProjects, type ProjectItem } from "@/services/projects";
import { formatDateTime } from "@/utils/format";
// RF-10 拆分：常量/展示映射与报告下载已下沉，本文件只保留页面编排
import {
  THRESHOLD_TYPE_LABEL,
  gradeColor,
  parseRecipients,
  statusMeta,
  storageColor,
  storageLabel,
  triggerLabel,
} from "../../inspect/display";
import {
  InspectItemFormModal,
  InspectPlanFormSection,
  InspectReportTemplateFormModal,
} from "../../inspect/plan-form-drawer";
import { downloadInspectPdf, openAuthorized, toReportBlob } from "@/utils/inspect-report-download";

export default function ProjectInspectPage() {
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
  const [itemPage, setItemPage] = useState(1);
  const [itemPageSize, setItemPageSize] = useState(20);
  const [itemModalOpen, setItemModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<InspectItem | null>(null);
  const [tplModalOpen, setTplModalOpen] = useState(false);
  const [editingTpl, setEditingTpl] = useState<InspectReportTemplate | null>(null);
  const [runDetail, setRunDetail] = useState<InspectRun | null>(null);
  const [storageInfo, setStorageInfo] = useState<InspectStorageInfo | null>(null);
  const [runTrends, setRunTrends] = useState<InspectRunTrendItem[]>([]);
  const [migratingReports, setMigratingReports] = useState(false);
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
        const [p, its, rs, ds, tpls, storage, trends] = await Promise.all([
          getInspectPlan(pid),
          listInspectItems(pid),
          listInspectRuns(pid, { page, page_size: pageSize }),
          listAlertDatasources({ project_id: pid, page: 1, page_size: 200 }),
          listInspectReportTemplates(pid),
          getInspectStorageInfo(pid),
          listInspectRunTrends(pid, 30),
        ]);
        setPlan(p);
        setItems(its || []);
        setRuns(rs.list || []);
        setRunTotal(rs.total || 0);
        setStorageInfo(storage);
        setRunTrends(trends || []);
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

  const trendChartData = useMemo(() => {
    const ordered = [...runTrends].reverse();
    return {
      labels: ordered.map((t) => {
        const d = t.finished_at ? new Date(t.finished_at) : null;
        return d && !Number.isNaN(d.getTime()) ? `${d.getMonth() + 1}/${d.getDate()}` : `#${t.id}`;
      }),
      scores: ordered.map((t) => t.score),
      criticals: ordered.map((t) => t.critical_count),
    };
  }, [runTrends]);

  async function handleMigrateReports() {
    if (!projectId) return;
    setMigratingReports(true);
    try {
      const r = await migrateInspectReportsToMinio(projectId);
      message.success(`已迁移 ${r.migrated ?? 0} 份报告到 MinIO`);
      await refresh(projectId);
    } catch (e: unknown) {
      message.error(extractApiErrorMessage(e, "迁移失败"));
    } finally {
      setMigratingReports(false);
    }
  }

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

  useEffect(() => {
    setItemPage(1);
  }, [itemTypeFilter, projectId]);

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
          <Tooltip title="html2canvas + jsPDF 按 HTML 样式导出（与 PromAI 相同方案）">
            <Button
              type="link"
              size="small"
              disabled={r.status !== "success"}
              onClick={() => downloadInspectPdf(projectId, r.id, projectName)}
            >
              PDF
            </Button>
          </Tooltip>
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
    <PageContainer
      className="project-inspect-page"
      header={{
        title: "项目巡检",
        subTitle: "基于 Prometheus 指标定时/手动巡检，生成 HTML / Excel / PDF 报告",
        extra: (
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
        ),
      }}
    >

      {!projectId ? (
        <Card className="table-card">
          <Empty description="暂无可用项目，请先创建项目或确认成员权限" />
        </Card>
      ) : (
        <>
          {storageInfo && (!storageInfo.minio_ready || storageInfo.require_minio) ? (
            <Alert
              type={storageInfo.require_minio && !storageInfo.minio_ready ? "error" : "warning"}
              showIcon
              style={{ marginBottom: 16 }}
              message={
                storageInfo.require_minio && !storageInfo.minio_ready
                  ? "巡检报告须写入 MinIO，当前 MinIO 不可用，无法执行巡检"
                  : "巡检报告当前写入本地目录，容器重启后可能丢失"
              }
              description={
                <span>
                  {storageInfo.require_minio ? (
                    <>
                      数据字典 <Typography.Text code>inspect_report.require_minio</Typography.Text> 已启用，须配置 MinIO。
                    </>
                  ) : (
                    <>请在数据字典启用并填写 MinIO 配置（minio_endpoint、minio_access_key、minio_secret_key、minio_bucket）。</>
                  )}
                  当前路径：
                  <Typography.Text code copyable>
                    {storageInfo.local_root || "logs/inspect-reports"}
                  </Typography.Text>
                  {storageInfo.minio_reason ? (
                    <>
                      {" "}
                      原因：{storageInfo.minio_reason}
                    </>
                  ) : null}
                  {storageInfo.minio_ready ? (
                    <div style={{ marginTop: 8 }}>
                      <Button size="small" loading={migratingReports} onClick={() => void handleMigrateReports()}>
                        将历史本地报告迁移到 MinIO
                      </Button>
                    </div>
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

          {runTrends.length > 1 ? (
            <Card className="table-card" title="巡检趋势（最近 30 次）" style={{ marginBottom: 16 }} loading={loading}>
              <LineChart
                yAxisLabel="分数 / 数量"
                labels={trendChartData.labels}
                series={[
                  { name: "健康分", data: trendChartData.scores, color: CHART_BRAND },
                  { name: "严重项", data: trendChartData.criticals, color: CHART_ERROR },
                ]}
                height={240}
              />
            </Card>
          ) : null}

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
                    <InspectPlanFormSection
                      planForm={planForm}
                      projectId={projectId}
                      dsList={dsList}
                      reportTemplates={reportTemplates}
                      onSaved={() => void refresh(projectId)}
                      onGoToItems={() => {
                        setActiveTab("items");
                        setItemPage(1);
                      }}
                      onGoToRuns={() => setActiveTab("runs")}
                    />
                  </Card>
                ),
              },
              {
                key: "items",
                label: `巡检项 (${enabledItems}/${items.length})`,
                children: (
                  <Card
                    className="table-card"
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
                      loading={loading}
                      columns={itemColumns}
                      dataSource={filteredItems}
                      pagination={{
                        current: itemPage,
                        pageSize: itemPageSize,
                        total: filteredItems.length,
                        showSizeChanger: true,
                        pageSizeOptions: ["10", "20", "50", "100"],
                        showTotal: (t) => `共 ${t} 项`,
                      }}
                      onChange={(pag) => {
                        const nextSize = Number(pag.pageSize) || itemPageSize;
                        const nextPage = Number(pag.current) || 1;
                        if (nextSize !== itemPageSize) {
                          setItemPageSize(nextSize);
                          setItemPage(1);
                          return;
                        }
                        setItemPage(nextPage);
                      }}
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
                  <Card className="table-card">
                    <Table
                      rowKey="id"
                      size="small"
                      loading={loading}
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
                        pageSizeOptions: ["10", "20", "50", "100"],
                        showTotal: (t) => `共 ${t} 次`,
                      }}
                      onChange={(pag) => {
                        const nextSize = Number(pag.pageSize) || runPageSize;
                        const nextPage = Number(pag.current) || 1;
                        if (nextSize !== runPageSize) {
                          setRunPageSize(nextSize);
                          setRunPage(1);
                          return;
                        }
                        setRunPage(nextPage);
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
              <Tooltip title="html2canvas + jsPDF 按 HTML 样式导出（与 PromAI 相同方案）">
                <Button onClick={() => downloadInspectPdf(projectId, runDetail.id, projectName)}>PDF</Button>
              </Tooltip>
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

      <InspectReportTemplateFormModal
        open={tplModalOpen}
        onClose={() => setTplModalOpen(false)}
        tplForm={tplForm}
        editingTpl={editingTpl}
        projectId={projectId}
        onSaved={() => void refresh(projectId)}
      />

      <InspectItemFormModal
        open={itemModalOpen}
        onClose={() => setItemModalOpen(false)}
        itemForm={itemForm}
        editingItem={editingItem}
        projectId={projectId}
        onSaved={() => void refresh(projectId)}
      />
    </PageContainer>
  );
}

