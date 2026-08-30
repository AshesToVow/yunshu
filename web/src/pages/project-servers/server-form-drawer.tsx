import { Button, Card, Form, Input, InputNumber, Modal, Select, Space, Typography } from "antd";
import type { FormInstance } from "antd/es/form";
import { DictLabelFillSelect, type DictFillOption } from "../../components/dict-fill-select";
import type { ServerItem, ServerUpsertPayload } from "../../services/projects";
import { CLOUD_PROVIDER_LABEL, type CloudTagKV } from "../../utils/cloud-server-display";

export type ServerFormValues = ServerUpsertPayload & {
  port_dict_label?: string;
  private_key_dict_label?: string;
};

export type ServerFormOptions = {
  serverOsOptions: Array<{ label: string; value: string | number }>;
  serverAuthOptions: Array<{ label: string; value: string | number }>;
  activePortDict: DictFillOption[];
  activeUserDict: DictFillOption[];
  activePwdDict: DictFillOption[];
  activeKeyDict: DictFillOption[];
};

type ServerFormDrawerProps = {
  open: boolean;
  submitting: boolean;
  current: ServerItem | null;
  form: FormInstance<ServerFormValues>;
  cloudTagRows: CloudTagKV[];
  onCloudTagRowsChange: React.Dispatch<React.SetStateAction<CloudTagKV[]>>;
  options: ServerFormOptions;
  onCancel: () => void;
  onSubmit: () => void;
};

