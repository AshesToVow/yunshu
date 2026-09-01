import {
  DeleteOutlined,
  EditOutlined,
  ExportOutlined,
  LockOutlined,
  PlusOutlined,
  UserSwitchOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns, ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  PageContainer,
  ProFormSelect,
  ProFormText,
  ProFormTreeSelect,
  ProTable,
} from '@ant-design/pro-components';
import { useModel } from '@umijs/max';
import { Button, Modal, Popconfirm, Space, Tag, Tree, Typography, message } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import { getDepartmentTree } from '@/services/departments';
import { getRoleOptions } from '@/services/roles';
import {
  assignUserRoles,
  createUser,
  deleteUser,
  exportUsers,
  getUser,
  getUsers,
  updateUser,
} from '@/services/users';
import type { DepartmentItem, RoleItem, UserCreatePayload, UserItem, UserUpdatePayload } from '@/types/api';
import { formatDateTime } from '@/utils/format';
import { buildRoleTreeData, normalizeCheckedKeys } from '@/utils/tree';

function toDeptTree(nodes: DepartmentItem[]): { title: string; value: number; children?: ReturnType<typeof toDeptTree> }[] {
  return nodes.map((n) => ({
    title: n.name,
    value: n.id,
    children: n.children?.length ? toDeptTree(n.children) : undefined,
  }));
}

function toDeptFilterOptions(nodes: DepartmentItem[], prefix = ''): { label: string; value: number }[] {
  const out: { label: string; value: number }[] = [];
  for (const n of nodes) {
    out.push({ label: `${prefix}${n.name}`, value: n.id });
    if (n.children?.length) out.push(...toDeptFilterOptions(n.children, `${prefix}${n.name} / `));
  }
  return out;
}

