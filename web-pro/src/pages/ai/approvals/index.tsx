import {
  CheckOutlined,
  CloseOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Button, Space, Tag, message } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  executeAIApproval,
  listAIApprovals,
  reviewAIApproval,
  type AIApprovalItem,
} from '@/services/ai';
import { getClusters, type ClusterItem } from '@/services/clusters';
import { extractApiErrorMessage } from '@/services/http';

export default function AiApprovalsPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [clusters, setClusters] = useState<ClusterItem[]>([]);

  const clusterNameById = useMemo(() => {
    const m = new Map<number, string>();
    for (const c of clusters) m.set(c.id, c.name || `集群 #${c.id}`);
    return m;
  }, [clusters]);

  useEffect(() => {
    void getClusters({ page: 1, page_size: 1000 })
      .then((res) => setClusters(res?.list || []))
      .catch(() => undefined);
  }, []);

  async function review(id: number, approve: boolean, execute?: boolean) {
    try {
      await reviewAIApproval(id, { approve, execute: !!execute, note: approve ? '同意' : '驳回' });
      message.success(approve ? '已批准' : '已驳回');
      actionRef.current?.reload();
    } catch (e) {
      message.error(extractApiErrorMessage(e, '审批失败'));
    }
  }

  async function execute(id: number) {
    try {
      await executeAIApproval(id);
      message.success('已执行');
      actionRef.current?.reload();
    } catch (e) {
      message.error(extractApiErrorMessage(e, '执行失败'));
    }
  }

  const statusTag = (s: string) => {
    const map: Record<string, string> = {
      pending: 'processing',
      approved: 'success',
      rejected: 'default',
      executed: 'success',
      failed: 'error',
    };
    return <Tag color={map[s] || 'default'}>{s}</Tag>;
  };

  const columns: ProColumns<AIApprovalItem>[] = [
    {
      title: '状态',
      dataIndex: 'status',
      hideInTable: true,
      valueType: 'select',
      initialValue: 'pending',
      valueEnum: {
        pending: { text: 'pending' },
        approved: { text: 'approved' },
        rejected: { text: 'rejected' },
        executed: { text: 'executed' },
        failed: { text: 'failed' },
      },
    },
    { title: 'ID', dataIndex: 'id', width: 70, search: false },
    { title: '工具', dataIndex: 'tool_name', width: 160, search: false },
    {
      title: '集群',
      dataIndex: 'cluster_id',
      width: 140,
      ellipsis: true,
      search: false,
      render: (_, row) =>
        row.cluster_id ? clusterNameById.get(row.cluster_id) || `#${row.cluster_id}` : '—',
    },
    { title: '命名空间', dataIndex: 'namespace', width: 120, search: false },
    { title: '资源', dataIndex: 'resource', ellipsis: true, search: false },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      search: false,
      render: (_, row) => statusTag(row.status),
    },
    { title: '结果', dataIndex: 'result_msg', ellipsis: true, search: false },
    {
      title: '操作',
      valueType: 'option',
      width: 260,
      render: (_, row) => (
        <Space>
          {row.status === 'pending' ? (
            <>
              <Button
                type="link"
                size="small"
                icon={<CheckOutlined />}
                onClick={() => void review(row.id, true, true)}
              >
                批准并执行
              </Button>
              <Button type="link" size="small" onClick={() => void review(row.id, true, false)}>
                仅批准
              </Button>
              <Button
                type="link"
                size="small"
                danger
                icon={<CloseOutlined />}
                onClick={() => void review(row.id, false)}
              >
                驳回
              </Button>
            </>
          ) : null}
          {row.status === 'approved' || row.status === 'failed' ? (
            <Button
              type="link"
              size="small"
              icon={<PlayCircleOutlined />}
              onClick={() => void execute(row.id)}
            >
              执行
            </Button>
          ) : null}
        </Space>
      ),
    },
  ];

  return (
    <PageContainer header={{ title: 'AI 审批', subTitle: '高风险 AI Tool 执行前的人工审批' }}>
      <ProTable<AIApprovalItem>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        search={{ labelWidth: 'auto' }}
        request={async (params) => {
          try {
            const res = await listAIApprovals({
              status: (params.status as string) || undefined,
              page: params.current ?? 1,
              page_size: params.pageSize ?? 10,
            });
            return {
              data: res.list || [],
              success: true,
              total: res.total || 0,
            };
          } catch (e) {
            message.error(extractApiErrorMessage(e, '加载审批失败'));
            return { data: [], success: false, total: 0 };
          }
        }}
        pagination={{ defaultPageSize: 10 }}
        toolBarRender={() => [
          <Button key="reload" icon={<ReloadOutlined />} onClick={() => actionRef.current?.reload()}>
            刷新
          </Button>,
        ]}
      />
    </PageContainer>
  );
}
