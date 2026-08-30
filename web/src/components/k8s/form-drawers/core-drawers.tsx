/**
 * 核心资源「表单创建」抽屉：Namespace / ConfigMap / Secret。
 *
 * 从 k8s-resource-form-drawers.tsx 原地搬迁（RF-11 第二步），仅移动不改语义：
 * 组件签名、props 契约、提交逻辑与 YAML 文档结构均未变动。
 * 对外导入路径仍为 ../k8s-resource-form-drawers（该文件已退化为桶文件）。
 */

import { Button, Form, Input, Select, Space, Switch, Typography, message } from "antd";
import { useState } from "react";
import YAML from "yaml";
import { extractApiErrorMessage } from "../../../services/http";
import { applyNamespace } from "../../../services/namespaces";
import { applyConfigMap, applySecret } from "../../../services/configs";
import { DrawerShellForm } from "./drawer-shell-form";
import { secretTypes } from "./options";

export function NamespaceFormCreateDrawer(props: {
  open: boolean;
  onClose: () => void;
  clusterId?: number;
  onSuccess: () => void;
  embedded?: boolean;
}) {
  const { open, onClose, clusterId, onSuccess, embedded } = props;
  const [form] = Form.useForm<{ name: string; label_pairs: { key: string; value: string }[] }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const name = String(v.name || "").trim();
    const labels: Record<string, string> = {};
    for (const p of v.label_pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      labels[k] = String(p?.value ?? "");
    }
    const doc: Record<string, unknown> = {
      apiVersion: "v1",
      kind: "Namespace",
      metadata: { name, ...(Object.keys(labels).length ? { labels } : {}) },
    };
    setLoading(true);
    try {
      await applyNamespace(clusterId, YAML.stringify(doc), { failIfExists: true, silentErrorToast: true });
      message.success("命名空间已创建");
      onSuccess();
      onClose();
    } catch (e: unknown) {
      message.error(extractApiErrorMessage(e, "创建失败"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <DrawerShellForm
      title="表单创建 Namespace"
      open={embedded ? true : open}
      embedded={embedded}
      width={640}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{ label_pairs: [{ key: "", value: "" }] }}
    >
      <Form.Item name="name" label="命名空间名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="例如：team-demo" />
      </Form.Item>
      <Typography.Text type="secondary">标签（可选）</Typography.Text>
      <Form.List name="label_pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}>
                  <Input placeholder="键" style={{ width: 160 }} />
                </Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}>
                  <Input placeholder="值" style={{ width: 220 }} />
                </Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加标签</Button>
          </Space>
        )}
      </Form.List>
    </DrawerShellForm>
  );
}

export function ConfigMapFormCreateDrawer(props: {
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
    immutable?: boolean;
    label_pairs?: { key: string; value: string }[];
    annotation_pairs?: { key: string; value: string }[];
    pairs: { key: string; value: string }[];
  }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const data: Record<string, string> = {};
    const labels: Record<string, string> = {};
    const annotations: Record<string, string> = {};
    for (const p of v.pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      data[k] = String(p?.value ?? "");
    }
    for (const p of v.label_pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      labels[k] = String(p?.value ?? "");
    }
    for (const p of v.annotation_pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      annotations[k] = String(p?.value ?? "");
    }
    if (Object.keys(data).length === 0) {
      message.warning("请至少填写一组配置键值");
      return;
    }
    const doc = {
      apiVersion: "v1",
      kind: "ConfigMap",
      metadata: {
        name: String(v.name).trim(),
        namespace,
        ...(Object.keys(labels).length ? { labels } : {}),
        ...(Object.keys(annotations).length ? { annotations } : {}),
      },
      immutable: typeof v.immutable === "boolean" ? v.immutable : undefined,
      data,
    };
    setLoading(true);
    try {
      await applyConfigMap(clusterId, YAML.stringify(doc));
      message.success("ConfigMap 已创建");
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
      title="表单创建 ConfigMap"
      open={embedded ? true : open}
      embedded={embedded}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{ pairs: [{ key: "", value: "" }], label_pairs: [{ key: "", value: "" }], annotation_pairs: [{ key: "", value: "" }] }}
    >
      <Form.Item label="目标命名空间">
        <Input value={namespace} readOnly />
      </Form.Item>
      <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="app-config" />
      </Form.Item>
      <Form.Item name="immutable" label="Immutable" valuePropName="checked">
        <Switch />
      </Form.Item>
      <Typography.Text type="secondary">Labels（可选）</Typography.Text>
      <Form.List name="label_pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}><Input placeholder="label key" style={{ width: 200 }} /></Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}><Input placeholder="label value" style={{ width: 220 }} /></Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加 Label</Button>
          </Space>
        )}
      </Form.List>
      <Typography.Text type="secondary">Annotations（可选）</Typography.Text>
      <Form.List name="annotation_pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}><Input placeholder="annotation key" style={{ width: 200 }} /></Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}><Input placeholder="annotation value" style={{ width: 220 }} /></Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加 Annotation</Button>
          </Space>
        )}
      </Form.List>
      <Typography.Text type="secondary">数据键值（至少一组，值可为多行）</Typography.Text>
      <Form.List name="pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="start" style={{ width: "100%" }}>
                <Form.Item name={[f.name, "key"]} rules={[{ required: true, message: "键必填" }]} style={{ marginBottom: 0 }}>
                  <Input placeholder="键" style={{ width: 200 }} />
                </Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ flex: 1, marginBottom: 0, minWidth: 200 }}>
                  <Input.TextArea placeholder="值" rows={2} style={{ minHeight: 48 }} />
                </Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加条目</Button>
          </Space>
        )}
      </Form.List>
    </DrawerShellForm>
  );
}

