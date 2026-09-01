// @ts-nocheck
import { CheckOutlined, CloseOutlined, DownloadOutlined, PlayCircleOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Descriptions, Input, Modal, Select, Space, Table, Tabs, Tag, message } from "antd";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from '@umijs/max';
import { DbmgmtSectionTitle, auditModeLabel, formatInstanceLabel, ticketTypeLabel } from "../components/dbmgmt/dbmgmt-ui-shared";
import {
  approveDbTicket,
  controlDbTicketOsc,
  executeDbTicket,
  getDbTicket,
  getDbTicketRollback,
  listDbInstances,
  listDbTicketOscJobs,
  listDbTicketSteps,
  rejectDbTicket,
  submitDbRollbackTicket,
  type DbInstance,
  type DbOscJob,
  type DbRollbackItem,
  type DbTicket,
  type DbTicketStep,
} from "../services/dbmgmt";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";
import { envLabel, ticketStatusLabel } from "../utils/dbmgmt-labels";

type ReviewRow = {
  order_id?: number;
  stage?: string;
  error_level?: number;
  stage_status?: string;
  error_message?: string;
  sql?: string;
  affected_rows?: number;
  execute_time?: string;
  backup_time?: string;
};

function parseReviewJSON(raw?: string): ReviewRow[] {
  if (!raw) return [];
  try {
    const obj = JSON.parse(raw) as { rows?: ReviewRow[] };
    return obj.rows ?? [];
  } catch {
    return [];
  }
}

