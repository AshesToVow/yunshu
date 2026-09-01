import { CheckOutlined, CloseOutlined, LinkOutlined, PlayCircleOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Link, request, useSearchParams } from '@umijs/max';
import { Button, Space, Tag, message } from 'antd';
import { useRef } from 'react';

type PendingTicketItem = {
  workflow_ticket_id: number;
  step_id: number;
  domain: string;
  ticket_type: string;
  project_id: number;
  title: string;
  status: string;
  current_stage_name: string;
  submitter_name?: string;
  ref_type: string;
  ref_id: number;
  deep_link: string;
  activated_at?: string;
  created_at: string;
  mine_status: 'mine_pending' | 'mine_done';
  action?: string;
};

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

async function listPending(params: Record<string, unknown>) {
  const res = await request<{ list: PendingTicketItem[]; total: number }>('/api/v1/workflow/tickets/pending', {
    method: 'GET',
    params: {
      page: params.current,
      page_size: params.pageSize,
      mine_scope: params.mine_scope ?? 'pending',
      domains: params.domains ?? params.domain ?? params.domainFromUrl,
      project_id: params.project_id,
    },
  });
  return { data: res.list ?? [], total: res.total ?? 0, success: true };
}

async function executeRelease(projectId: number, runId: number) {
  await request(`/api/v1/projects/${projectId}/cicd/release-runs/${runId}/execute`, { method: 'POST' });
}

async function reviewWorkflow(ticketId: number, stepId: number, approve: boolean, comment?: string) {
  await request(`/api/v1/workflow/tickets/${ticketId}/steps/${stepId}/review`, {
    method: 'POST',
    data: { approve, comment: comment ?? '' },
  });
}

export default function WorkflowInboxPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [searchParams] = useSearchParams();
  const domainFromUrl = searchParams.get('domain') ?? undefined;

  const columns: ProColumns<PendingTicketItem>[] = [
    {
      title: '域',
      dataIndex: 'domain',
      width: 88,
      hideInTable: Boolean(domainFromUrl),
      initialValue: domainFromUrl,
      render: (_, row) => <Tag>{domainLabel(row.domain)}</Tag>,
    },
    { title: '标题', dataIndex: 'title', ellipsis: true },
    { title: '当前节点', dataIndex: 'current_stage_name', width: 120, ellipsis: true },
    { title: '提交人', dataIndex: 'submitter_name', width: 100 },
    {
      title: '到达时间',
      dataIndex: 'activated_at',
      width: 168,
      valueType: 'dateTime',
      render: (_, row) => row.activated_at || row.created_at,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 220,
      render: (_, row) => {
        const pending = row.mine_status === 'mine_pending';
        const isExecute = row.action === 'execute';
        return (
          <Space size="small">
            {row.deep_link ? (
              <Link to={row.deep_link}>
                <Button type="link" size="small" icon={<LinkOutlined />}>
                  详情
                </Button>
              </Link>
            ) : null}
            {pending && isExecute ? (
              <Button
                type="link"
                size="small"
                icon={<PlayCircleOutlined />}
                onClick={() => {
                  void executeRelease(row.project_id, row.ref_id).then(() => {
                    message.success('已触发发布执行');
                    actionRef.current?.reload();
                  });
                }}
              >
                执行发布
              </Button>
            ) : pending ? (
              <>
                <Button
                  type="link"
                  size="small"
                  icon={<CheckOutlined />}
                  onClick={() => {
                    void reviewWorkflow(row.workflow_ticket_id, row.step_id, true).then(() => {
                      message.success('已通过');
                      actionRef.current?.reload();
                    });
                  }}
                >
                  通过
                </Button>
                <Button
                  type="link"
                  size="small"
                  danger
                  icon={<CloseOutlined />}
                  onClick={() => {
                    void reviewWorkflow(row.workflow_ticket_id, row.step_id, false).then(() => {
                      message.success('已驳回');
                      actionRef.current?.reload();
                    });
                  }}
                >
                  驳回
                </Button>
              </>
            ) : (
              <Tag>已处理</Tag>
            )}
          </Space>
        );
      },
    },
  ];

  return (
    <PageContainer title="我的待办" subTitle="统一入口：审批 + 发布执行">
      <ProTable<PendingTicketItem>
        actionRef={actionRef}
        rowKey={(r) => `${r.workflow_ticket_id}-${r.step_id}`}
        columns={columns}
        request={(params) => listPending({ ...params, domainFromUrl })}
        pagination={{ defaultPageSize: 20 }}
        search={false}
        toolBarRender={() => [
          <Button key="refresh" onClick={() => actionRef.current?.reload()}>
            刷新
          </Button>,
        ]}
      />
    </PageContainer>
  );
}
