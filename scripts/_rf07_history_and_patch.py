# -*- coding: utf-8 -*-
from pathlib import Path

OUT = Path("web/src/pages/alert-config")
SRC = Path("web/src/pages/alert-config-center-panel.tsx")
lines = SRC.read_text(encoding="utf-8").splitlines(keepends=True)


def get(a: int, b: int) -> str:
    return "".join(lines[a - 1 : b])


def dedent(s: str, n: int = 10) -> str:
    out = []
    for line in s.splitlines(keepends=True):
        if not line.strip():
            out.append("\n")
            continue
        lead = len(line) - len(line.lstrip(" "))
        out.append(line[min(n, lead) :])
    return "".join(out)


def indent(s: str, n: int = 4) -> str:
    pad = " " * n
    return "".join("\n" if not l.strip() else pad + l for l in s.splitlines(keepends=True))


hist = dedent(get(864, 1339), 10).strip()
if hist.startswith("<>"):
    hist = hist[2:]
if hist.endswith("</>"):
    hist = hist[:-3]
hist = hist.strip() + "\n"

header = r'''/**
 * 告警配置中心 · 历史告警 Tab（RF-07 拆分产物）
 * 从 alert-config-center-panel.tsx 原地搬迁 JSX。
 */
import { ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Input, Popover, Radio, Select, Space, Tag, Typography, message } from "antd";
import { ResizableTable } from "../../components/resizable-table";
import { ALERT_ROUTING_TERMS } from "../../constants/alert-routing-terms";
import { explainAlertByFingerprint, type AlertEventGroupItem, type AlertEventItem, type FingerprintDeliveryExplain } from "../../services/alerts";
import type { AIAlertExplainResult } from "../../services/ai";
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
import { parseLabelsFromAlertEventRequestPayload, prettifyAlertRequestPayload } from "./payload-parse";

export type HistoryTabProps = {
  embedded?: boolean;
  projectContextId?: number;
  eventHistoryMode: "list" | "grouped";
  setEventHistoryMode: (v: "list" | "grouped") => void;
  eventKeyword: string;
  setEventKeyword: (v: string) => void;
  eventAlertIP: string;
  setEventAlertIP: (v: string) => void;
  alertIPOptions: { label: string; value: string }[];
  eventSourceFilter: string;
  setEventSourceFilter: (v: string) => void;
  sourceFilterOptions: { label: string; value: string }[];
  eventStatus: string;
  setEventStatus: (v: string) => void;
  eventCategory: AlertEventCategory | "";
  setEventCategory: (v: AlertEventCategory | "") => void;
  eventGroupKey: string;
  setEventGroupKey: (v: string) => void;
  eventFingerprint: string;
  setEventFingerprint: (v: string) => void;
  eventsLoading: boolean;
  events: AlertEventItem[];
  groupedEvents: AlertEventGroupItem[];
  eventsPage: number;
  eventsPageSize: number;
  eventsTotal: number;
  reloadEvents: (page?: number, pageSize?: number) => void | Promise<void>;
  fpExplainLoading: boolean;
  setFpExplainLoading: (v: boolean) => void;
  setFpExplain: (v: FingerprintDeliveryExplain | null) => void;
  setFpAiResult: (v: AIAlertExplainResult | null) => void;
  setFpExplainOpen: (v: boolean) => void;
};

export function HistoryTab({
  embedded,
  projectContextId,
  eventHistoryMode,
  setEventHistoryMode,
  eventKeyword,
  setEventKeyword,
  eventAlertIP,
  setEventAlertIP,
  alertIPOptions,
  eventSourceFilter,
  setEventSourceFilter,
  sourceFilterOptions,
  eventStatus,
  setEventStatus,
  eventCategory,
  setEventCategory,
  eventGroupKey,
  setEventGroupKey,
  eventFingerprint,
  setEventFingerprint,
  eventsLoading,
  events,
  groupedEvents,
  eventsPage,
  eventsPageSize,
  eventsTotal,
  reloadEvents,
  fpExplainLoading,
  setFpExplainLoading,
  setFpExplain,
  setFpAiResult,
  setFpExplainOpen,
}: HistoryTabProps) {
  return (
'''

(OUT / "history-tab.tsx").write_text(header + indent(hist) + "  );\n}\n", encoding="utf-8")
print("history-tab", len((OUT / "history-tab.tsx").read_text(encoding="utf-8").splitlines()))

# Patch main panel: replace tabItems children and GLOBAL_ROUTING_PROJECT_ID
text = SRC.read_text(encoding="utf-8")

if "from \"./alert-config/subscriptions-tab\"" not in text and "from \"./alert-config/history-tab\"" not in text:
    # add imports after webhook-templates import
    needle = 'import { webhookPayloadTemplates } from "./alert-config/webhook-templates";'
    # find actual import
    for cand in [
        'import { webhookPayloadTemplates } from "./alert-config/webhook-templates";',
        "from \"./alert-config/webhook-templates\"",
        "from './alert-config/webhook-templates'",
    ]:
        if cand in text:
            print("found webhook import via", cand[:40])
            break
    else:
        # search
        import re
        m = re.search(r'import \{[^}]*webhookPayloadTemplates[^}]*\} from ["\'][^"\']+["\'];', text)
        print("webhook import:", m.group(0) if m else "NOT FOUND")
        if m:
            needle = m.group(0)

# Check AlertSubscriptionNode import path for routing-tree-editor
print("checking services for AlertSubscriptionNode...")
for p in Path("web/src/services").glob("*sub*"):
    print(" ", p)
