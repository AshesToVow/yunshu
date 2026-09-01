import { ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Button, Typography, message } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { listDbAuditLogs, listDbInstances, type DbAuditLogItem } from '@/services/dbmgmt';
import { getProjects, type ProjectItem } from '@/services/projects';
import { formatDateTime } from '@/utils/format';

const AUDIT_ACTION_OPTIONS = [
  { value: '', label: '全部操作' },
  { value: 'console_query', label: 'SQL 查询' },
  { value: 'sql_execute', label: 'SQL 执行' },
  { value: 'sql_import', label: 'SQL 导入' },
  { value: 'ticket_create', label: '创建工单' },
  { value: 'ticket_approve', label: '工单审批通过' },
  { value: 'ticket_reject', label: '工单审批驳回' },
  { value: 'ticket_execute', label: '工单执行' },
  { value: 'instance_upsert', label: '实例变更' },
  { value: 'instance_delete', label: '实例删除' },
  { value: 'grant_create', label: '授权创建' },
  { value: 'grant_update', label: '授权更新' },
  { value: 'grant_delete', label: '授权删除' },
  { value: 'access_request_create', label: '权限申请' },
  { value: 'access_request_approve', label: '权限申请通过' },
  { value: 'access_request_reject', label: '权限申请驳回' },
  { value: 'app_user_request_create', label: '应用用户申请' },
  { value: 'app_user_request_approve', label: '应用用户申请通过' },
  { value: 'app_user_request_reject', label: '应用用户申请驳回' },
] as const;

const ACTION_LABEL: Record<string, string> = Object.fromEntries(
  AUDIT_ACTION_OPTIONS.filter((o) => o.value).map((o) => [o.value, o.label]),
);

function auditActionLabel(action: string) {
  return ACTION_LABEL[action] ?? action;
}

function formatAuditDetail(row: DbAuditLogItem) {
  const detail = row.detail_json;
  if (!detail || typeof detail !== 'object') return '-';
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
  return parts.length ? parts.join(' · ') : '-';
}

export default function DbmgmtAuditPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [instances, setInstances] = useState<
    { id: number; name: string; host: string; port: number }[]
  >([]);

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      const list = res.list ?? [];
      setProjects(list);
      if (list.length) setProjectId(list[0].id);
    });
  }, []);

  useEffect(() => {
    if (!projectId) return;
    void listDbInstances(projectId, { page: 1, page_size: 500 }).then((res) => {
      setInstances(res.list ?? []);
    });
  }, [projectId]);

  const columns: ProColumns<DbAuditLogItem>[] = [
    {
      title: '项目',
      dataIndex: 'project_id',
      hideInTable: true,
      valueType: 'select',
      initialValue: projectId,
      fieldProps: {
        options: projects.map((p) => ({ label: p.name, value: p.id })),
        allowClear: false,
        style: { width: 200 },
        onChange: (v: number) => {
          setProjectId(v);
          actionRef.current?.reloadAndRest?.();
          actionRef.current?.reload();
        },
      },
    },
    {
      title: '实例',
      dataIndex: 'instance_id',
      hideInTable: true,
      valueType: 'select',
      fieldProps: {
        allowClear: true,
        placeholder: '全部实例',
        style: { width: 200 },
        options: instances.map((i) => ({
          value: i.id,
          label: i.name || `${i.host}:${i.port}`,
        })),
      },
    },
    {
      title: '操作类型',
      dataIndex: 'action',
      hideInTable: true,
      valueType: 'select',
      fieldProps: {
        allowClear: true,
        placeholder: '全部操作',
        style: { width: 180 },
        options: AUDIT_ACTION_OPTIONS.filter((o) => o.value).map((o) => ({
          value: o.value,
          label: o.label,
        })),
      },
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 170,
      search: false,
      render: (_, row) => formatDateTime(row.created_at),
    },
    { title: '操作人', dataIndex: 'actor_name', width: 120, search: false },
    {
      title: '操作',
      dataIndex: 'action',
      width: 140,
      search: false,
      render: (_, row) => auditActionLabel(row.action),
    },
    {
      title: '实例',
      width: 180,
      search: false,
      render: (_, row) =>
        row.instance_label || row.instance_name || (row.instance_id ? `#${row.instance_id}` : '—'),
    },
    {
      title: '详情',
      search: false,
      render: (_, row) => formatAuditDetail(row),
    },
  ];

  return (
    <PageContainer
      header={{
        title: '审计日志',
        subTitle: 'SQL 查询/执行、工单、授权、权限申请等操作记录',
      }}
    >
      <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
        记录 SQL 查询/执行、工单、授权、权限申请等操作。查询语句的完整执行历史另见「SQL 查询」页的查询历史。
      </Typography.Paragraph>
      <ProTable<DbAuditLogItem>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        search={{ labelWidth: 'auto' }}
        params={{ projectId }}
        request={async (params) => {
          const pid = Number(params.project_id || projectId || 0);
          if (!pid) return { data: [], success: true, total: 0 };
          if (pid !== projectId) setProjectId(pid);
          try {
            const query: Record<string, string | number> = {
              page: params.current ?? 1,
              page_size: params.pageSize ?? 20,
            };
            if (params.instance_id) query.instance_id = Number(params.instance_id);
            if (params.action) query.action = String(params.action);
            const res = await listDbAuditLogs(pid, query);
            return {
              data: res.list ?? [],
              success: true,
              total: res.total ?? 0,
            };
          } catch (e) {
            message.error(e instanceof Error ? e.message : '加载失败');
            return { data: [], success: false, total: 0 };
          }
        }}
        pagination={{ defaultPageSize: 20 }}
        toolBarRender={() => [
          <Button key="reload" icon={<ReloadOutlined />} onClick={() => actionRef.current?.reload()}>
            刷新
          </Button>,
        ]}
      />
    </PageContainer>
  );
}
