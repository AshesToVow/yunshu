import {
  ApiOutlined,
  CheckCircleOutlined,
  CompressOutlined,
  DeleteOutlined,
  EditOutlined,
  ExpandOutlined,
  PlusOutlined,
  PlusSquareOutlined,
  StopOutlined,
} from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  PageContainer,
  ProFormDigit,
  ProFormList,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProTable,
} from '@ant-design/pro-components';
import { Button, Form, Popconfirm, Space, Tag, Typography, message } from 'antd';
import { useMemo, useRef, useState, type Key } from 'react';
import {
  batchUpdateMenuStatus,
  createMenu,
  deleteMenu,
  getAdminMenuTree,
  getMenuBindings,
  replaceMenuBindings,
  updateMenu,
  type MenuCreatePayload,
  type MenuItem,
  type MenuPermissionBindingItem,
  type MenuUpdatePayload,
} from '@/services/menus';
import { getAntdIconSelectOptions } from '@/utils/antd-icon-options';

function normalizeTreeOrder(items: MenuItem[]): MenuItem[] {
  const sorted = [...items].sort((a, b) => (a.sort !== b.sort ? a.sort - b.sort : a.id - b.id));
  return sorted.map((it) => ({
    ...it,
    children: it.children?.length ? normalizeTreeOrder(it.children) : undefined,
  }));
}

function collectAllIDs(items: MenuItem[]): number[] {
  const ids: number[] = [];
  for (const item of items) {
    ids.push(item.id);
    if (item.children?.length) ids.push(...collectAllIDs(item.children));
  }
  return ids;
}

