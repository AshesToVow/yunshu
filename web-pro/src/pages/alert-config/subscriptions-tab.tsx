// @ts-nocheck
/**
 * 告警配置中心 · 订阅/路由 Tab（RF-07 拆分产物）
 */
import { CopyOutlined, DeleteOutlined, PlusOutlined, ReloadOutlined, ThunderboltOutlined } from "@ant-design/icons";
import { Alert, Button, Popconfirm, Space } from "antd";
import type { FormInstance } from "antd/es/form";
import { ALERT_ROUTING_TERMS } from "../../constants/alert-routing-terms";
import { GLOBAL_ROUTING_PROJECT_ID, RoutingTreeEditor, type RoutingTreeEditorProps } from "./routing-tree-editor";

export type SubscriptionsTabProps = RoutingTreeEditorProps & {
  subLoading: boolean;
  loadSubscriptions: () => void | Promise<void>;
  setRgDrawerOpen: (v: boolean) => void;
  wizardForm: FormInstance;
  projectContextId?: number;
  setWizardStep: (v: number) => void;
  setWizardOpen: (v: boolean) => void;
  projects: Array<{ id: number; name: string }>;
  cloneForm: FormInstance;
  setCloneModalOpen: (v: boolean) => void;
  createSubscription: (parentID?: number | null) => void | Promise<void>;
  subSelectedID: number | null;
  removeSubscription: () => void | Promise<void>;
  saveSubscription: () => void | Promise<void>;
};

export function SubscriptionsTab(p: SubscriptionsTabProps) {
  const {
    subLoading,
    loadSubscriptions,
    setRgDrawerOpen,
    wizardForm,
    projectContextId,
    setWizardStep,
    setWizardOpen,
    projects,
    cloneForm,
    setCloneModalOpen,
    createSubscription,
    subSelectedID,
    removeSubscription,
    saveSubscription,
  } = p;

  return (
    <>
      <Space className="ops-filter-bar" style={{ width: "100%", marginBottom: 12 }} wrap>
        <Button icon={<ReloadOutlined />} loading={subLoading} onClick={() => void loadSubscriptions()}>
          刷新
        </Button>
        <Button onClick={() => setRgDrawerOpen(true)}>
          {ALERT_ROUTING_TERMS.receiverGroupManage}
        </Button>
        <Button
          icon={<ThunderboltOutlined />}
          onClick={() => {
            wizardForm.setFieldsValue({
              project_id: projectContextId && projectContextId > 0 ? projectContextId : undefined,
              severity: "warning",
              channel_ids: [],
              extra_emails: [],
              name: "",
            });
            setWizardStep(0);
            setWizardOpen(true);
          }}
        >
          {ALERT_ROUTING_TERMS.routingWizard}
        </Button>
        <Button
          icon={<CopyOutlined />}
          disabled={projects.length < 1}
          onClick={() => {
            cloneForm.setFieldsValue({
              source_project_id: projects[0]?.id,
              target_project_id: GLOBAL_ROUTING_PROJECT_ID,
              replace_cluster: "",
              replace_route: "",
              include_disabled: false,
              skip_if_target_has_nodes: true,
            });
            setCloneModalOpen(true);
          }}
        >
          {ALERT_ROUTING_TERMS.copyTemplate}
        </Button>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => void createSubscription(null)}>
          新增根节点
        </Button>
        <Button disabled={!subSelectedID} icon={<PlusOutlined />} onClick={() => void createSubscription(subSelectedID)}>
          新增子节点
        </Button>
        <Popconfirm title="确认删除节点？（有子节点会失败）" onConfirm={() => void removeSubscription()}>
          <Button danger disabled={!subSelectedID} icon={<DeleteOutlined />}>
            删除
          </Button>
        </Popconfirm>
        <Button type="primary" onClick={() => void saveSubscription()}>
          保存
        </Button>
      </Space>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        message="本页编辑平台全局路由树（投递流水里的 global: 前缀）"
        description="不要按项目切树。用匹配级别 / match_labels（cluster、project_id、severity）分流。warning 误发企微邮箱时，在本页停用对应子节点（例如 policy_4），不要只改某个业务项目里的停用开关。"
      />
      <RoutingTreeEditor
        subLoading={p.subLoading}
        subscriptionTreeData={p.subscriptionTreeData}
        subSelectedID={p.subSelectedID}
        onSelectSubscriptionNode={p.onSelectSubscriptionNode}
        selectedSubscriptionNode={p.selectedSubscriptionNode}
        subForm={p.subForm}
        subscriptionSeverityOptions={p.subscriptionSeverityOptions}
        receiverGroupOptions={p.receiverGroupOptions}
      />
    </>
  );
}
