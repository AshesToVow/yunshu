/**
 * Pod 创建/编辑表单抽屉（RF-04 拆分产物）
 * 从 pod-page.tsx 原地搬迁 JSX；提交逻辑仍由页面持有。
 */
import { Button, Card, Divider, Drawer, Form, Input, InputNumber, Select, Space, Tabs, Typography } from "antd";
import type { FormInstance } from "antd/es/form";
import { AiYamlGeneratePanel } from "../../components/k8s/ai-yaml-generate-panel";
import { MonacoYamlEditor } from "../../components/k8s/monaco-yaml-editor";
import type { PodItem } from "../../services/pods";
import { POD_CREATE_YAML_TEMPLATE } from "./pod-create-template";
import type { PodSimpleFormValues } from "./pod-form-payload";

export type PodFormDrawerProps = {
  createOpen: boolean;
  setCreateOpen: (v: boolean) => void;
  namespace: string;
  clusterId?: number;
  simpleMode: "create" | "edit";
  setSimpleMode: (v: "create" | "edit") => void;
  editTarget: PodItem | null;
  setEditTarget: (v: PodItem | null) => void;
  creating: boolean;
  simpleForm: FormInstance<PodSimpleFormValues>;
  yamlForm: FormInstance<{ manifest: string }>;
  rfc1123Subdomain: RegExp;
  rfc1123Label: RegExp;
  submitCreateSimple: () => void | Promise<void>;
  submitCreateYAML: () => void | Promise<void>;
};

