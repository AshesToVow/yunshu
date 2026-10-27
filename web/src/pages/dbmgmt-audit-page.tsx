import { ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Modal, Select, Space, Table, Typography, message } from "antd";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { listDbAuditLogs, listDbInstances, type DbAuditLogItem } from "../services/dbmgmt";
import { extractApiErrorMessage } from "../services/http";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";

const AUDIT_ACTION_OPTIONS = [
  { value: "", label: "全部操作" },
  { value: "console_query", label: "SQL 查询" },
  { value: "sql_execute", label: "SQL 执行" },
  { value: "sql_import", label: "SQL 导入" },
  { value: "ticket_create", label: "创建工单" },
  { value: "ticket_approve", label: "工单审批通过" },
  { value: "ticket_reject", label: "工单审批驳回" },
  { value: "ticket_execute", label: "工单执行" },
  { value: "instance_upsert", label: "实例变更" },
  { value: "instance_delete", label: "实例删除" },
  { value: "grant_create", label: "授权创建" },
  { value: "grant_update", label: "授权更新" },
  { value: "grant_delete", label: "授权删除" },
  { value: "access_request_create", label: "权限申请" },
  { value: "access_request_approve", label: "权限申请通过" },
  { value: "access_request_reject", label: "权限申请驳回" },
  { value: "app_user_request_create", label: "应用用户申请" },
  { value: "app_user_request_approve", label: "应用用户申请通过" },
  { value: "app_user_request_reject", label: "应用用户申请驳回" },
] as const;

const ACTION_LABEL: Record<string, string> = Object.fromEntries(
  AUDIT_ACTION_OPTIONS.filter((o) => o.value).map((o) => [o.value, o.label]),
);

function auditActionLabel(action: string) {
  return ACTION_LABEL[action] ?? action;
}

function formatAuditDetail(row: DbAuditLogItem) {
  const detail = row.detail_json;
  if (!detail || typeof detail !== "object") return "-";
  const d = detail as Record<string, unknown>;
  const parts: string[] = [];
  if (d.database) parts.push(`库: ${String(d.database)}`);
  if (d.ticket_id) parts.push(`工单 #${String(d.ticket_id)}`);
  if (d.ticket_type) parts.push(`类型: ${String(d.ticket_type)}`);
  if (d.request_id) parts.push(`申请 #${String(d.request_id)}`);
  if (d.grant_id) parts.push(`授权 #${String(d.grant_id)}`);
  if (d.mysql_user) parts.push(`用户: ${String(d.mysql_user)}`);
  if (d.apply_type) parts.push(`申请类型: ${String(d.apply_type)}`);
  if (d.statement_count) parts.push(`${String(d.statement_count)} 条语句`);
  if (d.name) parts.push(`名称: ${String(d.name)}`);
  if (d.sql_file_ref) parts.push(`文件: ${String(d.sql_file_ref)}`);
  return parts.length ? parts.join(" · ") : "-";
}

function auditSqlText(row: DbAuditLogItem): string | undefined {
  const detail = row.detail_json;
  if (!detail || typeof detail !== "object") return undefined;
  const sql = (detail as Record<string, unknown>).sql;
  if (typeof sql !== "string") return undefined;
  const text = sql.trim();
  return text || undefined;
}

type AuditSqlView = { sql: string; meta: string };

