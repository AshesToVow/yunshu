// @ts-nocheck
import { CheckOutlined, CloseOutlined, LinkOutlined, PlayCircleOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, Modal, Segmented, Select, Space, Table, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from '@umijs/max';
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import {
  approveDbAccessRequest,
  approveDbAppUserRequest,
  approveDbTicket,
  rejectDbAccessRequest,
  rejectDbAppUserRequest,
  rejectDbTicket,
} from "../services/dbmgmt";
import { approveReleaseRun, executeReleaseRun, rejectReleaseRun } from "../services/cicd";
import {
  listPendingWorkflowTickets,
  reviewWorkflowStep,
  type PendingTicketItem,
} from "../services/workflow";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";

type MineScope = "pending" | "done" | "all";

const DOMAIN_OPTIONS = [
  { label: "全部域", value: "" },
  { label: "数据库", value: "dbmgmt" },
  { label: "发布", value: "cicd" },
  { label: "故障", value: "incident" },
  { label: "变更", value: "ops" },
];

function domainLabel(domain: string) {
  switch (domain) {
    case "dbmgmt":
      return "数据库";
    case "cicd":
      return "发布";
    case "incident":
      return "故障";
    case "ops":
      return "变更";
    default:
      return domain || "—";
  }
}

function domainColor(domain: string) {
  switch (domain) {
    case "dbmgmt":
      return "blue";
    case "cicd":
      return "purple";
    case "incident":
      return "red";
    case "ops":
      return "orange";
    default:
      return "default";
  }
}

function ticketTypeLabel(row: PendingTicketItem) {
  switch (row.ticket_type) {
    case "sql_ticket":
      return "SQL 工单";
    case "access_request":
      return "权限申请";
    case "app_user_apply":
      return "应用用户";
    case "release":
      return "发布审批";
    case "change":
      return "变更单";
    case "incident":
      return "故障单";
    default:
      return row.ticket_type || "工单";
  }
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
    const { ref_type, ref_id, project_id } = reviewTarget;
    try {
      if (ref_type === "db_sql_ticket") {
        if (reviewApprove) await approveDbTicket(project_id, ref_id, comment);
        else await rejectDbTicket(project_id, ref_id, comment);
      } else if (ref_type === "db_access_request") {
        if (reviewApprove) await approveDbAccessRequest(project_id, ref_id, { comment });
        else await rejectDbAccessRequest(project_id, ref_id, comment);
      } else if (ref_type === "db_app_user_request") {
        if (reviewApprove) await approveDbAppUserRequest(project_id, ref_id, comment);
        else await rejectDbAppUserRequest(project_id, ref_id, comment);
      } else if (ref_type === "cicd_release_run") {
        if (reviewApprove) await approveReleaseRun(project_id, ref_id, comment);
        else await rejectReleaseRun(project_id, ref_id, comment);
      } else {
        await reviewWorkflowStep(reviewTarget.workflow_ticket_id, reviewTarget.step_id, reviewApprove, comment);
      }
      message.success(reviewApprove ? "已通过" : "已驳回");
      setReviewOpen(false);
      void load();
    } catch {
      /* http 拦截器已 toast */
    }
  };

  const columns: ColumnsType<PendingTicketItem> = [
    {
      title: "域",
      dataIndex: "domain",
      width: 88,
      render: (v: string) => <Tag color={domainColor(v)}>{domainLabel(v)}</Tag>,
    },
    { title: "类型", width: 108, render: (_, row) => ticketTypeLabel(row) },
    {
      title: "项目",
      dataIndex: "project_id",
      width: 120,
      render: (id: number) => projectNameMap.get(id) ?? (id ? `#${id}` : "—"),
    },
    { title: "标题", dataIndex: "title", ellipsis: true },
    { title: "当前节点", dataIndex: "current_stage_name", width: 120, ellipsis: true },
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
      width: 220,
      fixed: "right",
      render: (_, row) => {
        const pending = row.mine_status === "mine_pending";
        const isExecute = row.action === "execute";
        return (
          <Space size="small">
            {row.deep_link ? (
              <Link to={row.deep_link}>
                <Button type="link" size="small" icon={<LinkOutlined />}>
                  详情
                </Button>
              </Link>
            ) : null}
            {pending && isExecute ? (
              <Button
                type="link"
                size="small"
                icon={<PlayCircleOutlined />}
                onClick={() => {
                  void executeReleaseRun(row.project_id, row.ref_id).then(() => {
                    message.success("已触发发布执行");
                    void load();
                  });
                }}
              >
                执行发布
              </Button>
            ) : pending ? (
              <>
                <Button type="link" size="small" icon={<CheckOutlined />} onClick={() => openReview(row, true)}>
                  通过
                </Button>
                <Button type="link" size="small" danger icon={<CloseOutlined />} onClick={() => openReview(row, false)}>
                  驳回
                </Button>
              </>
            ) : (
              <Tag>已处理</Tag>
            )}
          </Space>
        );
      },
    },
  ];

  return (
    <div>
      <PageTelemetryHeader
        label="Workflow"
        title="我的待办"
        subtitle="统一入口：审批 + 发布执行（审批完成后，提交人可在此执行发布）"
      />
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
            options={DOMAIN_OPTIONS}
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
          scroll={{ x: 1100 }}
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

      <Modal
        title={reviewApprove ? "审批通过" : "驳回工单"}
        open={reviewOpen}
        onCancel={() => setReviewOpen(false)}
        onOk={() => void submitReview()}
        destroyOnClose
      >
        {reviewTarget ? (
          <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
            {domainLabel(reviewTarget.domain)} · {ticketTypeLabel(reviewTarget)} · {reviewTarget.title}
          </Typography.Paragraph>
        ) : null}
        <Form form={reviewForm} layout="vertical">
          <Form.Item name="comment" label="意见">
            <Input.TextArea rows={3} maxLength={512} placeholder="可选" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
