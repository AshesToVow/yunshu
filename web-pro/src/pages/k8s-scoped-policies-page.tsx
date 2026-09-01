// @ts-nocheck
import { DeleteOutlined, ReloadOutlined } from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Divider,
  Empty,
  Form,
  Modal,
  Popconfirm,
  Segmented,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from '@umijs/max';
import { getClusters, listNamespaces } from "../services/clusters";
import {
  createK8sNamespaceDenyRule,
  deleteK8sNamespaceDenyRule,
  listK8sNamespaceDenyRules,
  type K8sNamespaceDenyRule,
} from "../services/k8s-namespace-deny";
import {
  deleteK8sClusterGrant,
  grantK8sScopedPoliciesPreset,
  listK8sCapabilities,
  listK8sClusterGrants,
  listK8sPoliciesByRole,
  splitK8sScopedPoliciesByNamespaces,
  type K8sCapabilityItem,
  type K8sClusterAccessItem,
} from "../services/k8s-policies";
import { getRoleOptions } from "../services/roles";
import { getUsers } from "../services/users";
import { listUserGroups } from "../services/user-groups";
import type { RoleItem, UserItem } from "../types/api";
import type { UserGroupItem } from "../services/user-groups";
import { DenyRulesCard } from "./k8s-policies/deny-rules-card";
import { PresetForm } from "./k8s-policies/preset-form";
import {
  PRESET_CAPS,
  presetLabel,
  renderCapabilityTags,
  subjectPrincipalRef,
  type BootstrapPref,
  type SubjectKind,
} from "./k8s-policies/scoped-subject";

