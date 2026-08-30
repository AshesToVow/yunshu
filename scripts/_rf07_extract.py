# -*- coding: utf-8 -*-
"""RF-07: extract subscriptions/history tabs + routing-tree-editor."""
from __future__ import annotations

from pathlib import Path

SRC = Path("web/src/pages/alert-config-center-panel.tsx")
OUT = Path("web/src/pages/alert-config")
lines = SRC.read_text(encoding="utf-8").splitlines(keepends=True)


def get(a: int, b: int) -> str:
    return "".join(lines[a - 1 : b])


def dedent(s: str, n: int = 10) -> str:
    """Subscriptions children are indented with 8-10 spaces inside tabItems."""
    out = []
    for line in s.splitlines(keepends=True):
        # strip common leading spaces carefully
        stripped = line.lstrip(" ")
        lead = len(line) - len(stripped)
        remove = min(n, lead) if stripped.strip() else 0
        # for blank lines keep as newline
        if not line.strip():
            out.append("\n")
        else:
            out.append(line[remove:] if lead >= n else line.lstrip(" ") if False else line[min(n, lead):])
    return "".join(out)


def indent(s: str, n: int = 4) -> str:
    pad = " " * n
    return "".join("\n" if not l.strip() else pad + l for l in s.splitlines(keepends=True))


# subscriptions children: inside `children: ( <> ... </> ),` — lines 700-857
# history children: 864-1339
sub_inner = get(700, 857)  # <> ... </>
hist_inner = get(864, 1339)

# routing tree editor: the grid div 763-856
tree_inner = get(763, 856)

# Write GLOBAL constant + routing tree editor
tree_body = dedent(tree_inner, 10)
(OUT / "routing-tree-editor.tsx").write_text(
    '''/**
 * 全局路由树编辑器（RF-07 拆分产物）
 * 从 alert-config-center-panel.tsx 原地搬迁。
 * GLOBAL_ROUTING_PROJECT_ID 仅在此定义一处，主面板与 Tab 均从本文件导入。
 */
import { Card, Form, Input, InputNumber, Select, Space, Switch, Tree } from "antd";
import type { FormInstance } from "antd/es/form";
import type { DataNode } from "antd/es/tree";
import { ALERT_ROUTING_TERMS, formatRouteNodeTreeTitle } from "../../constants/alert-routing-terms";
import type { AlertSubscriptionNode } from "../../services/alert-subscriptions";

/** 平台全局路由树 project_id（投递流水 global: 前缀）。禁止在其他文件再定义一份。 */
export const GLOBAL_ROUTING_PROJECT_ID = 0;

export type RoutingTreeEditorProps = {
  subLoading: boolean;
  subscriptionTreeData: DataNode[];
  subSelectedID: number | null;
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
'''
    + indent(tree_body.rstrip() + "\n")
    + "  );\n}\n",
    encoding="utf-8",
)

# subscriptions tab: toolbar + alert + RoutingTreeEditor call
# Build from lines 700-762 (toolbar+alert) then component
toolbar = dedent(get(700, 762), 10)
# replace closing of fragment - toolbar ends before tree grid

(OUT / "subscriptions-tab.tsx").write_text(
    '''/**
 * 告警配置中心 · 订阅/路由 Tab（RF-07 拆分产物）
 */
import { CopyOutlined, DeleteOutlined, PlusOutlined, ReloadOutlined, ThunderboltOutlined } from "@ant-design/icons";
import { Alert, Button, Form, Popconfirm, Space } from "antd";
import type { FormInstance } from "antd/es/form";
import { ALERT_ROUTING_TERMS } from "../../constants/alert-routing-terms";
import type { ProjectItem } from "../../services/projects";
import { GLOBAL_ROUTING_PROJECT_ID, RoutingTreeEditor, type RoutingTreeEditorProps } from "./routing-tree-editor";

export type SubscriptionsTabProps = RoutingTreeEditorProps & {
  subLoading: boolean;
  loadSubscriptions: () => void | Promise<void>;
  setRgDrawerOpen: (v: boolean) => void;
  wizardForm: FormInstance;
  projectContextId?: number;
  setWizardStep: (v: number) => void;
  setWizardOpen: (v: boolean) => void;
  projects: ProjectItem[];
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
'''
    ,
    encoding="utf-8",
)

# History tab - exact JSX relocation with same-name props
# Discover identifiers used
import re
hist_text = hist_inner
# Too many props — write component that accepts a context-like bag typed loosely
# Better: exact JSX with destructured props matching names in hist_inner

