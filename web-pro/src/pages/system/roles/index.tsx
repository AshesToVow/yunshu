import { DeleteOutlined, EditOutlined, PlusOutlined, UserOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  PageContainer,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { Button, Modal, Popconfirm, Tag, Transfer, message } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { createRole, deleteRole, getRole, getRoles, updateRole } from '@/services/roles';
import { assignUserRoles, getUsers } from '@/services/users';
import type { RoleItem, RolePayload, UserItem } from '@/types/api';
import { formatDateTime } from '@/utils/format';

export default function RolesPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [current, setCurrent] = useState<RoleItem | null>(null);
  const [assignOpen, setAssignOpen] = useState(false);
  const [assignTarget, setAssignTarget] = useState<RoleItem | null>(null);
  const [users, setUsers] = useState<UserItem[]>([]);
  const [checkedUserIds, setCheckedUserIds] = useState<number[]>([]);

  useEffect(() => {
    void getUsers({ page: 1, page_size: 1000 }).then((res) => setUsers(res.list ?? []));
  }, []);

  async function openEdit(record: RoleItem) {
    const detail = await getRole(record.id);
    setCurrent(detail);
    setEditOpen(true);
  }

  function openAssign(record: RoleItem) {
    setAssignTarget(record);
    setCheckedUserIds(
      users.filter((u) => u.roles.some((r) => r.id === record.id)).map((u) => u.id),
    );
    setAssignOpen(true);
  }

  const columns: ProColumns<RoleItem>[] = [
    { title: 'ID', dataIndex: 'id', width: 70, search: false },
    { title: '模板名称', dataIndex: 'name' },
    {
      title: '模板编码',
      dataIndex: 'code',
      render: (_, row) => <Tag color="blue">{row.code}</Tag>,
    },
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '搜索名称或编码' },
    },
    {
      title: '说明',
      dataIndex: 'description',
      search: false,
      ellipsis: true,
      render: (_, row) => row.description || '—',
    },
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
      title: '更新时间',
      dataIndex: 'updated_at',
      width: 168,
      search: false,
      render: (_, row) => formatDateTime(row.updated_at),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 220,
      render: (_, row) => [
        <Button key="edit" type="link" size="small" icon={<EditOutlined />} onClick={() => void openEdit(row)}>
          编辑
        </Button>,
        <Button key="assign" type="link" size="small" icon={<UserOutlined />} onClick={() => openAssign(row)}>
          分配用户
        </Button>,
        <Popconfirm
          key="del"
          title={`确认删除模板 ${row.name}？`}
          onConfirm={async () => {
            await deleteRole(row.id);
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
    <PageContainer title="角色管理" subTitle="角色模板用于责任域授权与权限绑定">
      <ProTable<RoleItem>
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
        request={async (params) => {
          const res = await getRoles({
            keyword: params.keyword as string | undefined,
            page: params.current,
            page_size: params.pageSize,
          });
          return { data: res.list ?? [], total: res.total ?? 0, success: true };
        }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            新建角色模板
          </Button>,
        ]}
      />

      <ModalForm<RolePayload>
        title="新建角色模板"
        open={createOpen}
        modalProps={{ destroyOnClose: true, onCancel: () => setCreateOpen(false) }}
        initialValues={{ status: 1 }}
        onFinish={async (values) => {
          await createRole(values);
          message.success('角色模板创建成功');
          setCreateOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText name="name" label="模板名称" rules={[{ required: true }]} />
        <ProFormText name="code" label="模板编码" rules={[{ required: true }]} />
        <ProFormTextArea name="description" label="说明" />
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

      <ModalForm<RolePayload>
        title={current ? `编辑角色 #${current.id}` : '编辑角色'}
        open={editOpen}
        modalProps={{ destroyOnClose: true, onCancel: () => setEditOpen(false) }}
        initialValues={
          current
            ? {
                name: current.name,
                code: current.code,
                description: current.description,
                status: current.status,
              }
            : undefined
        }
        onFinish={async (values) => {
          if (!current) return false;
          await updateRole(current.id, values);
          message.success('角色详情已更新');
          setEditOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText name="name" label="模板名称" rules={[{ required: true }]} />
        <ProFormText name="code" label="模板编码" rules={[{ required: true }]} />
        <ProFormTextArea name="description" label="说明" />
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
        title={assignTarget ? `分配用户 — ${assignTarget.name}` : '分配用户'}
        open={assignOpen}
        width={720}
        onCancel={() => setAssignOpen(false)}
        onOk={async () => {
          if (!assignTarget) return;
          await Promise.all(
            checkedUserIds.map(async (userId) => {
              const user = users.find((u) => u.id === userId);
              if (!user) return;
              const roleIds = user.roles.map((r) => r.id);
              if (!roleIds.includes(assignTarget.id)) {
                await assignUserRoles(userId, { role_ids: [...roleIds, assignTarget.id] });
              }
            }),
          );
          message.success('用户角色已更新');
          setAssignOpen(false);
          const refreshed = await getUsers({ page: 1, page_size: 1000 });
          setUsers(refreshed.list ?? []);
        }}
      >
        <Transfer
          dataSource={users.map((u) => ({ key: String(u.id), title: `${u.nickname} (${u.username})` }))}
          titles={['可选用户', '已选用户']}
          targetKeys={checkedUserIds.map(String)}
          onChange={(keys) => setCheckedUserIds(keys.map(Number))}
          render={(item) => item.title}
          listStyle={{ width: 300, height: 360 }}
          showSearch
        />
      </Modal>
    </PageContainer>
  );
}