export function K8sScopedPoliciesPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [loading, setLoading] = useState(false);
  const [presetSubmitting, setPresetSubmitting] = useState(false);
  const [subjectKind, setSubjectKind] = useState<SubjectKind>("role");
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [groups, setGroups] = useState<UserGroupItem[]>([]);
  const [users, setUsers] = useState<UserItem[]>([]);
  const [selectedRoleId, setSelectedRoleId] = useState<number>();
  const [selectedGroupId, setSelectedGroupId] = useState<number>();
  const [selectedUserId, setSelectedUserId] = useState<number>();
  const [clusterOptions, setClusterOptions] = useState<{ id: number; name: string }[]>([]);
  const [accessGrants, setAccessGrants] = useState<K8sClusterAccessItem[]>([]);
  const [denyRules, setDenyRules] = useState<K8sNamespaceDenyRule[]>([]);
  const [denyLoading, setDenyLoading] = useState(false);
  const [denySubmitting, setDenySubmitting] = useState(false);
  const [presetForm] = Form.useForm<{
    cluster_ids: number[];
    preset: "readonly" | "readonly_exec" | "admin";
    capabilities: string[];
    deny_namespaces?: string[];
    allow_namespaces?: string[];
  }>();
  const [denyForm] = Form.useForm<{ cluster_id?: number; namespace?: string }>();
  const [capCatalog, setCapCatalog] = useState<K8sCapabilityItem[]>([]);
  const [presetNsOptions, setPresetNsOptions] = useState<{ label: string; value: string }[]>([]);
  const [presetNsLoading, setPresetNsLoading] = useState(false);
  const [denyNsOptions, setDenyNsOptions] = useState<{ label: string; value: string }[]>([]);
  const [denyNsLoading, setDenyNsLoading] = useState(false);
  const [splitOpen, setSplitOpen] = useState(false);
  const [splitSubmitting, setSplitSubmitting] = useState(false);
  const [splitForm] = Form.useForm<{
    cluster_ids: number[];
    splits: Array<{ namespace?: string; preset?: "readonly" | "readonly_exec" | "admin" }>;
  }>();

  const watchedPresetClusterIds = Form.useWatch("cluster_ids", presetForm) ?? [];
  const watchedDenyClusterId = Form.useWatch("cluster_id", denyForm);
  const watchedCapabilities = Form.useWatch("capabilities", presetForm) ?? [];

  const capNameByCode = useMemo(() => {
    const m = new Map<string, string>();
    for (const c of capCatalog) m.set(c.code, c.name);
    return m;
  }, [capCatalog]);

  const selectedRole = useMemo(() => roles.find((r) => r.id === selectedRoleId) ?? null, [roles, selectedRoleId]);
  const selectedGroup = useMemo(() => groups.find((g) => g.id === selectedGroupId) ?? null, [groups, selectedGroupId]);
  const selectedUser = useMemo(() => users.find((u) => u.id === selectedUserId) ?? null, [users, selectedUserId]);
  const linkFromPolicies = searchParams.get("from") === "policies";
  const linkUsername = searchParams.get("username") ?? "";
  const activeSubjectReady =
    (subjectKind === "role" && selectedRole != null) ||
    (subjectKind === "group" && selectedGroup != null) ||
    (subjectKind === "user" && selectedUser != null);

  const clusterNameById = useMemo(() => new Map(clusterOptions.map((c) => [c.id, c.name])), [clusterOptions]);

  const presetClusterIds = useMemo(() => {
    const raw = Array.isArray(watchedPresetClusterIds) ? watchedPresetClusterIds : [];
    return raw.filter((x): x is number => typeof x === "number" && x > 0);
  }, [watchedPresetClusterIds]);
  const presetClusterKey = useMemo(() => presetClusterIds.slice().sort((a, b) => a - b).join(","), [presetClusterIds]);

  useEffect(() => {
    if (presetClusterIds.length === 0) {
      setPresetNsOptions([]);
      setPresetNsLoading(false);
      return;
    }
    let cancelled = false;
    setPresetNsLoading(true);
    void Promise.all(presetClusterIds.map((id) => listNamespaces(id)))
      .then((results) => {
        if (cancelled) return;
        const seen = new Set<string>();
        const opts: { label: string; value: string }[] = [];
        for (const res of results) {
          for (const n of res.list ?? []) {
            if (!seen.has(n.name)) {
              seen.add(n.name);
              opts.push({ label: n.name, value: n.name });
            }
          }
        }
        opts.sort((a, b) => a.label.localeCompare(b.label));
        setPresetNsOptions(opts);
      })
      .catch(() => {
        if (!cancelled) setPresetNsOptions([]);
      })
      .finally(() => {
        if (!cancelled) setPresetNsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [presetClusterKey]);

  useEffect(() => {
    const cid = typeof watchedDenyClusterId === "number" ? watchedDenyClusterId : undefined;
    if (!cid) {
      setDenyNsOptions([]);
      setDenyNsLoading(false);
      denyForm.setFieldsValue({ namespace: undefined });
      return;
    }
    let cancelled = false;
    setDenyNsLoading(true);
    void listNamespaces(cid)
      .then((res) => {
        if (cancelled) return;
        const opts = (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
        setDenyNsOptions(opts);
      })
      .catch(() => {
        if (!cancelled) setDenyNsOptions([]);
      })
      .finally(() => {
        if (!cancelled) setDenyNsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [watchedDenyClusterId]); // denyForm.setFieldsValue 稳定；仅集群变更时需重置命名空间

  useEffect(() => {
    const subject = searchParams.get("subject");
    const userId = Number(searchParams.get("user_id"));
    const roleId = Number(searchParams.get("role_id"));
    const groupId = Number(searchParams.get("group_id"));
    if (subject === "user" && userId > 0) {
      setSubjectKind("user");
      void bootstrap({ kind: "user", userId });
    } else if (subject === "group" && groupId > 0) {
      setSubjectKind("group");
      void bootstrap({ kind: "group", groupId });
    } else if (subject === "role" && roleId > 0) {
      setSubjectKind("role");
      void bootstrap({ kind: "role", roleId });
    } else {
      void bootstrap();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function bootstrap(pref?: BootstrapPref) {
    setLoading(true);
    try {
      const [roleData, clusterData, groupRes, userRes, capRes] = await Promise.all([
        getRoleOptions(),
        getClusters({ page: 1, page_size: 200 }),
        listUserGroups({ page: 1, page_size: 500 }),
        getUsers({ page: 1, page_size: 500 }),
        listK8sCapabilities().catch(() => ({ list: [] as K8sCapabilityItem[] })),
      ]);
      setRoles(roleData.list);
      setGroups(groupRes.list ?? []);
      setUsers(userRes.list ?? []);
      setClusterOptions(clusterData.list.map((c) => ({ id: c.id, name: c.name })));
      setCapCatalog(capRes.list ?? []);
      presetForm.setFieldsValue({
        capabilities: PRESET_CAPS.readonly,
        preset: "readonly",
      });

      const kind = pref?.kind ?? subjectKind;
      const nextRoleId = pref?.roleId ?? selectedRoleId ?? roleData.list[0]?.id;
      const nextGroupId = pref?.groupId ?? selectedGroupId ?? groupRes.list[0]?.id;
      const nextUserId = pref?.userId ?? selectedUserId ?? userRes.list[0]?.id;

      setSelectedRoleId(nextRoleId);
      setSelectedGroupId(nextGroupId);
      setSelectedUserId(nextUserId);

      if (kind === "role" && nextRoleId) {
        await refreshAccessGrants("role", nextRoleId, undefined, undefined);
        await refreshDenyRules("role", roleData.list.find((r) => r.id === nextRoleId)?.code ?? "");
      } else if (kind === "group" && nextGroupId) {
        await refreshAccessGrants("group", undefined, nextGroupId, undefined);
        await refreshDenyRules("group", groupRes.list.find((g) => g.id === nextGroupId)?.code ?? "");
      } else if (kind === "user" && nextUserId) {
        await refreshAccessGrants("user", undefined, undefined, nextUserId);
        await refreshDenyRules("user", String(nextUserId));
      } else {
        setAccessGrants([]);
        setDenyRules([]);
      }
    } finally {
      setLoading(false);
    }
  }

  async function refreshAccessGrants(
    kind: SubjectKind,
    roleId?: number,
    groupId?: number,
    userId?: number,
  ) {
    if (kind === "role" && roleId) {
      const result = await listK8sPoliciesByRole(roleId);
      setAccessGrants(result.list);
      return;
    }
    if (kind === "group" && groupId) {
      const result = await listK8sClusterGrants({ group_id: groupId });
      setAccessGrants(result.list);
      return;
    }
    if (kind === "user" && userId) {
      const result = await listK8sClusterGrants({ user_id: userId });
      setAccessGrants(result.list);
      return;
    }
    setAccessGrants([]);
  }

  async function refreshDenyRules(principalKind: SubjectKind, principalRef: string) {
    const ref = principalRef.trim();
    if (!ref) {
      setDenyRules([]);
      return;
    }
    setDenyLoading(true);
    try {
      const data = await listK8sNamespaceDenyRules({ principal_kind: principalKind, principal_ref: ref });
      setDenyRules(data.list ?? []);
    } catch {
      setDenyRules([]);
    } finally {
      setDenyLoading(false);
    }
  }

  function handleSavePreset() {
    if (subjectKind === "role" && !selectedRoleId) return;
    if (subjectKind === "group" && !selectedGroupId) return;
    if (subjectKind === "user" && !selectedUserId) return;
    void (async () => {
      const values = await presetForm.validateFields();
      setPresetSubmitting(true);
      try {
        const denyRaw = values.deny_namespaces ?? [];
        const denyList = (Array.isArray(denyRaw) ? denyRaw : []).map((s) => String(s).trim()).filter(Boolean);
        const allowRaw = values.allow_namespaces ?? [];
        const allowList = (Array.isArray(allowRaw) ? allowRaw : []).map((s) => String(s).trim()).filter(Boolean);
        const caps = Array.isArray(values.capabilities) ? values.capabilities : [];
        const base = {
          cluster_ids: values.cluster_ids ?? [],
          capabilities: caps,
          deny_namespaces: denyList.length ? denyList : undefined,
          allow_namespaces: allowList.length ? allowList : undefined,
        };
        const payload =
          subjectKind === "role"
            ? {
                ...base,
                principal_kind: "role" as const,
                role_id: selectedRoleId!,
              }
            : subjectKind === "user"
              ? {
                  ...base,
                  principal_kind: "user" as const,
                  user_id: selectedUserId!,
                }
              : {
                  ...base,
                  principal_kind: "group" as const,
                  group_id: selectedGroupId!,
                };
        const resp = await grantK8sScopedPoliciesPreset(payload);
        message.success(
          `能力包已保存：新增 ${resp.added}，更新 ${resp.skipped}；黑名单新增 ${resp.deny_rules_added}（跳过 ${resp.deny_rules_skipped}）；白名单新增 ${resp.allow_rules_added}（跳过 ${resp.allow_rules_skipped}）`,
        );
        const pref = subjectPrincipalRef(subjectKind, selectedRole, selectedGroup, selectedUserId);
        await refreshAccessGrants(subjectKind, selectedRoleId, selectedGroupId, selectedUserId);
        await refreshDenyRules(subjectKind, pref);
      } finally {
        setPresetSubmitting(false);
      }
    })();
  }

  function handleOpenSplit() {
    splitForm.resetFields();
    splitForm.setFieldsValue({
      cluster_ids: presetForm.getFieldValue("cluster_ids") ?? [],
      splits: [{ namespace: undefined, preset: "readonly" }],
    });
    setSplitOpen(true);
  }

  async function handleDenyFinish(v: { cluster_id?: number; namespace?: string }) {
    const cid = v.cluster_id;
    const ns = String(v.namespace ?? "").trim();
    if (!cid || !ns) {
      message.warning("请选择集群并填写命名空间");
      return;
    }
    const pk = subjectKind;
    const pref = subjectPrincipalRef(subjectKind, selectedRole, selectedGroup, selectedUserId);
    setDenySubmitting(true);
    try {
      await createK8sNamespaceDenyRule({
        principal_kind: pk,
        principal_ref: pref,
        cluster_id: cid,
        namespace: ns,
      });
      message.success("已添加黑名单规则");
      denyForm.resetFields();
      await refreshDenyRules(pk, pref);
    } finally {
      setDenySubmitting(false);
    }
  }

  async function handleDeleteDenyRule(r: K8sNamespaceDenyRule) {
    try {
      await deleteK8sNamespaceDenyRule(r.id);
      message.success("已删除");
      await refreshDenyRules(
        subjectKind,
        subjectPrincipalRef(subjectKind, selectedRole, selectedGroup, selectedUserId),
      );
    } catch {
      /* http 拦截器已提示 */
    }
  }

  return (
    <div>
      {linkFromPolicies ? (
        <Alert
          type="success"
          showIcon
          closable
          style={{ marginBottom: 12 }}
          message="API 权限已配置，请为角色模板下发 Kubernetes 集群档位"
          description={
            linkUsername
              ? `用户「${linkUsername}」所属角色的集群访问在此配置；也可切换为「用户」主体单独直授。`
              : "选择集群与档位后点击「按档位保存」完成授权。"
          }
          onClose={() => setSearchParams({})}
        />
      ) : null}
      <Card
        className="table-card"
        title="Kubernetes 集群访问档位（数据库维护，不经 Casbin）"
        loading={loading}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void bootstrap({ kind: subjectKind, roleId: selectedRoleId, groupId: selectedGroupId, userId: selectedUserId })}>
              刷新
            </Button>
          </Space>
        }
      >
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 12 }}
          message="档位不替代 API 授权"
          description="此处 GrantPreset 只写入 k8s_cluster_access_grants，用于范围校验（命名空间黑白名单 + 读写档位）。写操作仍须在「授权管理」中授予对应 API（POST/PUT/DELETE）；仅有档位而无 Casbin 权限时写接口仍会 403。"
        />
        <Space direction="vertical" size={12} style={{ width: "100%" }}>
          <Space wrap style={{ width: "100%", justifyContent: "space-between" }}>
            <Space wrap align="center">
              <Segmented
                value={subjectKind}
                options={[
                  { label: "角色模板", value: "role" },
                  { label: "用户", value: "user" },
                  { label: "用户组", value: "group" },
                ]}
                onChange={(v) => {
                  const k = v as SubjectKind;
                  setSubjectKind(k);
                  void (async () => {
                    if (k === "role") {
                      const rid = selectedRoleId ?? roles[0]?.id;
                      if (rid) {
                        setSelectedRoleId(rid);
                        await refreshAccessGrants("role", rid, undefined, undefined);
                        await refreshDenyRules("role", roles.find((r) => r.id === rid)?.code ?? "");
                      } else {
                        setAccessGrants([]);
                        setDenyRules([]);
                      }
                    } else if (k === "user") {
                      const uid = selectedUserId ?? users[0]?.id;
                      if (uid) {
                        setSelectedUserId(uid);
                        await refreshAccessGrants("user", undefined, undefined, uid);
                        await refreshDenyRules("user", String(uid));
                      } else {
                        setAccessGrants([]);
                        setDenyRules([]);
                      }
                    } else {
                      const gid = selectedGroupId ?? groups[0]?.id;
                      if (gid) {
                        setSelectedGroupId(gid);
                        await refreshAccessGrants("group", undefined, gid, undefined);
                        await refreshDenyRules("group", groups.find((g) => g.id === gid)?.code ?? "");
                      } else {
                        setAccessGrants([]);
                        setDenyRules([]);
                      }
                    }
                  })();
                }}
              />
              {subjectKind === "role" ? (
                <Select
                  placeholder="请选择角色模板"
                  style={{ minWidth: 300 }}
                  value={selectedRoleId}
                  onChange={(v) => {
                    setSelectedRoleId(v);
                    void refreshAccessGrants("role", v, undefined, undefined);
                    const rc = roles.find((r) => r.id === v)?.code ?? "";
                    void refreshDenyRules("role", rc);
                  }}
                  options={roles.map((role) => ({ label: `${role.name} (${role.code})`, value: role.id }))}
                />
              ) : subjectKind === "user" ? (
                <Select
                  placeholder="请选择用户"
                  style={{ minWidth: 300 }}
                  showSearch
                  optionFilterProp="label"
                  value={selectedUserId}
                  onChange={(v) => {
                    setSelectedUserId(v);
                    void refreshAccessGrants("user", undefined, undefined, v);
                    void refreshDenyRules("user", String(v));
                  }}
                  options={users.map((u) => ({
                    label: `${u.nickname || u.username} (${u.username})`,
                    value: u.id,
                  }))}
                />
              ) : (
                <Select
                  placeholder="请选择用户组"
                  style={{ minWidth: 300 }}
                  value={selectedGroupId}
                  onChange={(v) => {
                    setSelectedGroupId(v);
                    void refreshAccessGrants("group", undefined, v, undefined);
                    const gc = groups.find((g) => g.id === v)?.code ?? "";
                    void refreshDenyRules("group", gc);
                  }}
                  options={groups.map((g) => ({ label: `${g.name} (${g.code})`, value: g.id }))}
                />
              )}
            </Space>
          </Space>

          {activeSubjectReady ? (
            <>
              <Alert
                type="info"
                showIcon
                style={{ width: "100%" }}
                message="与 API / Casbin 的关系"
                description={
                  <span>
                    此处为<strong>主体</strong>（角色模板 / 用户 / 用户组）配置<strong>集群能力包</strong>（可快捷三档或自定义勾选），数据在表{" "}
                    <Typography.Text code>k8s_cluster_access_grants</Typography.Text>。
                    <strong>能力包不替代 API 授权</strong>：HTTP 接口能否调用仍由<strong>授权管理</strong>中的 Casbin
                    API 权限决定；仅勾选变更类能力而没有对应写接口 Casbin 权限时，POST/PUT/DELETE 仍会 403。带{" "}
                    <Typography.Text code>cluster_id</Typography.Text> 的 K8s 类请求在通过 API 鉴权后，再按此处能力包与<strong>命名空间黑/白名单</strong>校验。
                  </span>
                }
              />

              <Alert
                type="info"
                showIcon
                style={{ width: "100%" }}
                message="能力包下发"
                description={
                  <span>
                    快捷档位会预填能力勾选；也可自行勾选组合（保存时以勾选为准）。未勾「只读浏览」时后端会自动补上，否则无法列表。命名空间黑/白名单可选：须选择<strong>具体集群</strong>。
                  </span>
                }
              />

              <PresetForm
                form={presetForm}
                watchedCapabilities={watchedCapabilities}
                capCatalog={capCatalog}
                clusterOptions={clusterOptions}
                presetClusterIds={presetClusterIds}
                presetNsLoading={presetNsLoading}
                presetNsOptions={presetNsOptions}
                presetSubmitting={presetSubmitting}
                activeSubjectReady={activeSubjectReady}
                onSave={handleSavePreset}
                onOpenSplit={handleOpenSplit}
              />

              <Divider style={{ margin: "8px 0" }} />
              <Typography.Text strong>当前主体的集群能力包</Typography.Text>
              <Table<K8sClusterAccessItem>
                rowKey="id"
                dataSource={accessGrants}
                pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
                size="small"
                scroll={{ x: "max-content" }}
                columns={[
                  {
                    title: "主体",
                    key: "principal",
                    render: (_: unknown, r: K8sClusterAccessItem) => (
                      <span>
                        <Tag>{r.principal_kind || (r.role_code ? "role" : "")}</Tag>{" "}
                        <Typography.Text code>{r.principal_ref || r.role_code}</Typography.Text>
                      </span>
                    ),
                  },
                  {
                    title: "集群",
                    dataIndex: "cluster_id",
                    render: (v: number) =>
                      v === 0 ? (
                        <Tag color="blue">全部集群</Tag>
                      ) : (
                        <Tag>{clusterNameById.get(v) ?? `集群 #${v}`}</Tag>
                      ),
                  },
                  {
                    title: "档位",
                    dataIndex: "preset",
                    width: 120,
                    render: (v: string) => <Tag color="processing">{presetLabel(v)}</Tag>,
                  },
                  {
                    title: "能力包",
                    dataIndex: "capabilities",
                    render: (v: string[] | undefined) => renderCapabilityTags(v, capNameByCode),
                  },
                  {
                    title: "操作",
                    key: "op",
                    width: 100,
                    render: (_, r) => (
                      <Popconfirm
                        title="确定删除该集群授权？"
                        onConfirm={() =>
                          void (async () => {
                            try {
                              await deleteK8sClusterGrant(r.id);
                              message.success("已删除");
                              if (subjectKind === "role" && selectedRoleId) {
                                await refreshAccessGrants("role", selectedRoleId, undefined, undefined);
                              } else if (subjectKind === "group" && selectedGroupId) {
                                await refreshAccessGrants("group", undefined, selectedGroupId, undefined);
                              } else if (subjectKind === "user" && selectedUserId) {
                                await refreshAccessGrants("user", undefined, undefined, selectedUserId);
                              }
                            } catch {
                              /* http 拦截器已提示 */
                            }
                          })()
                        }
                      >
                        <Button type="link" danger size="small" icon={<DeleteOutlined />}>
                          删除
                        </Button>
                      </Popconfirm>
                    ),
                  },
                ]}
              />
            </>
          ) : (
            <Empty
              description={
                subjectKind === "role"
                  ? "暂无可配置角色模板"
                  : subjectKind === "user"
                    ? "暂无可选用户"
                    : "暂无可选用户组，请先在「用户组管理」创建并绑定成员"
              }
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
          )}
        </Space>
      </Card>

      <DenyRulesCard
        denyLoading={denyLoading}
        activeSubjectReady={activeSubjectReady}
        denyForm={denyForm}
        denySubmitting={denySubmitting}
        subjectKind={subjectKind}
        selectedRole={selectedRole}
        selectedGroup={selectedGroup}
        selectedUser={selectedUser}
        clusterOptions={clusterOptions}
        watchedDenyClusterId={typeof watchedDenyClusterId === "number" ? watchedDenyClusterId : undefined}
        denyNsLoading={denyNsLoading}
        denyNsOptions={denyNsOptions}
        denyRules={denyRules}
        clusterNameById={clusterNameById}
        onDenyFinish={handleDenyFinish}
        onDeleteDenyRule={handleDeleteDenyRule}
      />

      <Modal
        title="按命名空间拆分档位"
        open={splitOpen}
        onCancel={() => setSplitOpen(false)}
        width={720}
        onOk={() => {
          if (!activeSubjectReady) return;
          void (async () => {
            const values = await splitForm.validateFields();
            const clusterIds = values.cluster_ids ?? [];
            if (!clusterIds.length) {
              message.warning("请选择至少一个集群");
              return;
            }
            const splits = (values.splits ?? [])
              .map((s) => ({
                namespace: String(s.namespace || "").trim(),
                preset: s.preset || "readonly",
              }))
              .filter((s) => s.namespace);
            if (!splits.length) {
              message.warning("请至少填写一行命名空间");
              return;
            }
            setSplitSubmitting(true);
            try {
              const base =
                subjectKind === "role"
                  ? { principal_kind: "role" as const, role_id: selectedRoleId! }
                  : subjectKind === "user"
                    ? { principal_kind: "user" as const, user_id: selectedUserId! }
                    : { principal_kind: "group" as const, group_id: selectedGroupId! };
              const resp = await splitK8sScopedPoliciesByNamespaces({
                ...base,
                cluster_ids: clusterIds,
                splits,
              });
              message.success(`已拆分下发：新增 ${resp.added}，跳过 ${resp.skipped}`);
              setSplitOpen(false);
              const pref = subjectPrincipalRef(subjectKind, selectedRole, selectedGroup, selectedUserId);
              await refreshAccessGrants(subjectKind, selectedRoleId, selectedGroupId, selectedUserId);
              await refreshDenyRules(subjectKind, pref);
            } finally {
              setSplitSubmitting(false);
            }
          })();
        }}
        confirmLoading={splitSubmitting}
        destroyOnClose
      >
        <Typography.Paragraph type="secondary">
          为同一主体在不同命名空间下发不同档位（每行一条 NS + preset），须选择具体集群。
        </Typography.Paragraph>
        <Form form={splitForm} layout="vertical">
          <Form.Item label="集群" name="cluster_ids" rules={[{ required: true, type: "array", min: 1, message: "请选择集群" }]}>
            <Select
              mode="multiple"
              options={clusterOptions.map((c) => ({ label: c.name, value: c.id }))}
              placeholder="选择集群"
            />
          </Form.Item>
          <Form.List name="splits">
            {(fields, { add, remove }) => (
              <>
                {fields.map((field) => (
                  <Space key={field.key} align="baseline" style={{ display: "flex", marginBottom: 8 }}>
                    <Form.Item
                      {...field}
                      name={[field.name, "namespace"]}
                      rules={[{ required: true, message: "命名空间" }]}
                      style={{ width: 220 }}
                    >
                      <Select
                        showSearch
                        placeholder="命名空间"
                        options={presetNsOptions}
                        loading={presetNsLoading}
                      />
                    </Form.Item>
                    <Form.Item
                      {...field}
                      name={[field.name, "preset"]}
                      initialValue="readonly"
                      style={{ width: 200 }}
                    >
                      <Select
                        options={[
                          { value: "readonly", label: "只读" },
                          { value: "readonly_exec", label: "只读+Exec" },
                          { value: "admin", label: "管理" },
                        ]}
                      />
                    </Form.Item>
                    <Button type="link" onClick={() => remove(field.name)}>删除</Button>
                  </Space>
                ))}
                <Button type="dashed" onClick={() => add({ preset: "readonly" })} block>
                  添加命名空间行
                </Button>
              </>
            )}
          </Form.List>
        </Form>
      </Modal>
    </div>
  );
}
