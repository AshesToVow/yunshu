import { ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { history } from '@umijs/max';
import { Button, Tag } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { auditModeLabel, ticketTypeLabel } from '@/components/dbmgmt/dbmgmt-ui-shared';
import { listDbTickets, type DbTicket } from '@/services/dbmgmt';
import { getProjects, type ProjectItem } from '@/services/projects';
import { ticketStatusLabel } from '@/utils/dbmgmt-labels';
import { formatDateTime } from '@/utils/format';

function ticketTypeColor(t?: string) {
  if (t === 'sql_import') return 'purple';
  if (t === 'sql_execute') return 'blue';
  return 'cyan';
}

function statusTagColor(s?: string) {
  if (s === 'success') return 'green';
  if (s === 'executing') return 'orange';
  if (s === 'failed' || s === 'rejected') return 'red';
  if (s === 'pending_execution') return 'blue';
  return 'default';
}

export default function DbmgmtTicketsPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      const list = res.list ?? [];
      setProjects(list);
      if (list.length) setProjectId(list[0].id);
    });
  }, []);

  const columns: ProColumns<DbTicket>[] = [
    {
      title: '项目',
      dataIndex: 'project_id',
      hideInTable: true,
      valueType: 'select',
      initialValue: projectId,
      fieldProps: {
        options: projects.map((p) => ({ label: p.name, value: p.id })),
        allowClear: false,
        style: { width: 180 },
        onChange: (v: number) => {
          setProjectId(v);
          actionRef.current?.reload();
        },
      },
    },
    {
      title: '类型',
      dataIndex: 'ticket_type',
      hideInTable: true,
      valueType: 'select',
      valueEnum: {
        sql_execute: { text: 'SQL上线申请' },
        sql_import: { text: 'SQL文件上线' },
      },
      fieldProps: { allowClear: true, placeholder: '全部类型' },
    },
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '任务名称 / SQL 摘要' },
    },
    {
      title: '任务名称',
      search: false,
      ellipsis: true,
      render: (_, r) => r.reason || r.sql_excerpt || `工单 #${r.id}`,
    },
    {
      title: '类型',
      dataIndex: 'ticket_type',
      width: 140,
      search: false,
      render: (_, r) => <Tag color={ticketTypeColor(r.ticket_type)}>{ticketTypeLabel(r.ticket_type)}</Tag>,
    },
    { title: '提交人', dataIndex: 'submitter_name', width: 100, search: false },
    {
      title: '是否审核',
      width: 100,
      search: false,
      render: (_, r) => (
        <Tag color={r.audit_mode === 'manual' ? 'orange' : 'blue'}>{auditModeLabel(r.audit_mode)}</Tag>
      ),
    },
    {
      title: '提交时间',
      dataIndex: 'created_at',
      width: 170,
      search: false,
      render: (_, r) => formatDateTime(r.created_at),
    },
    {
      title: '完成时间',
      width: 170,
      search: false,
      render: (_, r) =>
        r.status === 'success' || r.status === 'failed'
          ? formatDateTime(r.updated_at ?? r.created_at)
          : '—',
    },
    {
      title: '任务状态',
      dataIndex: 'status',
      width: 100,
      search: false,
      render: (_, r) => <Tag color={statusTagColor(r.status)}>{ticketStatusLabel(r.status)}</Tag>,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 200,
      render: (_, r) => [
        <Button
          key="view"
          size="small"
          onClick={() =>
            history.push(`/dbmgmt/workflow/tickets/${r.id}?project=${projectId}&tab=info`)
          }
        >
          查看
        </Button>,
        <Button
          key="detail"
          size="small"
          type="primary"
          onClick={() =>
            history.push(`/dbmgmt/workflow/tickets/${r.id}?project=${projectId}&tab=log`)
          }
        >
          详情
        </Button>,
      ],
    },
  ];

  return (
    <PageContainer header={{ title: 'SQL 工单', subTitle: '数据库变更工单列表与历史' }}>
      <ProTable<DbTicket>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        search={{ labelWidth: 'auto' }}
        params={{ projectId }}
        request={async (params) => {
          const pid = Number(params.project_id || projectId || 0);
          if (!pid) return { data: [], success: true, total: 0 };
          if (pid !== projectId) setProjectId(pid);
          const res = await listDbTickets(pid, {
            page: params.current ?? 1,
            page_size: params.pageSize ?? 10,
            ...(params.ticket_type ? { ticket_type: String(params.ticket_type) } : {}),
          });
          let list = res.list ?? [];
          const kw = String(params.keyword || '')
            .trim()
            .toLowerCase();
          if (kw) {
            list = list.filter((r) =>
              (r.reason || r.sql_excerpt || '').toLowerCase().includes(kw),
            );
          }
          return { data: list, success: true, total: res.total ?? list.length };
        }}
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
        toolBarRender={() => [
          <Button key="reload" icon={<ReloadOutlined />} onClick={() => actionRef.current?.reload()}>
            刷新
          </Button>,
        ]}
      />
    </PageContainer>
  );
}
