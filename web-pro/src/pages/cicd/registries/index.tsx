import { ApiOutlined, DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import {
  Button,
  Drawer,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useCallback, useRef, useState } from 'react';
import {
  createCleanupPolicy,
  createRegistry,
  deleteCleanupPolicy,
  deleteRegistry,
  listCleanupPolicies,
  listRegistries,
  pingRegistry,
  runCleanupPolicy,
  updateCleanupPolicy,
  updateRegistry,
  type ImageCleanupPolicy,
  type ImageRegistryItem,
} from '@/services/cicd';
import { formatDateTime } from '@/utils/format';

type RegistryForm = {
  name: string;
  type: string;
  url: string;
  host_ip?: string;
  username?: string;
  password?: string;
  default_project?: string;
  is_default?: boolean;
  status?: number;
  remark?: string;
};

type PolicyForm = {
  harbor_project?: string;
  keep_last_n: number;
  retain_days: number;
  enabled: boolean;
  cron_spec: string;
};

export default function CicdRegistriesPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<ImageRegistryItem | null>(null);
  const [form] = Form.useForm<RegistryForm>();
  const [detail, setDetail] = useState<ImageRegistryItem | null>(null);
  const [policies, setPolicies] = useState<ImageCleanupPolicy[]>([]);
  const [policyOpen, setPolicyOpen] = useState(false);
  const [policyEditing, setPolicyEditing] = useState<ImageCleanupPolicy | null>(null);
  const [policyForm] = Form.useForm<PolicyForm>();

  const loadPolicies = useCallback(async (registryId: number) => {
    const list = await listCleanupPolicies(registryId);
    setPolicies(list || []);
  }, []);

  const columns: ProColumns<ImageRegistryItem>[] = [
    { title: '名称', dataIndex: 'name', width: 160, search: false },
    {
      title: '类型',
      dataIndex: 'type',
      width: 140,
      search: false,
      render: (_, row) => (row.type === 'docker_registry' ? 'Docker Registry' : 'Harbor'),
    },
    { title: '地址', dataIndex: 'url', ellipsis: true, search: false },
    {
      title: '默认项目',
      dataIndex: 'default_project',
      width: 120,
      search: false,
      render: (_, row) => row.default_project || '—',
    },
    {
      title: '默认',
      dataIndex: 'is_default',
      width: 80,
      search: false,
      render: (_, row) => (row.is_default ? <Tag color="blue">默认</Tag> : '—'),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 80,
      search: false,
      render: (_, row) => (row.status === 1 ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 280,
      render: (_, row) => [
        <Button
          key="detail"
          type="link"
          size="small"
          onClick={async () => {
            setDetail(row);
            await loadPolicies(row.id);
          }}
        >
          详情/清理
        </Button>,
        <Button
          key="ping"
          type="link"
          size="small"
          icon={<ApiOutlined />}
          onClick={async () => {
            const r = await pingRegistry(row.id);
            message.success(r?.['ok'] ? '连通成功' : '已完成探测');
          }}
        >
          测连
        </Button>,
        <Button
          key="edit"
          type="link"
          size="small"
          icon={<EditOutlined />}
          onClick={() => {
            setEditing(row);
            form.setFieldsValue({
              name: row.name,
              type: row.type,
              url: row.url,
              host_ip: row.host_ip,
              username: row.username,
              default_project: row.default_project,
              is_default: row.is_default,
              status: row.status,
              remark: row.remark,
            });
            setOpen(true);
          }}
        >
          编辑
        </Button>,
        <Popconfirm
          key="del"
          title="确认删除该注册中心？"
          onConfirm={async () => {
            await deleteRegistry(row.id);
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

  const policyColumns: ColumnsType<ImageCleanupPolicy> = [
    { title: 'Harbor 项目', dataIndex: 'harbor_project', render: (v) => v || '全部' },
    { title: '保留最近 N', dataIndex: 'keep_last_n', width: 100 },
    { title: '保留天数', dataIndex: 'retain_days', width: 90 },
    { title: 'Cron', dataIndex: 'cron_spec', width: 120 },
    {
      title: '启用',
      dataIndex: 'enabled',
      width: 70,
      render: (v: boolean) => (v ? <Tag color="success">是</Tag> : <Tag>否</Tag>),
    },
    {
      title: '上次执行',
      dataIndex: 'last_run_at',
      width: 168,
      render: (v, row) => (
        <div>
          <div>{formatDateTime(v)}</div>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {row.last_result || '—'}
          </Typography.Text>
        </div>
      ),
    },
    {
      title: '操作',
      width: 200,
      render: (_, row) => (
        <Space>
          <Button
            type="link"
            size="small"
            onClick={() => {
              setPolicyEditing(row);
              policyForm.setFieldsValue({
                harbor_project: row.harbor_project,
                keep_last_n: row.keep_last_n,
                retain_days: row.retain_days,
                enabled: row.enabled,
                cron_spec: row.cron_spec,
              });
              setPolicyOpen(true);
            }}
          >
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            onClick={async () => {
              const r = await runCleanupPolicy(row.id);
              message.success(r.result || '已触发');
              if (detail) await loadPolicies(detail.id);
            }}
          >
            立即执行
          </Button>
          <Popconfirm
            title="删除策略？"
            onConfirm={async () => {
              await deleteCleanupPolicy(row.id);
              if (detail) await loadPolicies(detail.id);
            }}
          >
            <Button type="link" size="small" danger>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  async function submit() {
    const values = await form.validateFields();
    const payload = { ...values, password: values.password || undefined };
    if (editing) {
      await updateRegistry(editing.id, payload);
      message.success('已更新');
    } else {
      await createRegistry(payload);
      message.success('已创建');
    }
    setOpen(false);
    actionRef.current?.reload();
  }

  async function submitPolicy() {
    if (!detail) return;
    const values = await policyForm.validateFields();
    const payload = { ...values, registry_id: detail.id };
    if (policyEditing) {
      await updateCleanupPolicy(policyEditing.id, payload);
      message.success('策略已更新');
    } else {
      await createCleanupPolicy(payload);
      message.success('策略已创建');
    }
    setPolicyOpen(false);
    await loadPolicies(detail.id);
  }

  return (
    <PageContainer
      header={{
        title: '镜像仓库注册中心',
        subTitle: 'Harbor / Docker Registry 连接配置、默认仓库与 Tag 清理策略',
      }}
    >
      <ProTable<ImageRegistryItem>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        search={false}
        pagination={false}
        request={async () => {
          const data = await listRegistries({ page: 1, page_size: 100 });
          return { data: data.list || [], success: true };
        }}
        toolBarRender={() => [
          <Button
            key="create"
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditing(null);
              form.resetFields();
              form.setFieldsValue({ type: 'harbor', status: 1, is_default: false });
              setOpen(true);
            }}
          >
            新建
          </Button>,
        ]}
      />

      <Modal
        title={editing ? '编辑注册中心' : '新建注册中心'}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void submit()}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label="类型" rules={[{ required: true }]}>
            <Select
              options={[
                { label: 'Harbor', value: 'harbor' },
                { label: 'Docker Registry', value: 'docker_registry' },
              ]}
            />
          </Form.Item>
          <Form.Item name="url" label="地址" rules={[{ required: true }]} extra="如 harbor.example.com">
            <Input />
          </Form.Item>
          <Form.Item name="host_ip" label="Host IP" extra="可选，用于 hostAliases 拨号">
            <Input allowClear />
          </Form.Item>
          <Form.Item name="username" label="用户名">
            <Input allowClear />
          </Form.Item>
          <Form.Item
            name="password"
            label="密码"
            extra={editing?.has_password ? '留空则保持原密码' : undefined}
          >
            <Input.Password allowClear placeholder={editing?.has_password ? '********' : undefined} />
          </Form.Item>
          <Form.Item name="default_project" label="默认 Harbor 项目">
            <Input allowClear />
          </Form.Item>
          <Form.Item name="is_default" label="设为默认" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select
              options={[
                { label: '启用', value: 1 },
                { label: '停用', value: 0 },
              ]}
            />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer title={detail ? `注册中心 — ${detail.name}` : '详情'} width={880} open={!!detail} onClose={() => setDetail(null)}>
        {detail && (
          <Tabs
            items={[
              {
                key: 'info',
                label: '基本信息',
                children: (
                  <div style={{ display: 'grid', gridTemplateColumns: '120px 1fr', gap: 8 }}>
                    {(
                      [
                        ['类型', detail.type],
                        ['地址', detail.url],
                        ['Host IP', detail.host_ip],
                        ['用户名', detail.username],
                        ['默认项目', detail.default_project],
                        ['备注', detail.remark],
                      ] as [string, string | undefined][]
                    ).map(([k, v]) => (
                      <span key={k}>
                        <Typography.Text type="secondary">{k}</Typography.Text>
                        <div>{v || '—'}</div>
                      </span>
                    ))}
                  </div>
                ),
              },
              {
                key: 'cleanup',
                label: '清理策略',
                children: (
                  <>
                    <Button
                      type="primary"
                      size="small"
                      icon={<PlusOutlined />}
                      style={{ marginBottom: 12 }}
                      onClick={() => {
                        setPolicyEditing(null);
                        policyForm.resetFields();
                        policyForm.setFieldsValue({
                          keep_last_n: 10,
                          retain_days: 30,
                          enabled: true,
                          cron_spec: '0 3 * * *',
                        });
                        setPolicyOpen(true);
                      }}
                    >
                      新增策略
                    </Button>
                    <Table
                      rowKey="id"
                      size="small"
                      columns={policyColumns}
                      dataSource={policies}
                      pagination={false}
                    />
                  </>
                ),
              },
            ]}
          />
        )}
      </Drawer>

      <Modal
        title={policyEditing ? '编辑清理策略' : '新增清理策略'}
        open={policyOpen}
        onCancel={() => setPolicyOpen(false)}
        onOk={() => void submitPolicy()}
        destroyOnClose
      >
        <Form form={policyForm} layout="vertical">
          <Form.Item name="harbor_project" label="Harbor 项目" extra="留空表示全部 project">
            <Input allowClear />
          </Form.Item>
          <Form.Item name="keep_last_n" label="保留最近 Tag 数" rules={[{ required: true }]}>
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="retain_days" label="保留天数" rules={[{ required: true }]}>
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="cron_spec" label="Cron" rules={[{ required: true }]}>
            <Input placeholder="0 3 * * *" />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
