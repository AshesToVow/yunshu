import { DeleteOutlined, EditOutlined, PlusOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import {
  ModalForm,
  PageContainer,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import { Link } from '@umijs/max';
import { Alert, Button, Modal, Popconfirm, Switch, Table, Tag, Tooltip, Typography, message } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { API_CATALOG_GROUPS, type ApiCatalogRow } from '@/constants/api-catalog';
import {
  batchSetPermissionK8sScope,
  createPermission,
  deletePermission,
  getPermission,
  getPermissions,
  listAllPermissions,
  updatePermission,
} from '@/services/permissions';
import { grantPolicy } from '@/services/policies';
import { getRoleOptions } from '@/services/roles';
import { extractApiErrorMessage } from '@/services/http';
import type { PermissionItem, PermissionPayload, RoleItem } from '@/types/api';
import { formatDateTime } from '@/utils/format';

const HTTP_METHOD_OPTIONS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'].map((m) => ({ label: m, value: m }));

function isK8sScopeNotApplicable(resource: string) {
  const p = resource.trim().toLowerCase();
  return (
    p.startsWith('/api/v1/k8s-policies') ||
    p.startsWith('/api/v1/k8s-namespace-deny-rules') ||
    p.startsWith('/api/v1/k8s-namespace-allow-rules') ||
    p.startsWith('/api/v1/k8s/event-forward')
  );
}

function k8sScopeSwitchTooltip(resource: string) {
  if (isK8sScopeNotApplicable(resource)) {
    return '此为 K8s 平台配置接口，路由未挂 K8sScopeAuthorize，无需纳入三元校验目录。';
  }
  return '打开后：该接口在请求带 cluster_id 时将进入 K8s 范围校验中间件。';
}

const PERMISSION_SYNC_SKIP = new Set([
  'GET /api/v1/health',
  'POST /api/v1/auth/verification-code',
  'POST /api/v1/auth/login-code',
  'POST /api/v1/auth/password-login-code',
  'POST /api/v1/auth/login',
  'POST /api/v1/auth/email-login',
  'POST /api/v1/auth/register',
  'POST /api/v1/auth/logout',
  'POST /api/v1/auth/ws-ticket',
  'GET /api/v1/auth/me',
  'PUT /api/v1/auth/me',
  'PUT /api/v1/auth/password',
  'GET /api/v1/menus/tree',
  'POST /api/v1/alerts/webhook/alertmanager',
  'POST /api/v1/alerts/ingress/k8s-events',
  'POST /api/v1/loggie/heartbeat/report',
]);

function catalogRouteKey(route: ApiCatalogRow) {
  return `${route.method.toUpperCase()} ${route.path.trim()}`;
}

function shouldSyncCatalogRoute(route: ApiCatalogRow) {
  if (!route.auth) return false;
  return !PERMISSION_SYNC_SKIP.has(catalogRouteKey(route));
}

function truncateText(value: string, max: number) {
  const s = value.trim();
  return s.length <= max ? s : s.slice(0, max);
}

export default function PermissionsPage() {
  const actionRef = useRef<ActionType>(undefined);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<PermissionItem | null>(null);
  const [assignOpen, setAssignOpen] = useState(false);
  const [assignTarget, setAssignTarget] = useState<PermissionItem | null>(null);
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [checkedRoleIds, setCheckedRoleIds] = useState<number[]>([]);
  const [syncingCatalog, setSyncingCatalog] = useState(false);
  const [batchK8sScopeLoading, setBatchK8sScopeLoading] = useState(false);
  const [lastQuery, setLastQuery] = useState<{ keyword?: string }>({});

  useEffect(() => {
    void getRoleOptions().then((res) => setRoles(res.list ?? []));
  }, []);

  async function openEdit(record: PermissionItem) {
    const detail = await getPermission(record.id);
    setEditTarget(detail);
    setEditOpen(true);
  }

  async function handleToggleK8sScope(record: PermissionItem, enabled: boolean) {
    await updatePermission(record.id, { k8s_scope_enabled: enabled });
    message.success(enabled ? '已纳入 K8s 范围校验目录' : '已取消 K8s 范围校验目录');
    actionRef.current?.reload();
  }

  async function handleSyncCatalog() {
    setSyncingCatalog(true);
    try {
      const existing = new Set<string>();
      const all = await listAllPermissions();
      for (const it of all) existing.add(`${it.action.toUpperCase()} ${it.resource}`);
      const missing: PermissionPayload[] = [];
      for (const group of API_CATALOG_GROUPS) {
        for (const route of group.routes) {
          if (!shouldSyncCatalogRoute(route)) continue;
          const action = route.method.toUpperCase();
          const resource = route.path.trim();
          if (existing.has(`${action} ${resource}`)) continue;
          missing.push({
            name: truncateText(route.summary, 64),
            resource,
            action,
            description: truncateText(`${group.title} · ${route.ui}`, 255),
            k8s_scope_enabled: false,
          });
        }
      }
      if (!missing.length) {
        message.info('接口能力记录已是最新，无需补全');
        return;
      }
      let created = 0;
      const failed: string[] = [];
      for (const it of missing) {
        try {
          await createPermission(it, { silentErrorToast: true });
          created += 1;
        } catch (e) {
          failed.push(`${it.action} ${it.resource}: ${extractApiErrorMessage(e, '创建失败')}`);
        }
      }
      if (created > 0) {
        message.success(`已补全 ${created} 条接口能力记录`);
        actionRef.current?.reload();
      }
      if (failed.length) message.warning(`有 ${failed.length} 条补全失败：${failed[0]}`);
    } catch (e) {
      message.error(extractApiErrorMessage(e, '补全接口失败'));
    } finally {
      setSyncingCatalog(false);
    }
  }

  async function handleBatchK8sScope(enabled: boolean) {
    setBatchK8sScopeLoading(true);
    try {
      const result = await batchSetPermissionK8sScope({
        enabled,
        k8s_related: 'on',
        keyword: lastQuery.keyword?.trim() || undefined,
      });
      message.success(
        enabled
          ? `已为 ${result.affected} 条 K8s 集群资源接口纳入范围校验`
          : `已关闭 ${result.affected} 条接口的范围校验`,
      );
      actionRef.current?.reload();
    } finally {
      setBatchK8sScopeLoading(false);
    }
  }

  const columns: ProColumns<PermissionItem>[] = [
    { title: 'ID', dataIndex: 'id', width: 70, search: false },
    { title: '能力名称', dataIndex: 'name' },
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: { placeholder: '能力名称或资源路径' },
    },
    {
      title: '资源路径',
      dataIndex: 'resource',
      search: false,
      ellipsis: true,
      render: (_, row) => <Tag>{row.resource}</Tag>,
    },
    {
      title: '动作',
      dataIndex: 'action',
      width: 90,
      search: false,
      render: (_, row) => <Tag color="processing">{row.action}</Tag>,
    },
    {
      title: 'K8s 范围校验',
      dataIndex: 'k8s_scope',
      valueType: 'select',
      valueEnum: {
        '': { text: '全部' },
        on: { text: '已纳入' },
        off: { text: '未纳入' },
      },
      hideInTable: true,
    },
    {
      title: '集群资源接口',
      dataIndex: 'k8s_related',
      valueType: 'select',
      valueEnum: {
        '': { text: '全部接口' },
        on: { text: '仅 K8s 集群资源' },
      },
      hideInTable: true,
    },
    {
      title: 'K8s 范围',
      dataIndex: 'k8s_scope_enabled',
      width: 110,
      search: false,
      render: (_, row) => {
        if (row.k8s_scope_enabled) return <Tag color="purple">已纳入</Tag>;
        if (isK8sScopeNotApplicable(row.resource)) return <Tag>不适用</Tag>;
        return <Tag>未纳入</Tag>;
      },
    },
    {
      title: '说明',
      dataIndex: 'description',
      search: false,
      ellipsis: true,
      render: (_, row) => row.description || '—',
    },
    {
      title: '操作',
      valueType: 'option',
      width: 280,
      render: (_, row) => [
        <Button key="edit" type="link" size="small" icon={<EditOutlined />} onClick={() => void openEdit(row)}>
          编辑
        </Button>,
        <Button
          key="assign"
          type="link"
          size="small"
          icon={<SafetyCertificateOutlined />}
          onClick={() => {
            setAssignTarget(row);
            setCheckedRoleIds([]);
            setAssignOpen(true);
          }}
        >
          分配角色
        </Button>,
        <Tooltip key="scope" title={k8sScopeSwitchTooltip(row.resource)}>
          <Switch
            size="small"
            disabled={isK8sScopeNotApplicable(row.resource)}
            checked={Boolean(row.k8s_scope_enabled)}
            checkedChildren="开"
            unCheckedChildren="关"
            onChange={(checked) => void handleToggleK8sScope(row, checked)}
          />
        </Tooltip>,
        <Popconfirm
          key="del"
          title="确认删除该能力项吗？"
          onConfirm={async () => {
            await deletePermission(row.id);
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
    <PageContainer title="权限管理" subTitle="接口能力目录、K8s 范围校验与角色分配">
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="接口目录与前端入口"
        description={
          <span>
            「一键补全接口」按 <Typography.Text code>constants/api-catalog.ts</Typography.Text> 补全缺失权限记录。告警相关入口见{' '}
            <Link to="/alert-monitor-platform/datasources">告警监控平台</Link>。
          </span>
        }
      />
      <ProTable<PermissionItem>
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
        request={async (params) => {
          setLastQuery({ keyword: params.keyword as string | undefined });
          const res = await getPermissions({
            keyword: params.keyword as string | undefined,
            k8s_scope: (params.k8s_scope as '' | 'on' | 'off' | undefined) ?? '',
            k8s_related: (params.k8s_related as '' | 'on' | undefined) ?? '',
            page: params.current,
            page_size: params.pageSize,
          });
          return { data: res.list ?? [], total: res.total ?? 0, success: true };
        }}
        toolBarRender={() => [
          <Popconfirm
            key="batch-on"
            title="确认一键纳入 K8s 校验？"
            onConfirm={() => void handleBatchK8sScope(true)}
          >
            <Button loading={batchK8sScopeLoading}>一键纳入 K8s 校验</Button>
          </Popconfirm>,
          <Popconfirm
            key="batch-off"
            title="确认一键关闭 K8s 校验？"
            onConfirm={() => void handleBatchK8sScope(false)}
          >
            <Button loading={batchK8sScopeLoading}>一键关闭 K8s 校验</Button>
          </Popconfirm>,
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            新建能力项
          </Button>,
          <Button key="sync" loading={syncingCatalog} onClick={() => void handleSyncCatalog()}>
            一键补全接口
          </Button>,
        ]}
      />

      <ModalForm<PermissionPayload>
        title="新建接口能力"
        open={createOpen}
        modalProps={{ destroyOnClose: true, onCancel: () => setCreateOpen(false) }}
        initialValues={{ action: 'GET', k8s_scope_enabled: false }}
        onFinish={async (values) => {
          await createPermission(values);
          message.success('接口能力创建成功');
          setCreateOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText name="name" label="能力名称" rules={[{ required: true }]} />
        <ProFormText
          name="resource"
          label="资源路径"
          rules={[
            { required: true },
            {
              validator: async (_, v: string) => {
                if (String(v || '').includes('*')) throw new Error('资源路径不能包含通配符 *');
              },
            },
          ]}
          placeholder="/api/v1/users"
        />
        <ProFormSelect name="action" label="HTTP 动作" options={HTTP_METHOD_OPTIONS} rules={[{ required: true }]} />
        <ProFormTextArea name="description" label="说明" />
      </ModalForm>

      <ModalForm<PermissionPayload>
        title={editTarget ? `编辑权限 #${editTarget.id}` : '编辑权限'}
        open={editOpen}
        modalProps={{ destroyOnClose: true, width: 640, onCancel: () => setEditOpen(false) }}
        initialValues={
          editTarget
            ? {
                name: editTarget.name,
                resource: editTarget.resource,
                action: editTarget.action,
                description: editTarget.description,
                k8s_scope_enabled: editTarget.k8s_scope_enabled,
              }
            : undefined
        }
        onFinish={async (values) => {
          if (!editTarget) return false;
          await updatePermission(editTarget.id, values);
          message.success('权限详情已更新');
          setEditOpen(false);
          actionRef.current?.reload();
          return true;
        }}
      >
        <ProFormText name="name" label="能力名称" rules={[{ required: true }]} />
        <ProFormText
          name="resource"
          label="资源路径"
          rules={[
            { required: true },
            {
              validator: async (_, v: string) => {
                if (String(v || '').includes('*')) throw new Error('资源路径不能包含通配符 *');
              },
            },
          ]}
        />
        <ProFormSelect name="action" label="HTTP 动作" options={HTTP_METHOD_OPTIONS} rules={[{ required: true }]} />
        <ProFormTextArea name="description" label="说明" />
        <ProFormSwitch
          name="k8s_scope_enabled"
          label="K8s 范围校验"
          extra="打开后纳入 K8s 三元中间件目录；与授权管理中的 API 勾选相互独立。"
        />
        {editTarget ? (
          <Typography.Paragraph type="secondary">
            创建：{formatDateTime(editTarget.created_at)}　更新：{formatDateTime(editTarget.updated_at)}
          </Typography.Paragraph>
        ) : null}
      </ModalForm>

      <Modal
        title={assignTarget ? `为权限 ${assignTarget.name} 分配角色` : '分配角色'}
        open={assignOpen}
        width={640}
        destroyOnClose
        onCancel={() => setAssignOpen(false)}
        onOk={async () => {
          if (!assignTarget) return;
          await Promise.all(checkedRoleIds.map((roleId) => grantPolicy({ role_id: roleId, permission_id: assignTarget.id })));
          message.success('角色权限已更新');
          setAssignOpen(false);
        }}
      >
        <Typography.Paragraph type="secondary">已选 {checkedRoleIds.length} 个角色</Typography.Paragraph>
        <Table
          rowKey="id"
          size="small"
          dataSource={roles}
          pagination={{ pageSize: 10 }}
          rowSelection={{
            selectedRowKeys: checkedRoleIds,
            onChange: (keys) => setCheckedRoleIds(keys as number[]),
          }}
          columns={[
            { title: '角色名称', dataIndex: 'name' },
            { title: '角色编码', dataIndex: 'code', render: (v: string) => <Tag color="blue">{v}</Tag> },
            {
              title: '状态',
              dataIndex: 'status',
              render: (v: number) => (v === 1 ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>),
            },
          ]}
        />
      </Modal>
    </PageContainer>
  );
}
