import { CheckOutlined, CloseOutlined, LinkOutlined, ReloadOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  Modal,
  Segmented,
  Select,
  Space,
  Steps,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { ApprovalCenterNav } from "../components/approval-center-nav";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import { reviewAIApproval } from "../services/ai";
import { approveReleaseRun, rejectReleaseRun } from "../services/cicd";
import {
  approveDbAccessRequest,
  approveDbAppUserRequest,
  approveDbTicket,
  rejectDbAccessRequest,
  rejectDbAppUserRequest,
  rejectDbTicket,
} from "../services/dbmgmt";
import { getProjects, type ProjectItem } from "../services/projects";
import {
  getWorkflowTicket,
  listPendingWorkflowTickets,
  reviewWorkflowStep,
  type PendingTicketItem,
  type WorkflowTicketDetail,
} from "../services/workflow";
import { formatDateTime } from "../utils/format";
import {
  WORKFLOW_DOMAIN_OPTIONS,
  workflowBusinessDeepLink,
  workflowDomainColor,
  workflowDomainLabel,
  workflowStatusColor,
  workflowStatusLabel,
  workflowStepStatusLabel,
  workflowTicketTypeLabel,
} from "../utils/workflow-labels";

type MineScope = "pending" | "done" | "all";

async function submitDomainReview(row: PendingTicketItem, approve: boolean, comment: string) {
  const { ref_type, ref_id, project_id } = row;
  if (ref_type === "db_sql_ticket") {
    if (approve) await approveDbTicket(project_id, ref_id, comment);
    else await rejectDbTicket(project_id, ref_id, comment);
    return;
  }
  if (ref_type === "db_access_request") {
    if (approve) await approveDbAccessRequest(project_id, ref_id, { comment });
    else await rejectDbAccessRequest(project_id, ref_id, comment);
    return;
  }
  if (ref_type === "db_app_user_request") {
    if (approve) await approveDbAppUserRequest(project_id, ref_id, comment);
    else await rejectDbAppUserRequest(project_id, ref_id, comment);
    return;
  }
  if (ref_type === "cicd_release_run") {
    if (approve) await approveReleaseRun(project_id, ref_id, comment);
    else await rejectReleaseRun(project_id, ref_id, comment);
    return;
  }
  if (ref_type === "ai_tool_approval") {
    await reviewAIApproval(ref_id, { approve, note: comment });
    return;
  }
  await reviewWorkflowStep(row.workflow_ticket_id, row.step_id, approve, comment);
}

export function WorkflowInboxPage() {
  const [searchParams] = useSearchParams();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number | undefined>(() => {
    const n = Number(searchParams.get("project") || 0);
    return n > 0 ? n : undefined;
  });
  const [domains, setDomains] = useState(() => searchParams.get("domain") || "");
  const [mineScope, setMineScope] = useState<MineScope>("pending");
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<PendingTicketItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const [reviewOpen, setReviewOpen] = useState(false);
  const [reviewTarget, setReviewTarget] = useState<PendingTicketItem | null>(null);
  const [reviewApprove, setReviewApprove] = useState(true);
  const [reviewForm] = Form.useForm<{ comment: string }>();
  const [submitting, setSubmitting] = useState(false);

  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewRow, setPreviewRow] = useState<PendingTicketItem | null>(null);
  const [previewDetail, setPreviewDetail] = useState<WorkflowTicketDetail | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);

  const highlightTicketId = Number(searchParams.get("ticket") || searchParams.get("highlight") || 0) || undefined;

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
    });
  }, []);

  useEffect(() => {
    const d = searchParams.get("domain");
    if (d !== null) setDomains(d);
    const p = Number(searchParams.get("project") || 0);
    if (p > 0) setProjectId(p);
  }, [searchParams]);

  const projectNameMap = useMemo(() => {
    const m = new Map<number, string>();
    for (const p of projects) m.set(p.id, p.name);
    return m;
  }, [projects]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listPendingWorkflowTickets({
        page,
        page_size: pageSize,
        mine_scope: mineScope,
        domains: domains || undefined,
        project_id: projectId,
      });
      setRows(res.list ?? []);
      setTotal(res.total ?? 0);
    } finally {
      setLoading(false);
    }
  }, [domains, mineScope, page, pageSize, projectId]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!highlightTicketId || rows.length === 0) return;
    const el = document.getElementById(`wf-row-${highlightTicketId}`);
    el?.scrollIntoView({ behavior: "smooth", block: "center" });
  }, [highlightTicketId, rows]);

  const openPreview = async (row: PendingTicketItem) => {
    setPreviewRow(row);
    setPreviewOpen(true);
    setPreviewDetail(null);
    setPreviewLoading(true);
    try {
      const detail = await getWorkflowTicket(row.workflow_ticket_id);
      setPreviewDetail(detail);
    } catch {
      /* toast */
    } finally {
      setPreviewLoading(false);
    }
  };

  const openReview = (row: PendingTicketItem, approve: boolean) => {
    setReviewTarget(row);
    setReviewApprove(approve);
    reviewForm.resetFields();
    setReviewOpen(true);
  };

  const submitReview = async () => {
    if (!reviewTarget) return;
    const values = await reviewForm.validateFields();
    const comment = values.comment ?? "";
    setSubmitting(true);
    try {
      await submitDomainReview(reviewTarget, reviewApprove, comment);
      message.success(reviewApprove ? "已通过" : "已驳回");
      setReviewOpen(false);
      setPreviewOpen(false);
      void load();
    } catch {
      /* http 拦截器已 toast */
    } finally {
      setSubmitting(false);
    }
  };

  const columns: ColumnsType<PendingTicketItem> = [
    {
      title: "域",
      dataIndex: "domain",
      width: 88,
      render: (v: string) => <Tag color={workflowDomainColor(v)}>{workflowDomainLabel(v)}</Tag>,
    },
    { title: "类型", width: 108, render: (_, row) => workflowTicketTypeLabel(row.ticket_type) },
    {
      title: "项目",
      dataIndex: "project_id",
      width: 120,
      render: (id: number) => projectNameMap.get(id) ?? (id ? `#${id}` : "—"),
    },
    {
      title: "标题",
      dataIndex: "title",
      ellipsis: true,
      render: (title: string, row) => (
        <Button type="link" style={{ padding: 0, height: "auto" }} onClick={() => void openPreview(row)}>
          {title}
        </Button>
      ),
    },
    { title: "当前节点", dataIndex: "current_stage_name", width: 120, ellipsis: true },
    {
      title: "事项",
      width: 88,
      render: (_, row) => {
        if (row.action === "execute" || row.action === "execute_sql") {
          return <Tag color="gold">待执行</Tag>;
        }
        if (row.mine_status === "mine_pending") {
          return <Tag color="processing">待审批</Tag>;
        }
        return <Tag>已处理</Tag>;
      },
    },
    { title: "提交人", dataIndex: "submitter_name", width: 100, render: (v) => v || "—" },
    {
      title: "到达时间",
      dataIndex: "activated_at",
      width: 168,
      render: (v, row) => formatDateTime(v || row.created_at),
    },
    {
      title: "操作",
      key: "actions",
      width: 200,
      fixed: "right",
      render: (_, row) => {
        const pending = row.mine_status === "mine_pending";
        const isExecute = row.action === "execute" || row.action === "execute_sql";
        const link = workflowBusinessDeepLink(row);
        return (
          <Space size="small">
            <Button type="link" size="small" onClick={() => void openPreview(row)}>
              预览
            </Button>
            {link ? (
              <Link to={link}>
                <Button type="link" size="small" icon={<LinkOutlined />}>
                  {pending && isExecute ? "去执行" : "详情"}
                </Button>
              </Link>
            ) : null}
            {pending && !isExecute ? (
              <>
                <Button type="link" size="small" icon={<CheckOutlined />} onClick={() => openReview(row, true)}>
                  通过
                </Button>
                <Button type="link" size="small" danger icon={<CloseOutlined />} onClick={() => openReview(row, false)}>
                  驳回
                </Button>
              </>
            ) : null}
          </Space>
        );
      },
    },
  ];

  const previewPending = previewRow?.mine_status === "mine_pending";
  const previewExecute = previewRow?.action === "execute" || previewRow?.action === "execute_sql";
  const previewLink = previewRow ? workflowBusinessDeepLink(previewRow) : "";

  return (
    <div>
      <PageTelemetryHeader
        label="审批中心"
        title="我的待办"
        subtitle="只处理审批；执行类事项请点「去执行」进入业务详情"
        meta={[`待办 ${total}`]}
      />
      <ApprovalCenterNav />
      <Card>
        <Space wrap style={{ marginBottom: 16 }}>
          <Select
            allowClear
            placeholder="全部项目"
            style={{ width: 200 }}
            value={projectId}
            onChange={(v) => {
              setProjectId(v);
              setPage(1);
            }}
            options={projects.map((p) => ({ label: p.name, value: p.id }))}
          />
          <Select
            style={{ width: 140 }}
            value={domains}
            onChange={(v) => {
              setDomains(v);
              setPage(1);
            }}
            options={[...WORKFLOW_DOMAIN_OPTIONS]}
          />
          <Segmented
            value={mineScope}
            onChange={(v) => {
              setMineScope(v as MineScope);
              setPage(1);
            }}
            options={[
              { label: "待我处理", value: "pending" },
              { label: "我已处理", value: "done" },
              { label: "全部相关", value: "all" },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
        </Space>

        <Table
          rowKey={(r) => `${r.workflow_ticket_id}-${r.step_id}`}
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: 1180 }}
          locale={{ emptyText: <Empty description={mineScope === "pending" ? "暂无待处理事项" : "暂无相关工单"} /> }}
          onRow={(row) => ({
            id: `wf-row-${row.workflow_ticket_id}`,
            style:
              highlightTicketId &&
              (row.workflow_ticket_id === highlightTicketId || row.ref_id === highlightTicketId)
                ? { background: "rgba(22, 119, 255, 0.08)" }
                : undefined,
          })}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>

      <Drawer
        title="工单预览"
        width={520}
        open={previewOpen}
        onClose={() => setPreviewOpen(false)}
        destroyOnClose
        extra={
          <Space>
            {previewLink ? (
              <Link to={previewLink}>
                <Button size="small" icon={<LinkOutlined />}>
                  {previewPending && previewExecute ? "去执行" : "业务详情"}
                </Button>
              </Link>
            ) : null}
            {previewPending && !previewExecute && previewRow ? (
              <>
                <Button
                  size="small"
                  type="primary"
                  icon={<CheckOutlined />}
                  onClick={() => openReview(previewRow, true)}
                >
                  通过
                </Button>
                <Button size="small" danger icon={<CloseOutlined />} onClick={() => openReview(previewRow, false)}>
                  驳回
                </Button>
              </>
            ) : null}
          </Space>
        }
      >
        {previewRow ? (
          <Descriptions column={1} size="small" style={{ marginBottom: 16 }}>
            <Descriptions.Item label="域">
              <Tag color={workflowDomainColor(previewRow.domain)}>{workflowDomainLabel(previewRow.domain)}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="类型">{workflowTicketTypeLabel(previewRow.ticket_type)}</Descriptions.Item>
            <Descriptions.Item label="项目">
              {projectNameMap.get(previewRow.project_id) ?? (previewRow.project_id ? `#${previewRow.project_id}` : "—")}
            </Descriptions.Item>
            <Descriptions.Item label="标题">{previewRow.title}</Descriptions.Item>
            <Descriptions.Item label="提交人">{previewRow.submitter_name || "—"}</Descriptions.Item>
            <Descriptions.Item label="当前节点">{previewRow.current_stage_name || "—"}</Descriptions.Item>
            <Descriptions.Item label="到达">{formatDateTime(previewRow.activated_at || previewRow.created_at)}</Descriptions.Item>
          </Descriptions>
        ) : null}

        <Typography.Title level={5} style={{ marginTop: 0 }}>
          审批进度
        </Typography.Title>
        {previewLoading ? (
          <Typography.Text type="secondary">加载中…</Typography.Text>
        ) : previewDetail?.steps?.length ? (
          <Steps
            direction="vertical"
            size="small"
            items={previewDetail.steps.map((st) => ({
              title: st.stage_name || st.stage_key,
              status:
                st.status === "approved" ? "finish" : st.status === "rejected" ? "error" : st.activated_at ? "process" : "wait",
              description: (
                <Space direction="vertical" size={0}>
                  <Typography.Text type="secondary">
                    {workflowStepStatusLabel(st.status)}
                    {st.reviewer_name ? ` · ${st.reviewer_name}` : ""}
                    {st.user_group_name ? ` · ${st.user_group_name}` : ""}
                  </Typography.Text>
                  {st.review_comment ? <Typography.Text>{st.review_comment}</Typography.Text> : null}
                </Space>
              ),
            }))}
          />
        ) : (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无步骤信息" />
        )}

        {previewDetail ? (
          <div style={{ marginTop: 16 }}>
            <Tag color={workflowStatusColor(previewDetail.status)}>{workflowStatusLabel(previewDetail.status)}</Tag>
            <Typography.Text type="secondary" style={{ marginLeft: 8 }}>
              统一工单 #{previewDetail.id}
            </Typography.Text>
          </div>
        ) : null}
      </Drawer>

      <Modal
        title={reviewApprove ? "审批通过" : "驳回工单"}
        open={reviewOpen}
        confirmLoading={submitting}
        onCancel={() => setReviewOpen(false)}
        onOk={() => void submitReview()}
        destroyOnClose
        okText={reviewApprove ? "确认通过" : "确认驳回"}
        okButtonProps={reviewApprove ? undefined : { danger: true }}
      >
        {reviewTarget ? (
          <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
            {workflowDomainLabel(reviewTarget.domain)} · {workflowTicketTypeLabel(reviewTarget.ticket_type)} ·{" "}
            {reviewTarget.title}
          </Typography.Paragraph>
        ) : null}
        <Form form={reviewForm} layout="vertical">
          <Form.Item name="comment" label="意见" rules={reviewApprove ? [] : [{ required: true, message: "驳回请填写意见" }]}>
            <Input.TextArea rows={3} maxLength={512} placeholder={reviewApprove ? "可选" : "请说明驳回原因"} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