export function SecretFormCreateDrawer(props: {
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
    type: string;
    data_mode?: "stringData" | "data";
    immutable?: boolean;
    label_pairs?: { key: string; value: string }[];
    annotation_pairs?: { key: string; value: string }[];
    pairs: { key: string; value: string }[];
  }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const stringData: Record<string, string> = {};
    const data: Record<string, string> = {};
    const labels: Record<string, string> = {};
    const annotations: Record<string, string> = {};
    for (const p of v.pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      if ((v.data_mode || "stringData") === "data") data[k] = String(p?.value ?? "");
      else stringData[k] = String(p?.value ?? "");
    }
    for (const p of v.label_pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      labels[k] = String(p?.value ?? "");
    }
    for (const p of v.annotation_pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      annotations[k] = String(p?.value ?? "");
    }
    if (Object.keys(stringData).length === 0 && Object.keys(data).length === 0) {
      message.warning("请至少填写一组键值");
      return;
    }
    const doc = {
      apiVersion: "v1",
      kind: "Secret",
      metadata: {
        name: String(v.name).trim(),
        namespace,
        ...(Object.keys(labels).length ? { labels } : {}),
        ...(Object.keys(annotations).length ? { annotations } : {}),
      },
      type: v.type || "Opaque",
      immutable: typeof v.immutable === "boolean" ? v.immutable : undefined,
      ...(Object.keys(stringData).length ? { stringData } : {}),
      ...(Object.keys(data).length ? { data } : {}),
    };
    setLoading(true);
    try {
      await applySecret(clusterId, YAML.stringify(doc));
      message.success("Secret 已创建");
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
      title="表单创建 Secret"
      open={embedded ? true : open}
      embedded={embedded}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{
        type: "Opaque",
        data_mode: "stringData",
        pairs: [{ key: "", value: "" }],
        label_pairs: [{ key: "", value: "" }],
        annotation_pairs: [{ key: "", value: "" }],
      }}
    >
      <Form.Item label="目标命名空间">
        <Input value={namespace} readOnly />
      </Form.Item>
      <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="app-secret" />
      </Form.Item>
      <Form.Item name="type" label="类型" rules={[{ required: true, message: "请选择类型" }]}>
        <Select options={secretTypes} showSearch optionFilterProp="label" />
      </Form.Item>
      <Form.Item name="data_mode" label="数据模式" rules={[{ required: true }]}>
        <Select options={[{ label: "stringData（明文输入）", value: "stringData" }, { label: "data（base64）", value: "data" }]} />
      </Form.Item>
      <Form.Item name="immutable" label="Immutable" valuePropName="checked">
        <Switch />
      </Form.Item>
      <Typography.Text type="secondary">Labels（可选）</Typography.Text>
      <Form.List name="label_pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}><Input placeholder="label key" style={{ width: 200 }} /></Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}><Input placeholder="label value" style={{ width: 220 }} /></Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加 Label</Button>
          </Space>
        )}
      </Form.List>
      <Typography.Text type="secondary">Annotations（可选）</Typography.Text>
      <Form.List name="annotation_pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}><Input placeholder="annotation key" style={{ width: 200 }} /></Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}><Input placeholder="annotation value" style={{ width: 220 }} /></Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加 Annotation</Button>
          </Space>
        )}
      </Form.List>
      <Typography.Text type="secondary">Secret 键值（至少一组）</Typography.Text>
      <Form.List name="pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="start" style={{ width: "100%" }}>
                <Form.Item name={[f.name, "key"]} rules={[{ required: true, message: "键必填" }]} style={{ marginBottom: 0 }}>
                  <Input placeholder="键" style={{ width: 200 }} />
                </Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ flex: 1, marginBottom: 0 }}>
                  <Input.Password placeholder="值（敏感）" style={{ minWidth: 200 }} />
                </Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加条目</Button>
          </Space>
        )}
      </Form.List>
    </DrawerShellForm>
  );
}
