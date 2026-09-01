import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Link, useLocation } from '@umijs/max';
import { Button, Form, Input, InputNumber, Modal, Popconfirm, message } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import { formatInstanceLabel } from '@/components/dbmgmt/dbmgmt-ui-shared';
import {
  GrantValidityCalendarPicker,
  expiresAtToGrantPeriod,
  grantPeriodToExpiresAt,
  type GrantValidityPeriod,
} from '@/components/dbmgmt/grant-validity-calendar';
import {
  deleteDbGrant,
  listDbGrants,
  listDbInstances,
  updateDbGrant,
  type DbGrant,
  type DbInstance,
} from '@/services/dbmgmt';
import { getProjects, type ProjectItem } from '@/services/projects';
import { getUsers } from '@/services/users';
import type { UserItem } from '@/types/api';

function isQueryGrant(g: DbGrant) {
  const privs = g.privileges ?? [];
  if (privs.length) return privs.every((p) => p.toLowerCase() === 'select');
  return g.can_query && !g.can_dml && !g.can_ddl && !g.can_import && !g.can_export;
}

function formatExpiry(v?: string) {
  if (!v) return '永久';
  const d = v.slice(0, 10);
  if (!d || d >= '9999') return '永久';
  return d;
}

function privSummary(g: DbGrant) {
  const parts: string[] = [];
  if (g.can_query) parts.push('查询');
  if (g.can_dml) parts.push('DML');
  if (g.can_ddl) parts.push('DDL');
  if (g.can_import) parts.push('导入');
  if (g.can_export) parts.push('导出');
  if (g.can_manage) parts.push('管理');
  if (g.privileges?.length) return g.privileges.join(', ');
  return parts.length ? parts.join(' / ') : '—';
}

