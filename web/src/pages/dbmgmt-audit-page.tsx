import { ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Select, Space, Table, Typography } from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import { formatInstanceLabel } from "../components/dbmgmt/dbmgmt-ui-shared";
import { listDbAuditLogs, listDbInstances, type DbAuditLogItem } from "../services/dbmgmt";
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
  if (d.sql) {
    const sql = String(d.sql);
    parts.push(sql.length > 120 ? `${sql.slice(0, 120)}...` : sql);
  }
  return parts.length ? parts.join(" · ") : "-";
}

export function DbmgmtAuditPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [instanceId, setInstanceId] = useState<number>();
  const [actionFilter, setActionFilter] = useState("");
  const [rows, setRows] = useState<DbAuditLogItem[]>([]);
  const [instances, setInstances] = useState<{ id: number; name: string; host: string; port: number; driver?: string }[]>([]);

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
      if (res.list?.length) setProjectId(res.list[0].id);
    });
  }, []);

  useEffect(() => {
    if (!projectId) return;
    void listDbInstances(projectId, { page: 1, page_size: 500 }).then((res) => {
      setInstances(res.list ?? []);
    });
  }, [projectId]);

  const load = useCallback(async () => {
    if (!projectId) return;
    const params: Record<string, string | number> = { page: 1, page_size: 200 };
    if (instanceId) params.instance_id = instanceId;
    const res = await listDbAuditLogs(projectId, params);
    let list = res.list ?? [];
    if (actionFilter) {
      list = list.filter((row) => row.action === actionFilter);
    }
    setRows(list);
  }, [projectId, instanceId, actionFilter]);

  useEffect(() => {
    void load();
  }, [load]);

  const instanceOptions = useMemo(
    () => [{ value: undefined as number | undefined, label: "全部实例" }, ...instances.map((i) => ({ value: i.id, label: i.name || formatInstanceLabel(i) }))],
    [instances],
  );

  return (
    <Card
      title="审计日志"
      extra={
        <Space wrap>
          <Select style={{ width: 200 }} value={projectId} options={projects.map((p) => ({ value: p.id, label: p.name }))} onChange={(v) => { setProjectId(v); setInstanceId(undefined); }} />
          <Select allowClear placeholder="实例" style={{ width: 200 }} value={instanceId} options={instanceOptions} onChange={setInstanceId} />
          <Select style={{ width: 160 }} value={actionFilter} options={AUDIT_ACTION_OPTIONS as unknown as { value: string; label: string }[]} onChange={setActionFilter} />
          <Button icon={<ReloadOutlined />} onClick={() => void load()} />
        </Space>
      }
    >
      <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
        记录 SQL 查询/执行、工单、授权、权限申请等操作。查询语句的完整执行历史另见「SQL 查询」页的查询历史。
      </Typography.Paragraph>
      <Table
        rowKey="id"
        dataSource={rows}
        scroll={{ x: 960 }}
        columns={[
          { title: "操作", dataIndex: "action", width: 140, render: (v: string) => auditActionLabel(v) },
          { title: "操作人", dataIndex: "actor_name", width: 120 },
          {
            title: "实例",
            width: 200,
            render: (_: unknown, row: DbAuditLogItem) => {
              if (row.instance_name) {
                return (
                  <span>
                    {row.instance_name}
                    {row.instance_label ? <Typography.Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>{row.instance_label}</Typography.Text> : null}
                  </span>
                );
              }
              return row.instance_id ? `#${row.instance_id}` : "-";
            },
          },
          { title: "详情", render: (_: unknown, row: DbAuditLogItem) => formatAuditDetail(row) },
          { title: "时间", dataIndex: "created_at", width: 180, render: (v) => formatDateTime(String(v)) },
        ]}
      />
    </Card>
  );
}