hist_body = dedent(hist_inner, 10)
# Strip outer <> </>
hb = hist_body.strip()
if hb.startswith("<>"):
    hb = hb[2:]
if hb.endswith("</>"):
    hb = hb[:-3]

(OUT / "history-tab.tsx").write_text(
    '''/**
 * 告警配置中心 · 历史告警 Tab（RF-07 拆分产物）
 * 从 alert-config-center-panel.tsx 原地搬迁 JSX。
 */
import { ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Input, Popover, Select, Space, Table, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { ResizableTable } from "../../components/resizable-table";
import { ALERT_ROUTING_TERMS } from "../../constants/alert-routing-terms";
import type { AIAlertExplainResult } from "../../services/ai";
import type { AlertEventGroupItem, AlertEventItem, FingerprintDeliveryExplain } from "../../services/alerts";
import {
  ALERT_EVENT_CATEGORY_OPTIONS,
  ALERT_HISTORY_PIPELINE_HELP,
  describeAlertEvent,
  summarizeAlertEventHint,
  type AlertEventCategory,
} from "../../utils/alert-event-reasons";
import { formatMatchedPolicyNamesDisplay } from "../../utils/alert-policy-display";
import { explainAlertRecipients } from "../../utils/alert-recipient-reason";
import { formatDateTime } from "../../utils/format";
import { tablePagination } from "../../utils/table-pagination";
import { prettifyAlertRequestPayload } from "./payload-parse";

export type HistoryTabProps = {
  eventsViewMode: "flat" | "grouped";
  setEventsViewMode: (v: "flat" | "grouped") => void;
  eventsKeyword: string;
  setEventsKeyword: (v: string) => void;
  eventsStatus: string | undefined;
  setEventsStatus: (v: string | undefined) => void;
  eventsCategory: AlertEventCategory | undefined;
  setEventsCategory: (v: AlertEventCategory | undefined) => void;
  eventsLoading: boolean;
  reloadEvents: (page?: number, pageSize?: number) => void | Promise<void>;
  eventsPage: number;
  eventsPageSize: number;
  eventsTotal: number;
  setEventsPage: (v: number) => void;
  setEventsPageSize: (v: number) => void;
  events: AlertEventItem[];
  eventGroups: AlertEventGroupItem[];
  groupsLoading: boolean;
  reloadGroups: (page?: number, pageSize?: number) => void | Promise<void>;
  groupsPage: number;
  groupsPageSize: number;
  groupsTotal: number;
  setGroupsPage: (v: number) => void;
  setGroupsPageSize: (v: number) => void;
  openEventDetail: (row: AlertEventItem) => void;
  openGroupDetail: (row: AlertEventGroupItem) => void;
  explainLoadingFp: string | null;
  fingerprintExplain: FingerprintDeliveryExplain | null;
  aiExplainLoadingFp: string | null;
  aiExplainByFp: Record<string, AIAlertExplainResult>;
  runFingerprintExplain: (fp: string) => void | Promise<void>;
  runAIExplain: (fp: string) => void | Promise<void>;
};

export function HistoryTab(props: HistoryTabProps) {
  // Bind original local names for exact JSX
  const {
    eventsViewMode, setEventsViewMode,
    eventsKeyword, setEventsKeyword,
    eventsStatus, setEventsStatus,
    eventsCategory, setEventsCategory,
    eventsLoading, reloadEvents,
    eventsPage, eventsPageSize, eventsTotal, setEventsPage, setEventsPageSize,
    events, eventGroups, groupsLoading, reloadGroups,
    groupsPage, groupsPageSize, groupsTotal, setGroupsPage, setGroupsPageSize,
    openEventDetail, openGroupDetail,
    explainLoadingFp, fingerprintExplain,
    aiExplainLoadingFp, aiExplainByFp,
    runFingerprintExplain, runAIExplain,
  } = props;

  return (
'''
    + indent(hb.rstrip() + "\n")
    + "  );\n}\n",
    encoding="utf-8",
)

print("Wrote routing-tree-editor", len((OUT / "routing-tree-editor.tsx").read_text(encoding="utf-8").splitlines()))
print("Wrote subscriptions-tab", len((OUT / "subscriptions-tab.tsx").read_text(encoding="utf-8").splitlines()))
print("Wrote history-tab", len((OUT / "history-tab.tsx").read_text(encoding="utf-8").splitlines()))
print("history body starts:", repr(hb[:120]))
print("tree body starts:", repr(tree_body[:120]))
