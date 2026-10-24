import { GiftOutlined } from "@ant-design/icons";
import { Button, Form, Select, Space } from "antd";
import type { FormInstance } from "antd";

export type PresetFormValues = {
  cluster_ids: number[];
  preset: "readonly" | "readonly_exec" | "admin";
  deny_namespaces?: string[];
  allow_namespaces?: string[];
};

export type PresetFormProps = {
  form: FormInstance<PresetFormValues>;
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
              { value: "readonly", label: "集群只读（对齐 k8m cluster_readonly）" },
              { value: "readonly_exec", label: "只读 + Pod Exec（两行角色）" },
              { value: "admin", label: "集群管理员（含重启/删除/Apply）" },
            ]}
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
          label="命名空间黑名单（可选）"
          name="deny_namespaces"
          tooltip="写入授权行 BlacklistNamespaces，优先于白名单；不选集群时也可保存（对全部集群通配行生效）"
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
          label="命名空间白名单（可选）"
          name="allow_namespaces"
          tooltip="空=该角色不限制命名空间；非空则仅允许所列 NS（对齐 k8m ClusterUserRole.Namespaces）"
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
      <Form.Item>
        <Space>
          <Button type="primary" ghost icon={<GiftOutlined />} loading={presetSubmitting} onClick={onSave}>
            保存集群角色
          </Button>
          <Button onClick={onOpenSplit} disabled={!activeSubjectReady}>
            按 NS 拆分档位
          </Button>
        </Space>
      </Form.Item>
    </Form>
  );
}
