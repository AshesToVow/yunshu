import { ExclamationCircleOutlined, UnlockOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Button, Popconfirm, message } from 'antd';
import { useRef } from 'react';
import { getBannedIPs, unbanIP, type BannedIPItem } from '@/services/admin';

export default function BannedIPsPage() {
  const actionRef = useRef<ActionType>(undefined);

  const columns: ProColumns<BannedIPItem>[] = [
    { title: 'IP', dataIndex: 'ip', copyable: true },
    {
      title: '剩余封禁(秒)',
      dataIndex: 'ttl_seconds',
      width: 160,
      valueType: 'digit',
    },
    {
      title: '操作',
      valueType: 'option',
      width: 140,
      render: (_, row) => [
        <Popconfirm
          key="unban"
          title={`确定解除 ${row.ip} 的封禁吗？`}
          okText="解除"
          cancelText="取消"
          icon={<ExclamationCircleOutlined />}
          onConfirm={async () => {
            await unbanIP(row.ip);
            message.success('已解除封禁');
            actionRef.current?.reload();
          }}
        >
          <Button type="link" size="small" icon={<UnlockOutlined />}>
            解除封禁
          </Button>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <PageContainer title="封禁 IP" subTitle="登录失败过多触发的临时封禁列表">
      <ProTable<BannedIPItem>
        actionRef={actionRef}
        rowKey="ip"
        columns={columns}
        search={false}
        pagination={false}
        request={async () => {
          const res = await getBannedIPs();
          return { data: res.list ?? [], success: true };
        }}
        toolBarRender={() => [
          <Button key="refresh" onClick={() => actionRef.current?.reload()}>
            刷新
          </Button>,
        ]}
      />
    </PageContainer>
  );
}
