import { DeleteOutlined, PlusOutlined, ThunderboltOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Link } from '@umijs/max';
import {
  Button,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Space,
  Switch,
  Tag,
  message,
} from 'antd';
import { useRef, useState } from 'react';
import {
  createEsmgmtConnection,
  deleteEsmgmtConnection,
  listEsmgmtConnections,
  pingEsmgmtConnection,
  testEsmgmtConnection,
  updateEsmgmtConnection,
  type EsmgmtConnection,
} from '@/services/esmgmt';
import { extractApiErrorMessage } from '@/services/http';

export default function EsmgmtConnectionsPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [open, setOpen] = useState(false);
  const [testing, setTesting] = useState(false);
  const [current, setCurrent] = useState<EsmgmtConnection | null>(null);
  const [form] = Form.useForm();

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ timeout_sec: 30, is_default: false });
    setOpen(true);
  }

  function openEdit(row: EsmgmtConnection) {
    setCurrent(row);
    form.setFieldsValue({
      name: row.name,
      addresses: row.addresses,
      username: row.username,
      timeout_sec: row.timeout_sec || 30,
      is_default: row.is_default,
      remark: row.remark,
      password: '',
    });
    setOpen(true);
  }

  async function onTest() {
    const values = await form.validateFields(['addresses', 'username', 'password', 'timeout_sec']);
    setTesting(true);
    try {
      const res = await testEsmgmtConnection({
        addresses: values.addresses,
        username: values.username,
        password: values.password,
        timeout_sec: values.timeout_sec,
        connection_id: current?.id,
      });
      if (res?.ok) message.success('连通成功');
      else message.error(res?.message || '连通失败');
    } catch (e) {
      message.error(extractApiErrorMessage(e, '连通失败'));
    } finally {
      setTesting(false);
    }
  }

  async function onSubmit() {
    const values = await form.validateFields();
    try {
      if (current) {
        await updateEsmgmtConnection(current.id, values);
        message.success('已更新');
      } else {
        await createEsmgmtConnection(values);
        message.success('已创建');
      }
      setOpen(false);
      actionRef.current?.reload();
    } catch (e) {
      message.error(extractApiErrorMessage(e, '保存失败'));
    }
  }

  const columns: ProColumns<EsmgmtConnection>[] = [
    { title: '名称', dataIndex: 'name', search: false },
    { title: '地址', dataIndex: 'addresses', ellipsis: true, search: false },
    { title: '用户', dataIndex: 'username', width: 120, search: false },
    {
      title: '默认',
      dataIndex: 'is_default',
      width: 80,
      search: false,
      render: (_, r) => (r.is_default ? <Tag color="blue">是</Tag> : '—'),
    },
    {
      title: '密码',
      width: 80,
      search: false,
      render: (_, r) => (r.has_password ? '已配置' : '—'),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 220,
      render: (_, row) => [
        <Button
          key="ping"
          type="link"
          size="small"
          icon={<ThunderboltOutlined />}
          onClick={async () => {
            try {
              const res = await pingEsmgmtConnection(row.id);
              if (res?.ok) message.success('连通成功');
              else message.error(res?.message || '连通失败');
            } catch (e) {
              message.error(extractApiErrorMessage(e, '连通失败'));
            }
          }}
        >
          探测
        </Button>,
        <Button key="edit" type="link" size="small" onClick={() => openEdit(row)}>
          编辑
        </Button>,
        <Popconfirm
          key="del"
          title="确认删除？"
          onConfirm={async () => {
            await deleteEsmgmtConnection(row.id);
            message.success('已删除');
            actionRef.current?.reload();
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
    <PageContainer
      header={{
        title: 'ES 连接管理',
        subTitle:
          '维护 Elasticsearch 连接（密码加密存储）。集群概览与 REST 控制台共用这些连接。',
        extra: (
          <Space>
            <Link to="/esmgmt/overview">集群概览</Link>
            <Link to="/esmgmt/console">REST 控制台</Link>
          </Space>
        ),
      }}
    >
      <ProTable<EsmgmtConnection>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        search={false}
        pagination={{ defaultPageSize: 20, showSizeChanger: true }}
        locale={{ emptyText: '暂无连接，请新建或检查权限' }}
        request={async () => {
          const list = (await listEsmgmtConnections()) || [];
          return { data: list, success: true, total: list.length };
        }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建连接
          </Button>,
        ]}
      />

      <Modal
        title={current ? '编辑连接' : '新建连接'}
        open={open}
        onCancel={() => setOpen(false)}
        destroyOnClose
        footer={
          <Space>
            <Button onClick={() => setOpen(false)}>取消</Button>
            <Button loading={testing} icon={<ThunderboltOutlined />} onClick={() => void onTest()}>
              连通测试
            </Button>
            <Button type="primary" onClick={() => void onSubmit()}>
              保存
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="addresses" label="地址（逗号分隔）" rules={[{ required: true }]}>
            <Input.TextArea rows={2} placeholder="http://127.0.0.1:9200" />
          </Form.Item>
          <Form.Item name="username" label="用户名">
            <Input autoComplete="username" />
          </Form.Item>
          <Form.Item name="password" label={current ? '密码（留空不改，测试时可复用已存密码）' : '密码'}>
            <Input.Password autoComplete="new-password" />
          </Form.Item>
          <Form.Item name="timeout_sec" label="超时秒">
            <InputNumber min={5} max={300} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="is_default" label="默认连接" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