export function ServerFormDrawer({
  open,
  submitting,
  current,
  form,
  cloudTagRows,
  onCloudTagRowsChange,
  options,
  onCancel,
  onSubmit,
}: ServerFormDrawerProps) {
  const { serverOsOptions, serverAuthOptions, activePortDict, activeUserDict, activePwdDict, activeKeyDict } = options;

  return (
    <Modal
      title={current ? "编辑服务器" : "新增服务器"}
      open={open}
      onCancel={onCancel}
      onOk={onSubmit}
      confirmLoading={submitting}
      destroyOnClose
      width={720}
    >
      <Form layout="vertical" form={form} autoComplete="off">
        <Form.Item name="id" hidden>
          <Input />
        </Form.Item>
        <Form.Item name="project_id" hidden>
          <Input />
        </Form.Item>
        <Form.Item name="group_id" hidden>
          <Input />
        </Form.Item>
        <Form.Item name="status" hidden>
          <InputNumber />
        </Form.Item>
        <Form.Item name="source_type" hidden>
          <Input />
        </Form.Item>
        <Form.Item name="provider" hidden>
          <Input />
        </Form.Item>
        <Form.Item label="名称" name="name" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Space style={{ width: "100%" }} size={16} align="start">
          <Form.Item label="Host" name="host" rules={[{ required: true }]} style={{ flex: 1 }}>
            <Input />
          </Form.Item>
          <Form.Item label="Port" name="port" style={{ width: 160 }}>
            <InputNumber min={1} max={65535} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="OS" name="os_type" style={{ width: 160 }}>
            <Select options={serverOsOptions} />
          </Form.Item>
        </Space>
        <Form.Item
          label="从数据字典填端口"
          name="port_dict_label"
          extra="按标签选择后自动写入上方 Port；手动修改 Port 不受影响。"
        >
          <DictLabelFillSelect
            form={form}
            labelFieldName="port_dict_label"
            targetFieldName="port"
            options={activePortDict}
            placeholder="选择字典中的端口模板"
          />
        </Form.Item>
        <Form.Item label="区域" name="cloud_region">
          <Input placeholder="例如：cn-hangzhou / 华东-1 / IDC-A" />
        </Form.Item>
        <Form.Item label="Tags（逗号分隔）" name="tags">
          <Input />
        </Form.Item>
        <Form.Item noStyle shouldUpdate={(prev, next) => prev.source_type !== next.source_type || prev.provider !== next.provider}>
          {({ getFieldValue }) => {
            const sourceType = String(getFieldValue("source_type") || "");
            const provider = String(getFieldValue("provider") || "");
            if (sourceType !== "cloud" || !["tencent", "alibaba", "jd"].includes(provider)) return null;
            return (
              <Card size="small" title={`云厂商标签（${CLOUD_PROVIDER_LABEL[provider] || provider || "-"}）`}>
                <Space direction="vertical" style={{ width: "100%" }} size={8}>
                  {cloudTagRows.map((row, idx) => (
                    <Space key={`${idx}-${row.key}`} style={{ width: "100%" }} size={8}>
                      <Input
                        placeholder="标签键（Tag Key）"
                        value={row.key}
                        onChange={(e) =>
                          onCloudTagRowsChange((prev) => prev.map((it, i) => (i === idx ? { ...it, key: e.target.value } : it)))
                        }
                      />
                      <Input
                        placeholder="标签值（Tag Value）"
                        value={row.value}
                        onChange={(e) =>
                          onCloudTagRowsChange((prev) => prev.map((it, i) => (i === idx ? { ...it, value: e.target.value } : it)))
                        }
                      />
                      <Button danger onClick={() => onCloudTagRowsChange((prev) => prev.filter((_, i) => i !== idx))}>
                        删除
                      </Button>
                    </Space>
                  ))}
                  <Space>
                    <Button onClick={() => onCloudTagRowsChange((prev) => [...prev, { key: "", value: "" }])}>新增标签</Button>
                    <Typography.Text type="secondary">保存后会回写腾讯云标签并同步到本地。</Typography.Text>
                  </Space>
                </Space>
              </Card>
            );
          }}
        </Form.Item>
        <Card size="small" title="SSH 凭据（可选）">
          <Space style={{ width: "100%" }} size={16} align="start">
            <Form.Item label="认证方式" name="auth_type" style={{ width: 180 }}>
              <Select options={serverAuthOptions} />
            </Form.Item>
          </Space>
          <Form.Item
            label="从数据字典填用户名"
            name="username_dict_label"
            extra="按标签选择，避免下拉展示过长内容；选后写入「用户名」。手改用户名会清空此处。"
          >
            <DictLabelFillSelect
              form={form}
              labelFieldName="username_dict_label"
              targetFieldName="username"
              options={activeUserDict}
              placeholder="选择字典中的用户名模板"
            />
          </Form.Item>
          <Form.Item label="用户名" name="username">
            <Input autoComplete="off" />
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(a, b) => a.auth_type !== b.auth_type}>
            {({ getFieldValue }) =>
              getFieldValue("auth_type") === "key" ? (
                <>
                  <Form.Item
                    label="从数据字典填私钥"
                    name="private_key_dict_label"
                    extra="按标签选择；选后写入下方私钥框。手改私钥会清空此处。"
                  >
                    <DictLabelFillSelect
                      form={form}
                      labelFieldName="private_key_dict_label"
                      targetFieldName="private_key"
                      options={activeKeyDict}
                      placeholder="选择字典中的私钥模板"
                    />
                  </Form.Item>
                  <Form.Item label="私钥（PEM）" name="private_key">
                    <Input.TextArea rows={6} />
                  </Form.Item>
                  <Form.Item label="私钥口令（可选）" name="passphrase">
                    <Input.Password />
                  </Form.Item>
                </>
              ) : (
                <>
                  <Form.Item
                    label="从数据字典填密码"
                    name="password_dict_label"
                    extra="按标签选择；选后写入下方密码框。编辑时密码可留空以保留原密码；手改密码会清空此处。"
                  >
                    <DictLabelFillSelect
                      form={form}
                      labelFieldName="password_dict_label"
                      targetFieldName="password"
                      options={activePwdDict}
                      placeholder="选择字典中的密码模板"
                    />
                  </Form.Item>
                  <Form.Item label="密码" name="password">
                    <Input.Password placeholder={current ? "留空表示保留原密码" : undefined} autoComplete="new-password" />
                  </Form.Item>
                </>
              )
            }
          </Form.Item>
        </Card>
      </Form>
    </Modal>
  );
}
