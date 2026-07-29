import { LinkOutlined, ReloadOutlined, SaveOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Empty, Input, InputNumber, Select, Space, Table, Tabs, Tag, Tree, Typography, message } from "antd";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { useCallback, useEffect, useMemo, useState } from "react";
import { listAllPermissions } from "../services/permissions";
import {
  fixMenuEntryAPIs,
  fixDisabledPluginPolicies,
  getPermissionTree,
  getPolicies,
  getPolicyConflicts,
  getPolicyMenuLinks,
  grantPolicy,
  revokePolicy,
  simulatePolicy,
} from "../services/policies";
import { getRoleOptions } from "../services/roles";
import type { PermissionItem, PermissionTreeNode, PolicyConflictItem, PolicyItem, RoleItem } from "../types/api";
import {
  buildPermissionTreeData,
  buildUnifiedPermissionTreeData,
  collectGrantedPermissionIds,
  normalizeCheckedKeys,
} from "../utils/tree";
import { isAPIResourceAllowedByPlugins } from "../modules/plugin-path";
import { usePlugins } from "../contexts/plugin-context";

export function PoliciesPage() {
  const { isPluginEnabled } = usePlugins();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [list, setList] = useState<PolicyItem[]>([]);
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [permissions, setPermissions] = useState<PermissionItem[]>([]);
  const [allPermissions, setAllPermissions] = useState<PermissionItem[]>([]);
  const [menuLinks, setMenuLinks] = useState<Record<string, { path: string }[]>>({});
  const [conflicts, setConflicts] = useState<PolicyConflictItem[]>([]);
  const [unifiedTree, setUnifiedTree] = useState<PermissionTreeNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [selectedRoleId, setSelectedRoleId] = useState<number>();
  const [checkedPermissionIds, setCheckedPermissionIds] = useState<number[]>([]);
  const [roleKeyword, setRoleKeyword] = useState("");
  const [permissionKeyword, setPermissionKeyword] = useState("");
  const [roleStatus, setRoleStatus] = useState<number | undefined>();
  const [assignPager, setAssignPager] = useState({ current: 1, pageSize: 10 });
  const [simulateUserId, setSimulateUserId] = useState<number>();
  const [simulatePath, setSimulatePath] = useState("/api/v1/projects/1/dbmgmt/instances");
  const [simulateMethod, setSimulateMethod] = useState("GET");
  const [simulateResult, setSimulateResult] = useState<string>("");

  const isResourceAllowed = useCallback(
    (resource: string) => isAPIResourceAllowedByPlugins(resource, isPluginEnabled),
    [isPluginEnabled],
  );

  const permissionTreeData = useMemo(
    () =>
      buildPermissionTreeData(allPermissions.length ? allPermissions : permissions, {
        menuLinks,
        isPluginAllowed: isResourceAllowed,
      }),
    [allPermissions, permissions, menuLinks, isResourceAllowed],
  );
  const permissionIdSet = useMemo(() => new Set(permissions.map((permission) => permission.id)), [permissions]);
  const selectedRole = useMemo(
    () => roles.find((role) => role.id === selectedRoleId) ?? null,
    [roles, selectedRoleId],
  );
  const currentRolePolicies = useMemo(
    () => (selectedRoleId ? list.filter((policy) => policy.role_id === selectedRoleId) : []),
    [list, selectedRoleId],
  );
  const unifiedTreeData = useMemo(() => buildUnifiedPermissionTreeData(unifiedTree), [unifiedTree]);
  const filteredRoles = useMemo(() => {
    const key = roleKeyword.trim().toLowerCase();
    return roles.filter((role) => {
      const matchKeyword = !key || role.name.toLowerCase().includes(key) || role.code.toLowerCase().includes(key);
      const matchStatus = roleStatus === undefined || role.status === roleStatus;
      return matchKeyword && matchStatus;
    });
  }, [roles, roleKeyword, roleStatus]);
  const filteredPermissionTree = useMemo(() => {
    const key = permissionKeyword.trim().toLowerCase();
    if (!key) return permissionTreeData;
    const walk = (nodes: any[]): any[] => {
      const next: any[] = [];
      for (const node of nodes) {
        const titleText = String(node.title ?? "").toLowerCase();
        const children = Array.isArray(node.children) ? walk(node.children) : [];
        if (titleText.includes(key) || children.length > 0) {
          next.push({ ...node, children });
        }
      }
      return next;
    };
    return walk(permissionTreeData as any[]);
  }, [permissionTreeData, permissionKeyword]);

  const linkFromUser = searchParams.get("from") === "user";
  const linkUsername = searchParams.get("username") ?? "";
  const linkUserId = Number(searchParams.get("user_id"));
  const linkNext = searchParams.get("next");
  const linkRoleId = Number(searchParams.get("role_id"));

  useEffect(() => {
    const preferredRoleId = linkRoleId > 0 ? linkRoleId : undefined;
    void bootstrap(preferredRoleId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    setAssignPager((p) => ({ ...p, current: 1 }));
  }, [selectedRoleId]);

  useEffect(() => {
    if (!selectedRoleId) {
      setConflicts([]);
      setUnifiedTree([]);
      return;
    }
    void loadRoleExtras(selectedRoleId);
  }, [selectedRoleId]);

  async function loadRoleExtras(roleId: number) {
    try {
      const [conflictData, treeData] = await Promise.all([getPolicyConflicts(roleId), getPermissionTree(roleId)]);
      setConflicts(conflictData.items ?? []);
      setUnifiedTree(treeData.tree ?? []);
    } catch {
      setConflicts([]);
      setUnifiedTree([]);
    }
  }

  async function handleFixMenuEntryConflicts() {
    if (!selectedRoleId) return;
    if (!conflicts.some((c) => c.type === "menu_needs_entry_api")) {
      message.info("没有可自动修复的入口 API 冲突");
      return;
    }
    setSubmitting(true);
    try {
      const result = await fixMenuEntryAPIs(selectedRoleId);
      message.success(
        `已补齐入口 API：新建 ${result.created}，授权 ${result.granted}，跳过 ${result.skipped}（共 ${result.total}）`,
      );
      await Promise.all([bootstrap(selectedRoleId), loadRoleExtras(selectedRoleId)]);
    } catch (e: unknown) {
      message.error(String((e as Error)?.message ?? e));
    } finally {
      setSubmitting(false);
    }
  }

  async function handleFixDisabledPluginPolicies() {
    if (!selectedRoleId) return;
    const hasPluginConflicts = conflicts.some(
      (c) => c.type === "plugin_disabled_policy_active" || c.type === "api_granted_plugin_disabled",
    );
    if (!hasPluginConflicts) {
      message.info("没有可清理的禁用插件策略冲突");
      return;
    }
    setSubmitting(true);
    try {
      const result = await fixDisabledPluginPolicies(selectedRoleId);
      message.success(
        `已清理禁用插件策略：撤销 ${result.revoked}，跳过 ${result.skipped}（共扫描 ${result.total}）`,
      );
      await Promise.all([bootstrap(selectedRoleId), loadRoleExtras(selectedRoleId)]);
    } catch (e: unknown) {
      message.error(String((e as Error)?.message ?? e));
    } finally {
      setSubmitting(false);
    }
  }

  async function bootstrap(preferredRoleId?: number) {
    setLoading(true);
    try {
      const [policyList, roleData, permissionData, allPerms, linksData] = await Promise.all([
        getPolicies(),
        getRoleOptions(),
        listAllPermissions().then((all) => ({
          list: all.filter((p) => isResourceAllowed(p.resource)),
          total: all.length,
          page: 1,
          page_size: all.length,
        })),
        listAllPermissions(),
        getPolicyMenuLinks(),
      ]);

      const allowedResources = new Set(permissionData.list.map((p) => `${p.resource}::${p.action}`));
      setList(policyList.filter((p) => allowedResources.has(`${p.resource}::${p.action}`)));
      setRoles(roleData.list);
      setPermissions(permissionData.list);
      setAllPermissions(allPerms);
      setMenuLinks(linksData.links ?? {});

      const nextRoleId = preferredRoleId ?? selectRoleId(roleData.list, selectedRoleId);
      setSelectedRoleId(nextRoleId);
      setCheckedPermissionIds(nextRoleId ? getRolePermissionIds(policyList, nextRoleId).filter((id) => id > 0) : []);
    } finally {
      setLoading(false);
    }
  }

  function handleRoleChange(value: number) {
    setSelectedRoleId(value);
    setCheckedPermissionIds(getRolePermissionIds(list, value).filter((id) => id > 0));
  }

  async function handleSave() {
    if (!selectedRoleId) {
      message.warning("请先选择一个角色模板");
      return;
    }

    const currentIds = getRolePermissionIds(list, selectedRoleId).filter((id) => id > 0);
    const desiredIds = checkedPermissionIds.filter((id) => id > 0 && permissionIdSet.has(id));
    const currentIdSet = new Set(currentIds);
    const desiredIdSet = new Set(desiredIds);
    const toGrant = desiredIds.filter((id) => !currentIdSet.has(id));
    const toRevoke = currentIds.filter((id) => !desiredIdSet.has(id));

    if (toGrant.length === 0 && toRevoke.length === 0) {
      if (linkNext === "k8s" && selectedRoleId) {
        navigateToK8s(selectedRoleId);
        return;
      }
      message.info("授权编排没有变化");
      return;
    }

    setSubmitting(true);
    try {
      await Promise.all([
        ...toGrant.map((permissionId) => grantPolicy({ role_id: selectedRoleId, permission_id: permissionId })),
        ...toRevoke.map((permissionId) => revokePolicy({ role_id: selectedRoleId, permission_id: permissionId })),
      ]);
      message.success("授权编排已同步");
      await bootstrap(selectedRoleId);
      await loadRoleExtras(selectedRoleId);
      if (linkNext === "k8s" && selectedRoleId) {
        navigateToK8s(selectedRoleId);
      }
    } finally {
      setSubmitting(false);
    }
  }

  function navigateToK8s(roleId: number) {
    const qs = new URLSearchParams({ subject: "role", role_id: String(roleId), from: "policies" });
    if (linkUserId > 0) qs.set("user_id", String(linkUserId));
    if (linkUsername) qs.set("username", linkUsername);
    navigate(`/k8s-scoped-policies?${qs.toString()}`);
  }

  async function handleSimulate() {
    if (!simulateUserId) {
      message.warning("请输入用户 ID");
      return;
    }
    try {
      const result = await simulatePolicy({
        user_id: simulateUserId,
        path: simulatePath.trim(),
        method: simulateMethod,
      });
      setSimulateResult(JSON.stringify(result, null, 2));
    } catch {
      setSimulateResult("");
    }
  }

  const roleTableScrollY = "calc(100dvh - 220px)";

  return (
    <div className="policies-auth-page">
      {linkFromUser && linkUsername ? (
        <Alert
          type="info"
          showIcon
          closable
          style={{ marginBottom: 12 }}
          message={`用户「${linkUsername}」已绑定角色，请为角色模板勾选 API 权限`}
          description="勾选完成后点击「同步权限」保存；侧栏菜单可见性将随入口 GET 权限联动。"
          action={
            linkNext === "k8s" && selectedRoleId ? (
              <Button size="small" onClick={() => navigateToK8s(selectedRoleId)}>
                跳过，直接配置 K8s
              </Button>
            ) : (
              <Button size="small" onClick={() => setSearchParams({})}>
                关闭提示
              </Button>
            )
          }
          onClose={() => setSearchParams({})}
        />
      ) : null}
      <Card className="table-card policies-auth-page__card" loading={loading}>
        <div className="toolbar auth-toolbar">
          <Space wrap>
            <Input
              allowClear
              value={roleKeyword}
              onChange={(e) => setRoleKeyword(e.target.value)}
              placeholder="分组名称/编码"
              style={{ width: 180 }}
            />
            <Input
              allowClear
              value={permissionKeyword}
              onChange={(e) => setPermissionKeyword(e.target.value)}
              placeholder="权限名称/资源路径"
              style={{ width: 220 }}
            />
            <Select
              allowClear
              placeholder="角色状态"
              style={{ width: 130 }}
              value={roleStatus}
              onChange={(v) => setRoleStatus(v)}
              options={[
                { value: 1, label: "启用" },
                { value: 0, label: "停用" },
              ]}
            />
          </Space>
          <div className="toolbar__actions">
            <Button
              onClick={() => {
                setRoleKeyword("");
                setPermissionKeyword("");
                setRoleStatus(undefined);
              }}
            >
              重置
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => void bootstrap(selectedRoleId)}>
              刷新
            </Button>
            <Button type="primary" icon={<SaveOutlined />} loading={submitting} onClick={() => void handleSave()}>
              同步权限
            </Button>
          </div>
        </div>

        <div className="auth-split">
          <Card
            className="glass-card auth-split__left"
            title="角色模板"
            extra={
              <Space size={4} wrap>
                <Tag className="status-chip status-chip--ok">共 {filteredRoles.length} 项</Tag>
                <Link to="/roles" className="policies-auth-page__roles-link">
                  <LinkOutlined /> 维护模板
                </Link>
              </Space>
            }
          >
            <Table
              rowKey="id"
              dataSource={filteredRoles}
              pagination={false}
              size="small"
              scroll={{ y: roleTableScrollY }}
              rowClassName={(record) => (record.id === selectedRoleId ? "is-selected-row" : "")}
              onRow={(record) => ({
                onClick: () => handleRoleChange(record.id),
              })}
              columns={[
                { title: "模板名称", dataIndex: "name" },
                { title: "模板编码", dataIndex: "code" },
                {
                  title: "状态",
                  dataIndex: "status",
                  width: 90,
                  render: (status: number) =>
                    status === 1 ? <Tag className="status-chip status-chip--ok">正常</Tag> : <Tag className="status-chip status-chip--off">停用</Tag>,
                },
              ]}
            />
          </Card>

          <Card
            className="glass-card auth-split__right"
            title="Casbin API 授权"
            extra={
              selectedRole ? (
                <Typography.Text className="inline-muted">
                  当前模板：{selectedRole.name}（树中已选 {checkedPermissionIds.length} 项）
                </Typography.Text>
              ) : null
            }
          >
            {selectedRole ? (
              <Tabs
                className="policies-auth-tabs"
                destroyInactiveTabPane={false}
                items={[
                  {
                    key: "unified",
                    label: "菜单+API 树",
                    children: (
                      <div className="auth-tab-pane auth-tab-pane--tree">
                        <Typography.Paragraph type="secondary" style={{ marginBottom: 8 }}>
                          按侧栏菜单分组展示入口 API；勾选后点击「同步权限」写入 Casbin。
                        </Typography.Paragraph>
                        <div className="tree-shell auth-tree-shell">
                          <Tree
                            checkable
                            defaultExpandAll
                            checkedKeys={checkedPermissionIds}
                            treeData={unifiedTreeData}
                            onCheck={(checkedKeys) => {
                              const nextIds = normalizeCheckedKeys(checkedKeys).filter((id) => permissionIdSet.has(id));
                              setCheckedPermissionIds(nextIds);
                            }}
                          />
                        </div>
                        <Button
                          size="small"
                          style={{ marginTop: 8 }}
                          onClick={() => setCheckedPermissionIds(collectGrantedPermissionIds(unifiedTree))}
                        >
                          重置为当前已授权
                        </Button>
                      </div>
                    ),
                  },
                  {
                    key: "tree",
                    label: "API 权限树",
                    children: (
                      <div className="auth-tab-pane auth-tab-pane--tree">
                        <Typography.Paragraph type="secondary" style={{ marginBottom: 8 }}>
                          叶子节点为可授权 API；未启用插件的菜单已从树中移除。
                        </Typography.Paragraph>
                        <div className="tree-shell auth-tree-shell">
                          <Tree
                            checkable
                            defaultExpandAll
                            checkedKeys={checkedPermissionIds}
                            treeData={filteredPermissionTree}
                            onCheck={(checkedKeys) => {
                              const nextIds = normalizeCheckedKeys(checkedKeys).filter((id) => permissionIdSet.has(id));
                              setCheckedPermissionIds(nextIds);
                            }}
                          />
                        </div>
                      </div>
                    ),
                  },
                  {
                    key: "conflicts",
                    label: `冲突分析（${conflicts.length}）`,
                    children: (
                      <div className="auth-tab-pane auth-tab-pane--table">
                        {conflicts.length === 0 ? (
                          <Empty description="未发现治理冲突" image={Empty.PRESENTED_IMAGE_SIMPLE} />
                        ) : (
                          <>
                            <Space style={{ marginBottom: 12 }} wrap>
                              <Button
                                type="primary"
                                loading={submitting}
                                disabled={!conflicts.some((c) => c.type === "menu_needs_entry_api")}
                                onClick={() => void handleFixMenuEntryConflicts()}
                              >
                                一键补齐入口 API
                              </Button>
                              <Button
                                loading={submitting}
                                disabled={
                                  !conflicts.some(
                                    (c) =>
                                      c.type === "plugin_disabled_policy_active" ||
                                      c.type === "api_granted_plugin_disabled",
                                  )
                                }
                                onClick={() => void handleFixDisabledPluginPolicies()}
                              >
                                清理禁用插件策略
                              </Button>
                              <Button onClick={() => selectedRoleId && void loadRoleExtras(selectedRoleId)}>重新分析</Button>
                              <Typography.Text type="secondary">
                                menu_needs_entry_api：补齐入口 GET；plugin_disabled_*：撤销未启用插件上的 Casbin 策略。
                              </Typography.Text>
                            </Space>
                            <Table
                            rowKey={(row, idx) => `${row.type}-${row.menu_path ?? row.resource}-${idx}`}
                            dataSource={conflicts}
                            size="small"
                            pagination={false}
                            scroll={{ y: "calc(100dvh - 360px)" }}
                            columns={[
                              {
                                title: "级别",
                                dataIndex: "severity",
                                width: 80,
                                render: (v: string) => (
                                  <Tag color={v === "error" ? "red" : v === "warning" ? "orange" : "blue"}>{v}</Tag>
                                ),
                              },
                              { title: "类型", dataIndex: "type", width: 180 },
                              { title: "说明", dataIndex: "message" },
                              { title: "菜单", dataIndex: "menu_path", width: 160 },
                              {
                                title: "入口 API",
                                width: 260,
                                render: (_: unknown, row: PolicyConflictItem) =>
                                  row.resource ? `${row.action || "GET"} ${row.resource}` : "—",
                              },
                              {
                                title: "建议",
                                dataIndex: "suggest_fix",
                                render: (v: string) => v || "—",
                              },
                            ]}
                          />
                          </>
                        )}
                      </div>
                    ),
                  },
                  {
                    key: "simulate",
                    label: "策略模拟",
                    children: (
                      <div className="auth-tab-pane">
                        <Space wrap style={{ marginBottom: 12 }}>
                          <InputNumber
                            placeholder="用户 ID"
                            value={simulateUserId}
                            onChange={(v) => setSimulateUserId(typeof v === "number" ? v : undefined)}
                            min={1}
                          />
                          <Input
                            placeholder="API path"
                            value={simulatePath}
                            onChange={(e) => setSimulatePath(e.target.value)}
                            style={{ width: 360 }}
                          />
                          <Select
                            value={simulateMethod}
                            onChange={setSimulateMethod}
                            style={{ width: 100 }}
                            options={["GET", "POST", "PUT", "DELETE", "PATCH"].map((m) => ({ value: m, label: m }))}
                          />
                          <Button type="primary" onClick={() => void handleSimulate()}>
                            模拟
                          </Button>
                        </Space>
                        {simulateResult ? (
                          <pre style={{ maxHeight: 360, overflow: "auto", background: "#f5f5f5", padding: 12, borderRadius: 6 }}>
                            {simulateResult}
                          </pre>
                        ) : (
                          <Typography.Text type="secondary">输入用户与 API，查看 super-admin / Casbin / 插件 / 项目成员分层判定。</Typography.Text>
                        )}
                      </div>
                    ),
                  },
                  {
                    key: "granted",
                    label: `已授权清单（${currentRolePolicies.length}）`,
                    children: (
                      <div className="auth-tab-pane auth-tab-pane--table">
                        <Table
                          rowKey={(record) => `${record.role_id}-${record.permission_id}`}
                          dataSource={currentRolePolicies}
                          pagination={{
                            current: assignPager.current,
                            pageSize: assignPager.pageSize,
                            total: currentRolePolicies.length,
                            showSizeChanger: true,
                            pageSizeOptions: [8, 10, 20, 50],
                            showTotal: (t) => `共 ${t} 条`,
                            hideOnSinglePage: currentRolePolicies.length <= assignPager.pageSize,
                            onChange: (page, pageSize) =>
                              setAssignPager({
                                current: page,
                                pageSize: pageSize ?? assignPager.pageSize,
                              }),
                          }}
                          size="small"
                          scroll={{ y: "calc(100dvh - 320px)" }}
                          columns={[
                            { title: "权限名称", dataIndex: "permission_name" },
                            {
                              title: "权限编码",
                              dataIndex: "resource",
                              render: (value: string, row: PolicyItem) => {
                                const key = `${row.resource}::${row.action.toUpperCase()}`;
                                const menus = menuLinks[key] ?? [];
                                return (
                                  <Space direction="vertical" size={0}>
                                    <Tag>{value}</Tag>
                                    {menus.length > 0 ? (
                                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                                        菜单: {menus.map((m) => m.path).join(", ")}
                                      </Typography.Text>
                                    ) : null}
                                  </Space>
                                );
                              },
                            },
                            { title: "模板名称", dataIndex: "role_name" },
                            { title: "Method", dataIndex: "action", width: 80 },
                          ]}
                        />
                      </div>
                    ),
                  },
                ]}
              />
            ) : (
              <Empty description="请选择左侧分组后进行授权" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            )}
          </Card>
        </div>
      </Card>
    </div>
  );
}

function selectRoleId(roles: RoleItem[], currentRoleId?: number) {
  if (currentRoleId && roles.some((role) => role.id === currentRoleId)) {
    return currentRoleId;
  }
  return roles[0]?.id;
}

function getRolePermissionIds(policies: PolicyItem[], roleId: number) {
  return policies.filter((policy) => policy.role_id === roleId).map((policy) => policy.permission_id);
}