export default function MenusPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [treeData, setTreeData] = useState<MenuItem[]>([]);
  const [expandedRowKeys, setExpandedRowKeys] = useState<readonly Key[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [editorOpen, setEditorOpen] = useState(false);
  const [current, setCurrent] = useState<MenuItem | null>(null);
  const [parentID, setParentID] = useState<number | undefined>();
  const [bindingOpen, setBindingOpen] = useState(false);
  const [bindingMenu, setBindingMenu] = useState<MenuItem | null>(null);
  const [bindingInitial, setBindingInitial] = useState<MenuPermissionBindingItem[]>([]);

  const iconOptions = useMemo(() => getAntdIconSelectOptions(), []);

  function openCreate(parentId?: number) {
    setCurrent(null);
    setParentID(parentId);
    setEditorOpen(true);
  }

  function openEdit(record: MenuItem) {
    setCurrent(record);
    setParentID(record.parent_id);
    setEditorOpen(true);
  }

  async function openBindings(record: MenuItem) {
    if (!record.path?.trim()) {
      message.info('目录节点无路由 path，无需配置入口权限');
      return;
    }
    setBindingMenu(record);
    const data = await getMenuBindings(record.id);
    setBindingInitial(
      data.custom?.length
        ? data.custom
        : data.default?.length
          ? data.default.map((b) => ({ resource: b.resource, action: b.action, mode: 'any' }))
          : [{ resource: '', action: 'GET', mode: 'any' }],
    );
    setBindingOpen(true);
  }

  async function handleBatchStatus(status: 0 | 1) {
    if (!selectedRowKeys.length) {
      message.warning('请先勾选菜单');
      return;
    }
    await batchUpdateMenuStatus({ ids: selectedRowKeys, status });
    message.success(status === 1 ? '批量启用成功' : '批量停用成功');
    setSelectedRowKeys([]);
    actionRef.current?.reload();
  }

  const columns: ProColumns<MenuItem>[] = [
    {
      title: '菜单名称',
      dataIndex: 'name',
      render: (_, row) => (
        <Space>
          {row.icon ? <Typography.Text type="secondary">{row.icon}</Typography.Text> : null}
          <span>{row.name}</span>
          {row.hidden ? <Tag>隐藏</Tag> : null}
        </Space>
      ),
    },
    { title: '路由路径', dataIndex: 'path', render: (_, row) => row.path || '—' },
    { title: '组件路径', dataIndex: 'component', ellipsis: true, render: (_, row) => row.component || '—' },
    { title: '重定向', dataIndex: 'redirect', ellipsis: true, render: (_, row) => row.redirect || '—' },
    { title: '排序', dataIndex: 'sort', width: 72 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 88,
      render: (_, row) => (row.status === 1 ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 280,
      render: (_, row) => [
        row.path ? (
          <Button key="bind" type="link" size="small" icon={<ApiOutlined />} onClick={() => void openBindings(row)}>
            入口权限
          </Button>
        ) : null,
        <Button key="child" type="link" size="small" icon={<PlusSquareOutlined />} onClick={() => openCreate(row.id)}>
          子菜单
        </Button>,
        <Button key="edit" type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(row)}>
          编辑
        </Button>,
        <Popconfirm
          key="del"
          title="确认删除该菜单吗？子菜单需先删除。"
          onConfirm={async () => {
            await deleteMenu(row.id);
            message.success('菜单已删除');
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
    <PageContainer title="菜单管理" subTitle="侧栏菜单树、组件路径与入口权限绑定">
      <ProTable<MenuItem>
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        search={false}
        pagination={false}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys as number[]),
        }}
        expandable={{
          expandedRowKeys: [...expandedRowKeys],
          onExpandedRowsChange: (keys) => setExpandedRowKeys(keys),
        }}
        request={async () => {
          const data = normalizeTreeOrder(await getAdminMenuTree());
          setTreeData(data);
          setExpandedRowKeys(collectAllIDs(data));
          return { data, success: true };
        }}
        toolBarRender={() => [
          <Button key="expand" icon={<ExpandOutlined />} onClick={() => setExpandedRowKeys(collectAllIDs(treeData))}>
            展开全部
          </Button>,
          <Button key="collapse" icon={<CompressOutlined />} onClick={() => setExpandedRowKeys([])}>
            折叠全部
          </Button>,
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => openCreate()}>
            创建菜单
          </Button>,
          <Button key="enable" icon={<CheckCircleOutlined />} onClick={() => void handleBatchStatus(1)}>
            批量启用
          </Button>,
          <Button key="disable" icon={<StopOutlined />} onClick={() => void handleBatchStatus(0)}>
            批量停用
          </Button>,
        ]}
      />

      <ModalForm<MenuCreatePayload>
        title={current ? `编辑菜单 #${current.id}` : parentID ? '新增子菜单' : '新增根菜单'}
        open={editorOpen}
        modalProps={{ destroyOnClose: true, width: 640, onCancel: () => setEditorOpen(false) }}
        initialValues={
          current
            ? {
                name: current.name,
                path: current.path,
                icon: current.icon,
                sort: current.sort,
                hidden: current.hidden,
                component: current.component,
                redirect: current.redirect,
                status: current.status,
              }
            : { status: 1, sort: 0, hidden: false }
        }
        onValuesChange={(changed) => {
          if (changed.hidden === true) {
            // ModalForm 内部 form 由 Pro 托管；隐藏时同步停用在 onFinish 前由用户手动选亦可
          }
        }}
        onFinish={async (values) => {
          const payload = { ...values, status: values.hidden ? 0 : values.status };
          if (current) {
            await updateMenu(current.id, payload as MenuUpdatePayload);
            message.success('菜单已更新');
          } else {
            await createMenu({ ...payload, parent_id: parentID });
            message.success('菜单已创建');
          }
          setEditorOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText name="name" label="菜单名称" rules={[{ required: true }]} placeholder="例如：系统管理" />
        <ProFormText name="path" label="路由路径" placeholder="/system" />
        <ProFormText
          name="component"
          label="组件路径"
          placeholder="例如：foo-bar-page"
          extra="与 src/pages 下 *-page.tsx 文件名一致"
        />
        <ProFormSelect
          name="icon"
          label="图标"
          showSearch
          allowClear
          options={iconOptions}
          fieldProps={{ optionFilterProp: 'value', virtual: true, listHeight: 280 }}
        />
        <ProFormDigit name="sort" label="排序" min={0} fieldProps={{ precision: 0 }} />
        <ProFormText name="redirect" label="重定向" placeholder="/redirect/path" />
        <ProFormSelect
          name="status"
          label="状态"
          options={[
            { label: '启用', value: 1 },
            { label: '停用', value: 0 },
          ]}
          rules={[{ required: true }]}
        />
        <ProFormSwitch name="hidden" label="是否隐藏" />
      </ModalForm>

      <ModalForm<{ bindings: MenuPermissionBindingItem[] }>
        title={bindingMenu ? `入口权限 · ${bindingMenu.name}` : '入口权限'}
        open={bindingOpen}
        modalProps={{ destroyOnClose: true, width: 680, onCancel: () => setBindingOpen(false) }}
        initialValues={{ bindings: bindingInitial }}
        onFinish={async (values) => {
          if (!bindingMenu) return false;
          const bindings = (values.bindings ?? []).filter((b) => b.resource?.trim() && b.action?.trim());
          await replaceMenuBindings(bindingMenu.id, bindings);
          message.success('入口权限绑定已保存');
          setBindingOpen(false);
          return true;
        }}
      >
        <Typography.Paragraph type="secondary">
          配置侧栏菜单可见性所需的 Casbin API（通常为 GET）。留空则使用内置 catalog 默认映射。
        </Typography.Paragraph>
        <ProFormList
          name="bindings"
          creatorButtonProps={{ creatorButtonText: '添加绑定' }}
          copyIconProps={false}
          itemRender={({ listDom, action }) => (
            <div style={{ display: 'flex', gap: 8, alignItems: 'flex-start', marginBottom: 8 }}>
              <div style={{ flex: 1 }}>{listDom}</div>
              {action}
            </div>
          )}
        >
          <Form.Item noStyle>
            <Space align="start" wrap>
              <ProFormText
                name="resource"
                width="md"
                placeholder="/api/v1/..."
                rules={[{ required: true, message: 'resource' }]}
              />
              <ProFormSelect
                name="action"
                width="xs"
                options={['GET', 'POST', 'PUT', 'DELETE'].map((a) => ({ value: a, label: a }))}
                rules={[{ required: true }]}
              />
            </Space>
          </Form.Item>
        </ProFormList>
      </ModalForm>
    </PageContainer>
  );
}