export default function DbmgmtGrantsPage() {
  const location = useLocation();
  const preset: 'all' | 'query' = location.pathname.includes('query-grants') ? 'query' : 'all';
  const actionRef = useRef<ActionType>(undefined);
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [instances, setInstances] = useState<DbInstance[]>([]);
  const [users, setUsers] = useState<UserItem[]>([]);
  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<DbGrant>();
  const [editForm] = Form.useForm();

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      const list = res.list ?? [];
      setProjects(list);
      if (list.length) setProjectId(list[0].id);
    });
    void getUsers({ page: 1, page_size: 500 }).then((res) => setUsers(res.list ?? []));
  }, []);

  useEffect(() => {
    if (!projectId) return;
    void listDbInstances(projectId, { page: 1, page_size: 200 }).then((res) =>
      setInstances(res.list ?? []),
    );
  }, [projectId]);

  const instanceMap = useMemo(() => new Map(instances.map((i) => [i.id, i])), [instances]);
  const userNameMap = useMemo(() => {
    const m = new Map<string, string>();
    for (const u of users) {
      m.set(String(u.id), u.nickname || u.username);
      m.set(u.username, u.nickname || u.username);
    }
    return m;
  }, [users]);

  const displayName = (g: DbGrant) => {
    if (g.principal_kind === 'user') return userNameMap.get(g.principal_ref) ?? g.principal_ref;
    return g.principal_ref;
  };

  const instanceIp = (id: number) => {
    const inst = instanceMap.get(id);
    return inst ? formatInstanceLabel(inst) : String(id);
  };

  const openEdit = (row: DbGrant) => {
    setEditing(row);
    editForm.setFieldsValue({
      query_limit_num: row.query_limit_num ?? 1000,
      grant_period: expiresAtToGrantPeriod(row.expires_at),
      remark: row.remark,
    });
    setEditOpen(true);
  };

  const submitEdit = async () => {
    if (!projectId || !editing) return;
    const values = await editForm.validateFields();
    const period = values.grant_period as GrantValidityPeriod | null | undefined;
    await updateDbGrant(projectId, editing.id, {
      query_limit_num: values.query_limit_num,
      expires_at: grantPeriodToExpiresAt(period ?? null),
      remark: values.remark,
    });
    message.success('已更新查询权限');
    setEditOpen(false);
    actionRef.current?.reload();
  };

  const queryColumns: ProColumns<DbGrant>[] = [
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
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '名称 / 实例 / 库' },
    },
    { title: 'ID', dataIndex: 'id', width: 70, search: false },
    {
      title: '中文名称',
      search: false,
      render: (_, r) => displayName(r),
    },
    {
      title: '实例IP',
      dataIndex: 'instance_id',
      search: false,
      render: (_, r) => instanceIp(r.instance_id),
    },
    {
      title: '权限级别',
      search: false,
      render: (_, r) => (r.table_names?.length ? 'TABLE' : 'DATABASE'),
    },
    {
      title: '数据库',
      dataIndex: 'database_name',
      search: false,
      render: (_, r) => r.database_name || '—',
    },
    {
      title: '表',
      search: false,
      render: (_, r) => (r.table_names?.length ? r.table_names.join(', ') : '—'),
    },
    {
      title: '结果集',
      dataIndex: 'query_limit_num',
      width: 90,
      search: false,
      render: (_, r) => r.query_limit_num ?? 1000,
    },
    {
      title: '有效时间',
      dataIndex: 'expires_at',
      width: 120,
      search: false,
      render: (_, r) => formatExpiry(r.expires_at),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 140,
      render: (_, r) => [
        <Button
          key="edit"
          size="small"
          type="primary"
          icon={<EditOutlined />}
          onClick={() => openEdit(r)}
        >
          编辑
        </Button>,
        <Popconfirm
          key="del"
          title="删除该查询权限？"
          onConfirm={async () => {
            if (!projectId) return;
            await deleteDbGrant(projectId, r.id);
            message.success('已删除');
            actionRef.current?.reload();
          }}
        >
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>,
      ],
    },
  ];

  const allColumns: ProColumns<DbGrant>[] = [
    {
      title: '项目',
      dataIndex: 'project_id',
      hideInTable: true,
      valueType: 'select',
      initialValue: projectId,
      fieldProps: {
        options: projects.map((p) => ({ label: p.name, value: p.id })),
        allowClear: false,
        style: { width: 200 },
        onChange: (v: number) => {
          setProjectId(v);
          actionRef.current?.reload();
        },
      },
    },
    { title: 'ID', dataIndex: 'id', width: 70, search: false },
    {
      title: '主体',
      search: false,
      render: (_, r) => displayName(r),
    },
    { title: '类型', dataIndex: 'principal_kind', width: 80, search: false },
    {
      title: '实例',
      dataIndex: 'instance_id',
      search: false,
      render: (_, r) => instanceIp(r.instance_id),
    },
    {
      title: '数据库',
      dataIndex: 'database_name',
      search: false,
      render: (_, r) => r.database_name || '—',
    },
    {
      title: '权限',
      search: false,
      render: (_, r) => privSummary(r),
    },
    {
      title: '有效期',
      dataIndex: 'expires_at',
      width: 120,
      search: false,
      render: (_, r) => formatExpiry(r.expires_at),
    },
    {
      title: '备注',
      dataIndex: 'remark',
      ellipsis: true,
      search: false,
      render: (_, r) => r.remark || '—',
    },
    {
      title: '操作',
      valueType: 'option',
      width: 80,
      render: (_, r) => [
        <Popconfirm
          key="del"
          title="删除该授权？"
          onConfirm={async () => {
            if (!projectId) return;
            await deleteDbGrant(projectId, r.id);
            message.success('已删除');
            actionRef.current?.reload();
          }}
        >
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>,
      ],
    },
  ];

  return (
    <PageContainer
      header={{
        title: preset === 'query' ? 'SQL查询权限管理' : '授权管理',
        subTitle:
          preset === 'query' ? '查询授权列表、结果集上限与有效期' : '项目内数据库主体授权一览',
      }}
    >
      <ProTable<DbGrant>
        rowKey="id"
        actionRef={actionRef}
        columns={preset === 'query' ? queryColumns : allColumns}
        search={preset === 'query' ? { labelWidth: 'auto' } : { filterType: 'light' }}
        params={{ projectId, preset }}
        request={async (params) => {
          const pid = Number(params.project_id || projectId || 0);
          if (!pid) return { data: [], success: true };
          if (pid !== projectId) setProjectId(pid);
          const grants = await listDbGrants(pid);
          let list = (grants ?? []).filter((g) => (preset === 'query' ? isQueryGrant(g) : true));
          const kw = String(params.keyword || '')
            .trim()
            .toLowerCase();
          if (kw) {
            list = list.filter((g) => {
              const inst = instanceMap.get(g.instance_id);
              const label = inst ? formatInstanceLabel(inst).toLowerCase() : '';
              return (
                displayName(g).toLowerCase().includes(kw) ||
                label.includes(kw) ||
                (g.database_name ?? '').toLowerCase().includes(kw)
              );
            });
          }
          return { data: list, success: true, total: list.length };
        }}
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
        toolBarRender={() => {
          const btns = [
            <Button key="reload" icon={<ReloadOutlined />} onClick={() => actionRef.current?.reload()}>
              刷新
            </Button>,
          ];
          if (preset === 'query') {
            btns.unshift(
              <Link key="apply" to="/dbmgmt/apply/query">
                <Button type="primary" icon={<PlusOutlined />}>
                  SQL查询权限申请
                </Button>
              </Link>,
            );
          }
          return btns;
        }}
      />

      <Modal
        title="编辑查询权限"
        open={editOpen}
        onCancel={() => setEditOpen(false)}
        onOk={() => void submitEdit()}
        destroyOnClose
        width={760}
      >
        {editing ? (
          <Form form={editForm} layout="vertical">
            <Form.Item label="用户">
              <Input value={displayName(editing)} disabled />
            </Form.Item>
            <Form.Item label="实例">
              <Input value={instanceIp(editing.instance_id)} disabled />
            </Form.Item>
            <Form.Item label="数据库">
              <Input value={editing.database_name || '—'} disabled />
            </Form.Item>
            <Form.Item name="query_limit_num" label="结果集行数上限" rules={[{ required: true }]}>
              <InputNumber min={1} max={100000} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item
              name="grant_period"
              label="授权有效期"
              extra="在日历上调整起止日期；点选「永久有效」表示不过期"
            >
              <GrantValidityCalendarPicker />
            </Form.Item>
            <Form.Item name="remark" label="备注">
              <Input.TextArea rows={2} />
            </Form.Item>
          </Form>
        ) : null}
      </Modal>
    </PageContainer>
  );
}
