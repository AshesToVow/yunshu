import { LinkOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Link, useSearchParams } from '@umijs/max';
import { Button, Tag } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import { getProjects, type ProjectItem } from '@/services/projects';
import { listWorkflowTickets, type WorkflowTicketRow } from '@/services/workflow';
import { formatDateTime } from '@/utils/format';

function domainLabel(domain: string) {
  switch (domain) {
    case 'dbmgmt':
      return '数据库';
    case 'cicd':
      return '发布';
    case 'incident':
      return '故障';
    case 'ops':
      return '变更';
    default:
      return domain || '—';
  }
}

function deepLink(row: WorkflowTicketRow) {
  switch (row.ref_type) {
    case 'db_sql_ticket':
      return `/dbmgmt/workflow/tickets/${row.ref_id}?project=${row.project_id}`;
    case 'db_access_request':
      return `/dbmgmt/apply/query?project=${row.project_id}&highlight=${row.ref_id}`;
    case 'db_app_user_request':
      return `/dbmgmt/apply/app-user?project=${row.project_id}&highlight=${row.ref_id}`;
    case 'cicd_release_run':
      return `/cicd/release-records?project=${row.project_id}&release=${row.ref_id}`;
    case 'alert_event':
      return `/alert-events?highlight=${row.ref_id}`;
    default:
      return `/workflow/inbox?project=${row.project_id}`;
  }
}

export default function WorkflowTicketsPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [searchParams] = useSearchParams();
  const [projects, setProjects] = useState<ProjectItem[]>([]);

  const initialProjectId = (() => {
    const n = Number(searchParams.get('project') || 0);
    return n > 0 ? n : undefined;
  })();
  const initialDomain = searchParams.get('domain') || undefined;
  const initialStatus = searchParams.get('status') || undefined;

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => setProjects(res.list ?? []));
  }, []);

  const projectNameMap = useMemo(() => {
    const m = new Map<number, string>();
    for (const p of projects) m.set(p.id, p.name);
    return m;
  }, [projects]);

  const columns: ProColumns<WorkflowTicketRow>[] = [
    {
      title: '域',
      dataIndex: 'domain',
      width: 88,
      valueType: 'select',
      initialValue: initialDomain || '',
      valueEnum: {
        '': { text: '全部域' },
        dbmgmt: { text: '数据库' },
        cicd: { text: '发布' },
        incident: { text: '故障' },
        ops: { text: '变更' },
      },
      render: (_, row) => <Tag>{domainLabel(row.domain)}</Tag>,
    },
    { title: '类型', dataIndex: 'ticket_type', width: 120, search: false },
    {
      title: '项目',
      dataIndex: 'project_id',
      width: 140,
      valueType: 'select',
      initialValue: initialProjectId,
      fieldProps: {
        options: projects.map((p) => ({ label: p.name, value: p.id })),
        allowClear: true,
        showSearch: true,
      },
      render: (_, row) => projectNameMap.get(row.project_id) ?? (row.project_id ? `#${row.project_id}` : '—'),
    },
    { title: '标题', dataIndex: 'title', ellipsis: true, search: false },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      valueType: 'select',
      initialValue: initialStatus,
      valueEnum: {
        pending: { text: '待审批', status: 'Processing' },
        approved: { text: '已通过', status: 'Success' },
        rejected: { text: '已驳回', status: 'Error' },
        cancelled: { text: '已取消', status: 'Default' },
      },
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 168,
      search: false,
      render: (_, row) => formatDateTime(row.created_at),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 100,
      render: (_, row) => [
        <Link key="detail" to={deepLink(row)}>
          <Button type="link" size="small" icon={<LinkOutlined />}>
            详情
          </Button>
        </Link>,
      ],
    },
  ];

  return (
    <PageContainer title="工单列表" subTitle="跨域工单历史；审批动作请到「我的待办」">
      <ProTable<WorkflowTicketRow>
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        scroll={{ x: 1000 }}
        pagination={{ defaultPageSize: 20, showSizeChanger: true }}
        search={{ labelWidth: 'auto' }}
        request={async (params) => {
          const domain = params.domain === '' ? undefined : (params.domain as string | undefined);
          const res = await listWorkflowTickets({
            page: params.current,
            page_size: params.pageSize,
            domain,
            project_id: params.project_id as number | undefined,
            status: params.status as string | undefined,
          });
          return { data: res.list ?? [], total: res.total ?? 0, success: true };
        }}
      />
    </PageContainer>
  );
}
