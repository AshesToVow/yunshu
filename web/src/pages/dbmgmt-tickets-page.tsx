import { ReloadOutlined, SearchOutlined } from "@ant-design/icons";
import { Button, Card, Input, Select, Space, Table, Tag } from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { auditModeLabel, ticketTypeLabel } from "../components/dbmgmt/dbmgmt-ui-shared";
import { listDbTickets, type DbTicket } from "../services/dbmgmt";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";
import { ticketStatusLabel } from "../utils/dbmgmt-labels";

function ticketTypeColor(t?: string) {
  if (t === "sql_import") return "purple";
  if (t === "sql_execute") return "blue";
  return "cyan";
}

function statusTagColor(s?: string) {
  if (s === "success") return "green";
  if (s === "executing") return "orange";
  if (s === "failed" || s === "rejected") return "red";
  if (s === "pending_execution") return "blue";
  return "default";
}

const TICKET_TYPE_OPTIONS = [
  { value: "", label: "全部类型" },
  { value: "sql_execute", label: "SQL上线申请" },
  { value: "sql_import", label: "SQL文件上线" },
];

export function DbmgmtTicketsPage({ variant = "list" }: { variant?: "list" | "history" }) {
  const navigate = useNavigate();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [rows, setRows] = useState<DbTicket[]>([]);
  const [ticketType, setTicketType] = useState("");
  const [keyword, setKeyword] = useState("");
  const [searchText, setSearchText] = useState("");
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
      if (res.list?.length) setProjectId(res.list[0].id);
    });
  }, []);

  const load = useCallback(async () => {
    if (!projectId) return;
    const res = await listDbTickets(projectId, {
      page,
      page_size: 10,
      ...(ticketType ? { ticket_type: ticketType } : {}),
    });
    setRows(res.list ?? []);
    setTotal(res.total ?? 0);
  }, [projectId, page, ticketType]);

  useEffect(() => {
    void load();
  }, [load]);

  const filteredRows = useMemo(() => {
    const kw = searchText.trim().toLowerCase();
    if (!kw) return rows;
    return rows.filter((r) => (r.reason || r.sql_excerpt || "").toLowerCase().includes(kw));
  }, [rows, searchText]);

  const reset = () => {
    setTicketType("");
    setKeyword("");
    setSearchText("");
    setPage(1);
    void load();
  };

  return (
    <Card title={variant === "history" ? "历史工单" : "SQL 工单"}>
      <Space style={{ marginBottom: 16 }} wrap>
        <Button icon={<ReloadOutlined />} onClick={reset}>
          重置
        </Button>
        <Select
          style={{ width: 180 }}
          placeholder="选择工单类型搜索"
          value={ticketType || undefined}
          allowClear
          options={TICKET_TYPE_OPTIONS}
          onChange={(v) => setTicketType(v ?? "")}
        />
        <Input
          placeholder="请输入任务名称进行搜索"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          onPressEnter={() => setSearchText(keyword)}
          style={{ width: 240 }}
        />
        <Button type="primary" icon={<SearchOutlined />} onClick={() => setSearchText(keyword)}>
          搜索
        </Button>
        <Select style={{ width: 160 }} value={projectId} options={projects.map((p) => ({ value: p.id, label: p.name }))} onChange={setProjectId} />
      </Space>
      <Table
        rowKey="id"
        dataSource={filteredRows}
        pagination={{
          current: page,
          pageSize: 10,
          total,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (p) => setPage(p),
        }}
        columns={[
          {
            title: "任务名称",
            render: (_, r) => r.reason || r.sql_excerpt || `工单 #${r.id}`,
            ellipsis: true,
          },
          {
            title: "类型",
            dataIndex: "ticket_type",
            width: 140,
            render: (v: string) => <Tag color={ticketTypeColor(v)}>{ticketTypeLabel(v)}</Tag>,
          },
          { title: "提交人", dataIndex: "submitter_name", width: 100 },
          {
            title: "是否审核",
            width: 100,
            render: (_, r) => <Tag color={r.audit_mode === "manual" ? "orange" : "blue"}>{auditModeLabel(r.audit_mode)}</Tag>,
          },
          { title: "提交时间", dataIndex: "created_at", width: 170, render: (v) => formatDateTime(v) },
          {
            title: "完成时间",
            width: 170,
            render: (_, r) =>
              r.status === "success" || r.status === "failed" ? formatDateTime(r.updated_at ?? r.created_at) : "—",
          },
          {
            title: "任务状态",
            dataIndex: "status",
            width: 100,
            render: (v: string) => <Tag color={statusTagColor(v)}>{ticketStatusLabel(v)}</Tag>,
          },
          {
            title: "操作",
            width: 200,
            render: (_, r) => (
              <Space>
                <Button size="small" onClick={() => navigate(`/dbmgmt/workflow/tickets/${r.id}?project=${projectId}&tab=info`)}>
                  查看
                </Button>
                <Button size="small" type="primary" style={{ background: "#52c41a", borderColor: "#52c41a" }} onClick={() => navigate(`/dbmgmt/workflow/tickets/${r.id}?project=${projectId}&tab=log`)}>
                  详情
                </Button>
              </Space>
            ),
          },
        ]}
      />
    </Card>
  );
}

export function DbmgmtWorkflowHistoryPage() {
  return <DbmgmtTicketsPage variant="history" />;
}
