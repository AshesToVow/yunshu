import { LinkOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Select, Space, Table, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import { getProjects, type ProjectItem } from "../services/projects";
import { listWorkflowTickets, type WorkflowTicketRow } from "../services/workflow";
import { formatDateTime } from "../utils/format";

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

function statusTag(status: string) {
  switch (status) {
    case "pending":
      return <Tag color="processing">待审批</Tag>;
    case "approved":
      return <Tag color="success">已通过</Tag>;
    case "rejected":
      return <Tag color="error">已驳回</Tag>;
    case "cancelled":
      return <Tag>已取消</Tag>;
    default:
      return <Tag>{status || "—"}</Tag>;
  }
}

function deepLink(row: WorkflowTicketRow) {
  switch (row.ref_type) {
    case "db_sql_ticket":
      return `/dbmgmt/workflow/tickets/${row.ref_id}?project=${row.project_id}`;
    case "db_access_request":
      return `/dbmgmt/apply/query?project=${row.project_id}&highlight=${row.ref_id}`;
    case "db_app_user_request":
      return `/dbmgmt/apply/app-user?project=${row.project_id}&highlight=${row.ref_id}`;
    case "cicd_release_run":
      return `/cicd/release-records?project=${row.project_id}&release=${row.ref_id}`;
    case "alert_event":
      return `/alert-events?highlight=${row.ref_id}`;
    default:
      return `/workflow/inbox?project=${row.project_id}`;
  }
}

export function WorkflowTicketsPage() {
  const [searchParams] = useSearchParams();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number | undefined>(() => {
    const n = Number(searchParams.get("project") || 0);
    return n > 0 ? n : undefined;
  });
  const [domain, setDomain] = useState(searchParams.get("domain") || "");
  const [status, setStatus] = useState(searchParams.get("status") || "");
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<WorkflowTicketRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => setProjects(res.list ?? []));
  }, []);

  const projectNameMap = useMemo(() => {
    const m = new Map<number, string>();
    for (const p of projects) m.set(p.id, p.name);
    return m;
  }, [projects]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listWorkflowTickets({
        page,
        page_size: pageSize,
        domain: domain || undefined,
        project_id: projectId,
        status: status || undefined,
      });
      setRows(res.list ?? []);
      setTotal(res.total ?? 0);
    } finally {
      setLoading(false);
    }
  }, [domain, page, pageSize, projectId, status]);

  useEffect(() => {
    void load();
  }, [load]);

  const columns: ColumnsType<WorkflowTicketRow> = [
    {
      title: "域",
      dataIndex: "domain",
      width: 88,
      render: (v: string) => <Tag>{domainLabel(v)}</Tag>,
    },
    { title: "类型", dataIndex: "ticket_type", width: 120 },
    {
      title: "项目",
      dataIndex: "project_id",
      width: 120,
      render: (id: number) => projectNameMap.get(id) ?? (id ? `#${id}` : "—"),
    },
    { title: "标题", dataIndex: "title", ellipsis: true },
    { title: "状态", dataIndex: "status", width: 100, render: (v: string) => statusTag(v) },
    {
      title: "创建时间",
      dataIndex: "created_at",
      width: 168,
      render: (v: string) => formatDateTime(v),
    },
    {
      title: "操作",
      key: "actions",
      width: 100,
      render: (_, row) => (
        <Link to={deepLink(row)}>
          <Button type="link" size="small" icon={<LinkOutlined />}>
            详情
          </Button>
        </Link>
      ),
    },
  ];

  return (
    <div>
      <PageTelemetryHeader label="Workflow" title="工单列表" subtitle="跨域工单历史；审批动作请到「我的待办」" />
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
            value={domain}
            onChange={(v) => {
              setDomain(v);
              setPage(1);
            }}
            options={DOMAIN_OPTIONS}
          />
          <Select
            allowClear
            placeholder="全部状态"
            style={{ width: 140 }}
            value={status || undefined}
            onChange={(v) => {
              setStatus(v ?? "");
              setPage(1);
            }}
            options={[
              { label: "待审批", value: "pending" },
              { label: "已通过", value: "approved" },
              { label: "已驳回", value: "rejected" },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
          <Typography.Text type="secondary">业务执行（SQL 执行等）仍在各业务详情页完成</Typography.Text>
        </Space>
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: 1000 }}
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
    </div>
  );
}
