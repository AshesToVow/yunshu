/**
 * RBAC「表单创建」抽屉：Role / ClusterRole / RoleBinding / ClusterRoleBinding。
 *
 * 从 k8s-resource-form-drawers.tsx 原地搬迁（RF-11 第二步），仅移动不改语义。
 * ServiceAccount 虽属 RBAC 域，但单体 256 行，合并后会突破 500 行目标线，
 * 故另置 ./service-account-drawer.tsx（与台账原方案的唯一偏离，已在 backlog 记录）。
 */

import { Form, Input, Select, Space, message } from "antd";
import { useState } from "react";
import YAML from "yaml";
import { applyRbac } from "../../../services/rbac";
import { DrawerShellForm } from "./drawer-shell-form";
import { apiGroupOptions, resourceOptions, subjectKindOptions, verbOptions } from "./options";

export function RbacRoleFormCreateDrawer(props: {
  open: boolean;
  onClose: () => void;
  clusterId?: number;
  namespace: string;
  onSuccess: () => void;
  embedded?: boolean;
}) {
  const { open, onClose, clusterId, namespace, onSuccess, embedded } = props;
  const [form] = Form.useForm<{ name: string; api_group: string; resources: string[]; verbs: string[] }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const rules = [
      {
        apiGroups: [v.api_group === "" ? "" : v.api_group],
        resources: v.resources?.length ? v.resources : ["pods"],
        verbs: v.verbs?.length ? v.verbs : ["get", "list"],
      },
    ];
    const doc = {
      apiVersion: "rbac.authorization.k8s.io/v1",
      kind: "Role",
      metadata: { name: String(v.name).trim(), namespace },
      rules,
    };
    setLoading(true);
    try {
      await applyRbac(clusterId, YAML.stringify(doc));
      message.success("Role 已创建");
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
      title="表单创建 Role"
      open={embedded ? true : open}
      embedded={embedded}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{ api_group: "", resources: ["pods"], verbs: ["get", "list"] }}
    >
      <Form.Item label="命名空间">
        <Input value={namespace} readOnly />
      </Form.Item>
      <Form.Item name="name" label="Role 名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="demo-role" />
      </Form.Item>
      <Form.Item name="api_group" label="API 组" rules={[{ required: true }]}>
        <Select options={apiGroupOptions} />
      </Form.Item>
      <Form.Item name="resources" label="资源" rules={[{ required: true, message: "请选择资源" }]}>
        <Select mode="multiple" options={resourceOptions} placeholder="选择资源类型" optionFilterProp="label" />
      </Form.Item>
      <Form.Item name="verbs" label="动词" rules={[{ required: true, message: "请选择动词" }]}>
        <Select mode="multiple" options={verbOptions} />
      </Form.Item>
    </DrawerShellForm>
  );
}

export function RbacClusterRoleFormCreateDrawer(props: {
  open: boolean;
  onClose: () => void;
  clusterId?: number;
  onSuccess: () => void;
  embedded?: boolean;
}) {
  const { open, onClose, clusterId, onSuccess, embedded } = props;
  const [form] = Form.useForm<{ name: string; api_group: string; resources: string[]; verbs: string[] }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const rules = [
      {
        apiGroups: [v.api_group === "" ? "" : v.api_group],
        resources: v.resources?.length ? v.resources : ["nodes"],
        verbs: v.verbs?.length ? v.verbs : ["get", "list"],
      },
    ];
    const doc = {
      apiVersion: "rbac.authorization.k8s.io/v1",
      kind: "ClusterRole",
      metadata: { name: String(v.name).trim() },
      rules,
    };
    setLoading(true);
    try {
      await applyRbac(clusterId, YAML.stringify(doc));
      message.success("ClusterRole 已创建");
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
      title="表单创建 ClusterRole"
      open={embedded ? true : open}
      embedded={embedded}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{ api_group: "", resources: ["nodes"], verbs: ["get", "list"] }}
    >
      <Form.Item name="name" label="ClusterRole 名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="demo-clusterrole" />
      </Form.Item>
      <Form.Item name="api_group" label="API 组" rules={[{ required: true }]}>
        <Select options={apiGroupOptions} />
      </Form.Item>
      <Form.Item name="resources" label="资源" rules={[{ required: true }]}>
        <Select mode="multiple" options={resourceOptions} optionFilterProp="label" />
      </Form.Item>
      <Form.Item name="verbs" label="动词" rules={[{ required: true }]}>
        <Select mode="multiple" options={verbOptions} />
      </Form.Item>
    </DrawerShellForm>
  );
}

