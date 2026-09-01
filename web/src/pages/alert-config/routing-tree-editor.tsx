/**
 * 全局路由树编辑器（RF-07 拆分产物）
 * 从 alert-config-center-panel.tsx 原地搬迁。
 * GLOBAL_ROUTING_PROJECT_ID 仅在此定义一处，主面板与 Tab 均从本文件导入。
 */
import { Card, Form, Input, InputNumber, Select, Space, Switch, Tree } from "antd";
import type { FormInstance } from "antd/es/form";
import { ALERT_ROUTING_TERMS, formatRouteNodeTreeTitle } from "../../constants/alert-routing-terms";
import type { AlertSubscriptionNode } from "../../services/alert-subscriptions";

/** 平台全局路由树 project_id（投递流水 global: 前缀）。禁止在其他文件再定义一份。 */
export const GLOBAL_ROUTING_PROJECT_ID = 0;

export type SubscriptionTreeNode = {
  key: string;
  title: string;
  children?: SubscriptionTreeNode[];
};

export type RoutingTreeEditorProps = {
  subLoading: boolean;
  subscriptionTreeData: SubscriptionTreeNode[];
  subSelectedID: number;
  onSelectSubscriptionNode: (id: number) => void | Promise<void>;
  selectedSubscriptionNode: AlertSubscriptionNode | null;
  subForm: FormInstance;
  subscriptionSeverityOptions: { label: string; value: string }[];
  receiverGroupOptions: { label: string; value: number }[];
};

export function RoutingTreeEditor({
  subLoading,
  subscriptionTreeData,
  subSelectedID,
  onSelectSubscriptionNode,
  selectedSubscriptionNode,
  subForm,
  subscriptionSeverityOptions,
  receiverGroupOptions,
}: RoutingTreeEditorProps) {
  return (
    <div style={{ display: "grid", gridTemplateColumns: "360px 1fr", gap: 12, alignItems: "start" }}>
      <Card size="small" title={ALERT_ROUTING_TERMS.treeTitle} loading={subLoading} styles={{ body: { padding: 8 } }}>
        <Tree
          treeData={subscriptionTreeData}
          selectedKeys={subSelectedID ? [String(subSelectedID)] : []}
          onSelect={(keys) => {
            const id = Number(keys?.[0] ?? 0);
            if (id > 0) void onSelectSubscriptionNode(id);
          }}
          defaultExpandAll
        />
      </Card>
      <Card
        size="small"
        title={
          selectedSubscriptionNode
            ? `编辑路由节点：${formatRouteNodeTreeTitle(selectedSubscriptionNode.name, true)}`
            : ALERT_ROUTING_TERMS.selectNodeHint
        }
      >
        <Form form={subForm} layout="vertical">
          <Form.Item name="id" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="parent_id" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="name" label={ALERT_ROUTING_TERMS.nodeName} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="code" label={ALERT_ROUTING_TERMS.nodeCode}>
            <Input />
          </Form.Item>
          <Space wrap style={{ width: "100%" }}>
            <Form.Item name="enabled" label="启用" valuePropName="checked" style={{ marginBottom: 0 }}>
              <Switch />
            </Form.Item>
            <Form.Item name="continue" label={ALERT_ROUTING_TERMS.continueMatchChildren} valuePropName="checked" style={{ marginBottom: 0 }}>
              <Switch />
            </Form.Item>
            <Form.Item name="notify_resolved" label="恢复通知" valuePropName="checked" style={{ marginBottom: 0 }}>
              <Switch />
            </Form.Item>
            <Form.Item name="silence_seconds" label="静默(s)" style={{ marginBottom: 0 }}>
              <InputNumber min={0} />
            </Form.Item>
          </Space>
          <Form.Item
            name="match_severity"
            label={`${ALERT_ROUTING_TERMS.matchSeverity}（可选，多选）`}
            extra="告警 labels.severity 命中任一即通过；不选表示不按级别过滤。"
          >
            <Select
              mode="multiple"
              allowClear
              placeholder="不选则不限级别"
              options={subscriptionSeverityOptions}
            />
          </Form.Item>
          <Form.Item
            name="receiver_group_ids"
            label={ALERT_ROUTING_TERMS.receiverGroup}
            dependencies={["parent_id"]}
            rules={[
              {
                validator: async (_, value) => {
                  const pid = subForm.getFieldValue("parent_id");
                  const isRoot = pid === null || pid === undefined || pid === "";
                  if (isRoot) return;
                  const ids = Array.isArray(value) ? value : [];
                  if (ids.length === 0) {
                    throw new Error("非根节点须至少选择一个接收组");
                  }
                },
              },
            ]}
            extra={
              <>
                根节点可留空：仅作路由分流，通知由子节点上的接收组发出。处理人邮件：wechat 等补发；钉钉/企微在无法 @ 时补发。请先点击上方「
                {ALERT_ROUTING_TERMS.receiverGroupManage}」创建接收组并绑定通道。
              </>
            }
          >
            <Select mode="multiple" options={receiverGroupOptions} placeholder="选择通知接收组" allowClear />
          </Form.Item>
          <Form.Item name="match_labels_json" label="match_labels_json（精确匹配 JSON）">
            <Input.TextArea rows={4} />
          </Form.Item>
          <Form.Item name="match_regex_json" label="match_regex_json（正则匹配 JSON）">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