export default function UsersPage() {
  const actionRef = useRef<ActionType>(undefined);
  const formRef = useRef<ProFormInstance>(undefined);
  const { initialState } = useModel('@@initialState');
  const currentUser = initialState?.currentUser;
  const isSuperAdmin = Boolean(
    (currentUser as { roles?: Array<{ code: string }> } | undefined)?.roles?.some((r) => r.code === 'super-admin'),
  );
  const currentUserId = Number(currentUser?.userid ?? (currentUser as { id?: number } | undefined)?.id);

  const [departments, setDepartments] = useState<DepartmentItem[]>([]);
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<UserItem | null>(null);
  const [assignOpen, setAssignOpen] = useState(false);
  const [assignTarget, setAssignTarget] = useState<UserItem | null>(null);
  const [checkedRoleIds, setCheckedRoleIds] = useState<number[]>([]);
  const [resetOpen, setResetOpen] = useState(false);
  const [resetTarget, setResetTarget] = useState<UserItem | null>(null);

  const deptTree = useMemo(() => toDeptTree(departments), [departments]);
  const deptFilterOptions = useMemo(() => toDeptFilterOptions(departments), [departments]);
  const roleOptions = useMemo(
    () => roles.map((r) => ({ label: `${r.name} (${r.code})`, value: r.id })),
    [roles],
  );
  const roleTreeData = useMemo(() => buildRoleTreeData(roles), [roles]);
  const roleIdSet = useMemo(() => new Set(roles.map((r) => r.id)), [roles]);

  useEffect(() => {
    void Promise.all([
      getDepartmentTree().then(setDepartments),
      getRoleOptions().then((res) => setRoles(res.list ?? [])),
    ]);
  }, []);

  async function openEdit(record: UserItem) {
    const detail = await getUser(record.id);
    setEditTarget(detail);
    setEditOpen(true);
  }

  function openAssign(record: UserItem) {
    setAssignTarget(record);
    setCheckedRoleIds(record.roles.map((r) => r.id));
    setAssignOpen(true);
  }

  function openReset(record: UserItem) {
    if (!isSuperAdmin) {
      message.warning('仅超级管理员可修改其他用户密码');
      return;
    }
    if (currentUserId && record.id === currentUserId) {
      message.warning('不能在此修改当前登录账号的密码');
      return;
    }
    setResetTarget(record);
    setResetOpen(true);
  }

  const columns: ProColumns<UserItem>[] = [
    { title: 'ID', dataIndex: 'id', width: 70, search: false },
    {
      title: '账号',
      dataIndex: 'username',
      render: (_, row) => (
        <Space direction="vertical" size={0}>
          <Typography.Text strong>{row.nickname}</Typography.Text>
          <Typography.Text type="secondary">{row.username}</Typography.Text>
        </Space>
      ),
    },
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '账号 / 昵称' },
    },
    {
      title: '部门',
      dataIndex: 'department_id',
      valueType: 'select',
      fieldProps: { options: deptFilterOptions, allowClear: true, showSearch: true },
      render: (_, row) => row.department_name || '—',
    },
    {
      title: '责任域角色',
      dataIndex: 'roles',
      search: false,
      render: (_, row) =>
        row.roles?.length ? (
          <Space wrap size={[4, 4]}>
            {row.roles.map((r) => (
              <Tag key={r.id}>{r.name}</Tag>
            ))}
          </Space>
        ) : (
          '—'
        ),
    },
    { title: '邮箱', dataIndex: 'email', search: false, ellipsis: true },
    {
      title: '状态',
      dataIndex: 'status',
      width: 88,
      search: false,
      valueEnum: {
        1: { text: '启用', status: 'Success' },
        0: { text: '停用', status: 'Default' },
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
      width: 260,
      render: (_, row) => [
        <Button key="edit" type="link" size="small" icon={<EditOutlined />} onClick={() => void openEdit(row)}>
          编辑
        </Button>,
        <Button key="roles" type="link" size="small" icon={<UserSwitchOutlined />} onClick={() => openAssign(row)}>
          角色
        </Button>,
        <Button key="pwd" type="link" size="small" icon={<LockOutlined />} onClick={() => openReset(row)}>
          重置密码
        </Button>,
        <Popconfirm
          key="del"
          title={`确认删除账号 ${row.username}？`}
          onConfirm={async () => {
            await deleteUser(row.id);
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
    <PageContainer title="用户管理" subTitle="账号、部门归属与责任域角色">
      <ProTable<UserItem>
        actionRef={actionRef}
        formRef={formRef}
        rowKey="id"
        columns={columns}
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
        request={async (params) => {
          const res = await getUsers({
            keyword: params.keyword as string | undefined,
            department_id: params.department_id as number | undefined,
            page: params.current,
            page_size: params.pageSize,
          });
          return { data: res.list ?? [], total: res.total ?? 0, success: true };
        }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            新建账号
          </Button>,
          <Button
            key="export"
            icon={<ExportOutlined />}
            onClick={() => {
              const form = formRef.current?.getFieldsValue?.() ?? {};
              void exportUsers({ keyword: form.keyword, department_id: form.department_id }).then((blob) => {
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = 'users.xlsx';
                a.click();
                window.URL.revokeObjectURL(url);
              });
            }}
          >
            导出
          </Button>,
        ]}
      />

      <ModalForm<UserCreatePayload>
        title="新建账号"
        open={createOpen}
        modalProps={{ destroyOnClose: true, onCancel: () => setCreateOpen(false) }}
        initialValues={{ status: 1, role_ids: [] }}
        onFinish={async (values) => {
          await createUser(values);
          message.success('账号创建成功');
          setCreateOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText name="username" label="用户名" rules={[{ required: true }]} />
        <ProFormText.Password name="password" label="初始密码" rules={[{ required: true, min: 8 }]} />
        <ProFormText name="nickname" label="昵称" rules={[{ required: true }]} />
        <ProFormText name="email" label="邮箱" rules={[{ required: true, type: 'email' }]} />
        <ProFormText name="phone" label="手机号" />
        <ProFormTreeSelect
          name="department_id"
          label="部门"
          allowClear
          fieldProps={{ treeDefaultExpandAll: true, treeData: deptTree }}
        />
        <ProFormSelect
          name="role_ids"
          label="责任域角色"
          mode="multiple"
          options={roleOptions}
          rules={[{ required: true, message: '请至少选择一个角色' }]}
        />
        <ProFormSelect
          name="status"
          label="状态"
          options={[
            { label: '启用', value: 1 },
            { label: '停用', value: 0 },
          ]}
          rules={[{ required: true }]}
        />
      </ModalForm>

      <ModalForm<UserUpdatePayload>
        title={editTarget ? `编辑账号 #${editTarget.id}` : '编辑账号'}
        open={editOpen}
        modalProps={{ destroyOnClose: true, onCancel: () => setEditOpen(false) }}
        initialValues={
          editTarget
            ? {
                nickname: editTarget.nickname,
                email: editTarget.email,
                phone: editTarget.phone,
                status: editTarget.status,
                department_id: editTarget.department_id,
              }
            : undefined
        }
        onFinish={async (values) => {
          if (!editTarget) return false;
          await updateUser(editTarget.id, values);
          message.success('账号已更新');
          setEditOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText name="nickname" label="昵称" rules={[{ required: true }]} />
        <ProFormText name="email" label="邮箱" rules={[{ required: true, type: 'email' }]} />
        <ProFormText name="phone" label="手机号" />
        <ProFormTreeSelect
          name="department_id"
          label="部门"
          allowClear
          fieldProps={{ treeDefaultExpandAll: true, treeData: deptTree }}
        />
        <ProFormSelect
          name="status"
          label="状态"
          options={[
            { label: '启用', value: 1 },
            { label: '停用', value: 0 },
          ]}
          rules={[{ required: true }]}
        />
      </ModalForm>

      <Modal
        title={assignTarget ? `分配角色 — ${assignTarget.nickname}` : '分配角色'}
        open={assignOpen}
        onCancel={() => setAssignOpen(false)}
        onOk={async () => {
          if (!assignTarget) return;
          const roleIds = checkedRoleIds.filter((id) => roleIdSet.has(id));
          await assignUserRoles(assignTarget.id, { role_ids: roleIds });
          message.success('责任域角色已同步');
          setAssignOpen(false);
          actionRef.current?.reload();
        }}
      >
        <Tree
          checkable
          defaultExpandAll
          checkedKeys={checkedRoleIds}
          treeData={roleTreeData}
          onCheck={(keys) => setCheckedRoleIds(normalizeCheckedKeys(keys).map(Number))}
        />
      </Modal>

      <ModalForm<{ password: string; confirm: string }>
        title={resetTarget ? `重置密码 — ${resetTarget.username}` : '重置密码'}
        open={resetOpen}
        modalProps={{ destroyOnClose: true, onCancel: () => setResetOpen(false) }}
        onFinish={async (values) => {
          if (!resetTarget) return false;
          if (values.password !== values.confirm) {
            message.error('两次输入的新密码不一致');
            return false;
          }
          await updateUser(resetTarget.id, { password: values.password });
          message.success('已更新该账号的登录密码');
          setResetOpen(false);
          return true;
        }}
      >
        <ProFormText.Password name="password" label="新密码" rules={[{ required: true, min: 8 }]} />
        <ProFormText.Password name="confirm" label="确认密码" rules={[{ required: true, min: 8 }]} />
      </ModalForm>
    </PageContainer>
  );
}
