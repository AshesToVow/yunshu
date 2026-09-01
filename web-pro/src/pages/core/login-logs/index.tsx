import { DeleteOutlined, ExportOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns, ProFormInstance } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { request } from '@umijs/max';
import { Button, Popconfirm, Tag, message } from 'antd';
import { useRef, useState } from 'react';

type LoginLogItem = {
  id: number;
  created_at: string;
  username: string;
  ip: string;
  status: number;
  detail: string;
  user_agent: string;
  source: string;
  user_id?: number;
};

function sourceLabel(source: string) {
  if (source === 'email') return '邮箱验证码';
  if (source === 'password') return '用户名密码';
  return source || '—';
}

function sourceColor(source: string) {
  if (source === 'email') return 'cyan';
  if (source === 'password') return 'blue';
  return 'default';
}

async function listLoginLogs(params: Record<string, unknown>) {
  const res = await request<{ list: LoginLogItem[]; total: number }>('/api/v1/login-logs', {
    method: 'GET',
    params: {
      page: params.current,
      page_size: params.pageSize,
      username: params.username,
      status: params.status,
      source: params.source,
    },
  });
  return { data: res.list ?? [], total: res.total ?? 0, success: true };
}

async function batchDelete(ids: number[]) {
  await request('/api/v1/login-logs/delete', { method: 'POST', data: { ids } });
}

async function deleteOne(id: number) {
  await request(`/api/v1/login-logs/${id}`, { method: 'DELETE' });
}

async function exportLogs(params: Record<string, unknown>) {
  return request<Blob>('/api/v1/login-logs/export', {
    method: 'GET',
    params: {
      username: params.username,
      status: params.status,
      source: params.source,
    },
    responseType: 'blob',
  });
}

export default function LoginLogsPage() {
  const actionRef = useRef<ActionType>(undefined);
  const formRef = useRef<ProFormInstance>(undefined);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);

  const columns: ProColumns<LoginLogItem>[] = [
    {
      title: '用户名',
      dataIndex: 'username',
      width: 120,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 88,
      valueType: 'select',
      valueEnum: {
        1: { text: '成功', status: 'Success' },
        0: { text: '失败', status: 'Error' },
      },
      render: (_, row) => (
        <Tag color={row.status === 1 ? 'success' : 'error'}>{row.status === 1 ? '成功' : '失败'}</Tag>
      ),
    },
    {
      title: '来源',
      dataIndex: 'source',
      width: 120,
      valueType: 'select',
      valueEnum: {
        password: { text: '用户名密码' },
        email: { text: '邮箱验证码' },
      },
      render: (_, row) => <Tag color={sourceColor(row.source)}>{sourceLabel(row.source)}</Tag>,
    },
    { title: 'IP', dataIndex: 'ip', width: 140, search: false },
    { title: '详情', dataIndex: 'detail', ellipsis: true, search: false },
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 168,
      valueType: 'dateTime',
      search: false,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 88,
      render: (_, row) => [
        <Popconfirm
          key="delete"
          title="确认删除该记录？"
          onConfirm={() => {
            void deleteOne(row.id).then(() => {
              message.success('已删除');
              actionRef.current?.reload();
            });
          }}
        >
          <Button type="link" size="small" danger icon={<DeleteOutlined />}>
            删除
          </Button>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <PageContainer title="登录日志">
      <ProTable<LoginLogItem>
        actionRef={actionRef}
        formRef={formRef}
        rowKey="id"
        columns={columns}
        request={listLoginLogs}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys as number[]),
        }}
        pagination={{ defaultPageSize: 10 }}
        search={{ labelWidth: 'auto' }}
        toolBarRender={() => [
          <Button
            key="export"
            icon={<ExportOutlined />}
            onClick={() => {
              const form = formRef.current?.getFieldsValue?.() ?? {};
              void exportLogs(form).then((blob) => {
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = 'login_logs.xlsx';
                a.click();
                window.URL.revokeObjectURL(url);
              });
            }}
          >
            导出
          </Button>,
          <Popconfirm
            key="batch"
            title={`确认删除选中的 ${selectedRowKeys.length} 条记录？`}
            disabled={selectedRowKeys.length === 0}
            onConfirm={() => {
              void batchDelete(selectedRowKeys).then(() => {
                message.success('已删除');
                setSelectedRowKeys([]);
                actionRef.current?.reload();
              });
            }}
          >
            <Button danger icon={<DeleteOutlined />} disabled={selectedRowKeys.length === 0}>
              批量删除
            </Button>
          </Popconfirm>,
        ]}
      />
    </PageContainer>
  );
}