export function PodFormDrawer({
  createOpen,
  setCreateOpen,
  namespace,
  clusterId,
  simpleMode,
  setSimpleMode,
  editTarget,
  setEditTarget,
  creating,
  simpleForm,
  yamlForm,
  rfc1123Subdomain,
  rfc1123Label,
  submitCreateSimple,
  submitCreateYAML,
}: PodFormDrawerProps) {
  return (
        <Drawer
          title={
            <Space direction="vertical" size={0}>
              <span>
                {simpleMode === "edit"
                  ? `编辑 Pod（重建） - ${editTarget?.namespace || namespace}/${editTarget?.name || ""}`
                  : "创建 Pod"}
              </span>
              <Typography.Text type="secondary" style={{ fontSize: 13, fontWeight: "normal" }}>
                目标命名空间：{namespace}
              </Typography.Text>
            </Space>
          }
          placement="right"
          width={960}
          open={createOpen}
          onClose={() => {
            setCreateOpen(false);
            setSimpleMode("create");
            setEditTarget(null);
          }}
          destroyOnClose
          maskClosable={false}
          styles={{ body: { paddingBottom: 24 } }}
          extra={
            <Button
              onClick={() => {
                setCreateOpen(false);
                setSimpleMode("create");
                setEditTarget(null);
              }}
            >
              取消
            </Button>
          }
        >
          <Tabs
            items={[
              {
                key: "simple",
                label: "表单创建",
                children: (
                  <Form
                    form={simpleForm}
                    layout="vertical"
                    requiredMark="optional"
                    scrollToFirstError
                    initialValues={{
                      name: "",
                      image: "nginx:latest",
                      command: "",
                      image_pull_policy: "IfNotPresent",
                      restart_policy: "Always",
                      env_pairs: [],
                      label_pairs: [],
                      node_selector_pairs: [],
                      tolerations: [],
                      priority_class_name: "",
                      affinity: {},
                    }}
                  >
                    <Form.Item
                      name="name"
                      label="Pod 名称"
                      rules={[
                        { required: true, message: "请输入 Pod 名称" },
                        {
                          validator: async (_, value) => {
                            const v = String(value || "").trim();
                            if (!v) return;
                            if (!rfc1123Subdomain.test(v)) {
                              throw new Error("Pod 名称不合法：必须全小写，且仅包含字母/数字/短横线/点，首尾为字母或数字");
                            }
                          },
                        },
                      ]}
                    >
                      <Input />
                    </Form.Item>
                    <Form.Item
                      name="container_name"
                      label="容器名称"
                      extra="默认同 Pod 名称"
                      rules={[
                        {
                          validator: async (_, value) => {
                            const v = String(value || "").trim();
                            if (!v) return;
                            if (!rfc1123Label.test(v)) {
                              throw new Error("容器名称不合法：必须全小写，且仅包含字母/数字/短横线，首尾为字母或数字");
                            }
                          },
                        },
                      ]}
                    >
                      <Input placeholder="默认同 Pod 名称" />
                    </Form.Item>
                    <Form.Item name="image" label="镜像" rules={[{ required: true, message: "请输入镜像" }]}>
                      <Input />
                    </Form.Item>
                    <Space style={{ width: "100%" }} size="middle">
                      <Form.Item name="image_pull_policy" label="镜像拉取策略" style={{ flex: 1 }}>
                        <Select
                          options={[
                            { label: "IfNotPresent", value: "IfNotPresent" },
                            { label: "Always", value: "Always" },
                            { label: "Never", value: "Never" },
                          ]}
                        />
                      </Form.Item>
                      <Form.Item name="restart_policy" label="重启策略" style={{ flex: 1 }}>
                        <Select
                          options={[
                            { label: "Always", value: "Always" },
                            { label: "OnFailure", value: "OnFailure" },
                            { label: "Never", value: "Never" },
                          ]}
                        />
                      </Form.Item>
                      <Form.Item name="port" label="容器端口" style={{ width: 140 }}>
                        <InputNumber min={1} max={65535} style={{ width: "100%" }} />
                      </Form.Item>
                    </Space>
                    <Form.Item name="command" label="启动命令" extra="例如覆盖镜像默认 CMD；留空则使用镜像入口">
                      <Input placeholder="例如：sleep 3600" />
                    </Form.Item>
                    <Space style={{ width: "100%" }} size="middle">
                      <Form.Item name="requests_cpu" label="CPU 请求" style={{ flex: 1 }}>
                        <Input placeholder="例如：100m" />
                      </Form.Item>
                      <Form.Item name="requests_memory" label="内存请求" style={{ flex: 1 }}>
                        <Input placeholder="例如：128Mi" />
                      </Form.Item>
                    </Space>
                    <Space style={{ width: "100%" }} size="middle">
                      <Form.Item name="limits_cpu" label="CPU 限制" style={{ flex: 1 }}>
                        <Input placeholder="例如：500m" />
                      </Form.Item>
                      <Form.Item name="limits_memory" label="内存限制" style={{ flex: 1 }}>
                        <Input placeholder="例如：512Mi" />
                      </Form.Item>
                    </Space>
                    <Form.List name="env_pairs">
                      {(fields, { add, remove }) => (
                        <Form.Item label="环境变量" extra="按键值对添加，KEY 不可重复">
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Space key={field.key} style={{ width: "100%" }} align="start">
                                <Form.Item
                                  {...field}
                                  name={[field.name, "key"]}
                                  rules={[
                                    { required: true, message: "请输入变量名" },
                                    {
                                      validator: async (_, value) => {
                                        const key = String(value || "").trim();
                                        if (!key) return;
                                        const list = simpleForm.getFieldValue("env_pairs") || [];
                                        const count = list.filter((it: { key?: string }) => String(it?.key || "").trim() === key).length;
                                        if (count > 1) throw new Error("变量名不能重复");
                                      },
                                    },
                                  ]}
                                  style={{ marginBottom: 0, flex: 1 }}
                                >
                                  <Input placeholder="KEY" />
                                </Form.Item>
                                <Form.Item
                                  {...field}
                                  name={[field.name, "value"]}
                                  style={{ marginBottom: 0, flex: 1 }}
                                >
                                  <Input placeholder="VALUE" />
                                </Form.Item>
                                <Button danger onClick={() => remove(field.name)}>
                                  删除
                                </Button>
                              </Space>
                            ))}
                            <Button type="dashed" onClick={() => add()}>
                              新增环境变量
                            </Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>
                    <Form.List name="node_selector_pairs">
                      {(fields, { add, remove }) => (
                        <Form.Item label="NodeSelector" extra="按键值对添加，用于节点选择">
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Space key={field.key} style={{ width: "100%" }} align="start">
                                <Form.Item
                                  {...field}
                                  name={[field.name, "key"]}
                                  rules={[{ required: true, message: "请输入选择器键" }]}
                                  style={{ marginBottom: 0, flex: 1 }}
                                >
                                  <Input placeholder="key" />
                                </Form.Item>
                                <Form.Item {...field} name={[field.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                  <Input placeholder="value" />
                                </Form.Item>
                                <Button danger onClick={() => remove(field.name)}>删除</Button>
                              </Space>
                            ))}
                            <Button type="dashed" onClick={() => add()}>新增 NodeSelector</Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>
                    <Form.Item name="priority_class_name" label="PriorityClassName">
                      <Input placeholder="例如：system-cluster-critical" />
                    </Form.Item>
                    <Divider style={{ margin: "8px 0" }} />
                    <Typography.Text strong>Affinity</Typography.Text>
                    <Typography.Paragraph className="inline-muted" style={{ margin: "6px 0 0" }}>
                      以表单方式配置 NodeAffinity / PodAffinity / PodAntiAffinity；未填写的块不会下发到 PodSpec。
                    </Typography.Paragraph>

                    <Card size="small" title="NodeAffinity（节点亲和）" style={{ marginTop: 10 }}>
                      <Form.List name={["affinity", "node", "required"]}>
                        {(fields, { add, remove }) => (
                          <Form.Item label="Required（必须满足）" style={{ marginBottom: 0 }}>
                            <Space direction="vertical" style={{ width: "100%" }}>
                              {fields.map((field) => (
                                <Card
                                  key={field.key}
                                  size="small"
                                  type="inner"
                                  title={`Term #${field.name + 1}`}
                                  extra={<Button danger onClick={() => remove(field.name)}>删除 Term</Button>}
                                >
                                  <Form.List name={[field.name, "match_expressions"]}>
                                    {(expFields, expOps) => (
                                      <Space direction="vertical" style={{ width: "100%" }}>
                                        {expFields.map((ef) => (
                                          <Space key={ef.key} style={{ width: "100%" }} align="start" wrap>
                                            <Form.Item
                                              {...ef}
                                              name={[ef.name, "key"]}
                                              rules={[{ required: true, message: "key 必填" }]}
                                              style={{ marginBottom: 0, width: 220 }}
                                            >
                                              <Input placeholder="key" />
                                            </Form.Item>
                                            <Form.Item
                                              {...ef}
                                              name={[ef.name, "operator"]}
                                              rules={[{ required: true, message: "operator 必填" }]}
                                              style={{ marginBottom: 0, width: 180 }}
                                            >
                                              <Select
                                                options={[
                                                  { label: "In", value: "In" },
                                                  { label: "NotIn", value: "NotIn" },
                                                  { label: "Exists", value: "Exists" },
                                                  { label: "DoesNotExist", value: "DoesNotExist" },
                                                  { label: "Gt", value: "Gt" },
                                                  { label: "Lt", value: "Lt" },
                                                ]}
                                              />
                                            </Form.Item>
                                            <Form.Item
                                              {...ef}
                                              name={[ef.name, "values"]}
                                              style={{ marginBottom: 0, width: 320 }}
                                              tooltip="In/NotIn 需要 values；Exists/DoesNotExist 可留空；Gt/Lt 建议填单个数字"
                                            >
                                              <Select mode="tags" placeholder="values" />
                                            </Form.Item>
                                            <Button danger onClick={() => expOps.remove(ef.name)}>删除</Button>
                                          </Space>
                                        ))}
                                        <Button type="dashed" onClick={() => expOps.add({ operator: "In", values: [] })}>
                                          新增 MatchExpression
                                        </Button>
                                      </Space>
                                    )}
                                  </Form.List>
                                </Card>
                              ))}
                              <Button type="dashed" onClick={() => add({ match_expressions: [] })}>
                                新增 Required Term
                              </Button>
                            </Space>
                          </Form.Item>
                        )}
                      </Form.List>

                      <Divider style={{ margin: "12px 0" }} />
                      <Form.List name={["affinity", "node", "preferred"]}>
                        {(fields, { add, remove }) => (
                          <Form.Item label="Preferred（尽量满足）" style={{ marginBottom: 0 }}>
                            <Space direction="vertical" style={{ width: "100%" }}>
                              {fields.map((field) => (
                                <Card
                                  key={field.key}
                                  size="small"
                                  type="inner"
                                  title={`Preference #${field.name + 1}`}
                                  extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}
                                >
                                  <Space style={{ width: "100%" }} align="start" wrap>
                                    <Form.Item
                                      {...field}
                                      name={[field.name, "weight"]}
                                      rules={[{ required: true, message: "weight 必填" }]}
                                      style={{ marginBottom: 0, width: 180 }}
                                    >
                                      <InputNumber min={1} max={100} style={{ width: "100%" }} placeholder="weight(1-100)" />
                                    </Form.Item>
                                  </Space>
                                  <Form.List name={[field.name, "match_expressions"]}>
                                    {(expFields, expOps) => (
                                      <Space direction="vertical" style={{ width: "100%", marginTop: 10 }}>
                                        {expFields.map((ef) => (
                                          <Space key={ef.key} style={{ width: "100%" }} align="start" wrap>
                                            <Form.Item
                                              {...ef}
                                              name={[ef.name, "key"]}
                                              rules={[{ required: true, message: "key 必填" }]}
                                              style={{ marginBottom: 0, width: 220 }}
                                            >
                                              <Input placeholder="key" />
                                            </Form.Item>
                                            <Form.Item
                                              {...ef}
                                              name={[ef.name, "operator"]}
                                              rules={[{ required: true, message: "operator 必填" }]}
                                              style={{ marginBottom: 0, width: 180 }}
                                            >
                                              <Select
                                                options={[
                                                  { label: "In", value: "In" },
                                                  { label: "NotIn", value: "NotIn" },
                                                  { label: "Exists", value: "Exists" },
                                                  { label: "DoesNotExist", value: "DoesNotExist" },
                                                  { label: "Gt", value: "Gt" },
                                                  { label: "Lt", value: "Lt" },
                                                ]}
                                              />
                                            </Form.Item>
                                            <Form.Item
                                              {...ef}
                                              name={[ef.name, "values"]}
                                              style={{ marginBottom: 0, width: 320 }}
                                            >
                                              <Select mode="tags" placeholder="values" />
                                            </Form.Item>
                                            <Button danger onClick={() => expOps.remove(ef.name)}>删除</Button>
                                          </Space>
                                        ))}
                                        <Button type="dashed" onClick={() => expOps.add({ operator: "In", values: [] })}>
                                          新增 MatchExpression
                                        </Button>
                                      </Space>
                                    )}
                                  </Form.List>
                                </Card>
                              ))}
                              <Button type="dashed" onClick={() => add({ weight: 50, match_expressions: [] })}>
                                新增 Preferred 规则
                              </Button>
                            </Space>
                          </Form.Item>
                        )}
                      </Form.List>
                    </Card>

                    <Card size="small" title="PodAffinity（Pod 亲和）" style={{ marginTop: 12 }}>
                      <Form.List name={["affinity", "pod", "required"]}>
                        {(fields, { add, remove }) => (
                          <Form.Item label="Required" style={{ marginBottom: 0 }}>
                            <Space direction="vertical" style={{ width: "100%" }}>
                              {fields.map((field) => (
                                <Card
                                  key={field.key}
                                  size="small"
                                  type="inner"
                                  title={`Term #${field.name + 1}`}
                                  extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}
                                >
                                  <Form.Item name={[field.name, "topology_key"]} rules={[{ required: true, message: "topologyKey 必填" }]}>
                                    <Input placeholder="topologyKey，例如：kubernetes.io/hostname" />
                                  </Form.Item>
                                  <Form.List name={[field.name, "match_labels"]}>
                                    {(kvFields, kvOps) => (
                                      <Space direction="vertical" style={{ width: "100%" }}>
                                        {kvFields.map((kv) => (
                                          <Space key={kv.key} style={{ width: "100%" }} align="start">
                                            <Form.Item {...kv} name={[kv.name, "key"]} rules={[{ required: true, message: "key 必填" }]} style={{ marginBottom: 0, flex: 1 }}>
                                              <Input placeholder="key" />
                                            </Form.Item>
                                            <Form.Item {...kv} name={[kv.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                              <Input placeholder="value" />
                                            </Form.Item>
                                            <Button danger onClick={() => kvOps.remove(kv.name)}>删除</Button>
                                          </Space>
                                        ))}
                                        <Button type="dashed" onClick={() => kvOps.add()}>新增 matchLabel</Button>
                                      </Space>
                                    )}
                                  </Form.List>
                                </Card>
                              ))}
                              <Button type="dashed" onClick={() => add({ match_labels: [] })}>新增 Required Term</Button>
                            </Space>
                          </Form.Item>
                        )}
                      </Form.List>

                      <Divider style={{ margin: "12px 0" }} />
                      <Form.List name={["affinity", "pod", "preferred"]}>
                        {(fields, { add, remove }) => (
                          <Form.Item label="Preferred" style={{ marginBottom: 0 }}>
                            <Space direction="vertical" style={{ width: "100%" }}>
                              {fields.map((field) => (
                                <Card
                                  key={field.key}
                                  size="small"
                                  type="inner"
                                  title={`Preferred #${field.name + 1}`}
                                  extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}
                                >
                                  <Space style={{ width: "100%" }} wrap>
                                    <Form.Item name={[field.name, "weight"]} rules={[{ required: true, message: "weight 必填" }]} style={{ width: 180 }}>
                                      <InputNumber min={1} max={100} style={{ width: "100%" }} placeholder="weight(1-100)" />
                                    </Form.Item>
                                    <Form.Item name={[field.name, "topology_key"]} rules={[{ required: true, message: "topologyKey 必填" }]} style={{ flex: 1 }}>
                                      <Input placeholder="topologyKey，例如：kubernetes.io/hostname" />
                                    </Form.Item>
                                  </Space>
                                  <Form.List name={[field.name, "match_labels"]}>
                                    {(kvFields, kvOps) => (
                                      <Space direction="vertical" style={{ width: "100%" }}>
                                        {kvFields.map((kv) => (
                                          <Space key={kv.key} style={{ width: "100%" }} align="start">
                                            <Form.Item {...kv} name={[kv.name, "key"]} rules={[{ required: true, message: "key 必填" }]} style={{ marginBottom: 0, flex: 1 }}>
                                              <Input placeholder="key" />
                                            </Form.Item>
                                            <Form.Item {...kv} name={[kv.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                              <Input placeholder="value" />
                                            </Form.Item>
                                            <Button danger onClick={() => kvOps.remove(kv.name)}>删除</Button>
                                          </Space>
                                        ))}
                                        <Button type="dashed" onClick={() => kvOps.add()}>新增 matchLabel</Button>
                                      </Space>
                                    )}
                                  </Form.List>
                                </Card>
                              ))}
                              <Button type="dashed" onClick={() => add({ weight: 50, match_labels: [] })}>新增 Preferred 规则</Button>
                            </Space>
                          </Form.Item>
                        )}
                      </Form.List>
                    </Card>

                    <Card size="small" title="PodAntiAffinity（Pod 反亲和）" style={{ marginTop: 12 }}>
                      <Form.List name={["affinity", "pod_anti", "required"]}>
                        {(fields, { add, remove }) => (
                          <Form.Item label="Required" style={{ marginBottom: 0 }}>
                            <Space direction="vertical" style={{ width: "100%" }}>
                              {fields.map((field) => (
                                <Card key={field.key} size="small" type="inner" title={`Term #${field.name + 1}`} extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}>
                                  <Form.Item name={[field.name, "topology_key"]} rules={[{ required: true, message: "topologyKey 必填" }]}>
                                    <Input placeholder="topologyKey，例如：kubernetes.io/hostname" />
                                  </Form.Item>
                                  <Form.List name={[field.name, "match_labels"]}>
                                    {(kvFields, kvOps) => (
                                      <Space direction="vertical" style={{ width: "100%" }}>
                                        {kvFields.map((kv) => (
                                          <Space key={kv.key} style={{ width: "100%" }} align="start">
                                            <Form.Item {...kv} name={[kv.name, "key"]} rules={[{ required: true, message: "key 必填" }]} style={{ marginBottom: 0, flex: 1 }}>
                                              <Input placeholder="key" />
                                            </Form.Item>
                                            <Form.Item {...kv} name={[kv.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                              <Input placeholder="value" />
                                            </Form.Item>
                                            <Button danger onClick={() => kvOps.remove(kv.name)}>删除</Button>
                                          </Space>
                                        ))}
                                        <Button type="dashed" onClick={() => kvOps.add()}>新增 matchLabel</Button>
                                      </Space>
                                    )}
                                  </Form.List>
                                </Card>
                              ))}
                              <Button type="dashed" onClick={() => add({ match_labels: [] })}>新增 Required Term</Button>
                            </Space>
                          </Form.Item>
                        )}
                      </Form.List>

                      <Divider style={{ margin: "12px 0" }} />
                      <Form.List name={["affinity", "pod_anti", "preferred"]}>
                        {(fields, { add, remove }) => (
                          <Form.Item label="Preferred" style={{ marginBottom: 0 }}>
                            <Space direction="vertical" style={{ width: "100%" }}>
                              {fields.map((field) => (
                                <Card key={field.key} size="small" type="inner" title={`Preferred #${field.name + 1}`} extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}>
                                  <Space style={{ width: "100%" }} wrap>
                                    <Form.Item name={[field.name, "weight"]} rules={[{ required: true, message: "weight 必填" }]} style={{ width: 180 }}>
                                      <InputNumber min={1} max={100} style={{ width: "100%" }} placeholder="weight(1-100)" />
                                    </Form.Item>
                                    <Form.Item name={[field.name, "topology_key"]} rules={[{ required: true, message: "topologyKey 必填" }]} style={{ flex: 1 }}>
                                      <Input placeholder="topologyKey，例如：kubernetes.io/hostname" />
                                    </Form.Item>
                                  </Space>
                                  <Form.List name={[field.name, "match_labels"]}>
                                    {(kvFields, kvOps) => (
                                      <Space direction="vertical" style={{ width: "100%" }}>
                                        {kvFields.map((kv) => (
                                          <Space key={kv.key} style={{ width: "100%" }} align="start">
                                            <Form.Item {...kv} name={[kv.name, "key"]} rules={[{ required: true, message: "key 必填" }]} style={{ marginBottom: 0, flex: 1 }}>
                                              <Input placeholder="key" />
                                            </Form.Item>
                                            <Form.Item {...kv} name={[kv.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                              <Input placeholder="value" />
                                            </Form.Item>
                                            <Button danger onClick={() => kvOps.remove(kv.name)}>删除</Button>
                                          </Space>
                                        ))}
                                        <Button type="dashed" onClick={() => kvOps.add()}>新增 matchLabel</Button>
                                      </Space>
                                    )}
                                  </Form.List>
                                </Card>
                              ))}
                              <Button type="dashed" onClick={() => add({ weight: 50, match_labels: [] })}>新增 Preferred 规则</Button>
                            </Space>
                          </Form.Item>
                        )}
                      </Form.List>
                    </Card>
                    <Form.List name="label_pairs">
                      {(fields, { add, remove }) => (
                        <Form.Item label="标签" extra="按键值对添加，key 不可重复">
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Space key={field.key} style={{ width: "100%" }} align="start">
                                <Form.Item
                                  {...field}
                                  name={[field.name, "key"]}
                                  rules={[
                                    { required: true, message: "请输入标签键" },
                                    {
                                      validator: async (_, value) => {
                                        const key = String(value || "").trim();
                                        if (!key) return;
                                        const list = simpleForm.getFieldValue("label_pairs") || [];
                                        const count = list.filter((it: { key?: string }) => String(it?.key || "").trim() === key).length;
                                        if (count > 1) throw new Error("标签键不能重复");
                                      },
                                    },
                                  ]}
                                  style={{ marginBottom: 0, flex: 1 }}
                                >
                                  <Input placeholder="key" />
                                </Form.Item>
                                <Form.Item
                                  {...field}
                                  name={[field.name, "value"]}
                                  style={{ marginBottom: 0, flex: 1 }}
                                >
                                  <Input placeholder="value" />
                                </Form.Item>
                                <Button danger onClick={() => remove(field.name)}>
                                  删除
                                </Button>
                              </Space>
                            ))}
                            <Button type="dashed" onClick={() => add()}>
                              新增标签
                            </Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>
                    <Form.List name="tolerations">
                      {(fields, { add, remove }) => (
                        <Form.Item label="容忍（Tolerations）" extra="用于匹配节点污点；污点(Taints)是节点配置，不在 Pod 内创建">
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Space key={field.key} style={{ width: "100%" }} align="start" wrap>
                                <Form.Item
                                  {...field}
                                  name={[field.name, "key"]}
                                  rules={[{ required: true, message: "请输入 key" }]}
                                  style={{ marginBottom: 0, width: 150 }}
                                >
                                  <Input placeholder="key" />
                                </Form.Item>
                                <Form.Item
                                  {...field}
                                  name={[field.name, "operator"]}
                                  initialValue="Equal"
                                  style={{ marginBottom: 0, width: 130 }}
                                >
                                  <Select
                                    options={[
                                      { label: "Equal", value: "Equal" },
                                      { label: "Exists", value: "Exists" },
                                    ]}
                                  />
                                </Form.Item>
                                <Form.Item
                                  {...field}
                                  name={[field.name, "value"]}
                                  style={{ marginBottom: 0, width: 160 }}
                                >
                                  <Input placeholder="value" />
                                </Form.Item>
                                <Form.Item
                                  {...field}
                                  name={[field.name, "effect"]}
                                  style={{ marginBottom: 0, width: 170 }}
                                >
                                  <Select
                                    allowClear
                                    placeholder="effect"
                                    options={[
                                      { label: "NoSchedule", value: "NoSchedule" },
                                      { label: "PreferNoSchedule", value: "PreferNoSchedule" },
                                      { label: "NoExecute", value: "NoExecute" },
                                    ]}
                                  />
                                </Form.Item>
                                <Form.Item
                                  {...field}
                                  name={[field.name, "toleration_seconds"]}
                                  style={{ marginBottom: 0, width: 160 }}
                                >
                                  <InputNumber min={1} style={{ width: "100%" }} placeholder="seconds" />
                                </Form.Item>
                                <Button danger onClick={() => remove(field.name)}>
                                  删除
                                </Button>
                              </Space>
                            ))}
                            <Button
                              type="dashed"
                              onClick={() =>
                                add({ operator: "Equal" })
                              }
                            >
                              新增容忍
                            </Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>
                    <Button type="primary" loading={creating} onClick={() => void submitCreateSimple()}>
                      {simpleMode === "edit" ? "保存并重建" : "创建"}
                    </Button>
                  </Form>
                ),
              },
              ...(simpleMode === "create"
                ? [
                    {
                      key: "yaml",
                      label: "YAML 创建",
                      children: (
                        <Form form={yamlForm} layout="vertical" requiredMark="optional" scrollToFirstError initialValues={{ manifest: "" }}>
                          <AiYamlGeneratePanel
                            resourceKind="Pod"
                            namespace={namespace}
                            clusterId={clusterId}
                            hintYaml={yamlForm.getFieldValue("manifest") || POD_CREATE_YAML_TEMPLATE}
                            onGenerated={(yaml) => yamlForm.setFieldsValue({ manifest: yaml })}
                          />
                          <Space wrap style={{ marginBottom: 8, marginTop: 8 }}>
                            <Button size="small" type="default" onClick={() => yamlForm.setFieldsValue({ manifest: POD_CREATE_YAML_TEMPLATE })}>
                              填入模板
                            </Button>
                            <Button size="small" onClick={() => yamlForm.setFieldsValue({ manifest: "" })}>
                              清空内容
                            </Button>
                          </Space>
                          <Form.Item name="manifest" label="YAML 内容" rules={[{ required: true, message: "请输入 YAML" }]}>
                            <MonacoYamlEditor height={420} />
                          </Form.Item>
                          <Button type="primary" loading={creating} onClick={() => void submitCreateYAML()}>
                            创建
                          </Button>
                        </Form>
                      ),
                    },
                  ]
                : []),
            ]}
          />
        </Drawer>
  );
}
