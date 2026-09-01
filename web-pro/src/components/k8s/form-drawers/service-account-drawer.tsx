// @ts-nocheck
/**
 * ServiceAccount「表单创建」抽屉。
 *
 * 从 k8s-resource-form-drawers.tsx 原地搬迁（RF-11 第二步），仅移动不改语义。
 * 单独成文件：本抽屉含 secrets / imagePullSecrets / 自动挂载 token 等多段配置，约 256 行。
 */

import { Button, Form, Input, Select, Space, Switch, Typography, message } from "antd";
import { useState } from "react";
import YAML from "yaml";
import { listClusterRoles, listRoles } from "../../../services/rbac";
import { applyServiceAccount } from "../../../services/serviceaccounts";
import { DrawerShellForm } from "./drawer-shell-form";

export function ServiceAccountFormCreateDrawer(props: {
  open: boolean;
  onClose: () => void;
  clusterId?: number;
  namespace: string;
  onSuccess: () => void;
  embedded?: boolean;
}) {
  const { open, onClose, clusterId, namespace, onSuccess, embedded } = props;
  const [form] = Form.useForm<{
    name: string;
    automount_token?: boolean;
    image_pull_secrets?: string[];
    annotations?: { key: string; value: string }[];
    labels?: { key: string; value: string }[];
    with_role_binding?: boolean;
    role_binding_name?: string;
    role_kind?: "Role" | "ClusterRole";
    role_name?: string;
    with_cluster_role_binding?: boolean;
    cluster_role_binding_name?: string;
    cluster_role_name?: string;
  }>();
  const [loading, setLoading] = useState(false);
  const [roleOptions, setRoleOptions] = useState<Array<{ label: string; value: string }>>([]);
  const [clusterRoleOptions, setClusterRoleOptions] = useState<Array<{ label: string; value: string }>>([]);
  const roleKind = Form.useWatch("role_kind", form) || "Role";

  async function preloadRoleOptions() {
    if (!clusterId) return;
    try {
      const [rolesRes, clusterRolesRes] = await Promise.all([
        listRoles(clusterId, namespace),
        listClusterRoles(clusterId),
      ]);
      setRoleOptions((rolesRes.list ?? []).map((it) => ({ label: it.name, value: it.name })));
      setClusterRoleOptions((clusterRolesRes.list ?? []).map((it) => ({ label: it.name, value: it.name })));
    } catch {
      // ignore options loading errors, user can still input manually
    }
  }

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const name = String(v.name || "").trim();
    const labels = Object.fromEntries(
      (v.labels ?? [])
        .map((p) => [String(p?.key ?? "").trim(), String(p?.value ?? "")] as const)
        .filter(([k]) => !!k),
    );
    const annotations = Object.fromEntries(
      (v.annotations ?? [])
        .map((p) => [String(p?.key ?? "").trim(), String(p?.value ?? "")] as const)
        .filter(([k]) => !!k),
    );

    const saDoc: Record<string, unknown> = {
      apiVersion: "v1",
      kind: "ServiceAccount",
      metadata: {
        name,
        namespace,
        ...(Object.keys(labels).length ? { labels } : {}),
        ...(Object.keys(annotations).length ? { annotations } : {}),
      },
      ...(typeof v.automount_token === "boolean" ? { automountServiceAccountToken: v.automount_token } : {}),
    };
    if ((v.image_pull_secrets ?? []).length > 0) {
      saDoc.imagePullSecrets = (v.image_pull_secrets ?? [])
        .map((it) => String(it || "").trim())
        .filter(Boolean)
        .map((it) => ({ name: it }));
    }

    const docs: Record<string, unknown>[] = [saDoc];

    if (v.with_role_binding) {
      const roleName = String(v.role_name || "").trim();
      if (!roleName) {
        message.error("已勾选 RoleBinding，请选择 Role/ClusterRole 名称");
        return;
      }
      docs.push({
        apiVersion: "rbac.authorization.k8s.io/v1",
        kind: "RoleBinding",
        metadata: {
          name: String(v.role_binding_name || `${name}-binding`).trim(),
          namespace,
        },
        subjects: [{ kind: "ServiceAccount", name, namespace }],
        roleRef: {
          apiGroup: "rbac.authorization.k8s.io",
          kind: v.role_kind || "Role",
          name: roleName,
        },
      });
    }

    if (v.with_cluster_role_binding) {
      const clusterRoleName = String(v.cluster_role_name || "").trim();
      if (!clusterRoleName) {
        message.error("已勾选 ClusterRoleBinding，请选择 ClusterRole 名称");
        return;
      }
      docs.push({
        apiVersion: "rbac.authorization.k8s.io/v1",
        kind: "ClusterRoleBinding",
        metadata: {
          name: String(v.cluster_role_binding_name || `${name}-cluster-binding`).trim(),
        },
        subjects: [{ kind: "ServiceAccount", name, namespace }],
        roleRef: {
          apiGroup: "rbac.authorization.k8s.io",
          kind: "ClusterRole",
          name: clusterRoleName,
        },
      });
    }

    setLoading(true);
    try {
      const manifest = docs.map((d) => YAML.stringify(d)).join("\n---\n");
      await applyServiceAccount(clusterId, manifest);
      message.success("ServiceAccount 已创建");
      onSuccess();
      onClose();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "创建失败");
    } finally {
      setLoading(false);
    }
  }

  return (
    <DrawerShellForm
      title="表单创建 ServiceAccount"
      open={embedded ? true : open}
      embedded={embedded}
      width={760}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{
        automount_token: true,
        image_pull_secrets: [],
        labels: [{ key: "", value: "" }],
        annotations: [{ key: "", value: "" }],
        with_role_binding: false,
        role_kind: "Role",
        with_cluster_role_binding: false,
      }}
    >
      <Form.Item label="目标命名空间">
        <Input value={namespace} readOnly />
      </Form.Item>
      <Form.Item name="name" label="ServiceAccount 名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="demo-sa" />
      </Form.Item>
      <Form.Item name="automount_token" label="自动挂载 ServiceAccount Token" valuePropName="checked">
        <Switch />
      </Form.Item>
      <Form.Item name="image_pull_secrets" label="ImagePullSecrets（可选）" extra="可输入多个 Secret 名称（支持回车）">
        <Select mode="tags" tokenSeparators={[",", " "]} placeholder="regcred" />
      </Form.Item>

      <Typography.Text type="secondary">Labels（可选）</Typography.Text>
      <Form.List name="labels">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}><Input placeholder="label key" style={{ width: 220 }} /></Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}><Input placeholder="label value" style={{ width: 240 }} /></Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加 Label</Button>
          </Space>
        )}
      </Form.List>

      <Typography.Text type="secondary">Annotations（可选）</Typography.Text>
      <Form.List name="annotations">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}><Input placeholder="annotation key" style={{ width: 220 }} /></Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}><Input placeholder="annotation value" style={{ width: 240 }} /></Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加 Annotation</Button>
          </Space>
        )}
      </Form.List>

      <Form.Item name="with_role_binding" label="同时创建 RoleBinding" valuePropName="checked">
        <Switch onChange={() => void preloadRoleOptions()} />
      </Form.Item>
      <Form.Item noStyle shouldUpdate={(prev, curr) => prev.with_role_binding !== curr.with_role_binding}>
        {({ getFieldValue }) =>
          getFieldValue("with_role_binding") ? (
            <Space direction="vertical" style={{ width: "100%", marginBottom: 12 }}>
              <Space style={{ width: "100%" }} align="start">
                <Form.Item name="role_binding_name" label="RoleBinding 名称" style={{ flex: 1 }}>
                  <Input placeholder="默认：<sa>-binding" />
                </Form.Item>
                <Form.Item name="role_kind" label="引用类型" style={{ width: 200 }}>
                  <Select options={[{ label: "Role", value: "Role" }, { label: "ClusterRole", value: "ClusterRole" }]} />
                </Form.Item>
              </Space>
              <Form.Item name="role_name" label={roleKind === "ClusterRole" ? "ClusterRole 名称" : "Role 名称"} rules={[{ required: true, message: "请选择角色名称" }]}>
                <Select
                  showSearch
                  allowClear
                  optionFilterProp="label"
                  options={roleKind === "ClusterRole" ? clusterRoleOptions : roleOptions}
                  placeholder={roleKind === "ClusterRole" ? "选择 ClusterRole" : "选择 Role"}
                  onFocus={() => void preloadRoleOptions()}
                />
              </Form.Item>
            </Space>
          ) : null
        }
      </Form.Item>

      <Form.Item name="with_cluster_role_binding" label="同时创建 ClusterRoleBinding" valuePropName="checked">
        <Switch onChange={() => void preloadRoleOptions()} />
      </Form.Item>
      <Form.Item noStyle shouldUpdate={(prev, curr) => prev.with_cluster_role_binding !== curr.with_cluster_role_binding}>
        {({ getFieldValue }) =>
          getFieldValue("with_cluster_role_binding") ? (
            <Space direction="vertical" style={{ width: "100%" }}>
              <Form.Item name="cluster_role_binding_name" label="ClusterRoleBinding 名称">
                <Input placeholder="默认：<sa>-cluster-binding" />
              </Form.Item>
              <Form.Item name="cluster_role_name" label="ClusterRole 名称" rules={[{ required: true, message: "请选择 ClusterRole 名称" }]}>
                <Select
                  showSearch
                  allowClear
                  optionFilterProp="label"
                  options={clusterRoleOptions}
                  placeholder="选择 ClusterRole"
                  onFocus={() => void preloadRoleOptions()}
                />
              </Form.Item>
            </Space>
          ) : null
        }
      </Form.Item>
    </DrawerShellForm>
  );
}
