// @ts-nocheck
/**
 * Ingress「表单创建」抽屉。
 *
 * 从 k8s-resource-form-drawers.tsx 原地搬迁（RF-11 第二步），仅移动不改语义。
 * 单独成文件：本抽屉含 rules/paths/tls 三段嵌套 Form.List，体量约 215 行。
 */

import { Button, Form, Input, InputNumber, Select, Space, Typography, message } from "antd";
import { useState } from "react";
import YAML from "yaml";
import { applyIngress } from "../../../services/ingresses";
import { DrawerShellForm } from "./drawer-shell-form";
import { pathTypes } from "./options";

export function IngressFormCreateDrawer(props: {
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
    ingress_class: string;
    annotations?: { key: string; value: string }[];
    tls?: { hosts?: string[]; secretName?: string }[];
    rules?: Array<{
      host: string;
      paths?: Array<{
        path?: string;
        path_type?: string;
        service_name?: string;
        service_port_name?: string;
        service_port_number?: number;
      }>;
    }>;
  }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const annotations = Object.fromEntries(
      (v.annotations ?? [])
        .map((p) => [String(p?.key ?? "").trim(), String(p?.value ?? "")] as const)
        .filter(([k]) => !!k),
    );
    const tls =
      (v.tls ?? [])
        .map((t) => ({
          hosts: (t.hosts ?? []).map((x) => String(x).trim()).filter(Boolean),
          secretName: String(t.secretName ?? "").trim() || undefined,
        }))
        .filter((t) => t.hosts.length || t.secretName) || undefined;
    const rules =
      (v.rules ?? [])
        .map((r) => ({
          host: String(r.host ?? "").trim(),
          http: {
            paths:
              (r.paths ?? [])
                .map((p) => {
                  const serviceName = String(p.service_name ?? "").trim();
                  const portName = String(p.service_port_name ?? "").trim();
                  const portNumber = typeof p.service_port_number === "number" ? p.service_port_number : undefined;
                  if (!serviceName) return null;
                  return {
                    path: String(p.path ?? "/").trim() || "/",
                    pathType: p.path_type || "Prefix",
                    backend: {
                      service: {
                        name: serviceName,
                        port: portName ? { name: portName } : { number: portNumber || 80 },
                      },
                    },
                  };
                })
                .filter(Boolean),
          },
        }))
        .filter((r) => r.host && r.http.paths.length) || undefined;
    const doc = {
      apiVersion: "networking.k8s.io/v1",
      kind: "Ingress",
      metadata: {
        name: String(v.name).trim(),
        namespace,
        ...(Object.keys(annotations).length ? { annotations } : {}),
      },
      spec: {
        ...(String(v.ingress_class || "").trim() ? { ingressClassName: String(v.ingress_class).trim() } : {}),
        rules,
        tls,
      },
    };
    setLoading(true);
    try {
      await applyIngress(clusterId, YAML.stringify(doc));
      message.success("Ingress 已创建");
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
      title="表单创建 Ingress"
      open={embedded ? true : open}
      embedded={embedded}
      width={760}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{
        annotations: [{ key: "nginx.ingress.kubernetes.io/rewrite-target", value: "/" }],
        tls: [{ hosts: [], secretName: "" }],
        rules: [{ host: "", paths: [{ path: "/", path_type: "Prefix", service_name: "", service_port_number: 80 }] }],
      }}
    >
      <Form.Item label="目标命名空间">
        <Input value={namespace} readOnly />
      </Form.Item>
      <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="demo-ingress" />
      </Form.Item>
      <Form.Item name="ingress_class" label="IngressClass 名称" extra="与集群 Ingress Controller 一致，如 nginx；可选">
        <Input placeholder="nginx" />
      </Form.Item>
      <Typography.Text type="secondary">Ingress 注解（可选）</Typography.Text>
      <Form.List name="annotations">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}>
                  <Input placeholder="annotation key" style={{ width: 280 }} />
                </Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}>
                  <Input placeholder="annotation value" style={{ width: 320 }} />
                </Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加注解</Button>
          </Space>
        )}
      </Form.List>
      <Typography.Text type="secondary" style={{ marginTop: 12, display: "block" }}>
        TLS（可选）
      </Typography.Text>
      <Form.List name="tls">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline" style={{ width: "100%" }}>
                <Form.Item name={[f.name, "hosts"]} label="Hosts" style={{ flex: 1 }}>
                  <Select mode="tags" tokenSeparators={[",", " "]} placeholder="example.com,api.example.com" />
                </Form.Item>
                <Form.Item name={[f.name, "secretName"]} label="SecretName" style={{ flex: 1 }}>
                  <Input placeholder="tls-secret" />
                </Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ hosts: [], secretName: "" })}>添加 TLS</Button>
          </Space>
        )}
      </Form.List>
      <Typography.Text type="secondary" style={{ marginTop: 12, display: "block" }}>
        Rules
      </Typography.Text>
      <Form.List name="rules">
        {(ruleFields, { add: addRule, remove: removeRule }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {ruleFields.map((rf) => (
              <div key={rf.key} style={{ border: "1px solid #f0f0f0", padding: 12, borderRadius: 8 }}>
                <Space style={{ width: "100%" }} align="start">
                  <Form.Item name={[rf.name, "host"]} label="域名 (Host)" rules={[{ required: true, message: "请输入域名" }]} style={{ flex: 1 }}>
                    <Input placeholder="app.example.com" />
                  </Form.Item>
                  <Button onClick={() => removeRule(rf.name)}>删除 Rule</Button>
                </Space>
                <Form.List name={[rf.name, "paths"]}>
                  {(pathFields, { add: addPath, remove: removePath }) => (
                    <Space direction="vertical" style={{ width: "100%" }}>
                      {pathFields.map((pf) => (
                        <div key={pf.key} style={{ border: "1px dashed #d9d9d9", padding: 10, borderRadius: 6 }}>
                          <Space style={{ width: "100%" }} align="start">
                            <Form.Item name={[pf.name, "path"]} label="路径" style={{ flex: 1 }}>
                              <Input placeholder="/" />
                            </Form.Item>
                            <Form.Item name={[pf.name, "path_type"]} label="路径匹配" style={{ width: 220 }}>
                              <Select options={pathTypes} />
                            </Form.Item>
                            <Button onClick={() => removePath(pf.name)}>删除 Path</Button>
                          </Space>
                          <Space style={{ width: "100%" }} align="start">
                            <Form.Item name={[pf.name, "service_name"]} label="后端 Service 名称" rules={[{ required: true, message: "请输入 Service" }]} style={{ flex: 1 }}>
                              <Input placeholder="my-service" />
                            </Form.Item>
                            <Form.Item name={[pf.name, "service_port_name"]} label="Service 端口名" style={{ width: 220 }}>
                              <Input placeholder="http" />
                            </Form.Item>
                            <Form.Item name={[pf.name, "service_port_number"]} label="Service 端口号" style={{ width: 220 }}>
                              <InputNumber min={1} max={65535} style={{ width: "100%" }} />
                            </Form.Item>
                          </Space>
                        </div>
                      ))}
                      <Button onClick={() => addPath({ path: "/", path_type: "Prefix", service_name: "", service_port_number: 80 })}>添加 Path</Button>
                    </Space>
                  )}
                </Form.List>
              </div>
            ))}
            <Button onClick={() => addRule({ host: "", paths: [{ path: "/", path_type: "Prefix", service_name: "", service_port_number: 80 }] })}>添加 Rule</Button>
          </Space>
        )}
      </Form.List>
    </DrawerShellForm>
  );
}