export function DbmgmtAuditPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [instanceId, setInstanceId] = useState<number>();
  const [actionFilter, setActionFilter] = useState("");
  const [rows, setRows] = useState<DbAuditLogItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string>();
  const [sqlView, setSqlView] = useState<AuditSqlView | null>(null);
  const [instances, setInstances] = useState<
    { id: number; name: string; host: string; port: number; driver?: string }[]
  >([]);
  const loadSeqRef = useRef(0);
  const instancesSeqRef = useRef(0);

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 })
      .then((res) => {
        setProjects(res.list ?? []);
        if (res.list?.length) setProjectId(res.list[0].id);
      })
      .catch(() => {
        /* 项目列表加载失败不阻断页面 */
      });
  }, []);

  useEffect(() => {
    const seq = ++instancesSeqRef.current;
    setInstances([]);
    if (!projectId) return;
    void listDbInstances(projectId, { page: 1, page_size: 500 })
      .then((res) => {
        if (seq === instancesSeqRef.current) setInstances(res.list ?? []);
      })
      .catch((e) => {
        if (seq !== instancesSeqRef.current) return;
        setInstances([]);
        message.error(extractApiErrorMessage(e, "加载实例列表失败"));
      });
  }, [projectId]);

  const load = useCallback(async () => {
    const seq = ++loadSeqRef.current;
    if (!projectId) {
      setRows([]);
      setTotal(0);
      setLoading(false);
      return;
    }
    setLoading(true);
    setErrorMsg(undefined);
    try {
      const params: Record<string, string | number> = { page, page_size: 20 };
      if (instanceId) params.instance_id = instanceId;
      if (actionFilter) params.action = actionFilter;
      const res = await listDbAuditLogs(projectId, params);
      if (seq !== loadSeqRef.current) return;
      setRows(res.list ?? []);
      setTotal(res.total ?? 0);
    } catch (e) {
      if (seq !== loadSeqRef.current) return;
      setErrorMsg(extractApiErrorMessage(e, "加载失败"));
    } finally {
      if (seq === loadSeqRef.current) setLoading(false);
    }
  }, [projectId, instanceId, actionFilter, page]);

  useEffect(() => {
    void load();
  }, [load]);

  const copySql = async () => {
    if (!sqlView) return;
    if (!navigator.clipboard) {
      message.error("当前浏览器不支持自动复制，请手动选择文本复制");
      return;
    }
    try {
      await navigator.clipboard.writeText(sqlView.sql);
      message.success("SQL 已复制");
    } catch {
      message.error("复制失败，请手动选择文本复制");
    }
  };

  const instanceOptions = useMemo(
    () => [
      { value: undefined as number | undefined, label: "全部实例" },
      ...instances.map((i) => ({ value: i.id, label: i.name || `${i.host}:${i.port}` })),
    ],
    [instances],
  );

  return (
    <Card className="table-card" title="审计日志">
      <div className="toolbar">
        <Space wrap>
          <Select
            style={{ width: 200 }}
            value={projectId}
            options={projects.map((p) => ({ value: p.id, label: p.name }))}
            onChange={(v) => {
              loadSeqRef.current += 1;
              instancesSeqRef.current += 1;
              setProjectId(v);
              setInstanceId(undefined);
              setPage(1);
              setInstances([]);
              setRows([]);
              setTotal(0);
            }}
          />
          <Select
            allowClear
            placeholder="实例"
            style={{ width: 200 }}
            value={instanceId}
            options={instanceOptions}
            onChange={(v) => {
              setInstanceId(v);
              setPage(1);
            }}
          />
          <Select
            style={{ width: 160 }}
            value={actionFilter}
            options={AUDIT_ACTION_OPTIONS as unknown as { value: string; label: string }[]}
            onChange={(v) => {
              setActionFilter(v);
              setPage(1);
            }}
          />
        </Space>
        <div className="toolbar__actions">
          <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>
            刷新
          </Button>
        </div>
      </div>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
        记录 SQL 查询/执行、工单、授权、权限申请等操作。查询语句的完整执行历史另见「SQL 查询」页的查询历史。
      </Typography.Paragraph>
      {errorMsg ? <Alert type="error" message={errorMsg} style={{ marginBottom: 16 }} /> : null}
      <Table
        rowKey="id"
        loading={loading}
        dataSource={rows}
        scroll={{ x: 1140 }}
        pagination={{ current: page, pageSize: 20, total, onChange: (p) => setPage(p) }}
        columns={[
          { title: "时间", dataIndex: "created_at", width: 170, render: (v?: string) => formatDateTime(v) },
          { title: "操作人", dataIndex: "actor_name", width: 120 },
          { title: "操作类型", dataIndex: "action", width: 140, render: (v: string) => auditActionLabel(v) },
          {
            title: "实例",
            width: 180,
            render: (_, row) =>
              row.instance_label || row.instance_name || (row.instance_id ? `#${row.instance_id}` : "—"),
          },
          {
            title: "详情",
            width: 420,
            render: (_, row) => {
              const summary = formatAuditDetail(row);
              return (
                <Typography.Text ellipsis={{ tooltip: summary }} style={{ maxWidth: "100%" }}>
                  {summary}
                </Typography.Text>
              );
            },
          },
          {
            title: "操作",
            key: "actions",
            width: 110,
            fixed: "right",
            render: (_, row) => {
              const sql = auditSqlText(row);
              if (!sql) return "—";
              return (
                <Button
                  type="link"
                  size="small"
                  onClick={() =>
                    setSqlView({
                      sql,
                      meta: `${auditActionLabel(row.action)} · ${row.actor_name} · ${formatDateTime(row.created_at)}`,
                    })
                  }
                >
                  查看 SQL
                </Button>
              );
            },
          },
        ]}
      />
      <Modal
        title="SQL 详情"
        open={Boolean(sqlView)}
        width={860}
        onCancel={() => setSqlView(null)}
        footer={[
          <Button key="copy" onClick={() => void copySql()}>
            复制 SQL
          </Button>,
          <Button key="close" type="primary" onClick={() => setSqlView(null)}>
            关闭
          </Button>,
        ]}
      >
        {sqlView ? (
          <>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
              {sqlView.meta}
            </Typography.Paragraph>
            <pre className="code-block-panel">{sqlView.sql}</pre>
          </>
        ) : null}
      </Modal>
    </Card>
  );
}
