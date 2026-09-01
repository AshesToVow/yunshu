import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
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
  Select,
  Space,
  Switch,
  Tag,
  Typography,
  message,
} from 'antd';
import { useEffect, useRef, useState } from 'react';
import {
  createDbInstance,
  deleteDbInstance,
  listDbInstances,
  pingDbInstance,
  updateDbInstance,
  type DbInstance,
  type DbInstancePayload,
} from '@/services/dbmgmt';
import { getProjectServers, getProjects, type ProjectItem, type ServerItem } from '@/services/projects';
import { envLabel, instanceRoleLabel } from '@/utils/dbmgmt-labels';
import { formatDateTime } from '@/utils/format';

const MYSQL_SSL_OPTIONS = [
  { value: 'disable', label: 'disable（不加密）' },
  { value: 'preferred', label: 'preferred（优先 TLS）' },
  { value: 'required', label: 'required（必须 TLS）' },
  { value: 'verify_ca', label: 'verify_ca' },
];

const PG_SSL_OPTIONS = [
  { value: 'disable', label: 'disable' },
  { value: 'prefer', label: 'prefer' },
  { value: 'require', label: 'require' },
  { value: 'verify-ca', label: 'verify-ca' },
  { value: 'verify-full', label: 'verify-full' },
];

export default function DbmgmtInstancesPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [servers, setServers] = useState<ServerItem[]>([]);
  const [rows, setRows] = useState<DbInstance[]>([]);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<DbInstance | null>(null);
  const [form] = Form.useForm<DbInstancePayload>();
  const driver = Form.useWatch('driver', form) ?? 'mysql';
  const connectMode = Form.useWatch('connect_mode', form) ?? 'direct';
  const instanceRole = Form.useWatch('role', form) ?? 'primary';

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      const list = res.list ?? [];
      setProjects(list);
      if (list.length) setProjectId(list[0].id);
    });
  }, []);

  useEffect(() => {
    if (!projectId) return;
    void getProjectServers(projectId, { page: 1, page_size: 200 }).then((res) =>
      setServers(res.list ?? []),
    );
  }, [projectId]);

  const primaryOptions = rows.filter(
    (r) => (r.role ?? 'primary') === 'primary' && r.id !== editing?.id,
  );

  const columns: ProColumns<DbInstance>[] = [
    {
      title: '项目',
      dataIndex: 'project_id',
      hideInTable: true,
      valueType: 'select',
      initialValue: projectId,
      fieldProps: {
        options: projects.map((p) => ({ label: p.name, value: p.id })),
        allowClear: false,
        style: { width: 220 },
        onChange: (v: number) => {
          setProjectId(v);
          actionRef.current?.reload();
        },
      },
    },
    {
      title: '名称',
      dataIndex: 'name',
      search: false,
      render: (_, r) => (
        <Link to={`/dbmgmt/instances/${r.id}?project=${projectId ?? ''}`}>{r.name}</Link>
      ),
    },
    {
      title: '环境',
      dataIndex: 'env',
      width: 90,
      search: false,
      render: (_, r) => <Tag>{envLabel(r.env)}</Tag>,
    },
    {
      title: '库角色',
      dataIndex: 'role',
      width: 90,
      search: false,
      render: (_, r) => (
        <Tag color={(r.role ?? 'primary') === 'replica' ? 'blue' : 'green'}>
          {instanceRoleLabel(r.role)}
        </Tag>
      ),
    },
    { title: '驱动', dataIndex: 'driver', width: 100, search: false },
    {
      title: '连接',
      search: false,
      render: (_, r) => `${r.host}:${r.port}${r.database ? ` / ${r.database}` : ''}`,
    },
    {
      title: '关联主库',
      width: 120,
      ellipsis: true,
      search: false,
      render: (_, r) =>
        r.role === 'replica'
          ? r.primary_instance_name || (r.primary_instance_id ? `#${r.primary_instance_id}` : '—')
          : '—',
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      search: false,
      render: (_, r) => (
        <Tag color={r.last_ping_ok ? 'green' : 'default'}>
          {r.last_ping_ok ? '正常' : r.status || '未知'}
        </Tag>
      ),
    },
    {
      title: '最近探活',
      dataIndex: 'last_ping_at',
      width: 170,
      search: false,
      render: (_, r) => (r.last_ping_at ? formatDateTime(r.last_ping_at) : '—'),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 260,
      render: (_, r) => [
        <Button
          key="ping"
          type="link"
          size="small"
          icon={<ThunderboltOutlined />}
          onClick={async () => {
            if (!projectId) return;
            await pingDbInstance(projectId, r.id);
            message.success('探活完成');
            actionRef.current?.reload();
          }}
        >
          探活
        </Button>,
        <Button
          key="edit"
          type="link"
          size="small"
          icon={<EditOutlined />}
          onClick={() => {
            setEditing(r);
            form.setFieldsValue({ ...r, role: r.role ?? 'primary', password: undefined });
            setOpen(true);
          }}
        >
          编辑
        </Button>,
        r.driver === 'mysql' && r.backup_link ? (
          <Link key="backup" to={r.backup_link}>
            备份
          </Link>
        ) : null,
        <Popconfirm
          key="del"
          title="确认删除？"
          onConfirm={async () => {
            if (!projectId) return;
            await deleteDbInstance(projectId, r.id);
            message.success('已删除');
            actionRef.current?.reload();
          }}
        >
          <Button type="link" size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>,
      ],
    },
  ];

  async function submit() {
    if (!projectId) return;
    const values = await form.validateFields();
    if (values.connect_mode !== 'ssh_tunnel') values.server_id = undefined;
    if ((values.role ?? 'primary') === 'primary') values.primary_instance_id = undefined;
    if (values.role === 'replica') values.read_only = true;
    if (editing) {
      await updateDbInstance(projectId, editing.id, values);
    } else {
      await createDbInstance(projectId, values);
    }
    message.success('已保存');
    setOpen(false);
    setEditing(null);
    form.resetFields();
    actionRef.current?.reload();
  }

  return (
    <PageContainer header={{ title: '数据库实例', subTitle: '项目内 MySQL / PostgreSQL 实例纳管与探活' }}>
      <ProTable<DbInstance>
        rowKey="id"
        actionRef={actionRef}
        columns={columns}
        search={{ labelWidth: 'auto' }}
        params={{ projectId }}
        pagination={false}
        request={async (params) => {
          const pid = Number(params.project_id || projectId || 0);
          if (!pid) return { data: [], success: true };
          if (pid !== projectId) setProjectId(pid);
          const res = await listDbInstances(pid, { page: 1, page_size: 200 });
          const list = res.list ?? [];
          setRows(list);
          return { data: list, success: true, total: res.total ?? list.length };
        }}
        toolBarRender={() => [
          <Button
            key="create"
            type="primary"
            icon={<PlusOutlined />}
            disabled={!projectId}
            onClick={() => {
              setEditing(null);
              form.resetFields();
              setOpen(true);
            }}
          >
            新建
          </Button>,
        ]}
      />

      <Modal
        title={editing ? '编辑实例' : '新建实例'}
        open={open}
        onCancel={() => {
          setOpen(false);
          setEditing(null);
        }}
        onOk={() => void submit()}
        width={720}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            env: 'dev',
            driver: 'mysql',
            connect_mode: 'direct',
            port: 3306,
            ssl_mode: 'disable',
            require_ticket_for_dml: true,
            role: 'primary',
          }}
        >
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="env" label="环境">
            <Select
              options={[
                { value: 'dev' },
                { value: 'test' },
                { value: 'prod' },
              ]}
            />
          </Form.Item>
          <Form.Item name="role" label="库角色" extra="从库须关联主库，并自动设为只读（禁止 DML/DDL）">
            <Select
              options={[
                { value: 'primary', label: '主库' },
                { value: 'replica', label: '从库' },
              ]}
              onChange={(v) => {
                if (v === 'replica') form.setFieldValue('read_only', true);
              }}
            />
          </Form.Item>
          {instanceRole === 'replica' ? (
            <Form.Item name="primary_instance_id" label="关联主库" rules={[{ required: true, message: '请选择主库' }]}>
              <Select
                placeholder="选择同项目主库实例"
                options={primaryOptions.map((p) => ({
                  value: p.id,
                  label: `${p.name} (${p.host}:${p.port})`,
                }))}
              />
            </Form.Item>
          ) : null}
          <Form.Item name="driver" label="驱动">
            <Select
              options={[
                { value: 'mysql', label: 'MySQL' },
                { value: 'postgres', label: 'PostgreSQL' },
              ]}
            />
          </Form.Item>
          <Form.Item name="connect_mode" label="连接模式" extra="直连：应用直接连数据库；SSH 隧道：经跳板机转发">
            <Select
              options={[
                { value: 'direct', label: '直连' },
                { value: 'ssh_tunnel', label: 'SSH 隧道' },
              ]}
            />
          </Form.Item>
          {connectMode === 'ssh_tunnel' ? (
            <Form.Item
              name="server_id"
              label="CMDB 跳板机"
              rules={[{ required: true, message: 'SSH 隧道须选择 CMDB 服务器' }]}
            >
              <Select
                allowClear
                options={servers.map((s) => ({ value: s.id, label: s.name }))}
                placeholder="选择项目内已纳管服务器"
              />
            </Form.Item>
          ) : (
            <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
              直连模式无需绑定 CMDB 服务器。
            </Typography.Text>
          )}
          <Space style={{ display: 'flex' }} align="start">
            <Form.Item name="host" label="主机" style={{ flex: 1 }} initialValue="127.0.0.1">
              <Input />
            </Form.Item>
            <Form.Item name="port" label="端口">
              <InputNumber min={1} max={65535} />
            </Form.Item>
          </Space>
          <Form.Item
            name="database"
            label="默认库"
            extra="可选。留空表示连接时不指定默认库；SQL 控制台中需手动选择目标库。"
          >
            <Input placeholder="如 yunshu（可留空）" />
          </Form.Item>
          <Form.Item name="username" label="用户名" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={editing ? [] : [{ required: true }]}>
            <Input.Password placeholder={editing ? '留空则不修改' : ''} />
          </Form.Item>
          <Form.Item
            name="ssl_mode"
            label={driver === 'postgres' ? 'SSL Mode（PostgreSQL）' : 'TLS（MySQL）'}
            extra={
              driver === 'postgres'
                ? 'PostgreSQL sslmode 参数'
                : 'MySQL 服务端若 require_secure_transport=ON，请选 required；自签证书可先选 verify_ca'
            }
          >
            <Select options={driver === 'postgres' ? PG_SSL_OPTIONS : MYSQL_SSL_OPTIONS} />
          </Form.Item>
          <Form.Item
            name="read_only"
            label="只读实例"
            valuePropName="checked"
            extra={instanceRole === 'replica' ? '从库固定为只读' : '开启后禁止 DML/DDL/导入，仅允许查询'}
          >
            <Switch disabled={instanceRole === 'replica'} />
          </Form.Item>
          <Form.Item name="require_ticket_for_dml" label="DML 须工单" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
