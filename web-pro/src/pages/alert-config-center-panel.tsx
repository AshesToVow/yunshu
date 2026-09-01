// @ts-nocheck
import { CopyOutlined, DeleteOutlined, EditOutlined, MinusCircleOutlined, PlusOutlined, ReloadOutlined, ThunderboltOutlined } from "@ant-design/icons";
import { Alert, AutoComplete, Button, Card, Drawer, Form, Input, InputNumber, Modal, Popconfirm, Popover, Radio, Select, Space, Statistic, Steps, Switch, Table, Tabs, Tag, Tree, Typography, message } from "antd";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  getAlertHistoryStats,
  listAlertChannels,
  listAlertEvents,
  listAlertEventsGrouped,
  explainAlertByFingerprint,
  sendAlertmanagerWebhook,
  type AlertEventGroupItem,
  type AlertEventItem,
  type FingerprintDeliveryExplain,
  debugAlertRouting,
  type AlertRoutingDebugResult,
} from "../services/alerts";
import { analyzeAlertExplainAI, type AIAlertExplainResult } from "../services/ai";
import { extractApiErrorMessage } from "../services/http";
import { useDictOptions } from "../hooks/use-dict-options";
import { revealDictEntryValue } from "../services/dict";
import {
  CHART_ERROR,
  CHART_INFO,
  CHART_SUCCESS,
} from "../constants/chart-colors";
import { formatDateTime } from "../utils/format";
import { formatMatchedPolicyNamesDisplay } from "../utils/alert-policy-display";
import { explainAlertRecipients } from "../utils/alert-recipient-reason";
import { DEFAULT_PAGE_SIZE, tablePagination } from "../utils/table-pagination";
import { ResizableTable } from "../components/resizable-table";
import {
  ALERT_EVENT_CATEGORY_OPTIONS,
  ALERT_HISTORY_PIPELINE_HELP,
  describeAlertEvent,
  summarizeAlertEventHint,
  type AlertEventCategory,
} from "../utils/alert-event-reasons";
import { ALERT_ROUTING_TERMS, formatReceiverGroupLabel, formatRouteNodeTreeTitle } from "../constants/alert-routing-terms";
import { listAlertDatasources, type AlertDatasourceItem } from "../services/alert-platform";
import { getProjects } from "../services/projects";
import {
  applyRoutingWizard,
  cloneSubscriptionFromProject,
  createReceiverGroup,
  createSubscriptionNode,
  deleteReceiverGroup,
  deleteSubscriptionNode,
  getSubscriptionTree,
  listReceiverGroups,
  updateReceiverGroup,
  updateSubscriptionNode,
  type AlertReceiverGroup,
  type AlertSubscriptionNode,
} from "../services/alert-subscriptions";
import {
  parseLabelsFromAlertEventRequestPayload,
  parseReceiverGroupChannelIds,
  parseReceiverGroupEmails,
  prettifyAlertRequestPayload,
} from "./alert-config/payload-parse";
import { webhookPayloadTemplates } from "./alert-config/webhook-templates";
import { GLOBAL_ROUTING_PROJECT_ID } from "./alert-config/routing-tree-editor";
import { SubscriptionsTab } from "./alert-config/subscriptions-tab";
import { HistoryTab } from "./alert-config/history-tab";

export type AlertConfigTab = "subscriptions" | "history";

/** 通知与路由只编辑平台全局树。投递仍可能合并各项目旧节点，本页改的是 global: 命中。 */

export type AlertConfigCenterPanelProps = {
  /** 当前子 Tab（策略 / 历史 / 模板） */
  activeTab: AlertConfigTab;
  onTabChange: (key: AlertConfigTab) => void;
  /** 嵌入「告警监控平台」时不显示最外层标题 Card */
  embedded?: boolean;
  /** 嵌入主页面时，仅展示当前视图内容，不再显示内部 tabs */
  hideTabs?: boolean;
  /** 历史 Tab 初始策略分类（如从抑制页跳转 ?event_category=inhibition） */
  initialEventCategory?: AlertEventCategory;
  /** 告警监控平台顶栏「全局项目上下文」；有值时同步订阅/历史筛选 */
  projectContextId?: number;
};