function downloadTextFile(filename: string, content: string) {
  const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

function syntaxLabel(v?: number) {
  if (v === 1) return "DDL";
  if (v === 2) return "DML";
  return "其他";
}

function buildTaskLogs(ticket: DbTicket, steps: DbTicketStep[]) {
  const logs: { op: string; user: string; time: string; info: string }[] = [];
  logs.push({
    op: "提交",
    user: ticket.submitter_name,
    time: formatDateTime(ticket.created_at),
    info: `等待审批，审批流程：${ticket.current_stage_name || "默认流程"}`,
  });
  for (const step of steps) {
    if (step.status === "approved" && step.reviewed_at) {
      logs.push({
        op: "审批通过",
        user: step.reviewer_name || "—",
        time: formatDateTime(step.reviewed_at),
        info: `审批备注：${step.review_comment || "—"}，审批完成，等待执行`,
      });
    }
    if (step.status === "rejected" && step.reviewed_at) {
      logs.push({
        op: "审批驳回",
        user: step.reviewer_name || "—",
        time: formatDateTime(step.reviewed_at),
        info: step.review_comment || "已驳回",
      });
    }
  }
  if (ticket.status === "executing" || ticket.status === "success" || ticket.status === "failed") {
    logs.push({
      op: "执行工单",
      user: ticket.submitter_name,
      time: ticket.updated_at ? formatDateTime(ticket.updated_at) : formatDateTime(ticket.created_at),
      info: "工单开始执行",
    });
  }
  if (ticket.status === "success" || ticket.status === "failed") {
    logs.push({
      op: "执行结束",
      user: ticket.submitter_name,
      time: ticket.updated_at ? formatDateTime(ticket.updated_at) : "—",
      info: `执行结果：${ticketStatusLabel(ticket.status)}`,
    });
  }
  return logs.sort((a, b) => (a.time < b.time ? 1 : -1));
}

export function DbmgmtTicketDetailPage() {
  const { ticketId: ticketIdParam } = useParams<{ ticketId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const ticketId = Number(ticketIdParam);
  const tab = searchParams.get("tab") ?? "info";
  const projectFromUrl = Number(searchParams.get("project")) || undefined;
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [ticket, setTicket] = useState<DbTicket>();
  const [steps, setSteps] = useState<DbTicketStep[]>([]);
  const [instances, setInstances] = useState<DbInstance[]>([]);
  const [rollbackRows, setRollbackRows] = useState<DbRollbackItem[]>([]);
  const [rollbackOpen, setRollbackOpen] = useState(false);
  const [rollbackLoading, setRollbackLoading] = useState(false);
  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [oscJobs, setOscJobs] = useState<DbOscJob[]>([]);
  const rejectCommentRef = useRef("");

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
      if (projectFromUrl) {
        setProjectId(projectFromUrl);
      } else if (res.list?.length) {
        setProjectId(res.list[0].id);
      }
    });
  }, [projectFromUrl]);

  const load = useCallback(async () => {
    if (!projectId || !ticketId) return;
    setLoading(true);
    try {
      const [t, st, inst] = await Promise.all([
        getDbTicket(projectId, ticketId),
        listDbTicketSteps(projectId, ticketId).catch(() => []),
        listDbInstances(projectId, { page: 1, page_size: 200 }),
      ]);
      setTicket(t);
      setSteps(st ?? []);
      setInstances(inst.list ?? []);
      if (t.project_id && t.project_id !== projectId) {
        setProjectId(t.project_id);
      }
      if (t.is_backup && (t.status === "success" || t.status === "failed")) {
        try {
          const rb = await getDbTicketRollback(projectId, ticketId);
          setRollbackRows(rb ?? []);
        } catch {
          setRollbackRows([]);
        }
      } else {
        setRollbackRows([]);
      }
    } catch (e) {
      setTicket(undefined);
      message.error(e instanceof Error ? e.message : "加载工单失败");
    } finally {
      setLoading(false);
    }
  }, [projectId, ticketId]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    const showOsc = ticket?.status === "executing" || ticket?.status === "success" || ticket?.status === "failed";
    if (!projectId || !ticketId || !showOsc) {
      setOscJobs([]);
      return;
    }
    void listDbTicketOscJobs(projectId, ticketId)
      .then((jobs) => setOscJobs(jobs ?? []))
      .catch(() => setOscJobs([]));
  }, [projectId, ticketId, ticket?.status]);

  const handleApprove = async () => {
    if (!projectId || !ticketId) return;
    setActionLoading(true);
    try {
      await approveDbTicket(projectId, ticketId);
      message.success("已审批通过");
      void load();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "审批失败");
    } finally {
      setActionLoading(false);
    }
  };

  const handleReject = () => {
    if (!projectId || !ticketId) return;
    rejectCommentRef.current = "";
    Modal.confirm({
      title: "驳回工单",
      content: (
        <Input.TextArea
          rows={3}
          placeholder="驳回原因（可选）"
          onChange={(e) => {
            rejectCommentRef.current = e.target.value;
          }}
        />
      ),
      okText: "确认驳回",
      okButtonProps: { danger: true },
      onOk: async () => {
        setActionLoading(true);
        try {
          await rejectDbTicket(projectId, ticketId, rejectCommentRef.current || undefined);
          message.success("已驳回");
          void load();
        } catch (e) {
          message.error(e instanceof Error ? e.message : "驳回失败");
          throw e;
        } finally {
          setActionLoading(false);
        }
      },
    });
  };

  const handleExecute = async () => {
    if (!projectId || !ticketId) return;
    setActionLoading(true);
    try {
      await executeDbTicket(projectId, ticketId);
      message.success("已开始执行");
      void load();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "执行失败");
    } finally {
      setActionLoading(false);
    }
  };

  const handleSubmitRollback = async () => {
    if (!projectId || !ticketId) return;
    setActionLoading(true);
    try {
      const res = await submitDbRollbackTicket(projectId, ticketId);
      message.success(res.message || "回滚工单已提交");
      if (res.ticket_id) {
        navigate(`/dbmgmt/workflow/tickets/${res.ticket_id}?project=${projectId}`);
      } else {
        void load();
      }
    } catch (e) {
      message.error(e instanceof Error ? e.message : "提交回滚失败");
    } finally {
      setActionLoading(false);
    }
  };

  const onProjectChange = (id: number) => {
    setProjectId(id);
    const next = new URLSearchParams(searchParams);
    next.set("project", String(id));
    setSearchParams(next);
  };

  const loadRollback = async () => {
    if (!projectId || !ticketId) return [];
    setRollbackLoading(true);
    try {
      const rb = await getDbTicketRollback(projectId, ticketId);
      setRollbackRows(rb ?? []);
      return rb ?? [];
    } catch (e) {
      message.error(e instanceof Error ? e.message : "加载回滚 SQL 失败");
      return [];
    } finally {
      setRollbackLoading(false);
    }
  };

  const executeRows = useMemo(() => parseReviewJSON(ticket?.execute_json), [ticket?.execute_json]);
  const reviewRows = useMemo(() => parseReviewJSON(ticket?.review_json), [ticket?.review_json]);
  const taskLogs = useMemo(() => (ticket ? buildTaskLogs(ticket, steps) : []), [ticket, steps]);
  const pendingStep = useMemo(() => steps.find((s) => s.status === "pending"), [steps]);
  const canApprove = ticket?.mine_status === "mine_pending" && ticket?.status === "pending_approval";
  const canExecute = ticket?.mine_status === "mine_pending" && ticket?.status === "pending_execution";
  const canSubmitRollback = ticket?.status === "success" && ticket?.is_backup && rollbackRows.length > 0;

  const downloadSqlFile = () => {
    const text = ticket?.sql_text?.trim() || ticket?.sql_excerpt?.trim();
    if (!text) {
      message.warning("暂无 SQL 文件内容");
      return;
    }
    const rawName = ticket?.sql_file_ref?.trim() || `ticket-${ticketId}.sql`;
    const filename = rawName.toLowerCase().endsWith(".sql") ? rawName : `${rawName}.sql`;
    downloadTextFile(filename, text);
    message.success("SQL 文件已下载");
  };

  const downloadExecLog = () => {
    if (!ticket) return;
    const lines: string[] = [
      `# 工单 #${ticket.id} 执行日志`,
      `提交人: ${ticket.submitter_name}`,
      `数据库: ${ticket.database_name || "—"}`,
      `状态: ${ticketStatusLabel(ticket.status)}`,
      "",
      "== 执行明细 ==",
    ];
    if (executeRows.length) {
      for (const r of executeRows) {
        lines.push(`[${r.order_id ?? "-"}] ${r.stage_status || "—"}`);
        lines.push(String(r.sql || ""));
        if (r.error_message) lines.push(`error: ${r.error_message}`);
        lines.push("");
      }
    } else {
      lines.push(ticket.sql_excerpt || "—");
    }
    lines.push("== 操作日志 ==");
    for (const log of taskLogs) {
      lines.push(`[${log.time}] ${log.op} / ${log.user}: ${log.info}`);
    }
    if (ticket.execute_json) {
      lines.push("", "== execute_json ==");
      lines.push(ticket.execute_json);
    }
    downloadTextFile(`ticket-${ticketId}-exec.log`, lines.join("\n"));
    message.success("执行日志已下载");
  };

  const openRollbackView = async () => {
    const rows = await loadRollback();
    if (rows.length) {
      setRollbackOpen(true);
      return;
    }
    message.warning(
      "暂无回滚 SQL。请确认 goInception 已开启备份（--backup=1），且实例连接账号对备份库（如 10_10_10_103_3306_test）有 SELECT 权限。",
    );
  };

  const downloadRollback = async () => {
    let rows = rollbackRows;
    if (!rows.length) {
      rows = await loadRollback();
    }
    if (!rows.length) {
      message.warning(
        "暂无回滚 SQL。可在 MySQL 上执行 SHOW DATABASES LIKE '%3306_test%' 确认备份库是否存在，并检查 goInception backup 配置。",
      );
      return;
    }
    const sql = rows.map((r) => r.rollback_sql).filter(Boolean).join("\n\n");
    downloadTextFile(`rollback-ticket-${ticketId}.sql`, sql);
    message.success("回滚 SQL 已下载");
  };

  const handleOscControl = async (sqlsha1: string, command: "kill" | "pause" | "resume") => {
    if (!projectId || !ticketId) return;
    setActionLoading(true);
    try {
      await controlDbTicketOsc(projectId, ticketId, sqlsha1, command);
      message.success(command === "kill" ? "已终止 OSC" : command === "pause" ? "已暂停 OSC" : "已恢复 OSC");
      const jobs = await listDbTicketOscJobs(projectId, ticketId);
      setOscJobs(jobs ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "OSC 操作失败");
    } finally {
      setActionLoading(false);
    }
  };

  const instance = instances.find((i) => i.id === ticket?.instance_id);
  const projectName = projects.find((p) => p.id === (ticket?.project_id ?? projectId))?.name ?? "—";

  const infoTab = ticket ? (
    <div>
      <DbmgmtSectionTitle>基本信息</DbmgmtSectionTitle>
      <Descriptions bordered column={3} size="small" style={{ marginBottom: 24 }}>
        <Descriptions.Item label="工单ID">{ticket.id}</Descriptions.Item>
        <Descriptions.Item label="工单名称">{ticket.reason || ticket.sql_excerpt || "—"}</Descriptions.Item>
        <Descriptions.Item label="工单类型">
          <Tag color="blue">{ticketTypeLabel(ticket.ticket_type)}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="申请人">{ticket.submitter_name}</Descriptions.Item>
        <Descriptions.Item label="是否需要审核">
          <Tag color={ticket.audit_mode === "manual" ? "orange" : "green"}>{auditModeLabel(ticket.audit_mode)}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="审批流程">
          <Tag color="blue">{ticket.current_stage_name || "默认流程"}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="提交时间">{formatDateTime(ticket.created_at)}</Descriptions.Item>
        <Descriptions.Item label="开始时间">{ticket.updated_at && ticket.status !== "pending_approval" ? formatDateTime(ticket.updated_at) : "—"}</Descriptions.Item>
        <Descriptions.Item label="完成时间">{ticket.status === "success" || ticket.status === "failed" ? formatDateTime(ticket.updated_at ?? ticket.created_at) : "—"}</Descriptions.Item>
        <Descriptions.Item label="当前审批">
          {pendingStep ? `${pendingStep.stage_name}${pendingStep.reviewer_name ? ` · ${pendingStep.reviewer_name}` : ""}` : "—"}
        </Descriptions.Item>
        <Descriptions.Item label="当前状态">
          <Tag color="blue">{ticketStatusLabel(ticket.status)}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="资源组">{projectName}</Descriptions.Item>
      </Descriptions>

      <DbmgmtSectionTitle>任务信息</DbmgmtSectionTitle>
      <Descriptions bordered column={3} size="small" style={{ marginBottom: 24 }}>
        <Descriptions.Item label="项目名称">{projectName}</Descriptions.Item>
        <Descriptions.Item label="环境信息">{envLabel(instance?.env)}</Descriptions.Item>
        <Descriptions.Item label="实例别名">
          <Tag color="blue">{instance?.name || "—"}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="实例名称">{instance ? formatInstanceLabel(instance) : ticket.instance_name}</Descriptions.Item>
        <Descriptions.Item label="数据库名">{ticket.database_name || "—"}</Descriptions.Item>
        <Descriptions.Item label="sql类型">{syntaxLabel(ticket.syntax_type)}</Descriptions.Item>
        <Descriptions.Item label="是否备份">{ticket.is_backup ? "是" : "否"}</Descriptions.Item>
        <Descriptions.Item label="审核方式">
          <Tag color={ticket.audit_mode === "manual" ? "orange" : "green"}>{auditModeLabel(ticket.audit_mode)}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="sql格式">{ticket.sql_file_ref ? `文件（${ticket.sql_file_ref}）` : "文本"}</Descriptions.Item>
      </Descriptions>

      <DbmgmtSectionTitle>审核结果</DbmgmtSectionTitle>
      <Table
        size="small"
        rowKey={(_, i) => `review-${i}`}
        dataSource={reviewRows.length ? reviewRows : []}
        locale={{ emptyText: "暂无审核明细（可能未走 goInception 或尚未审核）" }}
        pagination={false}
        columns={[
          { title: "序号", dataIndex: "order_id", width: 60, render: (v, _, i) => v ?? i + 1 },
          { title: "阶段", dataIndex: "stage", width: 100 },
          { title: "SQL内容", dataIndex: "sql", ellipsis: true },
          { title: "风险级别", dataIndex: "error_level", width: 90 },
          { title: "阶段状态", dataIndex: "stage_status", width: 100 },
          { title: "错误信息", dataIndex: "error_message", ellipsis: true, render: (v?: string) => v || "—" },
        ]}
        style={{ marginBottom: 24 }}
      />

      <DbmgmtSectionTitle>执行结果</DbmgmtSectionTitle>
      <Table
        size="small"
        rowKey={(_, i) => String(i)}
        dataSource={executeRows.length ? executeRows : [{ order_id: 1, sql: ticket.sql_file_ref ? "详见SQL文件" : ticket.sql_excerpt, stage_status: ticket.status }]}
        pagination={false}
        columns={[
          { title: "ID", dataIndex: "order_id", width: 60, render: (v, _, i) => v ?? i + 1 },
          { title: "SQL内容", dataIndex: "sql", ellipsis: true },
          {
            title: "执行状态",
            dataIndex: "stage_status",
            width: 100,
            render: (v?: string) => <Tag color={ticket.status === "success" ? "blue" : "default"}>{v || ticket.status}</Tag>,
          },
          { title: "影响行数", dataIndex: "affected_rows", width: 90, render: (v?: number) => (v != null ? v : "—") },
          { title: "错误信息", dataIndex: "error_message", ellipsis: true, render: (v?: string) => v || "—" },
          { title: "执行耗时", dataIndex: "execute_time", width: 90, render: (v?: string) => v || "—" },
          { title: "备份耗时", dataIndex: "backup_time", width: 90, render: (v?: string) => v || "—" },
          { title: "当前阶段", width: 100, render: () => ticketStatusLabel(ticket.status) },
        ]}
        style={{ marginBottom: 24 }}
      />

      {oscJobs.length ? (
        <>
          <DbmgmtSectionTitle>在线 DDL (OSC)</DbmgmtSectionTitle>
          <Table
            size="small"
            rowKey="sqlsha1"
            dataSource={oscJobs}
            pagination={false}
            columns={[
              { title: "序号", dataIndex: "order_id", width: 60 },
              { title: "阶段", dataIndex: "stage", width: 120 },
              { title: "状态", dataIndex: "stage_status", width: 100 },
              { title: "SQL", dataIndex: "sql", ellipsis: true },
              {
                title: "操作",
                width: 200,
                render: (_, row) => (
                  <Space size={4}>
                    <Button size="small" loading={actionLoading} onClick={() => void handleOscControl(row.sqlsha1, "pause")}>
                      暂停
                    </Button>
                    <Button size="small" loading={actionLoading} onClick={() => void handleOscControl(row.sqlsha1, "resume")}>
                      恢复
                    </Button>
                    <Button size="small" danger loading={actionLoading} onClick={() => void handleOscControl(row.sqlsha1, "kill")}>
                      终止
                    </Button>
                  </Space>
                ),
              },
            ]}
            style={{ marginBottom: 24 }}
          />
        </>
      ) : null}

      {(ticket.sql_file_ref || ticket.sql_text) ? (
        <>
          <DbmgmtSectionTitle>SQL文件信息</DbmgmtSectionTitle>
          <Space style={{ marginBottom: 24 }}>
            <Button type="link" style={{ color: "#52c41a", padding: 0 }} onClick={downloadSqlFile}>
              SQL文件下载
            </Button>
            <Button type="link" style={{ color: "#52c41a", padding: 0 }} onClick={downloadExecLog}>
              执行日志下载
            </Button>
          </Space>
        </>
      ) : null}

      {ticket.is_backup ? (
        <>
          <DbmgmtSectionTitle>回滚文件信息</DbmgmtSectionTitle>
          <Space direction="vertical" style={{ marginBottom: 24 }}>
            <Button
              type="primary"
              style={{ background: "#52c41a", borderColor: "#52c41a" }}
              loading={rollbackLoading}
              onClick={() => void openRollbackView()}
            >
              查看回滚
            </Button>
            {canSubmitRollback ? (
              <Button loading={actionLoading} onClick={() => void handleSubmitRollback()}>
                提交回滚工单
              </Button>
            ) : null}
            <Button type="link" icon={<DownloadOutlined />} style={{ color: "#52c41a", padding: 0 }} onClick={() => void downloadRollback()}>
              回滚信息下载
            </Button>
          </Space>
        </>
      ) : null}

      <DbmgmtSectionTitle>操作信息</DbmgmtSectionTitle>
      <Alert type="info" message={`执行结果：${ticketStatusLabel(ticket.status)}`} />
    </div>
  ) : null;

  const logTab = (
    <>
      <DbmgmtSectionTitle>任务日志</DbmgmtSectionTitle>
      <Table
        rowKey={(_, i) => String(i)}
        dataSource={taskLogs}
        pagination={{ pageSize: 10, showTotal: (t) => `共 ${t} 条` }}
        columns={[
          { title: "操作", dataIndex: "op", width: 120 },
          { title: "提交人", dataIndex: "user", width: 100 },
          { title: "操作时间", dataIndex: "time", width: 180 },
          { title: "操作信息", dataIndex: "info" },
        ]}
      />
    </>
  );

  return (
    <Card
      loading={loading}
      title="工单任务详情"
      extra={
        <Space wrap>
          {canApprove ? (
            <>
              <Button type="primary" icon={<CheckOutlined />} loading={actionLoading} onClick={() => void handleApprove()}>
                审批通过
              </Button>
              <Button danger icon={<CloseOutlined />} loading={actionLoading} onClick={handleReject}>
                驳回
              </Button>
            </>
          ) : null}
          {canExecute ? (
            <Button type="primary" icon={<PlayCircleOutlined />} loading={actionLoading} onClick={() => void handleExecute()}>
              执行工单
            </Button>
          ) : null}
          <Select style={{ width: 180 }} value={projectId} options={projects.map((p) => ({ value: p.id, label: p.name }))} onChange={onProjectChange} />
          <Button icon={<ReloadOutlined />} onClick={() => void load()} />
          <Button onClick={() => navigate(projectId ? `/dbmgmt/workflow/history?project=${projectId}` : "/dbmgmt/workflow/history")}>返回历史工单</Button>
        </Space>
      }
    >
      {ticket ? (
        <Tabs
          activeKey={tab}
          onChange={(k) => {
            const next = new URLSearchParams(searchParams);
            next.set("tab", k);
            if (projectId) next.set("project", String(projectId));
            setSearchParams(next);
          }}
          items={[
            { key: "info", label: "任务信息", children: infoTab },
            { key: "log", label: "任务日志", children: logTab },
          ]}
        />
      ) : !loading ? (
        <Alert type="warning" message="工单不存在或无权查看，请确认右上角项目选择与工单所属项目一致" />
      ) : null}

      <Modal title={`回滚 SQL · 工单 #${ticketId}`} open={rollbackOpen} onCancel={() => setRollbackOpen(false)} footer={null} width={900}>
        <Table
          size="small"
          rowKey={(_, i) => String(i)}
          dataSource={rollbackRows}
          pagination={false}
          columns={[
            { title: "原 SQL", dataIndex: "original_sql", ellipsis: true },
            {
              title: "回滚 SQL",
              dataIndex: "rollback_sql",
              render: (v: string) => <pre style={{ margin: 0, whiteSpace: "pre-wrap", maxHeight: 120, overflow: "auto" }}>{v}</pre>,
            },
          ]}
        />
        <div style={{ marginTop: 12, textAlign: "right" }}>
          <Button type="primary" icon={<DownloadOutlined />} onClick={() => void downloadRollback()}>
            下载回滚 SQL
          </Button>
        </div>
      </Modal>
    </Card>
  );
}
