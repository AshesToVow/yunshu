import { DeleteOutlined, ExportOutlined, EyeOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns, ProFormInstance } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { request } from '@umijs/max';
import { Button, Modal, Popconfirm, Tag, Typography, message } from 'antd';
import { useRef, useState } from 'react';

type OperationLogItem = {
  id: number;
  created_at: string;
  user_id: number;
  username: string;
  nickname: string;
  ip: string;
  method: string;
  path: string;
  status_code: number;
  request_body: string;
  response_body: string;
  latency_ms: number;
};

function methodColor(method: string) {
  switch (method) {
    case 'GET':
      return 'green';
    case 'POST':
      return 'blue';
    case 'PUT':
      return 'orange';
    case 'DELETE':
      return 'red';
    default:
      return 'default';
  }
}

function statusColor(code: number) {
  if (code >= 200 && code < 300) return 'success';
  if (code >= 400 && code < 500) return 'warning';
  if (code >= 500) return 'error';
  return 'default';
}

async function listOperationLogs(params: Record<string, unknown>) {
  const res = await request<{ list: OperationLogItem[]; total: number }>('/api/v1/operation-logs', {
    method: 'GET',
    params: {
      page: params.current,
      page_size: params.pageSize,
      method: params.method,
      path: params.path,
      status_code: params.status_code,
    },
  });
  return { data: res.list ?? [], total: res.total ?? 0, success: true };
}

async function batchDelete(ids: number[]) {
  await request('/api/v1/operation-logs/delete', { method: 'POST', data: { ids } });
}

async function deleteOne(id: number) {
  await request(`/api/v1/operation-logs/${id}`, { method: 'DELETE' });
}

async function exportLogs(params: Record<string, unknown>) {
  return request<Blob>('/api/v1/operation-logs/export', {
    method: 'GET',
    params: {
      method: params.method,
      path: params.path,
      status_code: params.status_code,
    },
    responseType: 'blob',
  });
}

export default function OperationLogsPage() {
  const actionRef = useRef<ActionType>(undefined);
  const formRef = useRef<ProFormInstance>(undefined);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [bodyModal, setBodyModal] = useState<{ title: string; content: string } | null>(null);

  const columns: ProColumns<OperationLogItem>[] = [
    { title: 'ID', dataIndex: 'id', width: 70, search: false },
    {
      title: '操作人',
      width: 140,
      search: false,
      render: (_, row) => `${row.username}${row.nickname ? ` (${row.nickname})` : ''}`,
    },
    { title: 'IP', dataIndex: 'ip', width: 130, search: false, render: (v) => v || '—' },
    {
      title: '方法',
      dataIndex: 'method',
      width: 90,
      valueType: 'select',
      valueEnum: {
        GET: { text: 'GET' },
        POST: { text: 'POST' },
        PUT: { text: 'PUT' },
        DELETE: { text: 'DELETE' },
      },
      render: (_, row) => <Tag color={methodColor(row.method)}>{row.method}</Tag>,
    },
    {
      title: '路径',
      dataIndex: 'path',
      width: 280,
      ellipsis: true,
    },
    {
      title: '状态码',
      dataIndex: 'status_code',
      width: 90,
      valueType: 'digit',
      fieldProps: { min: 100, max: 599, placeholder: 'HTTP 状态码' },
      render: (_, row) => <Tag color={statusColor(row.status_code)}>{row.status_code}</Tag>,
    },
    {
      title: '耗时(ms)',
      dataIndex: 'latency_ms',
      width: 100,
      search: false,
      render: (v) => v ?? 0,
    },
    {
      title: '请求体',
      width: 88,
      search: false,
      render: (_, row) =>
        row.request_body ? (
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => setBodyModal({ title: '请求体', content: row.request_body })}
          >
            查看
          </Button>
        ) : (
          '—'
        ),
    },
    {
      title: '响应体',
      width: 88,
      search: false,
      render: (_, row) =>
        row.response_body ? (
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => setBodyModal({ title: '响应体', content: row.response_body })}
          >
            查看
          </Button>
        ) : (
          '—'
        ),
    },
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
    <PageContainer title="操作日志">
      <ProTable<OperationLogItem>
        actionRef={actionRef}
        formRef={formRef}
        rowKey="id"
        columns={columns}
        request={listOperationLogs}
        scroll={{ x: 1400 }}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys as number[]),
        }}
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
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
                a.download = 'operation_logs.xlsx';
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
      <Modal
        open={Boolean(bodyModal)}
        title={bodyModal?.title}
        footer={null}
        width={720}
        onCancel={() => setBodyModal(null)}
      >
        <Typography.Paragraph copyable style={{ whiteSpace: 'pre-wrap', maxHeight: 480, overflow: 'auto' }}>
          {bodyModal?.content}
        </Typography.Paragraph>
      </Modal>
    </PageContainer>
  );
}
