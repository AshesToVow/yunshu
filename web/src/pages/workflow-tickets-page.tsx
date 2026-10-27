import { LinkOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Empty, Select, Space, Table, Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { ApprovalCenterNav } from "../components/approval-center-nav";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import { getProjects, type ProjectItem } from "../services/projects";
import { listWorkflowTickets, type WorkflowTicketRow } from "../services/workflow";
import { formatDateTime } from "../utils/format";
import {
  WORKFLOW_DOMAIN_OPTIONS,
  workflowBusinessDeepLink,
  workflowDomainColor,
  workflowDomainLabel,
  workflowStatusColor,
  workflowStatusLabel,
  workflowTicketTypeLabel,
} from "../utils/workflow-labels";

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
  const loadSeqRef = useRef(0);

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 })
      .then((res) => setProjects(res.list ?? []))
      .catch(() => {
        /* 项目列表加载失败不阻断工单列表 */
      });
  }, []);

  useEffect(() => {
    const d = searchParams.get("domain");
    if (d !== null && d !== domain) setDomain(d);
    const p = Number(searchParams.get("project") || 0);
    const nextProject = p > 0 ? p : undefined;
    if (nextProject !== undefined && nextProject !== projectId) {
      setProjectId(nextProject);
      setPage(1);
    }
    const st = searchParams.get("status");
    if (st !== null && st !== status) setStatus(st);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  const projectNameMap = useMemo(() => {
    const m = new Map<number, string>();
    for (const p of projects) m.set(p.id, p.name);
    return m;
  }, [projects]);

  const load = useCallback(async () => {
    const seq = ++loadSeqRef.current;
    setLoading(true);
    try {
      const res = await listWorkflowTickets({
        page,
        page_size: pageSize,
        domain: domain || undefined,
        project_id: projectId,
        status: status || undefined,
      });
      if (seq !== loadSeqRef.current) return;
      const nextTotal = res.total ?? 0;
      setRows(res.list ?? []);
      setTotal(nextTotal);
      if (nextTotal > 0 && page > Math.ceil(nextTotal / pageSize)) {
        setPage(1);
      }
    } catch {
      /* http 拦截器已 toast */
    } finally {
      if (seq === loadSeqRef.current) setLoading(false);
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
      render: (v: string) => <Tag color={workflowDomainColor(v)}>{workflowDomainLabel(v)}</Tag>,
    },
    {
      title: "类型",
      dataIndex: "ticket_type",
      width: 120,
      render: (v: string) => workflowTicketTypeLabel(v),
    },
    {
      title: "项目",
      dataIndex: "project_id",
      width: 120,
      render: (id: number) => projectNameMap.get(id) ?? (id ? `#${id}` : "—"),
    },
    { title: "标题", dataIndex: "title", ellipsis: true },
    {
      title: "状态",
      dataIndex: "status",
      width: 100,
      render: (v: string) => <Tag color={workflowStatusColor(v)}>{workflowStatusLabel(v)}</Tag>,
    },
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
      fixed: "right",
      className: "yunshu-table-actions-cell",
      render: (_, row) => (
        <Link to={workflowBusinessDeepLink(row)}>
          <Button type="link" size="small" icon={<LinkOutlined />}>
            详情
          </Button>
        </Link>
      ),
    },
  ];

  return (
    <div>
      <PageTelemetryHeader
        label="审批中心"
        title="全部工单"
        subtitle="跨域审计列表；审批请到「我的待办」，执行请到业务详情"
        meta={[`共 ${total}`]}
      />
      <ApprovalCenterNav />
      <Card className="table-card">
        <div className="toolbar">
          <Space wrap>
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
              options={[...WORKFLOW_DOMAIN_OPTIONS]}
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
                { label: "已关闭", value: "closed" },
                { label: "已取消", value: "cancelled" },
              ]}
            />
            <Button icon={<ReloadOutlined />} onClick={() => void load()} disabled={loading}>
              刷新
            </Button>
          </Space>
        </div>
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: 1000 }}
          locale={{ emptyText: <Empty description="暂无工单" /> }}
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
