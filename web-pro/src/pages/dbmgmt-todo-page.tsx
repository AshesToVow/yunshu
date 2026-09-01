// @ts-nocheck
import { CheckOutlined, CloseOutlined, CopyOutlined, EyeOutlined, PlayCircleOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Descriptions, Form, Input, Modal, Segmented, Select, Space, Spin, Table, Tabs, Tag, Typography, Alert, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from '@umijs/max';
import {
  GrantValidityCalendarPicker,
  expiresAtToGrantPeriod,
  formatGrantPeriodSummary,
  grantPeriodToExpiresAt,
  type GrantValidityPeriod,
} from "../components/dbmgmt/grant-validity-calendar";
import {
  approveDbAccessRequest,
  approveDbAppUserRequest,
  approveDbTicket,
  executeDbTicket,
  getDbTicket,
  listDbAccessRequests,
  listDbAppUserRequests,
  listDbTickets,
  rejectDbAccessRequest,
  rejectDbAppUserRequest,
  rejectDbTicket,
  type DbAccessRequest,
  type DbAppUserRequest,
  type DbTicket,
} from "../services/dbmgmt";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";
import { accessRequestStatusLabel, ticketStatusLabel } from "../utils/dbmgmt-labels";
import { riskLevelColor, riskLevelLabel } from "../utils/dbmgmt-console";
import { ticketTypeLabel } from "../components/dbmgmt/dbmgmt-ui-shared";

type TabKey = "access" | "app_user" | "ticket";
type MineScope = "all" | "pending" | "done";

function isAppUserPending(row: DbAppUserRequest) {
  if (row.mine_status === "mine_done") return false;
  if (row.mine_status === "mine_pending") return true;
  return row.status === "pending";
}

function isAccessPending(row: DbAccessRequest) {
  if (row.mine_status === "mine_done") return false;
  if (row.mine_status === "mine_pending") return true;
  return row.status === "pending";
}

function isTicketApprovalPending(row: DbTicket) {
  if (row.mine_status === "mine_done") return false;
  if (row.mine_status === "mine_pending") return true;
  return row.status === "pending_approval";
}

function isTicketExecutionPending(row: DbTicket) {
  if (row.mine_status === "mine_done") return false;
  if (row.mine_status === "mine_pending") return true;
  return row.status === "pending_execution";
}

const APP_USER_STATUS_OPTIONS = [
  { label: "全部状态", value: "" },
  { label: "待审批", value: "pending" },
  { label: "已通过", value: "approved" },
  { label: "执行成功", value: "success" },
  { label: "执行失败", value: "failed" },
  { label: "已驳回", value: "rejected" },
];

const ACCESS_STATUS_OPTIONS = [
  { label: "全部状态", value: "" },
  { label: "待审批", value: "pending" },
  { label: "已通过", value: "approved" },
  { label: "已驳回", value: "rejected" },
];

const TICKET_STATUS_OPTIONS = [
  { label: "全部状态", value: "" },
  { label: "待审批", value: "pending_approval" },
  { label: "待执行", value: "pending_execution" },
  { label: "执行成功", value: "success" },
  { label: "执行失败", value: "failed" },
  { label: "已驳回", value: "rejected" },
];

function TicketSqlMeta({ ticket }: { ticket: DbTicket }) {
  return (
    <div style={{ fontSize: 13, color: "#666" }}>
      <div>实例：{ticket.instance_name || "—"}</div>
      <div>数据库：{ticket.database_name || "—"}</div>
      <div>变更说明：{ticket.reason || "—"}</div>
    </div>
  );
}

export function DbmgmtTodoPage({ mode = "all" }: { mode?: "all" | "pending" }) {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [tab, setTab] = useState<TabKey>("access");
  const [mineScope, setMineScope] = useState<MineScope>(mode === "pending" ? "pending" : "all");
  const [status, setStatus] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [accessRows, setAccessRows] = useState<DbAccessRequest[]>([]);
  const [appUserRows, setAppUserRows] = useState<DbAppUserRequest[]>([]);
  const [ticketRows, setTicketRows] = useState<DbTicket[]>([]);
  const [accessTotal, setAccessTotal] = useState(0);
  const [appUserTotal, setAppUserTotal] = useState(0);
  const [ticketTotal, setTicketTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [reviewOpen, setReviewOpen] = useState(false);
  const [reviewType, setReviewType] = useState<"access" | "app_user" | "ticket">("access");
  const [reviewId, setReviewId] = useState<number>();
  const [reviewTicket, setReviewTicket] = useState<DbTicket | null>(null);
  const [reviewTicketLoading, setReviewTicketLoading] = useState(false);
  const [approve, setApprove] = useState(true);
  const [sqlOpen, setSqlOpen] = useState(false);
  const [sqlLoading, setSqlLoading] = useState(false);
  const [sqlTicket, setSqlTicket] = useState<DbTicket | null>(null);
  const [reviewAccess, setReviewAccess] = useState<DbAccessRequest | null>(null);
  const [form] = Form.useForm<{ comment: string; grant_period?: GrantValidityPeriod | null }>();

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
      if (res.list?.length) setProjectId(res.list[0].id);
    });
  }, []);

  const scopeOptions = useMemo(
    () => [
      { label: "全部", value: "all" as const },
      { label: tab === "ticket" && status === "pending_execution" ? "待我执行" : "待我审批", value: "pending" as const },
      { label: tab === "ticket" && status === "pending_execution" ? "我已执行" : "我已审批", value: "done" as const },
    ],
    [tab, status],
  );

  const load = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    try {
      const params: Record<string, string | number> = {
        mine: 1,
        mine_scope: mineScope,
        page,
        page_size: pageSize,
      };
      if (status) params.status = status;
      const [access, appUsers, tickets] = await Promise.all([
        listDbAccessRequests(projectId, params),
        listDbAppUserRequests(projectId, params),
        listDbTickets(projectId, params),
      ]);
      setAccessRows(access.list ?? []);
      setAccessTotal(access.total ?? 0);
      setAppUserRows(appUsers.list ?? []);
      setAppUserTotal(appUsers.total ?? 0);
      setTicketRows(tickets.list ?? []);
      setTicketTotal(tickets.total ?? 0);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "加载待办失败");
    } finally {
      setLoading(false);
    }
  }, [projectId, mineScope, status, page, pageSize]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    setPage(1);
  }, [tab, projectId, mineScope, status]);

  const openReview = (type: "access" | "app_user" | "ticket", id: number, isApprove: boolean, accessRow?: DbAccessRequest) => {
    setReviewType(type);
    setReviewId(id);
    setApprove(isApprove);
    setReviewTicket(null);
    setReviewAccess(type === "access" ? accessRow ?? null : null);
    setReviewTicketLoading(type === "ticket");
    form.resetFields();
    if (type === "access" && isApprove && accessRow) {
      form.setFieldsValue({ grant_period: expiresAtToGrantPeriod(accessRow.expires_at) });
    }
    setReviewOpen(true);
    if (type === "ticket" && projectId) {
      void getDbTicket(projectId, id)
        .then((t) => setReviewTicket(t))
        .catch(() => setReviewTicket(null))
        .finally(() => setReviewTicketLoading(false));
    }
  };

  const openSqlView = (ticketId: number) => {
    if (!projectId) return;
    setSqlOpen(true);
    setSqlLoading(true);
    setSqlTicket(null);
    void getDbTicket(projectId, ticketId)
      .then((t) => setSqlTicket(t))
      .catch((e) => {
        message.error(e instanceof Error ? e.message : "加载 SQL 失败");
        setSqlOpen(false);
      })
      .finally(() => setSqlLoading(false));
  };

  const copySql = async (text?: string) => {
    if (!text?.trim()) {
      message.warning("暂无 SQL 内容");
      return;
    }
    try {
      await navigator.clipboard.writeText(text);
      message.success("已复制 SQL");
    } catch {
      message.error("复制失败，请手动选择复制");
    }
  };

  const renderSqlPreview = (ticket: DbTicket | null, loading?: boolean) => {
    if (loading) return <Spin />;
    if (!ticket) return <Typography.Text type="secondary">无法加载 SQL 内容</Typography.Text>;
    return (
      <Space direction="vertical" style={{ width: "100%" }} size="small">
        <TicketSqlMeta ticket={ticket} />
        <pre
          style={{
            margin: 0,
            maxHeight: 360,
            overflow: "auto",
            background: "#1e1e1e",
            color: "#d4d4d4",
            padding: 12,
            borderRadius: 6,
            fontSize: 13,
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
          }}
        >
          {ticket.sql_text || ticket.sql_excerpt || "（无 SQL 内容）"}
        </pre>
      </Space>
    );
  };

  const grantPeriodRules = [
    {
      validator: (_: unknown, value: GrantValidityPeriod | null | undefined) => {
        if (value === null) return Promise.resolve();
        if (value?.start && value?.end) {
          if (!value.end.endOf("day").isBefore(dayjs().startOf("day"))) return Promise.resolve();
          return Promise.reject(new Error("结束日期不能早于今天"));
        }
        return Promise.reject(new Error("请在日历上选择授权起止日期，或点选「永久有效」"));
      },
    },
  ];

  const submitReview = async () => {
    if (!projectId || !reviewId) return;
    const values = await form.validateFields();
    const comment = values.comment;
    if (reviewType === "access") {
      if (approve) {
        const period =
          reviewAccess?.is_final_approval !== false
            ? (values.grant_period as GrantValidityPeriod | null | undefined)
            : undefined;
        const payload: { comment?: string; expires_at?: string } = { comment };
        if (period !== undefined) {
          payload.expires_at = period === null ? "" : grantPeriodToExpiresAt(period ?? null) ?? "";
        }
        await approveDbAccessRequest(projectId, reviewId, payload);
      } else await rejectDbAccessRequest(projectId, reviewId, comment);
    } else if (reviewType === "app_user") {
      if (approve) await approveDbAppUserRequest(projectId, reviewId, comment);
      else await rejectDbAppUserRequest(projectId, reviewId, comment);
    } else if (approve) await approveDbTicket(projectId, reviewId, comment);
    else await rejectDbTicket(projectId, reviewId, comment);
    message.success("已处理");
    setReviewOpen(false);
    void load();
  };

  const accessCols: ColumnsType<DbAccessRequest> = [
    { title: "实例", dataIndex: "instance_name" },
    { title: "库", dataIndex: "database_name" },
    {
      title: "表",
      dataIndex: "table_names",
      ellipsis: true,
      render: (v?: string[]) => (v?.length ? v.join(", ") : "整库"),
    },
    { title: "申请人", dataIndex: "requester_name" },
    { title: "理由", dataIndex: "reason", ellipsis: true },
    {
      title: "当前环节",
      dataIndex: "current_stage_name",
      render: (v, row) => {
        if (row.status === "approved") return "已完成";
        if (row.status === "rejected") return "已驳回";
        return v || "—";
      },
    },
    {
      title: "状态",
      dataIndex: "status",
      render: (v, row) => {
        const color = v === "approved" ? "green" : v === "rejected" ? "red" : "orange";
        let label = accessRequestStatusLabel(v);
        if (v === "pending" && row.mine_status === "mine_done") label = "待下一环节";
        return <Tag color={color}>{label}</Tag>;
      },
    },
    {
      title: "过期时间",
      dataIndex: "expires_at",
      render: (v?: string) => (v ? formatDateTime(v) : "永久"),
    },
    { title: "时间", dataIndex: "created_at", render: (v) => formatDateTime(v) },
    {
      title: "操作",
      render: (_, r) =>
        isAccessPending(r) ? (
          <Space>
            <Button size="small" type="primary" icon={<CheckOutlined />} onClick={() => openReview("access", r.id, true, r)} />
            <Button size="small" danger icon={<CloseOutlined />} onClick={() => openReview("access", r.id, false, r)} />
          </Space>
        ) : (
          <span style={{ color: "#999" }}>—</span>
        ),
    },
  ];

  const appUserCols: ColumnsType<DbAppUserRequest> = [
    { title: "实例", dataIndex: "instance_name" },
    {
      title: "类型",
      dataIndex: "apply_type",
      render: (v: string) => {
        const map: Record<string, string> = {
          new_user: "新建用户",
          add_priv: "追加权限",
          add_ip: "追加IP",
          revoke: "回收权限",
        };
        return map[v] ?? v;
      },
    },
    { title: "用户", render: (_, r) => `${r.mysql_user}@${r.mysql_host || "%"}` },
    { title: "库", dataIndex: "database_name", render: (v?: string) => v || "—" },
    { title: "权限", dataIndex: "privileges", ellipsis: true, render: (v?: string[]) => v?.join(", ") ?? "—" },
    { title: "申请人", dataIndex: "requester_name" },
    { title: "理由", dataIndex: "reason", ellipsis: true },
    {
      title: "状态",
      dataIndex: "status",
      render: (v, row) => {
        const color = v === "success" || v === "approved" ? "green" : v === "rejected" || v === "failed" ? "red" : "orange";
        let label = accessRequestStatusLabel(v);
        if (v === "pending" && row.mine_status === "mine_done") label = "待下一环节";
        return <Tag color={color}>{label}</Tag>;
      },
    },
    { title: "时间", dataIndex: "created_at", render: (v) => formatDateTime(v) },
    {
      title: "操作",
      render: (_, r) =>
        isAppUserPending(r) ? (
          <Space>
            <Button size="small" type="primary" icon={<CheckOutlined />} onClick={() => openReview("app_user", r.id, true)} />
            <Button size="small" danger icon={<CloseOutlined />} onClick={() => openReview("app_user", r.id, false)} />
          </Space>
        ) : (
          <span style={{ color: "#999" }}>—</span>
        ),
    },
  ];

  const ticketCols: ColumnsType<DbTicket> = [
    { title: "类型", dataIndex: "ticket_type", render: (v: string) => ticketTypeLabel(v) },
    { title: "实例", dataIndex: "instance_name" },
    { title: "风险", dataIndex: "risk_level", render: (v) => <Tag color={riskLevelColor(v ?? "")}>{riskLevelLabel(v ?? "")}</Tag> },
    { title: "提交人", dataIndex: "submitter_name" },
    { title: "SQL", dataIndex: "sql_excerpt", ellipsis: true, render: (v: string | undefined, r) => (
      <Space size={4}>
        <Typography.Text ellipsis style={{ maxWidth: 180 }}>{v || "—"}</Typography.Text>
        <Button type="link" size="small" style={{ padding: 0 }} onClick={() => openSqlView(r.id)}>
          查看
        </Button>
      </Space>
    ) },
    {
      title: "当前环节",
      dataIndex: "current_stage_name",
      render: (v, row) => {
        if (row.status === "success") return "已完成";
        if (row.status === "rejected") return "已驳回";
        return v || "—";
      },
    },
    {
      title: "状态",
      dataIndex: "status",
      render: (v, row) => {
        const color =
          v === "success" ? "green" : v === "rejected" || v === "failed" ? "red" : v === "pending_execution" ? "blue" : "orange";
        let label = ticketStatusLabel(v);
        if (v === "pending_approval" && row.mine_status === "mine_done") label = "待下一环节";
        return <Tag color={color}>{label}</Tag>;
      },
    },
    {
      title: "操作",
      render: (_, r) => (
        <Space>
          <Button size="small" icon={<EyeOutlined />} onClick={() => openSqlView(r.id)}>
            SQL
          </Button>
          {isTicketApprovalPending(r) ? (
            <>
              <Button size="small" type="primary" icon={<CheckOutlined />} onClick={() => openReview("ticket", r.id, true)} />
              <Button size="small" danger icon={<CloseOutlined />} onClick={() => openReview("ticket", r.id, false)} />
            </>
          ) : null}
          {isTicketExecutionPending(r) ? (
            <Button
              size="small"
              icon={<PlayCircleOutlined />}
              onClick={() =>
                void executeDbTicket(projectId!, r.id)
                  .then(() => {
                    message.success("执行成功");
                    void load();
                  })
                  .catch((e) => message.error(e instanceof Error ? e.message : "执行失败"))
              }
            >
              执行
            </Button>
          ) : null}
          {!isTicketApprovalPending(r) && !isTicketExecutionPending(r) ? <span style={{ color: "#999" }}>—</span> : null}
        </Space>
      ),
    },
  ];

  const total = tab === "access" ? accessTotal : tab === "app_user" ? appUserTotal : ticketTotal;
  const statusOptions = tab === "access" ? ACCESS_STATUS_OPTIONS : tab === "app_user" ? APP_USER_STATUS_OPTIONS : TICKET_STATUS_OPTIONS;

  return (
    <Card
      title={mode === "pending" ? "待审核" : "审批待办"}
      extra={
        <Space>
          <Select style={{ width: 200 }} value={projectId} options={projects.map((p) => ({ value: p.id, label: p.name }))} onChange={setProjectId} />
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
        </Space>
      }
    >
      <Space wrap style={{ marginBottom: 16 }}>
        <Segmented options={scopeOptions} value={mineScope} onChange={(v) => setMineScope(v as MineScope)} />
        <Select
          allowClear
          style={{ width: 160 }}
          placeholder="状态"
          value={status || undefined}
          options={statusOptions}
          onChange={(v) => setStatus(v ?? "")}
        />
      </Space>
      <Tabs
        activeKey={tab}
        onChange={(k) => setTab(k as TabKey)}
        items={[
          {
            key: "access",
            label: `权限申请 (${accessTotal})`,
            children: (
              <Table
                rowKey="id"
                loading={loading}
                columns={accessCols}
                dataSource={accessRows}
                pagination={{
                  current: page,
                  pageSize,
                  total,
                  showSizeChanger: true,
                  showTotal: (t) => `共 ${t} 条`,
                  onChange: (p, ps) => {
                    setPage(p);
                    setPageSize(ps);
                  },
                }}
              />
            ),
          },
          {
            key: "app_user",
            label: `应用用户 (${appUserTotal})`,
            children: (
              <Table
                rowKey="id"
                loading={loading}
                columns={appUserCols}
                dataSource={appUserRows}
                pagination={{
                  current: page,
                  pageSize,
                  total,
                  showSizeChanger: true,
                  showTotal: (t) => `共 ${t} 条`,
                  onChange: (p, ps) => {
                    setPage(p);
                    setPageSize(ps);
                  },
                }}
              />
            ),
          },
          {
            key: "ticket",
            label: `SQL 工单 (${ticketTotal})`,
            children: (
              <Table
                rowKey="id"
                loading={loading}
                columns={ticketCols}
                dataSource={ticketRows}
                pagination={{
                  current: page,
                  pageSize,
                  total,
                  showSizeChanger: true,
                  showTotal: (t) => `共 ${t} 条`,
                  onChange: (p, ps) => {
                    setPage(p);
                    setPageSize(ps);
                  },
                }}
              />
            ),
          },
        ]}
      />
      <Modal
        title={approve ? "审批通过" : "审批拒绝"}
        open={reviewOpen}
        onCancel={() => setReviewOpen(false)}
        onOk={() => void submitReview()}
        width={reviewType === "ticket" ? 720 : reviewType === "access" && approve ? 760 : 520}
      >
        {reviewType === "ticket" ? (
          <div style={{ marginBottom: 16 }}>{renderSqlPreview(reviewTicket, reviewTicketLoading)}</div>
        ) : null}
        {reviewType === "access" && reviewAccess ? (
          <Descriptions size="small" column={2} bordered style={{ marginBottom: 16 }}>
            <Descriptions.Item label="实例">{reviewAccess.instance_name || "—"}</Descriptions.Item>
            <Descriptions.Item label="申请人">{reviewAccess.requester_name}</Descriptions.Item>
            <Descriptions.Item label="数据库">{reviewAccess.database_name || "—"}</Descriptions.Item>
            <Descriptions.Item label="表">
              {reviewAccess.table_names?.length ? reviewAccess.table_names.join(", ") : "整库"}
            </Descriptions.Item>
            <Descriptions.Item label="申请理由" span={2}>
              {reviewAccess.reason || "—"}
            </Descriptions.Item>
            {approve ? (
              <Descriptions.Item label="申请人期望有效期" span={2}>
                {formatGrantPeriodSummary(expiresAtToGrantPeriod(reviewAccess.expires_at))}
              </Descriptions.Item>
            ) : null}
          </Descriptions>
        ) : null}
        <Form form={form} layout="vertical">
          {reviewType === "access" && approve && reviewAccess?.is_final_approval !== false ? (
            <Form.Item
              name="grant_period"
              label="授权有效期"
              rules={grantPeriodRules}
              extra="可在申请人期望基础上调整；审批通过后按此处日期生效"
            >
              <GrantValidityCalendarPicker />
            </Form.Item>
          ) : reviewType === "access" && approve ? (
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
              message={`当前为中间审批环节，授权有效期将保持申请人设定：${formatGrantPeriodSummary(expiresAtToGrantPeriod(reviewAccess?.expires_at))}`}
            />
          ) : null}
          <Form.Item name="comment" label="意见">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title="SQL 内容"
        open={sqlOpen}
        onCancel={() => setSqlOpen(false)}
        width={760}
        footer={
          <Space>
            <Button icon={<CopyOutlined />} onClick={() => void copySql(sqlTicket?.sql_text || sqlTicket?.sql_excerpt)}>
              复制 SQL
            </Button>
            {projectId && sqlTicket ? (
              <Link to={`/dbmgmt/workflow/tickets/${sqlTicket.id}?project=${projectId}`}>
                <Button type="primary">工单详情</Button>
              </Link>
            ) : null}
            <Button onClick={() => setSqlOpen(false)}>关闭</Button>
          </Space>
        }
      >
        {sqlLoading ? <Spin /> : renderSqlPreview(sqlTicket)}
      </Modal>
    </Card>
  );
}

export function DbmgmtWorkflowPendingPage() {
  return <DbmgmtTodoPage mode="pending" />;
}
