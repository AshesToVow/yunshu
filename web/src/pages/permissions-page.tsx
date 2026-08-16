import { PlusOutlined, ReloadOutlined, SafetyCertificateOutlined, EyeOutlined, DeleteOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Drawer, Form, Input, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Tooltip, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { StatusTag } from "../components/status-tag";
import { createPermission, deletePermission, getPermissions, getPermission, updatePermission, batchSetPermissionK8sScope, listAllPermissions } from "../services/permissions";
import { getRoleOptions } from "../services/roles";
import { grantPolicy } from "../services/policies";
import { API_CATALOG_GROUPS, type ApiCatalogRow } from "../constants/api-catalog";
import type { PermissionItem, PermissionPayload, PermissionQuery, RoleItem } from "../types/api";
import { extractApiErrorMessage } from "../services/http";
import { formatDateTime } from "../utils/format";

const defaultQuery: PermissionQuery = {
  keyword: "",
  page: 1,
  page_size: 10,
  k8s_scope: "",
  k8s_related: "",
};

const HTTP_METHOD_OPTIONS = ["GET", "POST", "PUT", "DELETE", "PATCH"].map((m) => ({ label: m, value: m }));

/** 平台侧 K8s 配置 API：只走 Casbin，不参与 K8sScopeAuthorize 目录。 */
function isK8sScopeNotApplicable(resource: string) {
  const p = resource.trim().toLowerCase();
  return (
    p.startsWith("/api/v1/k8s-policies") ||
    p.startsWith("/api/v1/k8s-namespace-deny-rules") ||
    p.startsWith("/api/v1/k8s-namespace-allow-rules")
  );
}

function k8sScopeSwitchTooltip(resource: string) {
  if (isK8sScopeNotApplicable(resource)) {
    return "此为 K8s 平台配置接口（档位/命名空间规则），路由未挂 K8sScopeAuthorize，无需纳入三元校验目录；权限由「授权管理」控制即可。";
  }
  return "打开后：该接口在请求带 cluster_id 时将进入 K8s 范围校验中间件。集群侧能力由「K8s 集群访问档位」配置，API 能否调用仍由「授权管理」决定。";
}

/** 无 Casbin authorize 的路由，不应写入 permissions 表（与 cmd/seed_permissions_cleanup.go 对齐）。 */
const PERMISSION_SYNC_SKIP = new Set([
  "GET /api/v1/health",
  "POST /api/v1/auth/verification-code",
  "POST /api/v1/auth/login-code",
  "POST /api/v1/auth/password-login-code",
  "POST /api/v1/auth/login",
  "POST /api/v1/auth/email-login",
  "POST /api/v1/auth/register",
  "POST /api/v1/auth/logout",
  "POST /api/v1/auth/ws-ticket",
  "GET /api/v1/auth/me",
  "PUT /api/v1/auth/me",
  "PUT /api/v1/auth/password",
  "POST /api/v1/alerts/ingress/k8s-events",
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

export function PermissionsPage() {
  const [list, setList] = useState<PermissionItem[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState<PermissionQuery>(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<PermissionPayload>();
  const [assignOpen, setAssignOpen] = useState(false);
  const [assignTarget, setAssignTarget] = useState<PermissionItem | null>(null);
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [checkedRoleIds, setCheckedRoleIds] = useState<number[]>([]);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailRecord, setDetailRecord] = useState<PermissionItem | null>(null);
  const [detailSubmitting, setDetailSubmitting] = useState(false);
  const [detailForm] = Form.useForm<PermissionPayload>();
  const [syncingCatalog, setSyncingCatalog] = useState(false);
  const [batchK8sScopeLoading, setBatchK8sScopeLoading] = useState(false);

  useEffect(() => {
    void loadPermissions(query);
  }, [query]);

  useEffect(() => {
    void loadRoles();
  }, []);

  async function loadPermissions(nextQuery = query) {
    setLoading(true);
    try {
      const result = await getPermissions(nextQuery);
      setList(result.list);
      setTotal(result.total);
    } finally {
      setLoading(false);
    }
  }

  async function loadRoles() {
    const result = await getRoleOptions();
    setRoles(result.list);
  }

  function openCreate() {
    form.resetFields();
    form.setFieldsValue({ action: "GET", k8s_scope_enabled: false });
    setOpen(true);
  }

  async function handleSubmit() {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      await createPermission(values);
      message.success("接口能力创建成功");
      setOpen(false);
      form.resetFields();
      void loadPermissions();
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(record: PermissionItem) {
    await deletePermission(record.id);
    message.success(`已删除能力项 ${record.name}`);
    void loadPermissions();
  }

  async function handleToggleK8sScope(record: PermissionItem, enabled: boolean) {
    await updatePermission(record.id, { k8s_scope_enabled: enabled });
    message.success(enabled ? "已纳入 K8s 范围校验目录" : "已取消 K8s 范围校验目录");
    setList((prev) => prev.map((item) => (item.id === record.id ? { ...item, k8s_scope_enabled: enabled } : item)));
    if (detailRecord?.id === record.id) {
      setDetailRecord((prev) => (prev ? { ...prev, k8s_scope_enabled: enabled } : prev));
    }
  }

  async function openDetail(record: PermissionItem) {
    const detail = await getPermission(record.id);
    setDetailRecord(detail);
    detailForm.setFieldsValue({
      name: detail.name,
      resource: detail.resource,
      action: detail.action,
      description: detail.description,
      k8s_scope_enabled: detail.k8s_scope_enabled,
    });
    setDetailOpen(true);
  }

  async function submitDetailEdit() {
    if (!detailRecord) return;
    const values = await detailForm.validateFields();
    setDetailSubmitting(true);
    try {
      await updatePermission(detailRecord.id, values);
      message.success("权限详情已更新");
      setDetailOpen(false);
      setDetailRecord(null);
      await loadPermissions();
    } finally {
      setDetailSubmitting(false);
    }
  }

  function openAssignRoles(record: PermissionItem) {
    setAssignTarget(record);
    setCheckedRoleIds([]);
    setAssignOpen(true);
  }

  async function submitAssignRoles() {
    if (!assignTarget) return;
    setSubmitting(true);
    try {
      const promises = checkedRoleIds.map((roleId) =>
        grantPolicy({ role_id: roleId, permission_id: assignTarget.id })
      );
      await Promise.all(promises);
      message.success("角色权限已更新");
      setAssignOpen(false);
    } finally {
      setSubmitting(false);
    }
  }

  async function handleSyncCatalog() {
    setSyncingCatalog(true);
    try {
      const existing = new Set<string>();
      const all = await listAllPermissions();
      for (const it of all) {
        existing.add(`${it.action.toUpperCase()} ${it.resource}`);
      }
      const missing: { name: string; resource: string; action: string; description: string }[] = [];
      for (const group of API_CATALOG_GROUPS) {
        for (const route of group.routes) {
          if (!shouldSyncCatalogRoute(route)) continue;
          const action = route.method.toUpperCase();
          const resource = route.path.trim();
          const key = `${action} ${resource}`;
          if (existing.has(key)) continue;
          missing.push({
            name: truncateText(route.summary, 64),
            resource,
            action,
            description: truncateText(`${group.title} · ${route.ui}`, 255),
          });
        }
      }
      if (missing.length === 0) {
        message.info("接口能力记录已是最新，无需补全");
        return;
      }
      let created = 0;
      const failed: string[] = [];
      for (const it of missing) {
        try {
          await createPermission(
            {
              name: it.name,
              resource: it.resource,
              action: it.action,
              description: it.description,
              k8s_scope_enabled: false,
            },
            { silentErrorToast: true },
          );
          created += 1;
        } catch (e) {
          failed.push(`${it.action} ${it.resource}: ${extractApiErrorMessage(e, "创建失败")}`);
        }
      }
      if (created > 0) {
        message.success(`已补全 ${created} 条接口能力记录`);
        await loadPermissions();
      }
      if (failed.length > 0) {
        message.warning(`有 ${failed.length} 条补全失败：${failed[0]}${failed.length > 1 ? " 等" : ""}`);
      }
      if (created === 0 && failed.length > 0) {
        message.error(`补全失败：${failed[0]}`);
      }
    } catch (e) {
      message.error(extractApiErrorMessage(e, "补全接口失败"));
    } finally {
      setSyncingCatalog(false);
    }
  }

  async function handleBatchK8sScope(enabled: boolean) {
    setBatchK8sScopeLoading(true);
    try {
      const result = await batchSetPermissionK8sScope({
        enabled,
        k8s_related: "on",
        keyword: query.keyword?.trim() || undefined,
      });
      message.success(
        enabled
          ? `已为 ${result.affected} 条 K8s 集群资源接口纳入范围校验`
          : `已关闭 ${result.affected} 条 K8s 集群资源接口的范围校验`
      );
      await loadPermissions();
    } finally {
      setBatchK8sScopeLoading(false);
    }
  }

  const totalCount = Number(total);

  return (
    <div className="permissions-admin-page">
      <Card className="table-card">
        <div className="toolbar">
          <Space wrap size="middle">
            <Input.Search
              allowClear
              placeholder="搜索能力名称或资源路径"
              style={{ width: 280 }}
              onSearch={(keyword) => setQuery((prev) => ({ ...prev, keyword, page: 1 }))}
            />
            <Select
              style={{ width: 160 }}
              placeholder="K8s 范围校验"
              options={[
                { label: "全部", value: "" },
                { label: "已纳入", value: "on" },
                { label: "未纳入", value: "off" },
              ]}
              value={query.k8s_scope}
              onChange={(v) =>
                setQuery((prev) => ({
                  ...prev,
                  k8s_scope: (v ?? "") as PermissionQuery["k8s_scope"],
                  page: 1,
                }))
              }
            />
            <Select
              style={{ width: 200 }}
              placeholder="集群资源接口"
              options={[
                { label: "全部接口", value: "" },
                { label: "仅 K8s 集群资源", value: "on" },
              ]}
              value={query.k8s_related}
              onChange={(v) =>
                setQuery((prev) => ({
                  ...prev,
                  k8s_related: (v ?? "") as PermissionQuery["k8s_related"],
                  page: 1,
                }))
              }
            />
          </Space>
          <div className="toolbar__actions">
            <Popconfirm
              title="确认一键纳入？"
              description={
                <span>
                  仅作用于已挂载 <Typography.Text code>K8sScopeAuthorize</Typography.Text> 的集群资源接口（pods、deployments、clusters 等，与「仅 K8s 集群资源」筛选一致）。
                  <br />
                  <Typography.Text type="secondary">
                    不包含：k8s-policies（档位配置）、k8s-namespace-*（命名空间黑/白名单）等平台管理接口——它们只走 Casbin，打开开关无实际效果。
                  </Typography.Text>
                </span>
              }
              onConfirm={() => void handleBatchK8sScope(true)}
              okText="确认纳入"
              cancelText="取消"
            >
              <Button loading={batchK8sScopeLoading}>一键纳入 K8s 校验</Button>
            </Popconfirm>
            <Popconfirm
              title="确认一键关闭？"
              description="仅关闭「仅 K8s 集群资源」筛选范围内的接口开关；k8s-policies / k8s-namespace-* 等不在范围内。"
              onConfirm={() => void handleBatchK8sScope(false)}
              okText="确认关闭"
              cancelText="取消"
            >
              <Button loading={batchK8sScopeLoading}>一键关闭 K8s 校验</Button>
            </Popconfirm>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建能力项
            </Button>
            <Button onClick={() => void handleSyncCatalog()} loading={syncingCatalog}>
              一键补全接口
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => void loadPermissions()}>
              刷新
            </Button>
          </div>
        </div>

        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message="接口目录与前端入口"
          description={
            <span>
              「一键补全接口」按 <Typography.Text code>constants/api-catalog.ts</Typography.Text> 中「告警中心」等分组补全缺失的权限记录（能力名称取各行的{" "}
              <Typography.Text code>summary</Typography.Text>，须与 <Typography.Text code>cmd/seed.go</Typography.Text> 中 Casbin 的{" "}
              <Typography.Text code>Name</Typography.Text> 一致）。数据源、静默、监控规则、处理人、值班、PromQL 与「策略与联调」（Webhook、策略、历史、模板）均在{" "}
              <Link to="/alert-monitor-platform/datasources">告警监控平台</Link>
              （<Link to="/alert-monitor-platform/policies">告警路由</Link>）。
              <br />
              <Typography.Text type="secondary">
                「K8s 范围校验」开关：标记该接口是否进入 <Typography.Text code>K8sScopeAuthorize</Typography.Text> 三元中间件目录（见权限设计文档 §0）；不等于角色授权。
                「一键纳入/关闭」与筛选「仅 K8s 集群资源」范围一致（pods、deployments、clusters…共约 150 条）。
                <strong>不在范围内</strong>：<Typography.Text code>/api/v1/k8s-policies/*</Typography.Text>（档位/矩阵配置）、
                <Typography.Text code>/api/v1/k8s-namespace-deny-rules</Typography.Text>、
                <Typography.Text code>/api/v1/k8s-namespace-allow-rules</Typography.Text>——仅 Casbin 鉴权，请求通常不带 cluster_id，保持「未纳入」即可。
              </Typography.Text>
            </span>
          }
        />

        <Table
          rowKey="id"
          loading={loading}
          dataSource={list}
          pagination={{
            current: query.page,
            pageSize: query.page_size,
            total: Number.isFinite(totalCount) ? totalCount : 0,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showQuickJumper: true,
            showTotal: (t, range) => `${range[0]}-${range[1]} / 共 ${t} 条`,
            onChange: (page, pageSize) => {
              setQuery((prev) => ({
                ...prev,
                page,
                page_size: pageSize ?? prev.page_size,
              }));
            },
            onShowSizeChange: (_page, size) => {
              setQuery((prev) => ({ ...prev, page: 1, page_size: size }));
            },
          }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "能力名称", dataIndex: "name" },
            { title: "资源路径", dataIndex: "resource", render: (value: string) => <Tag>{value}</Tag> },
            { title: "动作", dataIndex: "action", render: (value: string) => <Tag color="processing">{value}</Tag> },
            {
              title: "K8s 范围校验",
              dataIndex: "k8s_scope_enabled",
              width: 120,
              render: (v?: boolean, record?: PermissionItem) => {
                const na = record ? isK8sScopeNotApplicable(record.resource) : false;
                if (v) return <Tag color="purple">已纳入目录</Tag>;
                if (na) return <Tag>不适用</Tag>;
                return <Tag>未纳入</Tag>;
              },
            },
            { title: "说明", dataIndex: "description", render: (value?: string) => value || "-" },
            {
              title: "操作",
              key: "action",
              render: (_: unknown, record?: PermissionItem) =>
                record ? (
                <Space>
                  <Button type="link" icon={<EyeOutlined />} onClick={() => openDetail(record)}>
                    详情
                  </Button>
                  <Button type="link" icon={<SafetyCertificateOutlined />} onClick={() => openAssignRoles(record)}>
                    分配角色
                  </Button>
                  <Tooltip title={k8sScopeSwitchTooltip(record.resource)}>
                    <Switch
                      size="small"
                      disabled={isK8sScopeNotApplicable(record.resource)}
                      checked={Boolean(record.k8s_scope_enabled)}
                      checkedChildren="开"
                      unCheckedChildren="关"
                      onChange={(checked) => {
                        void handleToggleK8sScope(record, checked);
                      }}
                    />
                  </Tooltip>
                  <Popconfirm title="确认删除该能力项吗？" onConfirm={() => void handleDelete(record)}>
                    <Button type="link" danger icon={<DeleteOutlined />}>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              ) : null,
            },
          ]}
        />
      </Card>

      <Modal
        title="新建接口能力"
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void handleSubmit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ action: "GET", k8s_scope_enabled: false }}>
          <Form.Item label="能力名称" name="name" rules={[{ required: true, message: "请输入能力名称" }]}>
            <Input placeholder="例如：查询主机列表" />
          </Form.Item>
          <Form.Item label="资源路径" name="resource" rules={[{ required: true, message: "请输入资源路径" }]}>
            <Input placeholder="须与后端一致，例如 /api/v1/users 或 /api/v1/users/:id；撤销策略为 DELETE /api/v1/policies（勿写 :id）" />
          </Form.Item>
          <Form.Item label="HTTP 动作" name="action" rules={[{ required: true, message: "请选择动作" }]}>
            <Select options={HTTP_METHOD_OPTIONS} />
          </Form.Item>
          <Form.Item label="说明" name="description">
            <Input.TextArea rows={3} placeholder="请输入能力说明" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={assignTarget ? `为权限 ${assignTarget.name} 分配角色` : "分配角色"}
        open={assignOpen}
        onCancel={() => {
          setAssignOpen(false);
          setCheckedRoleIds([]);
        }}
        onOk={() => void submitAssignRoles()}
        confirmLoading={submitting}
        destroyOnClose
        width={600}
      >
        <Space direction="vertical" size={12} style={{ width: "100%" }}>
          <Typography.Text className="inline-muted">
            勾选需要分配该权限的角色，已选 {checkedRoleIds.length} 个角色。
          </Typography.Text>
          <Table
            rowKey="id"
            dataSource={roles}
            pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
            rowSelection={{
              selectedRowKeys: checkedRoleIds,
              onChange: (keys) => setCheckedRoleIds(keys as number[]),
            }}
            columns={[
              { title: "角色名称", dataIndex: "name" },
              { title: "角色编码", dataIndex: "code", render: (code) => <Tag color="blue">{code}</Tag> },
              { title: "状态", dataIndex: "status", render: (status) => <StatusTag status={status} /> },
            ]}
            size="small"
          />
        </Space>
      </Modal>

      <Drawer
        title="权限详情"
        open={detailOpen}
        onClose={() => {
          setDetailOpen(false);
          setDetailRecord(null);
        }}
        width={680}
        className="detail-edit-drawer"
        extra={
          <Button type="primary" loading={detailSubmitting} onClick={() => void submitDetailEdit()}>
            保存修改
          </Button>
        }
      >
        {detailRecord && (
          <Form form={detailForm} layout="vertical" className="detail-edit-form">
            <Form.Item label="ID">
              <Input value={String(detailRecord.id)} readOnly />
            </Form.Item>
            <Form.Item label="能力名称" name="name" rules={[{ required: true, message: "请输入能力名称" }]}>
              <Input />
            </Form.Item>
            <Form.Item label="资源路径" name="resource" rules={[{ required: true, message: "请输入资源路径" }]}>
              <Input />
            </Form.Item>
            <Form.Item label="HTTP 动作" name="action" rules={[{ required: true, message: "请选择动作" }]}>
              <Select options={HTTP_METHOD_OPTIONS} />
            </Form.Item>
            <Form.Item label="说明" name="description">
              <Input.TextArea rows={4} />
            </Form.Item>
            <Form.Item
              label="K8s 范围校验"
              name="k8s_scope_enabled"
              valuePropName="checked"
              extra="打开后该接口纳入 K8s 三元中间件目录（permissions.k8s_scope_enabled）；与「授权管理」中的 API 勾选相互独立。"
            >
              <Switch checkedChildren="已纳入" unCheckedChildren="未纳入" />
            </Form.Item>
            <Form.Item label="创建时间">
              <Input value={formatDateTime(detailRecord.created_at)} readOnly />
            </Form.Item>
            <Form.Item label="更新时间">
              <Input value={formatDateTime(detailRecord.updated_at)} readOnly />
            </Form.Item>
          </Form>
        )}
      </Drawer>
    </div>
  );
}