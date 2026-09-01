import { DeleteOutlined, EditOutlined, PlusOutlined, PlusSquareOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  PageContainer,
  ProFormDigit,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
  ProFormTreeSelect,
  ProTable,
} from '@ant-design/pro-components';
import { Button, Popconfirm, Space, Tag, Typography, message } from 'antd';
import { useEffect, useMemo, useRef, useState, type Key } from 'react';
import {
  createDepartment,
  deleteDepartment,
  getDepartmentTree,
  updateDepartment,
} from '@/services/departments';
import { getUsers } from '@/services/users';
import type { DepartmentItem, UserItem } from '@/types/api';

type DeptForm = {
  parent_id?: number;
  name: string;
  code: string;
  sort?: number;
  status: number;
  leader_id?: number;
  phone?: string;
  email?: string;
  remark?: string;
};

function collectIds(nodes: DepartmentItem[]): number[] {
  const ids: number[] = [];
  const walk = (list: DepartmentItem[]) => {
    for (const n of list) {
      ids.push(n.id);
      if (n.children?.length) walk(n.children);
    }
  };
  walk(nodes);
  return ids;
}

function toTreeSelect(nodes: DepartmentItem[]): { title: string; value: number; children?: ReturnType<typeof toTreeSelect> }[] {
  return nodes.map((n) => ({
    title: `${n.name} (${n.code})`,
    value: n.id,
    children: n.children?.length ? toTreeSelect(n.children) : undefined,
  }));
}

export default function DepartmentsPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [tree, setTree] = useState<DepartmentItem[]>([]);
  const [users, setUsers] = useState<UserItem[]>([]);
  const [expandedRowKeys, setExpandedRowKeys] = useState<readonly Key[]>([]);
  const [editorOpen, setEditorOpen] = useState(false);
  const [current, setCurrent] = useState<DepartmentItem | null>(null);
  const [defaultParentId, setDefaultParentId] = useState<number | undefined>();

  const treeSelectData = useMemo(() => toTreeSelect(tree), [tree]);
  const leaderOptions = useMemo(
    () => users.map((u) => ({ value: u.id, label: `${u.nickname} (${u.username})` })),
    [users],
  );

  useEffect(() => {
    void getUsers({ page: 1, page_size: 500 }).then((res) => setUsers(res.list ?? []));
  }, []);

  function openCreate(parentId?: number) {
    setCurrent(null);
    setDefaultParentId(parentId);
    setEditorOpen(true);
  }

  function openEdit(record: DepartmentItem) {
    setCurrent(record);
    setDefaultParentId(undefined);
    setEditorOpen(true);
  }

  const columns: ProColumns<DepartmentItem>[] = [
    {
      title: '部门名称',
      dataIndex: 'name',
      render: (_, row) => (
        <Space>
          <Typography.Text strong>{row.name}</Typography.Text>
          <Typography.Text type="secondary">({row.code})</Typography.Text>
        </Space>
      ),
    },
    { title: '负责人', dataIndex: 'leader_name', width: 120, render: (v) => v || '—' },
    { title: '成员数', dataIndex: 'user_count', width: 88, align: 'center', render: (v) => v ?? 0 },
    { title: '层级', dataIndex: 'level', width: 72, align: 'center' },
    { title: '排序', dataIndex: 'sort', width: 72, align: 'center' },
    {
      title: '状态',
      dataIndex: 'status',
      width: 88,
      render: (_, row) =>
        row.status === 1 ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>,
    },
    {
      title: '操作',
      valueType: 'option',
      width: 220,
      render: (_, row) => [
        <Button key="child" type="link" size="small" icon={<PlusSquareOutlined />} onClick={() => openCreate(row.id)}>
          子部门
        </Button>,
        <Button key="edit" type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(row)}>
          编辑
        </Button>,
        <Popconfirm key="del" title="确认删除该部门吗？" onConfirm={async () => {
          await deleteDepartment(row.id);
          message.success('部门已删除');
          actionRef.current?.reload();
        }}>
          <Button type="link" size="small" danger icon={<DeleteOutlined />}>
            删除
          </Button>
        </Popconfirm>,
      ],
    },
  ];

  return (
    <PageContainer title="部门管理" subTitle="组织架构支持多级部门与负责人绑定">
      <ProTable<DepartmentItem>
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        pagination={false}
        expandable={{
          expandedRowKeys: [...expandedRowKeys],
          onExpandedRowsChange: (keys) => setExpandedRowKeys(keys),
        }}
        request={async () => {
          const data = await getDepartmentTree();
          setTree(data);
          setExpandedRowKeys(collectIds(data));
          return { data, success: true };
        }}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => openCreate()}>
            新建部门
          </Button>,
          <Button key="refresh" onClick={() => actionRef.current?.reload()}>
            刷新
          </Button>,
        ]}
      />

      <ModalForm<DeptForm>
        title={current ? `编辑部门 #${current.id}` : '新建部门'}
        open={editorOpen}
        modalProps={{ destroyOnClose: true, onCancel: () => setEditorOpen(false) }}
        initialValues={
          current
            ? {
                parent_id: current.parent_id,
                name: current.name,
                code: current.code,
                sort: current.sort,
                status: current.status,
                leader_id: current.leader_id,
                phone: current.phone,
                email: current.email,
                remark: current.remark,
              }
            : { parent_id: defaultParentId, status: 1, sort: 0 }
        }
        onFinish={async (values) => {
          if (current) {
            await updateDepartment(current.id, values);
            message.success('部门信息已更新');
          } else {
            await createDepartment(values);
            message.success('部门创建成功');
          }
          setEditorOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormTreeSelect
          name="parent_id"
          label="上级部门"
          placeholder="不选则为根部门"
          allowClear
          fieldProps={{ treeDefaultExpandAll: true, treeData: treeSelectData }}
        />
        <ProFormText name="name" label="部门名称" rules={[{ required: true, message: '请输入部门名称' }]} />
        <ProFormText name="code" label="部门编码" rules={[{ required: true, message: '请输入部门编码' }]} />
        <ProFormDigit name="sort" label="排序" min={0} fieldProps={{ precision: 0 }} />
        <ProFormSelect
          name="status"
          label="状态"
          options={[
            { label: '启用', value: 1 },
            { label: '停用', value: 0 },
          ]}
          rules={[{ required: true }]}
        />
        <ProFormSelect name="leader_id" label="负责人" options={leaderOptions} showSearch allowClear />
        <ProFormText name="phone" label="联系电话" />
        <ProFormText name="email" label="邮箱" />
        <ProFormTextArea name="remark" label="备注" />
      </ModalForm>
    </PageContainer>
  );
}