export function RbacRoleBindingFormCreateDrawer(props: {
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
    role_kind: "Role" | "ClusterRole";
    role_name: string;
    subject_kind: string;
    subject_name: string;
    subject_namespace: string;
  }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const sub: Record<string, unknown> = { kind: v.subject_kind, name: String(v.subject_name).trim() };
    if (v.subject_kind === "ServiceAccount" && String(v.subject_namespace || "").trim()) {
      sub.namespace = String(v.subject_namespace).trim();
    }
    const doc = {
      apiVersion: "rbac.authorization.k8s.io/v1",
      kind: "RoleBinding",
      metadata: { name: String(v.name).trim(), namespace },
      subjects: [sub],
      roleRef: {
        apiGroup: "rbac.authorization.k8s.io",
        kind: v.role_kind,
        name: String(v.role_name).trim(),
      },
    };
    setLoading(true);
    try {
      await applyRbac(clusterId, YAML.stringify(doc));
      message.success("RoleBinding 已创建");
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
      title="表单创建 RoleBinding"
      open={embedded ? true : open}
      embedded={embedded}
      width={760}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{ role_kind: "Role", subject_kind: "User" }}
    >
      <Form.Item label="命名空间">
        <Input value={namespace} readOnly />
      </Form.Item>
      <Form.Item name="name" label="RoleBinding 名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="demo-binding" />
      </Form.Item>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="role_kind" label="引用类型" rules={[{ required: true }]} style={{ width: 200 }}>
          <Select options={[{ label: "Role", value: "Role" }, { label: "ClusterRole", value: "ClusterRole" }]} />
        </Form.Item>
        <Form.Item name="role_name" label="角色名称" rules={[{ required: true, message: "请输入角色名" }]} style={{ flex: 1 }}>
          <Input placeholder="demo-role" />
        </Form.Item>
      </Space>
      <Form.Item name="subject_kind" label="主体类型" rules={[{ required: true }]}>
        <Select options={subjectKindOptions} />
      </Form.Item>
      <Form.Item name="subject_name" label="主体名称" rules={[{ required: true, message: "请输入用户名/SA 名" }]}>
        <Input placeholder="alice 或 default/my-sa" />
      </Form.Item>
      <Form.Item name="subject_namespace" label="ServiceAccount 所在命名空间" extra="仅当主体为 ServiceAccount 时填写">
        <Input placeholder="default" />
      </Form.Item>
    </DrawerShellForm>
  );
}

export function RbacClusterRoleBindingFormCreateDrawer(props: {
  open: boolean;
  onClose: () => void;
  clusterId?: number;
  onSuccess: () => void;
  embedded?: boolean;
}) {
  const { open, onClose, clusterId, onSuccess, embedded } = props;
  const [form] = Form.useForm<{
    name: string;
    role_name: string;
    subject_kind: string;
    subject_name: string;
    subject_namespace: string;
  }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const sub: Record<string, unknown> = { kind: v.subject_kind, name: String(v.subject_name).trim() };
    if (v.subject_kind === "ServiceAccount" && String(v.subject_namespace || "").trim()) {
      sub.namespace = String(v.subject_namespace).trim();
    }
    const doc = {
      apiVersion: "rbac.authorization.k8s.io/v1",
      kind: "ClusterRoleBinding",
      metadata: { name: String(v.name).trim() },
      subjects: [sub],
      roleRef: {
        apiGroup: "rbac.authorization.k8s.io",
        kind: "ClusterRole",
        name: String(v.role_name).trim(),
      },
    };
    setLoading(true);
    try {
      await applyRbac(clusterId, YAML.stringify(doc));
      message.success("ClusterRoleBinding 已创建");
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
      title="表单创建 ClusterRoleBinding"
      open={embedded ? true : open}
      embedded={embedded}
      width={760}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{ subject_kind: "User" }}
    >
      <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="demo-crb" />
      </Form.Item>
      <Form.Item name="role_name" label="ClusterRole 名称" rules={[{ required: true, message: "请输入 ClusterRole 名" }]}>
        <Input />
      </Form.Item>
      <Form.Item name="subject_kind" label="主体类型" rules={[{ required: true }]}>
        <Select options={subjectKindOptions} />
      </Form.Item>
      <Form.Item name="subject_name" label="主体名称" rules={[{ required: true, message: "请输入" }]}>
        <Input />
      </Form.Item>
      <Form.Item name="subject_namespace" label="ServiceAccount 命名空间" extra="可选">
        <Input />
      </Form.Item>
    </DrawerShellForm>
  );
}
