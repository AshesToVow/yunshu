# -*- coding: utf-8 -*-
"""Patch alert-config-center-panel to use extracted tabs."""
from pathlib import Path
import re

SRC = Path("web/src/pages/alert-config-center-panel.tsx")
text = SRC.read_text(encoding="utf-8")

# 1) Replace GLOBAL_ROUTING const with import
text2 = text.replace(
    "const GLOBAL_ROUTING_PROJECT_ID = 0;\n",
    "",
    1,
)
imp = '''import { webhookPayloadTemplates } from "./alert-config/webhook-templates";
import { GLOBAL_ROUTING_PROJECT_ID } from "./alert-config/routing-tree-editor";
import { SubscriptionsTab } from "./alert-config/subscriptions-tab";
import { HistoryTab } from "./alert-config/history-tab";
'''
text2 = text2.replace(
    'import { webhookPayloadTemplates } from "./alert-config/webhook-templates";\n',
    imp,
    1,
)

# 2) Replace subscriptions children block (from children: ( through ), before history key)
# Find markers
m_sub = re.search(
    r'(    \{\n      key: "subscriptions",\n      label: ALERT_ROUTING_TERMS\.tabRouting,\n      children: \()([\s\S]*?)(\n      \),\n    \},\n    \{\n      key: "history",)',
    text2,
)
if not m_sub:
    raise SystemExit("subscriptions block not found")

sub_replacement = '''    {
      key: "subscriptions",
      label: ALERT_ROUTING_TERMS.tabRouting,
      children: (
        <SubscriptionsTab
          subLoading={subLoading}
          loadSubscriptions={loadSubscriptions}
          setRgDrawerOpen={setRgDrawerOpen}
          wizardForm={wizardForm}
          projectContextId={projectContextId}
          setWizardStep={setWizardStep}
          setWizardOpen={setWizardOpen}
          projects={projects}
          cloneForm={cloneForm}
          setCloneModalOpen={setCloneModalOpen}
          createSubscription={createSubscription}
          subSelectedID={subSelectedID}
          removeSubscription={removeSubscription}
          saveSubscription={saveSubscription}
          subscriptionTreeData={subscriptionTreeData}
          onSelectSubscriptionNode={onSelectSubscriptionNode}
          selectedSubscriptionNode={selectedSubscriptionNode}
          subForm={subForm}
          subscriptionSeverityOptions={subscriptionSeverityOptions}
          receiverGroupOptions={receiverGroupOptions}
        />
      ),
    },
    {
      key: "history",'''

text2 = text2[: m_sub.start()] + sub_replacement + text2[m_sub.end() :]

# 3) Replace history children
m_hist = re.search(
    r'(      key: "history",\n      label: "历史告警记录",\n      children: \()([\s\S]*?)(\n      \),\n    \},\n  \] as const;)',
    text2,
)
if not m_hist:
    raise SystemExit("history block not found")

hist_replacement = '''      key: "history",
      label: "历史告警记录",
      children: (
        <HistoryTab
          embedded={embedded}
          projectContextId={projectContextId}
          eventHistoryMode={eventHistoryMode}
          setEventHistoryMode={setEventHistoryMode}
          eventKeyword={eventKeyword}
          setEventKeyword={setEventKeyword}
          eventAlertIP={eventAlertIP}
          setEventAlertIP={setEventAlertIP}
          alertIPOptions={alertIPOptions}
          eventSourceFilter={eventSourceFilter}
          setEventSourceFilter={setEventSourceFilter}
          sourceFilterOptions={sourceFilterOptions}
          eventStatus={eventStatus}
          setEventStatus={setEventStatus}
          eventCategory={eventCategory}
          setEventCategory={setEventCategory}
          eventGroupKey={eventGroupKey}
          setEventGroupKey={setEventGroupKey}
          eventFingerprint={eventFingerprint}
          setEventFingerprint={setEventFingerprint}
          eventsLoading={eventsLoading}
          events={events}
          groupedEvents={groupedEvents}
          eventsPage={eventsPage}
          eventsPageSize={eventsPageSize}
          eventsTotal={eventsTotal}
          reloadEvents={reloadEvents}
          fpExplainLoading={fpExplainLoading}
          setFpExplainLoading={setFpExplainLoading}
          setFpExplain={setFpExplain}
          setFpAiResult={setFpAiResult}
          setFpExplainOpen={setFpExplainOpen}
        />
      ),
    },
  ] as const;'''

text2 = text2[: m_hist.start()] + hist_replacement + text2[m_hist.end() :]

SRC.write_text(text2, encoding="utf-8")
print("panel lines", len(text2.splitlines()))

# Fix subscriptions-tab projects type
st = Path("web/src/pages/alert-config/subscriptions-tab.tsx")
st_text = st.read_text(encoding="utf-8")
st_text = st_text.replace(
    'import type { ProjectItem } from "../../services/projects";\n',
    "",
)
st_text = st_text.replace(
    "  projects: ProjectItem[];\n",
    "  projects: Array<{ id: number; name: string }>;\n",
)
# Fix Form unused import if any
st.write_text(st_text, encoding="utf-8")

# Fix routing-tree-editor subSelectedID type
rt = Path("web/src/pages/alert-config/routing-tree-editor.tsx")
rt_text = rt.read_text(encoding="utf-8")
rt_text = rt_text.replace(
    "  subSelectedID: number | null;\n",
    "  subSelectedID: number;\n",
)
# Use a looser tree data type
rt_text = rt_text.replace(
    "  subscriptionTreeData: DataNode[];\n",
    "  subscriptionTreeData: DataNode[] | Array<{ key: string; title: string; children?: unknown[] }>;\n",
)
rt.write_text(rt_text, encoding="utf-8")

print("patched")
