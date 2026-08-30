import { Form, Input, Modal, Select } from "antd";
import type { FormInstance } from "antd/es/form";
import type { CicdServiceItem } from "../../services/cicd";
import type { UserItem } from "../../types/api";

type DictOption = { label: string; value: string | number };

type Props = {
  open: boolean;
  editingService: CicdServiceItem | null;
  form: FormInstance;
  pipelineTypes: DictOption[];
  userOptions: UserItem[];
  onCancel: () => void;
  onOk: () => void | Promise<void>;
};

export function ServiceFormDrawer({
  open,
  editingService,
  form,
  pipelineTypes,
  userOptions,
  onCancel,
  onOk,
}: Props) {
  return (
    <Modal
      title={editingService ? "编辑应用" : "新建应用"}
      open={open}
      onCancel={onCancel}
      onOk={() => void onOk()}
    >
      <Form form={form} layout="vertical">
        <Form.Item name="identifier" label="唯一标识符" rules={[{ required: true }]}>
          <Input placeholder="cityos-account" disabled={!!editingService} />
        </Form.Item>
        <Form.Item name="name" label="应用名称" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="service_type" label="应用类型" rules={[{ required: true }]}>
          <Select options={pipelineTypes.map((o) => ({ label: o.label, value: o.value }))} />
        </Form.Item>
        <Form.Item name="owner" label="Owner" extra="从平台用户列表选择，保存为用户名">
          <Select
            showSearch
            allowClear
            placeholder="选择负责人"
            optionFilterProp="label"
            options={userOptions.map((u) => ({
              value: u.username,
              label: `${u.nickname || u.username} (${u.username})`,
            }))}
          />
        </Form.Item>
        <Form.Item name="product_line" label="产品线">
          <Input />
        </Form.Item>
        <Form.Item name="jenkins_job" label="Jenkins Job 名" extra="留空则自动生成 cicd-p{projectId}-{identifier}">
          <Input />
        </Form.Item>
        <Form.Item name="remark" label="备注">
          <Input.TextArea rows={2} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