export function AlertConfigCenterPanel({
  activeTab: tab,
  onTabChange: setTab,
  embedded,
  hideTabs,
  initialEventCategory,
  projectContextId,
}: AlertConfigCenterPanelProps) {
  const [stats, setStats] = useState<{
    total: number;
    firing: number;
    resolved: number;
    success: number;
    failed: number;
    today_created: number;
    cluster_values?: string[];
    monitor_pipeline_values?: string[];
    datasource_filter_options?: Array<{ id: number; name: string }>;
  }>();

  const [channels, setChannels] = useState<Array<{ id: number; name: string }>>([]);
  const [baseLoading, setBaseLoading] = useState(false);

  // subscriptions (新策略)
  const [projects, setProjects] = useState<Array<{ id: number; name: string }>>([]);
  const [subProjectID, setSubProjectID] = useState<number>(0);
  const [subTree, setSubTree] = useState<AlertSubscriptionNode[]>([]);
  const [subSelectedID, setSubSelectedID] = useState<number>(0);
  const [subLoading, setSubLoading] = useState(false);
  const [receiverGroups, setReceiverGroups] = useState<AlertReceiverGroup[]>([]);
  const [projectDatasources, setProjectDatasources] = useState<AlertDatasourceItem[]>([]);
  const [subForm] = Form.useForm();
  const [cloneModalOpen, setCloneModalOpen] = useState(false);
  const [cloneSubmitting, setCloneSubmitting] = useState(false);
  const [cloneForm] = Form.useForm<{
    source_project_id: number;
    target_project_id: number;
    replace_cluster?: string;
    replace_route?: string;
    include_disabled?: boolean;
    skip_if_target_has_nodes?: boolean;
  }>();
  const [wizardOpen, setWizardOpen] = useState(false);
  const [wizardStep, setWizardStep] = useState(0);
  const [wizardSubmitting, setWizardSubmitting] = useState(false);
  const [wizardForm] = Form.useForm<{
    project_id?: number;
    severity?: string;
    channel_ids?: number[];
    extra_emails?: string[];
    name?: string;
  }>();
  const [rgDrawerOpen, setRgDrawerOpen] = useState(false);
  const [rgModalOpen, setRgModalOpen] = useState(false);
  const [rgEditingId, setRgEditingId] = useState<number | null>(null);
  const [rgSaving, setRgSaving] = useState(false);
  const [rgForm] = Form.useForm<{
    name: string;
    description?: string;
    channel_ids?: number[];
    email_recipients?: string[];
    enabled?: boolean;
    escalation_level?: number;
    escalation_delay_seconds?: number;
  }>();

  const [eventsLoading, setEventsLoading] = useState(false);
  const [events, setEvents] = useState<AlertEventItem[]>([]);
  const [groupedEvents, setGroupedEvents] = useState<AlertEventGroupItem[]>([]);
  const [eventHistoryMode, setEventHistoryMode] = useState<"list" | "grouped">("list");
  const [eventsPage, setEventsPage] = useState(1);
  const [eventsPageSize, setEventsPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [eventsTotal, setEventsTotal] = useState(0);
  const [eventKeyword, setEventKeyword] = useState("");
  const [eventAlertIP, setEventAlertIP] = useState("");
  const [eventStatus, setEventStatus] = useState("");
  /** 格式：`ds:<数据源ID>` 或 `mp:<monitor_pipeline slug>`（兼容历史 prometheus/platform） */
  const [eventSourceFilter, setEventSourceFilter] = useState("");
  const [eventGroupKey, setEventGroupKey] = useState("");
  const [eventFingerprint, setEventFingerprint] = useState("");
  const [fpExplain, setFpExplain] = useState<FingerprintDeliveryExplain | null>(null);
  const [fpExplainOpen, setFpExplainOpen] = useState(false);
  const [fpExplainLoading, setFpExplainLoading] = useState(false);
  const [fpAiLoading, setFpAiLoading] = useState(false);
  const [fpAiResult, setFpAiResult] = useState<AIAlertExplainResult | null>(null);
  const [eventCategory, setEventCategory] = useState<AlertEventCategory | "">(initialEventCategory ?? "");
  const eventsPageSizeRef = useRef(eventsPageSize);
  eventsPageSizeRef.current = eventsPageSize;
  const loadEventsSeqRef = useRef(0);
  const loadSubscriptionsSeqRef = useRef(0);
  const [webhookSending, setWebhookSending] = useState(false);
  const [routingDebugLoading, setRoutingDebugLoading] = useState(false);
  const [routingDebugResult, setRoutingDebugResult] = useState<AlertRoutingDebugResult | null>(null);
  const [routingLabelsJSON, setRoutingLabelsJSON] = useState('{"alertname":"DemoAlert","severity":"warning","cluster":"demo"}');
  const [routingSeverity, setRoutingSeverity] = useState("warning");
  const [webhookToken, setWebhookToken] = useState("");
  const [webhookTokenSensitivePickId, setWebhookTokenSensitivePickId] = useState<number | undefined>();
  const [webhookTemplate, setWebhookTemplate] = useState<"warning_prod" | "critical_prod" | "resolved_prod">("warning_prod");
  const [webhookPayload, setWebhookPayload] = useState(
    JSON.stringify(webhookPayloadTemplates.warning_prod, null, 2),
  );
  const webhookTokenOptions = useDictOptions("alert_webhook_token");
  const promqlLabelKeyOptions = useDictOptions("alert_promql_label_key");
  const webhookTokenPick = useMemo(() => {
    if (webhookTokenSensitivePickId != null) return webhookTokenSensitivePickId;
    const t = String(webhookToken || "").trim();
    if (!t) return undefined;
    const m = webhookTokenOptions.find((o) => !o.sensitive && String(o.value ?? "").trim() === t);
    return m?.value;
  }, [webhookToken, webhookTokenOptions, webhookTokenSensitivePickId]);

  const channelOptions = useMemo(() => channels.map((c) => ({ label: c.name, value: c.id })), [channels]);
  const labelKeyAutoCompleteOptions = useMemo(
    () => {
      return promqlLabelKeyOptions
        .map((opt) => {
          const value = String(opt.value ?? "").trim();
          const label = String(opt.label ?? "").trim() || value;
          return { value, label: `${label} (${value})` };
        })
        .filter((it) => it.value)
        .sort((a, b) => a.value.localeCompare(b.value, "zh-CN"));
    },
    [promqlLabelKeyOptions],
  );

  const alertIPOptions = useMemo(() => {
    const fromPage = (events ?? [])
      .map((it) => String(it.alertIP || "").trim())
      .filter(Boolean);
    const merged = Array.from(new Set([...fromPage])).sort((a, b) => a.localeCompare(b, "zh-CN"));
    return merged.map((v) => ({ label: v, value: v }));
  }, [events]);

  const sourceFilterOptions = useMemo(() => {
    const opts: { label: string; value: string }[] = [];
    const seen = new Set<string>();
    const push = (label: string, value: string) => {
      if (!value || seen.has(value)) return;
      seen.add(value);
      opts.push({ label, value });
    };
    push("Alertmanager", "mp:alertmanager");
    push("云资源到期", "mp:cloud_expiry");
    for (const ds of projectDatasources) {
      const id = Number(ds.id);
      if (!Number.isFinite(id) || id <= 0) continue;
      const name = String(ds.name ?? "").trim() || `数据源 ${id}`;
      push(name, `ds:${id}`);
    }
    if (opts.length <= 2) {
    for (const row of stats?.datasource_filter_options ?? []) {
      const id = Number(row?.id);
      if (!Number.isFinite(id) || id <= 0) continue;
        const name = String(row?.name ?? "").trim() || `数据源 ${id}`;
        push(name, `ds:${id}`);
      }
    }
    return opts;
  }, [projectDatasources, stats?.datasource_filter_options]);
  const webhookTemplateOptions = useMemo(
    () => [
      { label: "warning（prod）", value: "warning_prod" },
      { label: "critical（prod）", value: "critical_prod" },
      { label: "resolved（prod）", value: "resolved_prod" },
    ],
    [],
  );

  function applyWebhookTemplate(key: "warning_prod" | "critical_prod" | "resolved_prod") {
    setWebhookTemplate(key);
    setWebhookPayload(JSON.stringify(webhookPayloadTemplates[key], null, 2));
  }

  async function loadBase(projectId?: number) {
    setBaseLoading(true);
    try {
      const [statsRes, channelRes] = await Promise.all([
        getAlertHistoryStats({ project_id: projectId && projectId > 0 ? projectId : undefined }),
        listAlertChannels(),
      ]);
      setStats(statsRes);
      setChannels((channelRes.list ?? []).map((c) => ({ id: c.id, name: c.name })));
    } catch {
      // http 拦截器已 toast；保留上次成功的 stats，避免失败时整页变 0
    } finally {
      setBaseLoading(false);
    }
  }

  async function loadProjects() {
    try {
      const res = await getProjects({ page: 1, page_size: 200 });
      const list = (res?.list ?? []) as Array<{ id: number; name: string }>;
      const normalized = list.map((it) => ({ id: Number((it as any).id), name: String((it as any).name || "") })).filter((it) => it.id > 0);
      setProjects(normalized);
    } catch {
      // http 拦截器已 toast
    }
  }

  /** 同名接收组只展示一条，保留 id 较大者（通常为最近迁移/创建），避免下拉重复 */
  const receiverGroupOptions = useMemo(() => {
    const byNameKey = new Map<string, { label: string; value: number }>();
    for (const g of receiverGroups) {
      const id = Number(g.id);
      if (!Number.isFinite(id) || id <= 0) continue;
      const label = formatReceiverGroupLabel(String(g.name ?? ""), id);
      const key = label.toLowerCase();
      const prev = byNameKey.get(key);
      if (!prev || id > prev.value) {
        byNameKey.set(key, { label, value: id });
      }
    }
    return Array.from(byNameKey.values()).sort((a, b) => a.label.localeCompare(b.label, "zh-CN"));
  }, [receiverGroups]);

  const receiverGroupDuplicateCount = useMemo(() => {
    const counts = new Map<string, number>();
    for (const g of receiverGroups) {
      const key = formatReceiverGroupLabel(String(g.name ?? ""), Number(g.id)).toLowerCase();
      counts.set(key, (counts.get(key) || 0) + 1);
    }
    let dup = 0;
    counts.forEach((n) => {
      if (n > 1) dup += n - 1;
    });
    return dup;
  }, [receiverGroups]);

  const subscriptionSeverityOptions = useMemo(
    () =>
      ["critical", "warning", "info", "error", "none"].map((v) => ({
        label: v,
        value: v,
      })),
    [],
  );

  type SubscriptionAntTreeNode = { key: string; title: string; children?: SubscriptionAntTreeNode[] };

  const subscriptionTreeData = useMemo(() => {
    const toTree = (nodes: AlertSubscriptionNode[]): SubscriptionAntTreeNode[] =>
      (nodes ?? []).map((n) => {
        const ch = toTree(n.children ?? []);
        const row: SubscriptionAntTreeNode = {
          key: String(n.id),
          title: formatRouteNodeTreeTitle(n.name, n.enabled),
        };
        if (ch.length > 0) row.children = ch;
        return row;
      });
    return toTree(subTree);
  }, [subTree]);

  const selectedSubscriptionNode = useMemo(() => {
    const walk = (nodes: AlertSubscriptionNode[]): AlertSubscriptionNode | null => {
      for (const n of nodes ?? []) {
        if (n.id === subSelectedID) return n;
        const hit = walk(n.children ?? []);
        if (hit) return hit;
      }
      return null;
    };
    return walk(subTree);
  }, [subTree, subSelectedID]);

  const loadSubscriptions = useCallback(async (_overrideProjectId?: number) => {
    const seq = ++loadSubscriptionsSeqRef.current;
    setSubLoading(true);
    try {
      const [tree, groups] = await Promise.all([
        getSubscriptionTree({ project_id: GLOBAL_ROUTING_PROJECT_ID }),
        listReceiverGroups({ page: 1, page_size: 200 }),
      ]);
      if (seq !== loadSubscriptionsSeqRef.current) return;
      setSubTree(tree ?? []);
      setReceiverGroups(groups.list ?? groups.items ?? []);
    } catch {
      if (seq !== loadSubscriptionsSeqRef.current) return;
    } finally {
      if (seq === loadSubscriptionsSeqRef.current) setSubLoading(false);
    }
  }, []);

  function openReceiverGroupCreate() {
    setRgEditingId(null);
    rgForm.resetFields();
    rgForm.setFieldsValue({ enabled: true, channel_ids: [], email_recipients: [], escalation_level: 0, escalation_delay_seconds: 900 });
    setRgModalOpen(true);
  }

  function openReceiverGroupEdit(g: AlertReceiverGroup) {
    setRgEditingId(g.id);
    rgForm.setFieldsValue({
      name: g.name,
      description: g.description ?? "",
      channel_ids: parseReceiverGroupChannelIds(g),
      email_recipients: parseReceiverGroupEmails(g),
      enabled: g.enabled,
      escalation_level: g.escalation_level ?? 0,
      escalation_delay_seconds: g.escalation_delay_seconds ?? (g.escalation_level > 0 ? 900 : 0),
    });
    setRgModalOpen(true);
  }

  async function saveReceiverGroup() {
    const pid = GLOBAL_ROUTING_PROJECT_ID;
    const values = await rgForm.validateFields();
    const level = Number(values.escalation_level ?? 0);
    const payload = {
      project_id: pid,
      name: String(values.name ?? "").trim(),
      description: String(values.description ?? "").trim(),
      channel_ids_json: JSON.stringify(values.channel_ids ?? []),
      email_recipients_json: JSON.stringify(values.email_recipients ?? []),
      escalation_level: Number.isFinite(level) ? Math.max(0, Math.min(10, Math.trunc(level))) : 0,
      escalation_delay_seconds: Math.max(0, Math.trunc(Number(values.escalation_delay_seconds ?? 0))),
      enabled: values.enabled !== false,
    };
    setRgSaving(true);
    try {
      if (rgEditingId) {
        await updateReceiverGroup(rgEditingId, payload);
      } else {
        await createReceiverGroup(payload);
      }
      message.success("接收组已保存");
      setRgModalOpen(false);
      await loadSubscriptions();
    } finally {
      setRgSaving(false);
    }
  }

  async function removeReceiverGroup(id: number) {
    await deleteReceiverGroup(id);
    message.success("已删除");
    await loadSubscriptions();
  }

  async function onSelectSubscriptionNode(id: number) {
    setSubSelectedID(id);
    const node = (() => {
      const walk = (nodes: AlertSubscriptionNode[]): AlertSubscriptionNode | null => {
        for (const n of nodes ?? []) {
          if (n.id === id) return n;
          const hit = walk(n.children ?? []);
          if (hit) return hit;
        }
        return null;
      };
      return walk(subTree);
    })();
    if (!node) return;
    subForm.setFieldsValue({
      id: node.id,
      project_id: node.project_id,
      parent_id: node.parent_id ?? null,
      name: node.name,
      code: node.code,
      enabled: node.enabled,
      continue: node.continue,
      match_labels_json: node.match_labels_json ?? "{}",
      match_regex_json: node.match_regex_json ?? "{}",
      match_severity: node.match_severity
        ? String(node.match_severity)
            .split(/[,，;|]/)
            .map((s) => s.trim())
            .filter(Boolean)
        : [],
      receiver_group_ids: node.receiver_group_ids ?? [],
      silence_seconds: node.silence_seconds ?? 0,
      notify_resolved: node.notify_resolved,
    });
  }

  async function createSubscription(parentID?: number | null) {
    const payload: any = {
      project_id: GLOBAL_ROUTING_PROJECT_ID,
      parent_id: parentID ?? null,
      name: !parentID ? ALERT_ROUTING_TERMS.rootPolicyName : "新路由节点",
      code: "",
      enabled: true,
      continue: false,
      match_labels_json: "{}",
      match_regex_json: "{}",
      match_severity: "",
      receiver_group_ids_json: "[]",
      silence_seconds: 0,
      notify_resolved: true,
    };
    const created = await createSubscriptionNode(payload);
    message.success("已创建");
    await loadSubscriptions();
    await onSelectSubscriptionNode(created.id);
  }

  async function saveSubscription() {
    const v = await subForm.validateFields();
    const id = Number(v.id || 0);
    const payload: any = {
      project_id: GLOBAL_ROUTING_PROJECT_ID,
      parent_id: v.parent_id ?? null,
      name: String(v.name || "").trim(),
      code: String(v.code || "").trim(),
      enabled: !!v.enabled,
      continue: !!v.continue,
      match_labels_json: String(v.match_labels_json || "{}"),
      match_regex_json: String(v.match_regex_json || "{}"),
      match_severity: Array.isArray(v.match_severity)
        ? (v.match_severity as string[]).map((x) => String(x).trim()).filter(Boolean).join(",")
        : String(v.match_severity || "")
            .trim()
            .split(/[,，;|]/)
            .map((s) => s.trim())
            .filter(Boolean)
            .join(","),
      receiver_group_ids_json: JSON.stringify((v.receiver_group_ids ?? []).map((x: any) => Number(x)).filter((x: number) => x > 0)),
      silence_seconds: Number(v.silence_seconds || 0),
      notify_resolved: !!v.notify_resolved,
    };
    if (!id) {
      const created = await createSubscriptionNode(payload);
      message.success("已保存");
      await loadSubscriptions();
      await onSelectSubscriptionNode(created.id);
      return;
    }
    await updateSubscriptionNode(id, payload);
    message.success("已保存");
    await loadSubscriptions();
  }

  async function removeSubscription() {
    if (!subSelectedID) return;
    await deleteSubscriptionNode(subSelectedID);
    message.success("已删除");
    setSubSelectedID(0);
    subForm.resetFields();
    await loadSubscriptions();
  }

  const effectiveProjectId = projectContextId && projectContextId > 0 ? projectContextId : subProjectID;

  const loadEvents = useCallback(
    async (page: number, pageSize: number) => {
      const seq = ++loadEventsSeqRef.current;
      setEventsLoading(true);
      try {
        const src = String(eventSourceFilter || "").trim();
        let datasourceId: number | undefined;
        let monitorPipeline: string | undefined;
        if (src.startsWith("ds:")) {
          const id = Number(src.slice(3));
          if (Number.isFinite(id) && id > 0) datasourceId = id;
        } else if (src.startsWith("mp:")) {
          const slug = src.slice(3).trim();
          if (slug) monitorPipeline = slug;
        }
        const res = await listAlertEvents({
          page,
          page_size: pageSize,
          keyword: eventKeyword.trim() || undefined,
          alertIP: eventAlertIP.trim() || undefined,
          status: eventStatus.trim() || undefined,
          monitorPipeline,
          datasourceId,
          groupKey: eventGroupKey.trim() || undefined,
          fingerprint: eventFingerprint.trim() || undefined,
          category: eventCategory || undefined,
          projectId: effectiveProjectId > 0 ? effectiveProjectId : undefined,
        });
        if (seq !== loadEventsSeqRef.current) return;
        setEvents(res.list ?? []);
        setEventsTotal(res.total ?? 0);
        setEventsPage(res.page ?? page);
        setEventsPageSize(res.page_size ?? pageSize);
      } catch {
        if (seq !== loadEventsSeqRef.current) return;
      } finally {
        if (seq === loadEventsSeqRef.current) setEventsLoading(false);
      }
    },
    [eventKeyword, eventAlertIP, eventStatus, eventSourceFilter, eventGroupKey, eventFingerprint, eventCategory, effectiveProjectId],
  );

  const loadEventsGrouped = useCallback(
    async (page: number, pageSize: number) => {
      const seq = ++loadEventsSeqRef.current;
      setEventsLoading(true);
      try {
        const res = await listAlertEventsGrouped({
          page,
          page_size: pageSize,
          keyword: eventKeyword.trim() || undefined,
          projectId: effectiveProjectId > 0 ? effectiveProjectId : undefined,
        });
        if (seq !== loadEventsSeqRef.current) return;
        setGroupedEvents(res.list ?? []);
        setEventsTotal(res.total ?? 0);
        setEventsPage(res.page ?? page);
        setEventsPageSize(res.page_size ?? pageSize);
      } catch {
        if (seq !== loadEventsSeqRef.current) return;
      } finally {
        if (seq === loadEventsSeqRef.current) setEventsLoading(false);
      }
    },
    [eventKeyword, effectiveProjectId],
  );

  const reloadEvents = useCallback(
    (page: number, pageSize: number) => {
      if (eventHistoryMode === "grouped") return loadEventsGrouped(page, pageSize);
      return loadEvents(page, pageSize);
    },
    [eventHistoryMode, loadEvents, loadEventsGrouped],
  );

  useEffect(() => {
    if (initialEventCategory) {
      setEventCategory(initialEventCategory);
    }
  }, [initialEventCategory]);

  useEffect(() => {
    void loadProjects();
  }, []);

  useEffect(() => {
    void loadBase(effectiveProjectId > 0 ? effectiveProjectId : undefined);
  }, [effectiveProjectId]);

  useEffect(() => {
    if (projectContextId && projectContextId > 0) {
      setSubProjectID(projectContextId);
    }
  }, [projectContextId]);

  useEffect(() => {
    if (tab !== "history") {
      return;
    }
    const delay =
      eventKeyword || eventAlertIP || eventStatus || eventSourceFilter || eventGroupKey || eventFingerprint || eventCategory
        ? 300
        : 0;
    const timer = window.setTimeout(() => {
      void reloadEvents(1, eventsPageSizeRef.current);
    }, delay);
    return () => window.clearTimeout(timer);
  }, [
    tab,
    eventKeyword,
    eventAlertIP,
    eventStatus,
    eventSourceFilter,
    eventGroupKey,
    eventFingerprint,
    eventCategory,
    eventHistoryMode,
    effectiveProjectId,
    reloadEvents,
  ]);

  useEffect(() => {
    if (tab !== "subscriptions") {
      return;
    }
    void loadSubscriptions();
  }, [tab, loadSubscriptions]);

  useEffect(() => {
    if (tab !== "history") return;
    let cancelled = false;
    const pid = effectiveProjectId > 0 ? effectiveProjectId : undefined;
    void listAlertDatasources({ project_id: pid, page: 1, page_size: 200 })
      .then((r) => {
        if (cancelled) return;
        setProjectDatasources(r.list ?? r.items ?? []);
      })
      .catch(() => {
        if (!cancelled) setProjectDatasources([]);
      });
    return () => {
      cancelled = true;
    };
  }, [tab, effectiveProjectId]);

  async function sendWebhookDemo() {
    if (!String(webhookToken || "").trim()) {
      message.warning("请先选择或填写 Webhook Token（与后端 alert.webhook_token 一致，空 Token 将被拒绝）");
      return;
    }
    let payloadObj: Record<string, unknown>;
    try {
      payloadObj = JSON.parse(webhookPayload || "{}") as Record<string, unknown>;
    } catch {
      message.error("Webhook Payload 不是合法 JSON");
      return;
    }
    setWebhookSending(true);
    try {
      await sendAlertmanagerWebhook(payloadObj, webhookToken);
      message.success("Webhook 已发送，告警链路已触发");
      await reloadEvents(1, eventsPageSize);
      await loadBase();
    } finally {
      setWebhookSending(false);
    }
  }

  const tabItems = [
    {
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
      key: "history",
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
  ] as const;

  const activeContent = tabItems.find((item) => item.key === tab)?.children ?? null;

  const showOverviewAndDebug = !(embedded && hideTabs && tab === "subscriptions");

  const body = (
    <>
      {showOverviewAndDebug ? (
        <>
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fit, minmax(130px, 1fr))",
              gap: 12,
              marginBottom: 12,
            }}
          >
            <Card size="small" styles={{ body: { padding: 12 } }}>
              <Statistic title="总告警数" value={stats?.total ?? 0} />
            </Card>
            <Card size="small" styles={{ body: { padding: 12 } }}>
              <Statistic title="Firing" value={stats?.firing ?? 0} valueStyle={{ color: CHART_ERROR }} />
            </Card>
            <Card size="small" styles={{ body: { padding: 12 } }}>
              <Statistic title="Resolved" value={stats?.resolved ?? 0} valueStyle={{ color: CHART_SUCCESS }} />
            </Card>
            <Card size="small" styles={{ body: { padding: 12 } }}>
              <Statistic title="发送成功" value={stats?.success ?? 0} valueStyle={{ color: CHART_INFO }} />
            </Card>
            <Card size="small" styles={{ body: { padding: 12 } }}>
              <Statistic title="发送失败" value={stats?.failed ?? 0} valueStyle={{ color: CHART_ERROR }} />
            </Card>
            <Card size="small" styles={{ body: { padding: 12 } }}>
              <Statistic title="今日新增" value={stats?.today_created ?? 0} />
            </Card>
          </div>
          <Card size="small" title="路由调试器" style={{ marginBottom: 12 }}>
            <Space direction="vertical" style={{ width: "100%" }} size={8}>
              <Alert
                type="info"
                showIcon
                message="调试针对全局路由树（project_id=0）。投递仍可能合并各项目旧订阅树。"
              />
              <Typography.Text type="secondary">
                模拟标签命中全局订阅树，查看接收组、通道与静默/维护窗口抑制。
              </Typography.Text>
              <Input value={routingSeverity} onChange={(e) => setRoutingSeverity(e.target.value)} placeholder="severity" style={{ width: 160 }} addonBefore="级别" />
              <Input.TextArea rows={4} value={routingLabelsJSON} onChange={(e) => setRoutingLabelsJSON(e.target.value)} placeholder='{"alertname":"..."}' />
              <Button
                type="primary"
                loading={routingDebugLoading}
                onClick={() => {
                  let labels: Record<string, string> = {};
                  try {
                    labels = JSON.parse(routingLabelsJSON) as Record<string, string>;
                  } catch {
                    message.error("labels JSON 无效");
                    return;
                  }
                  setRoutingDebugLoading(true);
                  void debugAlertRouting({ project_id: GLOBAL_ROUTING_PROJECT_ID, labels, severity: routingSeverity, status: "firing" })
                    .then(setRoutingDebugResult)
                    .finally(() => setRoutingDebugLoading(false));
                }}
              >
                运行路由调试
              </Button>
              {routingDebugResult ? (
                <Typography.Paragraph>
                  命中：{routingDebugResult.matched ? "是" : "否"}
                  {routingDebugResult.matched_from_project ? " · 来自当前项目" : ""}
                  {routingDebugResult.matched_from_global ? " · 来自全局(project_id=0)" : ""}
                  {routingDebugResult.matched_path ? ` · 路径 ${routingDebugResult.matched_path}` : ""}
                  {routingDebugResult.matched_node_names?.length
                    ? ` · 节点 ${routingDebugResult.matched_node_names.join(", ")}`
                    : ""}
                  {routingDebugResult.silenced ? " · 已抑制" : ""}
                  {routingDebugResult.channels?.length
                    ? ` · 通道 ${routingDebugResult.channels.map((c) => c.name).join("、")}`
                    : ""}
                  {routingDebugResult.hint ? (
                    <>
                      <br />
                      <Typography.Text type="warning">{routingDebugResult.hint}</Typography.Text>
                    </>
                  ) : null}
                </Typography.Paragraph>
              ) : null}
            </Space>
          </Card>
          <Card size="small" title="内部入站联调（K8s Event）" style={{ marginBottom: 12 }}>
            <Space direction="vertical" style={{ width: "100%" }} size={12}>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                Alertmanager Webhook 已下线。本面板用于模拟 K8s Event 转发载荷：{" "}
                <Typography.Text code>POST /api/v1/alerts/ingress/k8s-events</Typography.Text>，携带与配置一致的 Token（
                <Typography.Text code>X-Alert-Token</Typography.Text> / <Typography.Text code>X-Webhook-Token</Typography.Text>
                ）。主路径告警请在「规则中心」配置 PromQL。记录出现在「事件台」。
              </Typography.Paragraph>
              <Space wrap>
                <Input
                  style={{ width: 360 }}
                  value={webhookToken}
                  placeholder="Webhook Token（可为空）"
                  onChange={(e) => {
                    setWebhookTokenSensitivePickId(undefined);
                    setWebhookToken(e.target.value);
                  }}
                />
                <Select
                  style={{ width: 300 }}
                  allowClear
                  value={webhookTokenPick}
                  options={webhookTokenOptions.map((o) => ({
                    label: o.label,
                    value: o.sensitive && o.id != null ? o.id : o.value,
                  }))}
                  placeholder="从字典选择 webhook token"
                  onChange={(v) => {
                    void (async () => {
                      if (v === undefined || v === null || v === "") {
                        setWebhookTokenSensitivePickId(undefined);
                        setWebhookToken("");
                        return;
                      }
                      const byId = webhookTokenOptions.find((o) => o.sensitive && o.id === v);
                      if (byId?.id != null) {
                        const hide = message.loading("正在获取字典明文…", 0);
                        try {
                          const { value } = await revealDictEntryValue(byId.id);
                          setWebhookTokenSensitivePickId(byId.id);
                          setWebhookToken(value);
                        } catch (e) {
                          message.error(e instanceof Error ? e.message : String(e));
                        } finally {
                          hide();
                        }
                        return;
                      }
                      setWebhookTokenSensitivePickId(undefined);
                      setWebhookToken(String(v ?? ""));
                    })();
                  }}
                />
                <Button type="primary" loading={webhookSending} onClick={() => void sendWebhookDemo()}>
                  发送模拟Webhook
                </Button>
                <Select
                  style={{ width: 220 }}
                  value={webhookTemplate}
                  options={webhookTemplateOptions}
                  onChange={(v) => applyWebhookTemplate(v as "warning_prod" | "critical_prod" | "resolved_prod")}
                />
                <Button onClick={() => applyWebhookTemplate(webhookTemplate)}>套用模板</Button>
              </Space>
              <Typography.Text type="secondary">
                已选择模板会立即同步到下方 JSON；发送前请确认 status 和 alerts[0].status 是否符合预期（firing/resolved）。
              </Typography.Text>
              <Input.TextArea
                rows={8}
                value={webhookPayload}
                onChange={(e) => setWebhookPayload(e.target.value)}
                placeholder="Alertmanager webhook JSON"
              />
            </Space>
          </Card>
        </>
      ) : null}

      {hideTabs ? (
        activeContent
      ) : (
        <Tabs activeKey={tab} onChange={(k) => setTab(k as AlertConfigTab)} items={tabItems as never} />
      )}

      <Drawer
        title={ALERT_ROUTING_TERMS.receiverGroupManage}
        width={720}
        open={rgDrawerOpen}
        onClose={() => setRgDrawerOpen(false)}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={openReceiverGroupCreate}>
            新建接收组
          </Button>
        }
      >
        <Alert type="info" showIcon style={{ marginBottom: 12 }} message={ALERT_ROUTING_TERMS.receiverGroupManageHint} />
        {receiverGroupDuplicateCount > 0 ? (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 12 }}
            message={`检测到 ${receiverGroupDuplicateCount} 条同名重复接收组`}
            description="多半是「从项目复制路由模板」或策略迁移重复执行，且目标只清了订阅节点、残留了接收组。下拉已自动按同名去重；请保留较大 ID 的一组，删掉其余重复项。"
          />
        ) : null}
        <Table
          rowKey="id"
          size="small"
          loading={subLoading}
          dataSource={receiverGroups}
          pagination={false}
          columns={[
            { title: "ID", dataIndex: "id", width: 72 },
            { title: "名称", dataIndex: "name", width: 180, render: (n: string, r: AlertReceiverGroup) => formatReceiverGroupLabel(n, r.id) },
            {
              title: "告警通道",
              render: (_: unknown, r: AlertReceiverGroup) => {
                const ids = parseReceiverGroupChannelIds(r);
                if (!ids.length) return <Typography.Text type="secondary">未绑定</Typography.Text>;
                return ids
                  .map((id) => channels.find((c) => c.id === id)?.name ?? `#${id}`)
                  .join("、");
              },
            },
            {
              title: ALERT_ROUTING_TERMS.receiverGroupStaticCC,
              render: (_: unknown, r: AlertReceiverGroup) => {
                const emails = parseReceiverGroupEmails(r);
                return emails.length ? emails.join("、") : "—";
              },
            },
            {
              title: "升级层级",
              dataIndex: "escalation_level",
              width: 88,
              render: (v: number) => (v > 0 ? `L${v}` : "L0 初始"),
            },
            {
              title: "升级延迟",
              dataIndex: "escalation_delay_seconds",
              width: 100,
              render: (v: number, r: AlertReceiverGroup) => {
                if (!(r.escalation_level > 0)) return "—";
                const sec = v > 0 ? v : 900;
                return sec >= 60 ? `${Math.round(sec / 60)} 分钟` : `${sec} 秒`;
              },
            },
            {
              title: "状态",
              dataIndex: "enabled",
              width: 72,
              render: (v: boolean) => (v ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>),
            },
            {
              title: "操作",
              width: 120,
              render: (_: unknown, r: AlertReceiverGroup) => (
                <Space>
                  <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openReceiverGroupEdit(r)}>
                    编辑
                  </Button>
                  <Popconfirm title="确认删除该接收组？" onConfirm={() => void removeReceiverGroup(r.id)}>
                    <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Drawer>

      <Modal
        title={rgEditingId ? "编辑通知接收组" : "新建通知接收组"}
        open={rgModalOpen}
        confirmLoading={rgSaving}
        onCancel={() => setRgModalOpen(false)}
        onOk={() => void saveReceiverGroup()}
        destroyOnClose
      >
        <Form form={rgForm} layout="vertical">
          <Form.Item name="name" label="接收组名称" rules={[{ required: true, message: "请输入名称" }]}>
            <Input placeholder="例如 prod-critical-dingding" />
          </Form.Item>
          <Form.Item name="description" label="说明">
            <Input.TextArea rows={2} placeholder="可选" />
          </Form.Item>
          <Form.Item
            name="channel_ids"
            label="绑定告警通道"
            rules={[{ required: true, message: "请至少绑定一个告警通道" }]}
            extra="钉钉/企微：处理人手机号不在企业通讯录或无法 @ 时自动补邮件；wechat 等会补邮件。已在钉钉/企微群内可 @ 的处理人不重复发邮件。"
          >
            <Select
              mode="multiple"
              allowClear
              placeholder="选择钉钉 / 邮件 / 企微等通道"
              options={channels.map((c) => ({ label: c.name, value: c.id }))}
            />
          </Form.Item>
          <Form.Item
            name="email_recipients"
            label={ALERT_ROUTING_TERMS.receiverGroupStaticCC}
            extra="可选：除规则处理人外，额外抄送固定邮箱；不填则仅按处理人与通道投递。"
          >
            <Select mode="tags" tokenSeparators={[",", " ", ";"]} placeholder="可选，输入后回车" />
          </Form.Item>
          <Form.Item
            name="escalation_level"
            label="升级层级"
            extra="0=首次 firing 立即通知；1+=未认领且持续 firing 达到延迟后通知。同一订阅节点可绑定多个不同层级的接收组。"
            rules={[{ required: true, message: "请填写升级层级" }]}
          >
            <InputNumber min={0} max={10} precision={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.escalation_level !== cur.escalation_level}>
            {() =>
              Number(rgForm.getFieldValue("escalation_level") ?? 0) > 0 ? (
                <Form.Item
                  name="escalation_delay_seconds"
                  label="升级延迟（秒）"
                  extra="进入本层前等待秒数；填 0 时默认 900 秒（15 分钟）。认领/恢复/静默会取消待升级。"
                >
                  <InputNumber min={0} max={604800} precision={0} style={{ width: "100%" }} />
                </Form.Item>
              ) : null
            }
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={ALERT_ROUTING_TERMS.routingWizard}
        open={wizardOpen}
        confirmLoading={wizardSubmitting}
        onCancel={() => setWizardOpen(false)}
        footer={
          <Space>
            <Button onClick={() => setWizardOpen(false)}>取消</Button>
            <Button disabled={wizardStep === 0} onClick={() => setWizardStep((s) => Math.max(0, s - 1))}>
              上一步
            </Button>
            {wizardStep < 3 ? (
              <Button
                type="primary"
                onClick={async () => {
                  if (wizardStep === 0) {
                    await wizardForm.validateFields(["project_id"]);
                  } else if (wizardStep === 1) {
                    await wizardForm.validateFields(["severity"]);
                  } else if (wizardStep === 2) {
                    await wizardForm.validateFields(["channel_ids"]);
                  }
                  setWizardStep((s) => Math.min(3, s + 1));
                }}
              >
                下一步
              </Button>
            ) : (
              <Button
                type="primary"
                loading={wizardSubmitting}
                onClick={async () => {
                  const v = await wizardForm.validateFields();
                  const channelIds = (v.channel_ids ?? []).filter((id) => Number(id) > 0);
                  if (channelIds.length === 0) {
                    message.error("请至少选择一个通知通道");
                    setWizardStep(2);
                    return;
                  }
                  setWizardSubmitting(true);
                  try {
                    const res = await applyRoutingWizard({
                      project_id: v.project_id && v.project_id > 0 ? v.project_id : 0,
                      severity: v.severity ?? "",
                      channel_ids: channelIds,
                      extra_emails: v.extra_emails ?? [],
                      name: v.name?.trim() || undefined,
                    });
                    message.success(
                      `已创建路由节点「${res.node?.name ?? ""}」与接收组「${res.receiver_group?.name ?? ""}」`,
                    );
                    setWizardOpen(false);
                    await loadSubscriptions();
                    if (res.node?.id) {
                      void onSelectSubscriptionNode(res.node.id);
                    }
                  } finally {
                    setWizardSubmitting(false);
                  }
                }}
              >
                创建
              </Button>
            )}
          </Space>
        }
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          {ALERT_ROUTING_TERMS.routingWizardHint}
        </Typography.Paragraph>
        <Steps
          size="small"
          current={wizardStep}
          style={{ marginBottom: 16 }}
          items={[{ title: "项目" }, { title: "级别" }, { title: "通道" }, { title: "抄送" }]}
        />
        <Form form={wizardForm} layout="vertical">
          <div style={{ display: wizardStep === 0 ? "block" : "none" }}>
            <Form.Item
              name="project_id"
              label="匹配项目"
              extra="不选表示不按 project_id 过滤（仍可按级别匹配）。写入全局路由树。"
            >
              <Select
                allowClear
                showSearch
                optionFilterProp="label"
                placeholder="全部项目（不写 project_id）"
                options={projects.map((p) => ({ label: p.name, value: p.id }))}
              />
            </Form.Item>
            <Form.Item name="name" label="节点名称（可选）">
              <Input placeholder="留空则自动生成，例如「向导 · 项目 3 · warning」" />
            </Form.Item>
          </div>
          <div style={{ display: wizardStep === 1 ? "block" : "none" }}>
            <Form.Item name="severity" label={ALERT_ROUTING_TERMS.matchSeverity}>
              <Radio.Group
                options={[
                  { label: "全部级别", value: "" },
                  { label: "critical", value: "critical" },
                  { label: "warning", value: "warning" },
                  { label: "info", value: "info" },
                  { label: "critical + warning", value: "critical,warning" },
                ]}
              />
            </Form.Item>
          </div>
          <div style={{ display: wizardStep === 2 ? "block" : "none" }}>
            <Form.Item
              name="channel_ids"
              label="通知通道"
              rules={[{ required: true, type: "array", min: 1, message: "请至少选择一个通知通道" }]}
            >
              <Select
                mode="multiple"
                allowClear
                placeholder="钉钉 / 邮件 / 企微等"
                options={channels.map((c) => ({ label: c.name, value: c.id }))}
              />
            </Form.Item>
          </div>
          <div style={{ display: wizardStep === 3 ? "block" : "none" }}>
            <Form.Item
              name="extra_emails"
              label={ALERT_ROUTING_TERMS.receiverGroupStaticCC}
              extra="可选。规则处理人仍按规则配置投递；此处为接收组静态抄送。"
            >
              <Select mode="tags" tokenSeparators={[",", " ", ";"]} placeholder="可选，输入后回车" />
            </Form.Item>
          </div>
        </Form>
      </Modal>

      <Modal
        title={ALERT_ROUTING_TERMS.copyTemplate}
        open={cloneModalOpen}
        confirmLoading={cloneSubmitting}
        onCancel={() => setCloneModalOpen(false)}
        onOk={async () => {
          const v = await cloneForm.validateFields();
          setCloneSubmitting(true);
          try {
            const rep = await cloneSubscriptionFromProject({
              source_project_id: v.source_project_id,
              target_project_id: GLOBAL_ROUTING_PROJECT_ID,
              replace_cluster: v.replace_cluster?.trim() || undefined,
              replace_route: v.replace_route?.trim() || undefined,
              include_disabled: !!v.include_disabled,
              skip_if_target_has_nodes: v.skip_if_target_has_nodes !== false,
            });
            if (rep.skipped) {
              message.warning(rep.message || "目标项目已有节点，已跳过");
            } else {
              message.success(
                `已复制：接收组 ${rep.receiver_groups_created} 个，订阅节点 ${rep.nodes_created} 个${rep.message ? `（${rep.message}）` : ""}`,
              );
            }
            setCloneModalOpen(false);
            await loadSubscriptions();
          } finally {
            setCloneSubmitting(false);
          }
        }}
      >
        <Form form={cloneForm} layout="vertical">
          <Form.Item name="source_project_id" label="源项目（已调配好的模板）" rules={[{ required: true }]}>
            <Select options={projects.map((p) => ({ label: p.name, value: p.id }))} showSearch />
          </Form.Item>
          <Typography.Paragraph type="secondary">将复制到<strong>平台全局路由树</strong>（本页正在编辑的树）。</Typography.Paragraph>
          <Form.Item name="replace_cluster" label="覆盖 cluster（可选，写入 match_labels）">
            <Input placeholder="例如 腾讯云告警链路" />
          </Form.Item>
          <Form.Item name="replace_route" label="覆盖 route（可选）">
            <Input placeholder="例如 prod-critical-dingding" />
          </Form.Item>
          <Form.Item name="include_disabled" label="包含已停用节点/接收组" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item
            name="skip_if_target_has_nodes"
            label="目标已有订阅树时跳过（推荐）"
            valuePropName="checked"
            extra="关闭后将清空全局树已有订阅节点与接收组再复制（慎用）"
          >
            <Switch defaultChecked />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={fpExplain ? `指纹追溯：${fpExplain.fingerprint}` : "指纹追溯"}
        open={fpExplainOpen}
        onCancel={() => {
          setFpExplainOpen(false);
          setFpAiResult(null);
        }}
        footer={
          <Space>
            <Button
              type="primary"
              loading={fpAiLoading}
              disabled={!fpExplain?.fingerprint}
              onClick={async () => {
                if (!fpExplain?.fingerprint) return;
                setFpAiLoading(true);
                try {
                  const res = await analyzeAlertExplainAI({
                    fingerprint: fpExplain.fingerprint,
                    project_id: projectContextId,
                    window_hours: 24,
                  });
                  setFpAiResult(res);
                  message.success("AI 解释完成");
                } catch (e) {
                  message.error(extractApiErrorMessage(e, "AI 解释失败"));
                } finally {
                  setFpAiLoading(false);
                }
              }}
            >
              AI 解释
            </Button>
            <Button onClick={() => setFpExplainOpen(false)}>关闭</Button>
          </Space>
        }
        width={900}
        destroyOnClose
      >
        {fpExplain ? (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            <Alert
              type={fpExplain.firing_delivered ? "success" : "warning"}
              showIcon
              message={
                fpExplain.firing_delivered
                  ? `已有成功 firing 投递（来源：${fpExplain.firing_delivered_source || "-"}）`
                  : "尚未记录成功 firing 投递（恢复通知可能被抑制）"
              }
            />
            {fpAiResult ? (
              <Card size="small" title={`AI 解释（${fpAiResult.provider} / ${fpAiResult.model}）`}>
                <Space direction="vertical" style={{ width: "100%" }} size="small">
                  <Typography.Paragraph style={{ marginBottom: 0 }}>
                    {fpAiResult.ai_summary || "（无摘要）"}
                  </Typography.Paragraph>
                  {(fpAiResult.root_causes || []).map((c, i) => (
                    <Alert
                      key={`ai-cause-${i}`}
                      type="warning"
                      showIcon
                      message={String(c.title || c.cause || `原因 ${i + 1}`)}
                      description={String(c.evidence || c.detail || c.description || JSON.stringify(c))}
                    />
                  ))}
                  {(fpAiResult.actions || []).map((a, i) => (
                    <Alert
                      key={`ai-action-${i}`}
                      type="info"
                      showIcon
                      message={String(a.title || a.action || `建议 ${i + 1}`)}
                      description={String(a.command_hint || a.detail || a.description || JSON.stringify(a))}
                    />
                  ))}
                  {!fpAiResult.root_causes?.length && !fpAiResult.actions?.length && fpAiResult.raw_reply ? (
                    <pre style={{ maxHeight: 200, overflow: "auto", fontSize: 12, whiteSpace: "pre-wrap" }}>
                      {fpAiResult.raw_reply}
                    </pre>
                  ) : null}
                </Space>
              </Card>
            ) : null}
            {(fpExplain.skip_summary || []).length > 0 ? (
              <Card size="small" title="跳过/失败原因汇总">
                <Table
                  size="small"
                  pagination={false}
                  rowKey={(r) => r.error_message}
                  dataSource={fpExplain.skip_summary}
                  columns={[
                    { title: "原因码", dataIndex: "error_message", width: 220, ellipsis: true },
                    { title: "分类", dataIndex: "category", width: 100 },
                    { title: "次数", dataIndex: "count", width: 70 },
                    { title: "说明", dataIndex: "hint", ellipsis: true },
                  ]}
                />
              </Card>
            ) : null}
            <Card size="small" title="最近留痕（最多 200 条）">
              <Table
                size="small"
                pagination={tablePagination()}
                rowKey="id"
                dataSource={fpExplain.events || []}
                columns={[
                  { title: "时间", dataIndex: "created_at", width: 160 },
                  { title: "状态", dataIndex: "status", width: 80 },
                  { title: "通道", dataIndex: "channel_name", width: 140, ellipsis: true },
                  { title: "分类", dataIndex: "category", width: 90 },
                  { title: "原因码", dataIndex: "error_message", width: 180, ellipsis: true },
                  { title: "说明", dataIndex: "reason_hint", ellipsis: true },
                ]}
              />
            </Card>
          </Space>
        ) : null}
      </Modal>

    </>
  );

  if (embedded) {
    return (
      <div className="alert-config-embedded">
        <div style={{ display: "flex", justifyContent: "flex-end", marginBottom: 8 }}>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => void loadBase(effectiveProjectId > 0 ? effectiveProjectId : undefined)}
          >
            刷新统计
          </Button>
        </div>
        {body}
      </div>
    );
  }

  return (
    <Card
      className="table-card"
      title="告警配置中心"
      extra={
        <Button
          icon={<ReloadOutlined />}
          onClick={() => void loadBase(effectiveProjectId > 0 ? effectiveProjectId : undefined)}
        >
          刷新统计
        </Button>
      }
    >
      {body}
    </Card>
  );
}

