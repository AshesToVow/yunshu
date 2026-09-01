// @ts-nocheck
import { GiftOutlined } from "@ant-design/icons";
import { Button, Checkbox, Form, Select, Space, Typography } from "antd";
import type { FormInstance } from "antd";
import type { K8sCapabilityItem } from "../../services/k8s-policies";
import { PRESET_CAPS } from "./scoped-subject";

export type PresetFormValues = {
  cluster_ids: number[];
  preset: "readonly" | "readonly_exec" | "admin";
  capabilities: string[];
  deny_namespaces?: string[];
  allow_namespaces?: string[];
};

export type PresetFormProps = {
  form: FormInstance<PresetFormValues>;
  watchedCapabilities: string[];
  capCatalog: K8sCapabilityItem[];
  clusterOptions: { id: number; name: string }[];
  presetClusterIds: number[];
  presetNsLoading: boolean;
  presetNsOptions: { label: string; value: string }[];
  presetSubmitting: boolean;
  activeSubjectReady: boolean;
  onSave: () => void;
  onOpenSplit: () => void;
};

export function PresetForm({
  form,
  watchedCapabilities,
  capCatalog,
  clusterOptions,
  presetClusterIds,
  presetNsLoading,
  presetNsOptions,
  presetSubmitting,
  activeSubjectReady,
  onSave,
  onOpenSplit,
}: PresetFormProps) {
  return (
    <Form
      form={form}
      layout="vertical"
      initialValues={{
        cluster_ids: [],
        preset: "readonly" as const,
        capabilities: PRESET_CAPS.readonly,
        deny_namespaces: [],
        allow_namespaces: [],
      }}
      style={{ maxWidth: 960 }}
    >
      <Space wrap style={{ width: "100%", alignItems: "flex-start" }}>
        <Form.Item label="快捷档位" name="preset" style={{ minWidth: 240 }}>
          <Select
            style={{ minWidth: 220 }}
            options={[
              { value: "readonly", label: "只读（控制台资源 GET）" },
              { value: "readonly_exec", label: "只读 + Pod Exec" },
              { value: "admin", label: "集群管理（全部能力）" },
            ]}
            onChange={(v: "readonly" | "readonly_exec" | "admin") => {
              form.setFieldsValue({ capabilities: PRESET_CAPS[v] });
            }}
          />
        </Form.Item>
        <Form.Item label="集群" name="cluster_ids" style={{ minWidth: 260 }}>
          <Select
            mode="multiple"
            allowClear
            style={{ minWidth: 260 }}
            placeholder="不选 = 全部集群"
            options={clusterOptions.map((c) => ({ label: c.name, value: c.id }))}
          />
        </Form.Item>
        <Form.Item
          label="同步命名空间黑名单（可选）"
          name="deny_namespaces"
          tooltip="须在「集群」中选择至少一个具体集群；命名空间列表为所选集群的合并结果（同名去重）；保存时对每个所选集群写入禁止规则"
        >
          <Select
            mode="multiple"
            allowClear
            showSearch
            optionFilterProp="label"
            loading={presetNsLoading}
            disabled={presetClusterIds.length === 0}
            style={{ minWidth: 320 }}
            placeholder={
              presetClusterIds.length > 0
                ? "从下拉选择命名空间（可多选）"
                : "请先在「集群」中选择至少一个集群以加载列表"
            }
            options={presetNsOptions}
          />
        </Form.Item>
        <Form.Item
          label="同步命名空间白名单（可选）"
          name="allow_namespaces"
          tooltip="须选择至少一个具体集群；写入后该主体在各所选集群仅允许访问所列命名空间（黑名单优先）；列表为所选集群命名空间合并去重"
        >
          <Select
            mode="multiple"
            allowClear
            showSearch
            optionFilterProp="label"
            loading={presetNsLoading}
            disabled={presetClusterIds.length === 0}
            style={{ minWidth: 320 }}
            placeholder={
              presetClusterIds.length > 0
                ? "从下拉选择命名空间（可多选）"
                : "请先在「集群」中选择至少一个集群以加载列表"
            }
            options={presetNsOptions}
          />
        </Form.Item>
      </Space>
      <Form.Item
        label="能力包（勾选）"
        name="capabilities"
        rules={[{ required: true, type: "array", min: 1, message: "请至少勾选一项能力" }]}
        extra={
          watchedCapabilities.length
            ? `已选 ${watchedCapabilities.length} 项`
            : "请勾选能力，或先选快捷档位自动填充"
        }
      >
        <Checkbox.Group style={{ width: "100%" }}>
          <Space direction="vertical" size={8} style={{ width: "100%" }}>
            {(capCatalog.length
              ? capCatalog
              : [
                  { code: "read", name: "只读浏览", description: "" },
                  { code: "exec", name: "Pod 终端", description: "" },
                  { code: "restart", name: "重启", description: "" },
                  { code: "scale", name: "扩缩容", description: "" },
                  { code: "apply", name: "YAML 变更", description: "" },
                  { code: "delete", name: "删除资源", description: "" },
                  { code: "secret_reveal", name: "Secret 明文", description: "" },
                  { code: "destructive", name: "高危运维", description: "" },
                ]
            ).map((c) => (
              <Checkbox key={c.code} value={c.code} disabled={c.code === "read"}>
                <Space direction="vertical" size={0}>
                  <span>{c.name}</span>
                  {c.description ? (
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      {c.description}
                    </Typography.Text>
                  ) : null}
                </Space>
              </Checkbox>
            ))}
          </Space>
        </Checkbox.Group>
      </Form.Item>
      <Form.Item>
        <Space>
          <Button
            type="primary"
            ghost
            icon={<GiftOutlined />}
            loading={presetSubmitting}
            onClick={onSave}
          >
            保存能力包
          </Button>
          <Button onClick={onOpenSplit} disabled={!activeSubjectReady}>
            按 NS 拆分档位
          </Button>
        </Space>
      </Form.Item>
    </Form>
  );
}
