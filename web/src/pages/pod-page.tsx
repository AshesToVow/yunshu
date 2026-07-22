import { CodeOutlined, DeleteOutlined, DownOutlined, DownloadOutlined, EditOutlined, FileSearchOutlined, FileTextOutlined, FolderOpenOutlined, MedicineBoxOutlined, PlusOutlined, ReloadOutlined, UndoOutlined, UploadOutlined } from "@ant-design/icons";
import type { MenuProps } from "antd";
import { Alert, Button, Card, Checkbox, Divider, Drawer, Dropdown, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Tabs, Tooltip, Typography, message } from "antd";
import { useCallback, useEffect, useMemo, useRef, useState, type MouseEvent } from "react";
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import "xterm/css/xterm.css";
import { K8sPageToolbar } from "../components/ops/k8s-page-toolbar";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import { PhaseTag } from "../components/ops/phase-tag";
import { PodDetailPanel } from "../components/pod/pod-detail-panel";
import { PodLogsPanel } from "../components/pod/pod-logs-panel";
import { useK8sContext } from "../hooks/use-k8s-context";
import { useK8sClusterTier } from "../hooks/use-k8s-cluster-tier";
import { useK8sWatch } from "../hooks/use-k8s-watch";
import { useEditGuardStore } from "../stores/edit-guard-store";
import { formatDateTime } from "../utils/format";
import { K8sDeleteDialog } from "../components/k8s/k8s-delete-dialog";
import { MonacoYamlEditor, validateYaml } from "../components/k8s/monaco-yaml-editor";
import { RealtimeUsageText } from "../components/k8s/k8s-resource-usage-cells";
import type { K8sDeleteOptions } from "../services/service-factory";
import { createPodByYAML, createPodSimple, deletePod, deletePodFile, downloadPodFile, downloadPodLogs, getPodDetail, getPodDiagnose, getPodEvents, getPodLogs, getPods, listPodFiles, readPodFile, restartPod, updatePodSimple, uploadPodFile, type PodDetail, type PodDiagnoseResult, type PodEventItem, type PodFileItem, type PodItem, type PodLogsQuery } from "../services/pods";
import { getToken } from "../services/storage";
import { openAuthenticatedWebSocket } from "../services/ws-auth";
import { extractApiErrorMessage } from "../services/http";

const POD_CREATE_YAML_TEMPLATE = `apiVersion: v1
kind: Pod
metadata:
  name: demo-pod
spec:
  containers:
  - name: main
    image: nginx:latest
`;

export function PodPage() {
  const rfc1123Subdomain = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/;
  const rfc1123Label = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;
  const {
    clusterId,
    namespace = "default",
    setClusterId,
    setNamespace,
    clusterOptions,
    namespaceOptions,
  } = useK8sContext({ needNamespace: true, syncUrl: true });
  const { canExec, canMutate } = useK8sClusterTier(clusterId);
  const beginEdit = useEditGuardStore((s) => s.beginEdit);
  const endEdit = useEditGuardStore((s) => s.endEdit);
  const [keyword, setKeyword] = useState("");
  const [loading, setLoading] = useState(false);
  const [pods, setPods] = useState<PodItem[]>([]);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<PodItem | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const [selected, setSelected] = useState<PodItem | null>(null);
  const [detail, setDetail] = useState<PodDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [events, setEvents] = useState<PodEventItem[]>([]);
  const [detailTab, setDetailTab] = useState("overview");

  const [logsOpen, setLogsOpen] = useState(false);
  const [logsLoading, setLogsLoading] = useState(false);
  const [logsText, setLogsText] = useState("");
  const [logsTitle, setLogsTitle] = useState("");
  const [logsKeyword, setLogsKeyword] = useState("");
  const [logsStartTime, setLogsStartTime] = useState("");
  const [logsEndTime, setLogsEndTime] = useState("");
  const [logsPrevious, setLogsPrevious] = useState(false);
  const [logsTimestamps, setLogsTimestamps] = useState(false);
  const [logsSinceSeconds, setLogsSinceSeconds] = useState<number | undefined>();
  const [logsSinceTime, setLogsSinceTime] = useState("");
  const [logsContainer, setLogsContainer] = useState<string>();
  const [logContainerOptions, setLogContainerOptions] = useState<string[]>([]);
  const [diagnoseOpen, setDiagnoseOpen] = useState(false);
  const [diagnoseLoading, setDiagnoseLoading] = useState(false);
  const [diagnoseResult, setDiagnoseResult] = useState<PodDiagnoseResult | null>(null);
  const streamAbortRef = useRef<AbortController | null>(null);
  const prevPodKeyRef = useRef("");
  const [streaming, setStreaming] = useState(false);
  const [execOpen, setExecOpen] = useState(false);
  const [fileOpen, setFileOpen] = useState(false);
  const [filePath, setFilePath] = useState("/");
  const [fileList, setFileList] = useState<PodFileItem[]>([]);
  const [fileLoading, setFileLoading] = useState(false);
  const [fileContent, setFileContent] = useState("");
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [execCommand, setExecCommand] = useState("sh");
  const execCommandRef = useRef(execCommand);
  execCommandRef.current = execCommand;
  const filterRef = useRef({ clusterId, namespace, keyword });
  filterRef.current = { clusterId, namespace, keyword };
  const [watchLive, setWatchLive] = useState(false);
  const execTermHostRef = useRef<HTMLDivElement | null>(null);
  const execTermRef = useRef<Terminal | null>(null);
  const execFitRef = useRef<FitAddon | null>(null);
  const execWsRef = useRef<WebSocket | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [simpleMode, setSimpleMode] = useState<"create" | "edit">("create");
  const [editTarget, setEditTarget] = useState<PodItem | null>(null);
  const [simpleForm] = Form.useForm<{
    name: string;
    image: string;
    command?: string;
    container_name?: string;
    image_pull_policy?: "Always" | "IfNotPresent" | "Never";
    restart_policy?: "Always" | "OnFailure" | "Never";
    port?: number;
    env_pairs?: Array<{ key?: string; value?: string }>;
    label_pairs?: Array<{ key?: string; value?: string }>;
    requests_cpu?: string;
    requests_memory?: string;
    limits_cpu?: string;
    limits_memory?: string;
    tolerations?: Array<{
      key?: string;
      operator?: "Equal" | "Exists";
      value?: string;
      effect?: "NoSchedule" | "PreferNoSchedule" | "NoExecute";
      toleration_seconds?: number;
    }>;
    node_selector_pairs?: Array<{ key?: string; value?: string }>;
    priority_class_name?: string;
    affinity?: {
      node?: {
        required?: Array<{
          match_expressions?: Array<{
            key?: string;
            operator?: "In" | "NotIn" | "Exists" | "DoesNotExist" | "Gt" | "Lt";
            values?: string[];
          }>;
        }>;
        preferred?: Array<{
          weight?: number;
          match_expressions?: Array<{
            key?: string;
            operator?: "In" | "NotIn" | "Exists" | "DoesNotExist" | "Gt" | "Lt";
            values?: string[];
          }>;
        }>;
      };
      pod?: {
        required?: Array<{
          match_labels?: Array<{ key?: string; value?: string }>;
          topology_key?: string;
        }>;
        preferred?: Array<{
          weight?: number;
          match_labels?: Array<{ key?: string; value?: string }>;
          topology_key?: string;
        }>;
      };
      pod_anti?: {
        required?: Array<{
          match_labels?: Array<{ key?: string; value?: string }>;
          topology_key?: string;
        }>;
        preferred?: Array<{
          weight?: number;
          match_labels?: Array<{ key?: string; value?: string }>;
          topology_key?: string;
        }>;
      };
    };
  }>();
  const [yamlForm] = Form.useForm<{ manifest: string }>();

  const loadPods = useCallback(async (overrideKeyword?: string) => {
    const { clusterId: cid, namespace: ns, keyword: kw } = filterRef.current;
    if (!cid) {
      setPods([]);
      return;
    }
    setLoading(true);
    try {
      const effectiveKeyword = (overrideKeyword ?? kw).trim();
      const res = await getPods({ cluster_id: cid, namespace: ns, keyword: effectiveKeyword || undefined });
      setPods(res.list || []);
    } catch {
      // http 拦截器已 toast
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadPods();
  }, [clusterId, namespace, loadPods]);

  useK8sWatch({
    enabled: watchLive,
    clusterId,
    namespace,
    resource: "pods",
    onRefresh: loadPods,
    onDisabled: () => setWatchLive(false),
  });

  const anyPanelOpen =
    createOpen || execOpen || detailOpen || logsOpen || diagnoseOpen || fileOpen || deleteDialogOpen;
  const pauseWatch = anyPanelOpen || (detailTab === "logs" && streaming);

  useEffect(() => {
    if (!pauseWatch) return;
    beginEdit();
    return () => endEdit();
  }, [pauseWatch, beginEdit, endEdit]);

  async function handleDeletePod(record: PodItem, deleteOpts?: K8sDeleteOptions) {
    if (!clusterId) return;
    await deletePod({
      cluster_id: clusterId,
      namespace: record.namespace,
      name: record.name,
      ...deleteOpts,
    });
    message.success("Pod 已删除");
    await loadPods();
  }

  function buildPodLogsQuery(record: PodItem, tailLines?: number): PodLogsQuery {
    const q: PodLogsQuery = {
      cluster_id: clusterId!,
      namespace: record.namespace,
      name: record.name,
      container: logsContainer || undefined,
      previous: logsPrevious || undefined,
      timestamps: logsTimestamps || undefined,
      since_seconds: logsSinceSeconds,
      since_time: logsSinceTime || undefined,
      keyword: logsKeyword || undefined,
      start_time: logsStartTime || undefined,
      end_time: logsEndTime || undefined,
    };
    if (tailLines != null) q.tail_lines = tailLines;
    return q;
  }

  async function loadLogsForPod(record: PodItem, opts?: { resetFilters?: boolean; tailLines?: number }) {
    if (!clusterId) return;
    setLogsLoading(true);
    if (opts?.resetFilters) {
      setLogsText("");
      setLogsKeyword("");
      setLogsStartTime("");
      setLogsEndTime("");
      setLogsPrevious(false);
      setLogsTimestamps(false);
      setLogsSinceSeconds(undefined);
      setLogsSinceTime("");
    }
    setLogsTitle(`${record.namespace}/${record.name}`);
    try {
      const detailRes = await getPodDetail({ cluster_id: clusterId, namespace: record.namespace, name: record.name });
      const names = (detailRes.containers ?? []).map((c) => c.name).filter(Boolean);
      setLogContainerOptions(names);
      const container = logsContainer && names.includes(logsContainer) ? logsContainer : names[0];
      setLogsContainer(container);
      const res = await getPodLogs({
        ...buildPodLogsQuery(record, opts?.tailLines ?? 500),
        container,
      });
      setLogsText(res.logs || "");
    } finally {
      setLogsLoading(false);
    }
  }

  async function openPodLogsInline(record: PodItem) {
    prevPodKeyRef.current = `${record.namespace}/${record.name}`;
    setSelected(record);
    setDetailTab("logs");
    void loadDetail(record);
    await loadLogsForPod(record, { resetFilters: true, tailLines: 500 });
  }

  async function handleViewLogs(record: PodItem, mode: "inline" | "modal" = "modal") {
    if (!clusterId) return;
    setSelected(record);
    if (mode === "inline") {
      await openPodLogsInline(record);
      return;
    }
    setLogsOpen(true);
    await loadLogsForPod(record, { resetFilters: true, tailLines: 500 });
  }

  async function handleDiagnose(record: PodItem) {
    if (!clusterId) return;
    setSelected(record);
    setDiagnoseOpen(true);
    setDiagnoseLoading(true);
    setDiagnoseResult(null);
    try {
      const res = await getPodDiagnose({ cluster_id: clusterId, namespace: record.namespace, name: record.name });
      setDiagnoseResult(res);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "排障诊断失败"));
      setDiagnoseOpen(false);
    } finally {
      setDiagnoseLoading(false);
    }
  }

  async function handleFilterLogs() {
    if (!clusterId || !selected) return;
    setLogsLoading(true);
    try {
      const res = await getPodLogs(buildPodLogsQuery(selected, 1000));
      setLogsText(res.logs || "");
    } finally {
      setLogsLoading(false);
    }
  }

  async function handleDownloadLogs() {
    if (!clusterId || !selected) return;
    const blob = await downloadPodLogs(buildPodLogsQuery(selected, 2000));
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${selected.namespace}-${selected.name}.log`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    window.URL.revokeObjectURL(url);
  }

  async function startLogStream() {
    if (!clusterId || !selected) return;
    stopLogStream();
    const aborter = new AbortController();
    streamAbortRef.current = aborter;
    setStreaming(true);
    const token = getToken();
    const q = buildPodLogsQuery(selected, 50);
    const params = new URLSearchParams({
      cluster_id: String(clusterId),
      namespace: q.namespace,
      name: q.name,
      tail_lines: "50",
    });
    if (q.container) params.set("container", q.container);
    if (q.previous) params.set("previous", "true");
    if (q.timestamps) params.set("timestamps", "true");
    if (q.since_seconds) params.set("since_seconds", String(q.since_seconds));
    if (q.since_time) params.set("since_time", q.since_time);
    try {
      const resp = await fetch(`/api/v1/pods/logs/stream?${params.toString()}`, {
        headers: { Authorization: token ? `Bearer ${token}` : "" },
        signal: aborter.signal,
      });
      if (!resp.ok || !resp.body) throw new Error("日志流连接失败");
      const reader = resp.body.getReader();
      const decoder = new TextDecoder("utf-8");
      let buffer = "";
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";
        for (const line of lines) {
          if (line.startsWith("data: ")) setLogsText((prev) => `${prev}${line.slice(6)}\n`);
        }
      }
    } catch (error) {
      if ((error as Error).name !== "AbortError") message.error((error as Error).message || "日志流断开");
    } finally {
      setStreaming(false);
    }
  }

  function stopLogStream() {
    streamAbortRef.current?.abort();
    streamAbortRef.current = null;
    setStreaming(false);
  }

  useEffect(() => {
    stopLogStream();
    setLogsText("");
    setDetailTab("overview");
    prevPodKeyRef.current = "";
  }, [clusterId]);

  useEffect(() => {
    const key = selected ? `${selected.namespace}/${selected.name}` : "";
    if (prevPodKeyRef.current === key) return;
    prevPodKeyRef.current = key;
    stopLogStream();
    setLogsText("");
    setDetailTab("overview");
  }, [selected?.namespace, selected?.name]);

  async function loadDetail(record: PodItem) {
    if (!clusterId) return;
    setSelected(record);
    setDetailLoading(true);
    setDetail(null);
    setEvents([]);
    try {
      const [d, e] = await Promise.all([
        getPodDetail({ cluster_id: clusterId, namespace: record.namespace, name: record.name }),
        getPodEvents({ cluster_id: clusterId, namespace: record.namespace, name: record.name }),
      ]);
      setDetail(d);
      setEvents(e.list || []);
    } catch {
      setDetail(null);
      setEvents([]);
    } finally {
      setDetailLoading(false);
    }
  }

  function closeExecSocket() {
    try {
      execWsRef.current?.close();
    } catch {
      // ignore
    }
    execWsRef.current = null;
  }

  useEffect(() => {
    if (!execOpen) return;
    if (!clusterId || !selected) return;
    const host = execTermHostRef.current;
    if (!host) return;

    let disposed = false;
    let removeWindowResize: (() => void) | undefined;
    let onDataDispose: { dispose: () => void } | undefined;
    let onResizeDispose: { dispose: () => void } | undefined;

    host.innerHTML = "";

    const term = new Terminal({
      cursorBlink: true,
      fontFamily:
        'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
      fontSize: 12,
      theme: { background: "#141414" },
      scrollback: 5000,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(host);
    fit.fit();
    term.focus();

    execTermRef.current = term;
    execFitRef.current = fit;

    void (async () => {
      let container: string | undefined;
      if (detail?.namespace === selected.namespace && detail.name === selected.name) {
        container = detail.containers?.[0]?.name;
      } else {
        try {
          const d = await getPodDetail({
            cluster_id: clusterId,
            namespace: selected.namespace,
            name: selected.name,
          });
          container = d.containers?.[0]?.name;
        } catch {
          // 后端会解析默认容器；此处失败时不阻塞 Exec 连接
        }
      }

      let ws: WebSocket;
      try {
        ws = await openAuthenticatedWebSocket(
          "/api/v1/pods/exec/ws",
          {
            cluster_id: clusterId,
            namespace: selected.namespace,
            name: selected.name,
            ...(container ? { container } : {}),
          },
          "pod-exec",
        );
      } catch {
        if (!disposed) term.writeln("\r\n[connection error: ticket failed]\r\n");
        return;
      }
      if (disposed) {
        ws.close();
        return;
      }
      execWsRef.current = ws;

      const sendResize = () => {
        try {
          const cols = term.cols;
          const rows = term.rows;
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: "resize", cols, rows }));
          }
        } catch {
          // ignore
        }
      };

      onDataDispose = term.onData((data) => {
        if (ws.readyState !== WebSocket.OPEN) return;
        ws.send(JSON.stringify({ type: "input", data }));
      });
      onResizeDispose = term.onResize(({ cols, rows }) => {
        if (ws.readyState !== WebSocket.OPEN) return;
        ws.send(JSON.stringify({ type: "resize", cols, rows }));
      });

      ws.onopen = () => {
        term.writeln(`Connected: ${selected.namespace}/${selected.name}`);
        sendResize();
        ws.send(JSON.stringify({ type: "input", data: `${execCommandRef.current}\n` }));
      };
      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(String(ev.data));
          if (msg.type === "stdout" && typeof msg.data === "string") {
            term.write(msg.data);
          } else if (msg.type === "error") {
            term.writeln(`\r\n[error] ${msg.data || "unknown"}`);
          } else if (msg.type === "exit") {
            term.writeln("\r\n[disconnected]");
          }
        } catch {
          term.write(String(ev.data));
        }
      };
      ws.onclose = () => {
        term.writeln("\r\n[connection closed]");
      };
      ws.onerror = () => {
        term.writeln("\r\n[connection error]");
      };

      const onWindowResize = () => {
        try {
          fit.fit();
          sendResize();
        } catch {
          // ignore
        }
      };
      window.addEventListener("resize", onWindowResize);
      removeWindowResize = () => window.removeEventListener("resize", onWindowResize);
    })();

    return () => {
      disposed = true;
      removeWindowResize?.();
      try {
        onDataDispose?.dispose();
        onResizeDispose?.dispose();
      } catch {
        // ignore
      }
      closeExecSocket();
      try {
        term.dispose();
      } catch {
        // ignore
      }
      execTermRef.current = null;
      execFitRef.current = null;
    };
  }, [execOpen, clusterId, selected?.namespace, selected?.name, detail?.namespace, detail?.name, detail?.containers]);

  async function handleRestartPod(record: PodItem) {
    if (!clusterId) return;
    await restartPod({ cluster_id: clusterId, namespace: record.namespace, name: record.name });
    message.success("Pod 已重启");
    await loadPods();
  }

  async function loadFiles(target: PodItem, path: string) {
    if (!clusterId) return;
    setFileLoading(true);
    try {
      const res = await listPodFiles({
        cluster_id: clusterId,
        namespace: target.namespace,
        name: target.name,
        path: path || "/",
      });
      setFileList(res.list || []);
      setFilePath(path || "/");
    } finally {
      setFileLoading(false);
    }
  }

  async function submitCreateSimple() {
    if (!clusterId) return;
    const values = await simpleForm.validateFields();
    const env: Record<string, string> = {};
    const labels: Record<string, string> = {};
    const nodeSelector: Record<string, string> = {};
    (values.env_pairs || []).forEach((item) => {
      const k = (item?.key || "").trim();
      const v = (item?.value || "").trim();
      if (k) env[k] = v;
    });
    (values.label_pairs || []).forEach((item) => {
      const k = (item?.key || "").trim();
      const v = (item?.value || "").trim();
      if (k) labels[k] = v;
    });
    (values.node_selector_pairs || []).forEach((item) => {
      const k = (item?.key || "").trim();
      const v = (item?.value || "").trim();
      if (k) nodeSelector[k] = v;
    });
    let affinityObj: Record<string, unknown> | undefined;
    if (values.affinity) {
      const a = values.affinity;
      const nodeRequiredTerms =
        a.node?.required
          ?.map((t) => ({
            matchExpressions: (t.match_expressions || [])
              .map((e) => ({
                key: (e.key || "").trim(),
                operator: e.operator,
                values: (e.values || []).map((v) => String(v).trim()).filter(Boolean),
              }))
              .filter((e) => e.key && e.operator),
          }))
          .filter((t) => t.matchExpressions.length > 0) || [];

      const nodePreferred =
        a.node?.preferred
          ?.map((p) => ({
            weight: Math.min(100, Math.max(1, Number(p.weight || 1))),
            preference: {
              matchExpressions: (p.match_expressions || [])
                .map((e) => ({
                  key: (e.key || "").trim(),
                  operator: e.operator,
                  values: (e.values || []).map((v) => String(v).trim()).filter(Boolean),
                }))
                .filter((e) => e.key && e.operator),
            },
          }))
          .filter((p) => p.preference.matchExpressions.length > 0) || [];

      const buildPodAffinityTerms = (
        list?: Array<{ match_labels?: Array<{ key?: string; value?: string }>; topology_key?: string }>,
      ) =>
        (list || [])
          .map((it) => {
            const labels = (it.match_labels || [])
              .map((kv) => ({ key: (kv.key || "").trim(), value: (kv.value || "").trim() }))
              .filter((kv) => kv.key);
            const matchLabels: Record<string, string> = {};
            labels.forEach((kv) => {
              matchLabels[kv.key] = kv.value;
            });
            const topologyKey = (it.topology_key || "").trim();
            if (!topologyKey || Object.keys(matchLabels).length === 0) return null;
            return {
              labelSelector: { matchLabels },
              topologyKey,
            };
          })
          .filter(Boolean);

      const buildPodPreferredTerms = (
        list?: Array<{ weight?: number; match_labels?: Array<{ key?: string; value?: string }>; topology_key?: string }>,
      ) =>
        (list || [])
          .map((it) => {
            const labels = (it.match_labels || [])
              .map((kv) => ({ key: (kv.key || "").trim(), value: (kv.value || "").trim() }))
              .filter((kv) => kv.key);
            const matchLabels: Record<string, string> = {};
            labels.forEach((kv) => {
              matchLabels[kv.key] = kv.value;
            });
            const topologyKey = (it.topology_key || "").trim();
            if (!topologyKey || Object.keys(matchLabels).length === 0) return null;
            return {
              weight: Math.min(100, Math.max(1, Number(it.weight || 1))),
              podAffinityTerm: {
                labelSelector: { matchLabels },
                topologyKey,
              },
            };
          })
          .filter(Boolean);

      const podRequired = buildPodAffinityTerms(a.pod?.required);
      const podPreferred = buildPodPreferredTerms(a.pod?.preferred);
      const podAntiRequired = buildPodAffinityTerms(a.pod_anti?.required);
      const podAntiPreferred = buildPodPreferredTerms(a.pod_anti?.preferred);

      const affinity: any = {};
      if (nodeRequiredTerms.length > 0 || nodePreferred.length > 0) {
        affinity.nodeAffinity = {};
        if (nodeRequiredTerms.length > 0) {
          affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution = {
            nodeSelectorTerms: nodeRequiredTerms,
          };
        }
        if (nodePreferred.length > 0) {
          affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution = nodePreferred;
        }
      }
      if ((podRequired as any[]).length > 0 || (podPreferred as any[]).length > 0) {
        affinity.podAffinity = {};
        if ((podRequired as any[]).length > 0) affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution = podRequired;
        if ((podPreferred as any[]).length > 0) affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution = podPreferred;
      }
      if ((podAntiRequired as any[]).length > 0 || (podAntiPreferred as any[]).length > 0) {
        affinity.podAntiAffinity = {};
        if ((podAntiRequired as any[]).length > 0) affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution = podAntiRequired;
        if ((podAntiPreferred as any[]).length > 0) affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution = podAntiPreferred;
      }
      if (Object.keys(affinity).length > 0) affinityObj = affinity;
    }
    setCreating(true);
    try {
      const payload = {
        cluster_id: clusterId,
        namespace,
        name: values.name,
        image: values.image,
        command: values.command,
        container_name: values.container_name,
        image_pull_policy: values.image_pull_policy,
        restart_policy: values.restart_policy,
        port: values.port,
        env: Object.keys(env).length > 0 ? env : undefined,
        labels: Object.keys(labels).length > 0 ? labels : undefined,
        requests_cpu: values.requests_cpu,
        requests_memory: values.requests_memory,
        limits_cpu: values.limits_cpu,
        limits_memory: values.limits_memory,
        node_selector: Object.keys(nodeSelector).length > 0 ? nodeSelector : undefined,
        priority_class_name: values.priority_class_name,
        affinity: affinityObj,
        tolerations: (values.tolerations || [])
          .filter((item) => (item.key || "").trim() !== "")
          .map((item) => ({
            key: (item.key || "").trim(),
            operator: item.operator || "Equal",
            value: (item.value || "").trim(),
            effect: item.effect,
            toleration_seconds: item.toleration_seconds,
          })),
      };
      if (simpleMode === "edit") {
        await updatePodSimple(payload);
        message.success("Pod 已更新并重建");
      } else {
        await createPodSimple(payload);
      message.success("Pod 创建成功");
      }
      setCreateOpen(false);
      await loadPods();
    } finally {
      setCreating(false);
    }
  }

  async function openEditPod(record: PodItem) {
    if (!clusterId) return;
    const d = await getPodDetail({ cluster_id: clusterId, namespace: record.namespace, name: record.name });
    setSimpleMode("edit");
    setEditTarget(record);
    setCreateOpen(true);
    simpleForm.setFieldsValue({
      name: d.name,
      container_name: d.containers?.[0]?.name || "",
      image: d.containers?.[0]?.image || "",
      command: "",
      image_pull_policy: "IfNotPresent",
      restart_policy: "Always",
      env_pairs: [],
      label_pairs: Object.entries(d.labels || {}).map(([key, value]) => ({ key, value })),
      node_selector_pairs: Object.entries(d.node_selector || {}).map(([key, value]) => ({ key, value })),
      priority_class_name: d.priority_class_name || "",
      affinity: (() => {
        const a: any = d.affinity || {};
        const out: any = {};
        if (a.nodeAffinity) {
          const na: any = {};
          const reqTerms = a.nodeAffinity?.requiredDuringSchedulingIgnoredDuringExecution?.nodeSelectorTerms || [];
          na.required = reqTerms.map((t: any) => ({
            match_expressions: (t.matchExpressions || []).map((e: any) => ({
              key: e.key,
              operator: e.operator,
              values: e.values || [],
            })),
          }));
          const pref = a.nodeAffinity?.preferredDuringSchedulingIgnoredDuringExecution || [];
          na.preferred = pref.map((p: any) => ({
            weight: p.weight,
            match_expressions: (p.preference?.matchExpressions || []).map((e: any) => ({
              key: e.key,
              operator: e.operator,
              values: e.values || [],
            })),
          }));
          out.node = na;
        }
        function parsePodTerms(list: any[]) {
          return (list || []).map((t: any) => {
            const ml = t.labelSelector?.matchLabels || {};
            return {
              topology_key: t.topologyKey,
              match_labels: Object.entries(ml).map(([key, value]) => ({ key, value })),
            };
          });
        }
        function parsePodPreferred(list: any[]) {
          return (list || []).map((p: any) => {
            const term = p.podAffinityTerm || {};
            const ml = term.labelSelector?.matchLabels || {};
            return {
              weight: p.weight,
              topology_key: term.topologyKey,
              match_labels: Object.entries(ml).map(([key, value]) => ({ key, value })),
            };
          });
        }
        if (a.podAffinity) {
          out.pod = {
            required: parsePodTerms(a.podAffinity.requiredDuringSchedulingIgnoredDuringExecution || []),
            preferred: parsePodPreferred(a.podAffinity.preferredDuringSchedulingIgnoredDuringExecution || []),
          };
        }
        if (a.podAntiAffinity) {
          out.pod_anti = {
            required: parsePodTerms(a.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution || []),
            preferred: parsePodPreferred(a.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution || []),
          };
        }
        return out;
      })(),
      tolerations: (d.tolerations || []).map((t) => ({
        key: t.key,
        operator: t.operator,
        value: t.value,
        effect: t.effect,
        toleration_seconds: t.tolerationSeconds,
      })),
    });
  }

  async function submitCreateYAML() {
    if (!clusterId) return;
    const values = await yamlForm.validateFields();
    if (validateYaml(values.manifest)) {
      message.warning("请先修正 YAML 语法错误");
      return;
    }
    setCreating(true);
    try {
      await createPodByYAML({ cluster_id: clusterId, namespace, manifest: values.manifest });
      message.success("YAML 创建成功");
      setCreateOpen(false);
      await loadPods();
    } finally {
      setCreating(false);
    }
  }

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="Pod 管理"
        description="跨集群 Pod 生命周期、诊断、日志与终端会话"
        breadcrumbs={[{ title: "Kubernetes" }, { title: "Pod" }]}
        meta={
          <span>
            {clusterId ? `集群 #${clusterId} · ${namespace}` : "未选择集群"} · {loading ? "加载中" : `${pods.length} 条`}
          </span>
        }
      />
      <K8sPageToolbar
        clusterId={clusterId}
        namespace={namespace}
        clusterOptions={clusterOptions}
        namespaceOptions={namespaceOptions}
        searchPlaceholder="搜索 Pod 名称/节点"
        onClusterChange={setClusterId}
        onNamespaceChange={setNamespace}
        onSearch={(v) => {
          setKeyword(v);
          void loadPods(v);
        }}
        onRefresh={() => void loadPods()}
        watchLive={watchLive}
        onWatchChange={setWatchLive}
        primaryAction={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setSimpleMode("create");
              setEditTarget(null);
              simpleForm.resetFields();
              yamlForm.resetFields();
              setCreateOpen(true);
            }}
          >
            创建 Pod
          </Button>
        }
      />
      <div className="pod-workbench">
        <div className="pod-workbench__list">
          <Table
            size="small"
            rowKey={(r) => `${r.namespace}/${r.name}`}
            loading={loading}
            dataSource={pods}
            pagination={{ pageSize: 10, showSizeChanger: true, size: "small" }}
            scroll={{ x: 1200 }}
            onRow={(record) => ({ onClick: () => void loadDetail(record) })}
            rowClassName={(record) =>
              selected && `${record.namespace}/${record.name}` === `${selected.namespace}/${selected.name}`
                ? "ant-table-row-selected"
                : ""
            }
            columns={[
              { title: "Pod 名称", dataIndex: "name", width: 160, fixed: "left", ellipsis: true },
              { title: "节点", dataIndex: "node_name", width: 120, ellipsis: true },
              {
                title: "镜像",
                dataIndex: "containers_text",
                width: 200,
                ellipsis: true,
                render: (v?: string) => v || "-",
              },
              { title: "重启", dataIndex: "restart_count", width: 64 },
              {
                title: "用量",
                key: "usage_rt",
                width: 110,
                render: (_: unknown, r: PodItem) => (
                  <Tooltip title="CPU / 内存">
                    <RealtimeUsageText cpu={r.cpu_usage} mem={r.mem_usage} />
                  </Tooltip>
                ),
              },
              { title: "状态", dataIndex: "phase", width: 96, render: (phase: string) => <PhaseTag phase={phase} /> },
              { title: "启动时间", dataIndex: "start_time", width: 150, render: (v: string) => formatDateTime(v) },
              {
                title: "操作",
                key: "action",
                width: 140,
                fixed: "right",
                render: (_: unknown, record: PodItem) => {
                  const stop = (e: MouseEvent) => e.stopPropagation();
                  const moreItems: MenuProps["items"] = [
                    canMutate
                      ? {
                          key: "edit",
                          icon: <EditOutlined />,
                          label: "高级编辑",
                          onClick: () => void openEditPod(record),
                        }
                      : null,
                    {
                      key: "diagnose",
                      icon: <MedicineBoxOutlined />,
                      label: "排障诊断",
                      onClick: () => void handleDiagnose(record),
                    },
                    canExec
                      ? {
                          key: "files",
                          icon: <FolderOpenOutlined />,
                          label: "文件",
                          onClick: () => {
                            setSelected(record);
                            setFileOpen(true);
                            setFileContent("");
                            void loadFiles(record, "/");
                          },
                        }
                      : null,
                    canExec
                      ? {
                          key: "exec",
                          icon: <CodeOutlined />,
                          label: "Exec",
                          onClick: () => {
                            setSelected(record);
                            setExecOpen(true);
                          },
                        }
                      : null,
                    canMutate
                      ? {
                          key: "restart",
                          icon: <UndoOutlined />,
                          label: "重启",
                          onClick: () => void handleRestartPod(record),
                        }
                      : null,
                    canMutate ? { type: "divider" } : null,
                    canMutate
                      ? {
                          key: "delete",
                          danger: true,
                          icon: <DeleteOutlined />,
                          label: "删除",
                          onClick: () => {
                            setDeleteTarget(record);
                            setDeleteDialogOpen(true);
                          },
                        }
                      : null,
                  ].filter(Boolean) as MenuProps["items"];
                  return (
                    <Space size={0} wrap onClick={stop}>
                      <Button type="link" size="small" icon={<FileTextOutlined />} onClick={() => void openPodLogsInline(record)}>
                        日志
                      </Button>
                      <Dropdown menu={{ items: moreItems }} trigger={["click"]}>
                        <Button type="link" size="small" onClick={stop}>
                          更多 <DownOutlined />
                        </Button>
                      </Dropdown>
                    </Space>
                  );
                },
              },
            ]}
          />
        </div>
        <div className="pod-workbench__detail">
          <PodDetailPanel
            selected={selected}
            detail={detail}
            events={events}
            loading={detailLoading}
            activeTab={detailTab}
            onTabChange={(key) => {
              setDetailTab(key);
              if (key === "logs" && selected && !logsText && !logsLoading) {
                void loadLogsForPod(selected, { tailLines: 500 });
              }
            }}
            logsPanel={
              selected ? (
                <PodLogsPanel
                  loading={logsLoading}
                  streaming={streaming}
                  logsText={logsText}
                  containerOptions={logContainerOptions}
                  container={logsContainer}
                  onContainerChange={(v) => setLogsContainer(v)}
                  onFetch={() => selected && void loadLogsForPod(selected, { tailLines: 1000 })}
                  onDownload={() => void handleDownloadLogs()}
                  onStartStream={() => void startLogStream()}
                  onStopStream={stopLogStream}
                />
              ) : null
            }
            onExec={
              selected && canExec
                ? () => {
                    setExecOpen(true);
                  }
                : undefined
            }
            onDiagnose={selected ? () => void handleDiagnose(selected) : undefined}
            onFiles={
              selected && canExec
                ? () => {
                    setFileOpen(true);
                    setFileContent("");
                    void loadFiles(selected, "/");
                  }
                : undefined
            }
            onRestart={selected && canMutate ? () => void handleRestartPod(selected) : undefined}
            onDelete={
              selected && canMutate
                ? () => {
                    setDeleteTarget(selected);
                    setDeleteDialogOpen(true);
                  }
                : undefined
            }
            onEdit={selected ? () => void openEditPod(selected) : undefined}
            onExpand={
              selected
                ? () => {
                    void loadDetail(selected);
                    setDetailOpen(true);
                  }
                : undefined
            }
          />
        </div>
      </div>

      {selected && detailTab === "logs" ? (
        <Typography.Paragraph type="secondary" style={{ marginTop: 8, marginBottom: 0, fontSize: 12 }}>
          需要关键字、时间范围等高级筛选？
          <Button type="link" size="small" style={{ paddingInline: 4 }} onClick={() => void handleViewLogs(selected, "modal")}>
            打开日志对话框
          </Button>
        </Typography.Paragraph>
      ) : null}

      {/* 日志高级筛选对话框 */}
      <Modal
        title={`Pod 日志 - ${logsTitle}`}
        open={logsOpen}
        onCancel={() => {
          stopLogStream();
          setLogsOpen(false);
        }}
        footer={null}
        width={980}
      >
        <Space wrap style={{ marginBottom: 12 }}>
          {logContainerOptions.length > 1 ? (
            <Select
              allowClear
              placeholder="容器"
              style={{ width: 140 }}
              value={logsContainer}
              options={logContainerOptions.map((n) => ({ label: n, value: n }))}
              onChange={(v) => setLogsContainer(v)}
            />
          ) : null}
          <Select
            allowClear
            placeholder="最近时间"
            style={{ width: 130 }}
            value={logsSinceSeconds}
            options={[
              { label: "5 分钟", value: 300 },
              { label: "1 小时", value: 3600 },
              { label: "6 小时", value: 21600 },
              { label: "24 小时", value: 86400 },
            ]}
            onChange={(v) => setLogsSinceSeconds(v)}
          />
          <Input placeholder="since-time RFC3339" value={logsSinceTime} onChange={(e) => setLogsSinceTime(e.target.value)} style={{ width: 200 }} />
          <Checkbox checked={logsPrevious} onChange={(e) => setLogsPrevious(e.target.checked)}>上一实例</Checkbox>
          <Switch size="small" checked={logsTimestamps} onChange={setLogsTimestamps} checkedChildren="时间戳" unCheckedChildren="时间戳" />
          <Input placeholder="关键字过滤" value={logsKeyword} onChange={(e) => setLogsKeyword(e.target.value)} style={{ width: 160 }} />
          <Input placeholder="开始时间 2026-01-02 15:04:05" value={logsStartTime} onChange={(e) => setLogsStartTime(e.target.value)} style={{ width: 210 }} />
          <Input placeholder="结束时间 2026-01-02 15:04:05" value={logsEndTime} onChange={(e) => setLogsEndTime(e.target.value)} style={{ width: 210 }} />
          <Button icon={<FileSearchOutlined />} onClick={() => void handleFilterLogs()}>拉取/过滤</Button>
          <Button icon={<DownloadOutlined />} onClick={() => void handleDownloadLogs()}>下载</Button>
          <Button type={streaming ? "default" : "primary"} onClick={() => void startLogStream()} disabled={streaming}>开始实时流</Button>
          <Button danger={streaming} onClick={stopLogStream} disabled={!streaming}>停止流</Button>
          <Button onClick={() => selected && void handleViewLogs(selected)}>获取当前日志</Button>
        </Space>
        {logsLoading ? (
          <Typography.Text>日志加载中...</Typography.Text>
        ) : (
          <pre className="code-block-panel">{logsText || "暂无日志"}</pre>
        )}
      </Modal>
      <Drawer
        title={selected ? `Pod 排障 - ${selected.namespace}/${selected.name}` : "Pod 排障"}
        open={diagnoseOpen}
        onClose={() => setDiagnoseOpen(false)}
        width={920}
      >
        {diagnoseLoading ? (
          <Typography.Text>诊断中...</Typography.Text>
        ) : diagnoseResult ? (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            <Alert
              type={diagnoseResult.ready ? "success" : "warning"}
              showIcon
              message={diagnoseResult.summary}
              description={`阶段 ${diagnoseResult.phase} · 节点 ${diagnoseResult.node_name || "-"} · ${diagnoseResult.ready ? "已就绪" : "未就绪"}`}
            />
            {diagnoseResult.hints.map((h, i) => (
              <Alert
                key={`${h.title}-${i}`}
                type={h.level === "error" ? "error" : h.level === "warning" ? "warning" : "info"}
                showIcon
                message={h.title}
                description={
                  <>
                    <div>{h.detail}</div>
                    {h.action ? <Typography.Text type="secondary">建议：{h.action}</Typography.Text> : null}
                  </>
                }
              />
            ))}
            <Typography.Text strong>容器状态</Typography.Text>
            <Table
              size="small"
              rowKey="name"
              pagination={false}
              dataSource={diagnoseResult.containers}
              columns={[
                { title: "容器", dataIndex: "name", width: 120 },
                { title: "状态", dataIndex: "state", width: 90 },
                { title: "原因", dataIndex: "reason", width: 140 },
                { title: "重启", dataIndex: "restart_count", width: 70 },
                {
                  title: "日志片段",
                  dataIndex: "log_snippet",
                  render: (v: string) =>
                    v ? (
                      <pre className="code-block-panel" style={{ maxHeight: 120, fontSize: 11, padding: 8 }}>
                        {v}
                      </pre>
                    ) : (
                      "-"
                    ),
                },
              ]}
            />
            <Typography.Text strong>相关事件</Typography.Text>
            <Table
              size="small"
              rowKey={(r) => `${r.reason}-${r.last_timestamp}`}
              pagination={{ pageSize: 5 }}
              dataSource={diagnoseResult.events}
              columns={[
                { title: "类型", dataIndex: "type", width: 70 },
                { title: "原因", dataIndex: "reason", width: 120 },
                { title: "消息", dataIndex: "message", ellipsis: true },
              ]}
            />
          </Space>
        ) : (
          <Typography.Text type="secondary">暂无诊断数据</Typography.Text>
        )}
      </Drawer>
      <Drawer
        title={selected ? `Pod 文件管理 - ${selected.namespace}/${selected.name}` : "Pod 文件管理"}
        open={fileOpen}
        onClose={() => setFileOpen(false)}
        width={920}
      >
        <Space wrap style={{ marginBottom: 12 }}>
          <Input
            value={filePath}
            onChange={(e) => setFilePath(e.target.value)}
            placeholder="目录路径，例如 / /tmp /var/log"
            style={{ width: 360 }}
          />
          <Button onClick={() => selected && void loadFiles(selected, filePath)} icon={<ReloadOutlined />}>
            刷新目录
          </Button>
          <Button
            icon={<UploadOutlined />}
            onClick={() => fileInputRef.current?.click()}
            disabled={!selected}
          >
            上传到当前目录
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            style={{ display: "none" }}
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (!f || !selected || !clusterId) return;
              void (async () => {
                await uploadPodFile({
                  cluster_id: clusterId,
                  namespace: selected.namespace,
                  name: selected.name,
                  path: filePath || "/",
                  file: f,
                });
                message.success("上传成功");
                await loadFiles(selected, filePath);
              })();
              e.currentTarget.value = "";
            }}
          />
        </Space>
        <Table
          rowKey={(r) => r.path}
          loading={fileLoading}
          dataSource={fileList}
          size="small"
          pagination={{ pageSize: 8 }}
          columns={[
            { title: "名称", dataIndex: "name" },
            { title: "类型", dataIndex: "type", width: 100 },
            { title: "大小", dataIndex: "size", width: 110 },
            { title: "权限", dataIndex: "permissions", width: 120 },
            { title: "修改时间", dataIndex: "mod_time", width: 140 },
            {
              title: "操作",
              width: 280,
              render: (_: unknown, row: PodFileItem) => (
                <Space>
                  {row.is_dir ? (
                    <Button type="link" onClick={() => selected && void loadFiles(selected, row.path)}>
                      进入
                    </Button>
                  ) : (
                    <>
                      <Button
                        type="link"
                        onClick={() => {
                          if (!selected || !clusterId) return;
                          void (async () => {
                            const res = await readPodFile({
                              cluster_id: clusterId,
                              namespace: selected.namespace,
                              name: selected.name,
                              path: row.path,
                            });
                            setFileContent(res.content || "");
                          })();
                        }}
                      >
                        查看
                      </Button>
                      <Button
                        type="link"
                        icon={<DownloadOutlined />}
                        onClick={() => {
                          if (!selected || !clusterId) return;
                          void (async () => {
                            const blob = await downloadPodFile({
                              cluster_id: clusterId,
                              namespace: selected.namespace,
                              name: selected.name,
                              path: row.path,
                            });
                            const url = window.URL.createObjectURL(blob);
                            const a = document.createElement("a");
                            a.href = url;
                            a.download = row.name;
                            document.body.appendChild(a);
                            a.click();
                            a.remove();
                            window.URL.revokeObjectURL(url);
                          })();
                        }}
                      >
                        下载
                      </Button>
                    </>
                  )}
                  <Popconfirm
                    title={`确认删除 ${row.path} ?`}
                    onConfirm={() => {
                      if (!selected || !clusterId) return;
                      void (async () => {
                        await deletePodFile({
                          cluster_id: clusterId,
                          namespace: selected.namespace,
                          name: selected.name,
                          path: row.path,
                        });
                        message.success("删除成功");
                        await loadFiles(selected, filePath);
                      })();
                    }}
                  >
                    <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
        <Divider />
        <Typography.Text strong>文件内容预览</Typography.Text>
        <Input.TextArea rows={14} value={fileContent} readOnly style={{ marginTop: 8 }} placeholder="点击“查看”显示文本内容" />
      </Drawer>
      <Drawer
        title={selected ? `Exec 进入容器 - ${selected.namespace}/${selected.name}` : "Exec 进入容器"}
        open={execOpen}
        onClose={() => {
          setExecOpen(false);
          closeExecSocket();
        }}
        width={760}
      >
        <div style={{ display: "flex", gap: 12, marginBottom: 10 }}>
          <Input
            value={execCommand}
            onChange={(e) => setExecCommand(e.target.value)}
            placeholder="启动命令（默认 sh），例如：bash"
            style={{ maxWidth: 320 }}
          />
          <Button
            icon={<CodeOutlined />}
            onClick={() => {
              // reopen to restart session
              closeExecSocket();
              execTermRef.current?.reset();
              if (execTermRef.current) execTermRef.current.writeln("\r\n[reconnecting…]");
              // trigger effect by toggling
              setExecOpen(false);
              setTimeout(() => setExecOpen(true), 0);
            }}
          >
            重连
          </Button>
        </div>
        <div
          ref={execTermHostRef}
          style={{
            height: "calc(100vh - 230px)",
            maxHeight: 720,
            borderRadius: 12,
            overflow: "hidden",
            border: "1px solid rgba(142, 162, 192, 0.28)",
          }}
        />
      </Drawer>
      <Drawer
        title={selected ? `Pod 详情 - ${selected.namespace}/${selected.name}` : "Pod 详情"}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        width={760}
        className="detail-edit-drawer"
      >
        {!detail ? (
          <Typography.Text type="secondary">加载详情中...</Typography.Text>
        ) : (
          <Space direction="vertical" size={12} style={{ width: "100%" }}>
            <Form layout="vertical" className="detail-edit-form">
              <Form.Item label="名称">
                <Input value={`${detail.namespace}/${detail.name}`} readOnly />
              </Form.Item>
              <Form.Item label="UID">
                <Input value={detail.uid || "-"} readOnly />
              </Form.Item>
              <Form.Item label="状态">
                <Input value={detail.phase || "-"} readOnly />
              </Form.Item>
              <Form.Item label="节点">
                <Input value={detail.node_name || "-"} readOnly />
              </Form.Item>
              <Form.Item label="IP">
                <Input value={detail.pod_ip || "-"} readOnly />
              </Form.Item>
              <Form.Item label="宿主机 IP">
                <Input value={detail.host_ip || "-"} readOnly />
              </Form.Item>
              <Form.Item label="QoS">
                <Input value={detail.qos_class || "-"} readOnly />
              </Form.Item>
              <Form.Item label="ServiceAccount">
                <Input value={detail.service_account || "-"} readOnly />
              </Form.Item>
              <Form.Item label="创建时间">
                <Input value={formatDateTime(detail.creation_time)} readOnly />
              </Form.Item>
              <Form.Item label="启动时间">
                <Input value={formatDateTime(detail.start_time)} readOnly />
              </Form.Item>
              <Form.Item label="镜像">
                <Input value={detail.containers?.[0]?.image || "-"} readOnly />
              </Form.Item>
              <Form.Item label="容器名">
                <Input value={detail.containers?.[0]?.name || detail.name} readOnly />
              </Form.Item>
              <Form.Item label="启动命令">
                <Input value="-（仅支持通过高级编辑修改）" readOnly />
              </Form.Item>
              <Space style={{ width: "100%" }} size="middle">
                <Form.Item label="镜像拉取策略" style={{ flex: 1 }}>
                  <Input value={(detail as any).image_pull_policy || "-"} readOnly />
                </Form.Item>
                <Form.Item label="重启策略" style={{ flex: 1 }}>
                  <Input value={(detail as any).restart_policy || "-"} readOnly />
                </Form.Item>
              </Space>
              <Form.Item label="PriorityClassName">
                <Input value={detail.priority_class_name || "-"} readOnly />
              </Form.Item>
              <Form.Item label="标签（每行 key=value）">
                <Input.TextArea
                  rows={3}
                  value={
                    detail.labels && Object.keys(detail.labels).length > 0
                      ? Object.entries(detail.labels).map(([k, v]) => `${k}=${v}`).join("\n")
                      : "-"
                  }
                  readOnly
                />
              </Form.Item>
              <Form.Item label="NodeSelector（每行 key=value）">
                <Input.TextArea
                  rows={3}
                  value={
                    detail.node_selector && Object.keys(detail.node_selector).length > 0
                      ? Object.entries(detail.node_selector).map(([k, v]) => `${k}=${v}`).join("\n")
                      : "-"
                  }
                  readOnly
                />
              </Form.Item>
              <Form.Item label="注解">
                <Input.TextArea
                  rows={3}
                  value={
                    detail.annotations && Object.keys(detail.annotations).length > 0
                      ? Object.entries(detail.annotations).map(([k, v]) => `${k}=${v}`).join("\n")
                      : "-"
                  }
                  readOnly
                />
              </Form.Item>
            </Form>
            <Divider style={{ margin: "8px 0" }} />
            <Typography.Text strong>容器信息</Typography.Text>
            <Table
              size="small"
              rowKey="name"
              pagination={false}
              dataSource={detail.containers}
              columns={[
                { title: "容器", dataIndex: "name" },
                {
                  title: "镜像",
                  dataIndex: "image",
                  render: (image: string) => (
                    <Typography.Text
                      copyable
                      ellipsis={{ tooltip: image }}
                      style={{ maxWidth: 340, display: "inline-block" }}
                    >
                      {image}
                    </Typography.Text>
                  ),
                },
                { title: "状态", dataIndex: "state", width: 90, render: (v: string) => <Tag>{v}</Tag> },
                { title: "重启", dataIndex: "restart_count", width: 70 },
              ]}
            />
            <Divider style={{ margin: "8px 0" }} />
            <Typography.Text strong>卷信息</Typography.Text>
            <Table
              size="small"
              rowKey={(r) => r.name}
              pagination={false}
              dataSource={detail.volumes || []}
              locale={{ emptyText: "无卷信息" }}
              columns={[
                { title: "卷名", dataIndex: "name", width: 160 },
                {
                  title: "类型",
                  render: (_: unknown, v: any) => {
                    if (v.configMap) return "ConfigMap";
                    if (v.secret) return "Secret";
                    if (v.persistentVolumeClaim) return "PVC";
                    if (v.emptyDir) return "EmptyDir";
                    if (v.hostPath) return "HostPath";
                    return "其他";
                  },
                  width: 120,
                },
                {
                  title: "详情",
                  render: (_: unknown, v: any) => (
                    <Typography.Text ellipsis={{ tooltip: JSON.stringify(v) }} style={{ maxWidth: 320 }}>
                      {v.configMap?.name || v.secret?.secretName || v.persistentVolumeClaim?.claimName || v.hostPath?.path || "-"}
                    </Typography.Text>
                  ),
                },
              ]}
            />
            <Divider style={{ margin: "8px 0" }} />
            <Typography.Text strong>最近事件</Typography.Text>
            <Table
              size="small"
              rowKey={(r) => `${r.reason}-${r.last_timestamp}-${r.message}`}
              pagination={{ pageSize: 5 }}
              dataSource={events}
              columns={[
                { title: "类型", dataIndex: "type", width: 70, render: (v: string) => <Tag>{v}</Tag> },
                { title: "原因", dataIndex: "reason", width: 110 },
                { title: "消息", dataIndex: "message", ellipsis: true },
                { title: "时间", dataIndex: "last_timestamp", width: 140, render: (v: string) => formatDateTime(v) },
              ]}
            />
          </Space>
        )}
      </Drawer>
      <Drawer
        title={
          <Space direction="vertical" size={0}>
            <span>
              {simpleMode === "edit"
                ? `编辑 Pod（重建） - ${editTarget?.namespace || namespace}/${editTarget?.name || ""}`
                : "创建 Pod"}
            </span>
            <Typography.Text type="secondary" style={{ fontSize: 13, fontWeight: "normal" }}>
              目标命名空间：{namespace}
            </Typography.Text>
          </Space>
        }
        placement="right"
        width={960}
        open={createOpen}
        onClose={() => {
          setCreateOpen(false);
          setSimpleMode("create");
          setEditTarget(null);
        }}
        destroyOnClose
        maskClosable={false}
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Button
            onClick={() => {
              setCreateOpen(false);
              setSimpleMode("create");
              setEditTarget(null);
            }}
          >
            取消
          </Button>
        }
      >
        <Tabs
          items={[
            {
              key: "simple",
              label: "表单创建",
              children: (
                <Form
                  form={simpleForm}
                  layout="vertical"
                  requiredMark="optional"
                  scrollToFirstError
                  initialValues={{
                    name: "",
                    image: "nginx:latest",
                    command: "",
                    image_pull_policy: "IfNotPresent",
                    restart_policy: "Always",
                    env_pairs: [],
                    label_pairs: [],
                    node_selector_pairs: [],
                    tolerations: [],
                    priority_class_name: "",
                    affinity: {},
                  }}
                >
                  <Form.Item
                    name="name"
                    label="Pod 名称"
                    rules={[
                      { required: true, message: "请输入 Pod 名称" },
                      {
                        validator: async (_, value) => {
                          const v = String(value || "").trim();
                          if (!v) return;
                          if (!rfc1123Subdomain.test(v)) {
                            throw new Error("Pod 名称不合法：必须全小写，且仅包含字母/数字/短横线/点，首尾为字母或数字");
                          }
                        },
                      },
                    ]}
                  >
                    <Input />
                  </Form.Item>
                  <Form.Item
                    name="container_name"
                    label="容器名称"
                    extra="默认同 Pod 名称"
                    rules={[
                      {
                        validator: async (_, value) => {
                          const v = String(value || "").trim();
                          if (!v) return;
                          if (!rfc1123Label.test(v)) {
                            throw new Error("容器名称不合法：必须全小写，且仅包含字母/数字/短横线，首尾为字母或数字");
                          }
                        },
                      },
                    ]}
                  >
                    <Input placeholder="默认同 Pod 名称" />
                  </Form.Item>
                  <Form.Item name="image" label="镜像" rules={[{ required: true, message: "请输入镜像" }]}>
                    <Input />
                  </Form.Item>
                  <Space style={{ width: "100%" }} size="middle">
                    <Form.Item name="image_pull_policy" label="镜像拉取策略" style={{ flex: 1 }}>
                      <Select
                        options={[
                          { label: "IfNotPresent", value: "IfNotPresent" },
                          { label: "Always", value: "Always" },
                          { label: "Never", value: "Never" },
                        ]}
                      />
                    </Form.Item>
                    <Form.Item name="restart_policy" label="重启策略" style={{ flex: 1 }}>
                      <Select
                        options={[
                          { label: "Always", value: "Always" },
                          { label: "OnFailure", value: "OnFailure" },
                          { label: "Never", value: "Never" },
                        ]}
                      />
                    </Form.Item>
                    <Form.Item name="port" label="容器端口" style={{ width: 140 }}>
                      <InputNumber min={1} max={65535} style={{ width: "100%" }} />
                    </Form.Item>
                  </Space>
                  <Form.Item name="command" label="启动命令" extra="例如覆盖镜像默认 CMD；留空则使用镜像入口">
                    <Input placeholder="例如：sleep 3600" />
                  </Form.Item>
                  <Space style={{ width: "100%" }} size="middle">
                    <Form.Item name="requests_cpu" label="CPU 请求" style={{ flex: 1 }}>
                      <Input placeholder="例如：100m" />
                    </Form.Item>
                    <Form.Item name="requests_memory" label="内存请求" style={{ flex: 1 }}>
                      <Input placeholder="例如：128Mi" />
                    </Form.Item>
                  </Space>
                  <Space style={{ width: "100%" }} size="middle">
                    <Form.Item name="limits_cpu" label="CPU 限制" style={{ flex: 1 }}>
                      <Input placeholder="例如：500m" />
                    </Form.Item>
                    <Form.Item name="limits_memory" label="内存限制" style={{ flex: 1 }}>
                      <Input placeholder="例如：512Mi" />
                    </Form.Item>
                  </Space>
                  <Form.List name="env_pairs">
                    {(fields, { add, remove }) => (
                      <Form.Item label="环境变量" extra="按键值对添加，KEY 不可重复">
                        <Space direction="vertical" style={{ width: "100%" }}>
                          {fields.map((field) => (
                            <Space key={field.key} style={{ width: "100%" }} align="start">
                              <Form.Item
                                {...field}
                                name={[field.name, "key"]}
                                rules={[
                                  { required: true, message: "请输入变量名" },
                                  {
                                    validator: async (_, value) => {
                                      const key = String(value || "").trim();
                                      if (!key) return;
                                      const list = simpleForm.getFieldValue("env_pairs") || [];
                                      const count = list.filter((it: { key?: string }) => String(it?.key || "").trim() === key).length;
                                      if (count > 1) throw new Error("变量名不能重复");
                                    },
                                  },
                                ]}
                                style={{ marginBottom: 0, flex: 1 }}
                              >
                                <Input placeholder="KEY" />
                              </Form.Item>
                              <Form.Item
                                {...field}
                                name={[field.name, "value"]}
                                style={{ marginBottom: 0, flex: 1 }}
                              >
                                <Input placeholder="VALUE" />
                              </Form.Item>
                              <Button danger onClick={() => remove(field.name)}>
                                删除
                              </Button>
                            </Space>
                          ))}
                          <Button type="dashed" onClick={() => add()}>
                            新增环境变量
                          </Button>
                        </Space>
                      </Form.Item>
                    )}
                  </Form.List>
                  <Form.List name="node_selector_pairs">
                    {(fields, { add, remove }) => (
                      <Form.Item label="NodeSelector" extra="按键值对添加，用于节点选择">
                        <Space direction="vertical" style={{ width: "100%" }}>
                          {fields.map((field) => (
                            <Space key={field.key} style={{ width: "100%" }} align="start">
                              <Form.Item
                                {...field}
                                name={[field.name, "key"]}
                                rules={[{ required: true, message: "请输入选择器键" }]}
                                style={{ marginBottom: 0, flex: 1 }}
                              >
                                <Input placeholder="key" />
                              </Form.Item>
                              <Form.Item {...field} name={[field.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                <Input placeholder="value" />
                              </Form.Item>
                              <Button danger onClick={() => remove(field.name)}>删除</Button>
                            </Space>
                          ))}
                          <Button type="dashed" onClick={() => add()}>新增 NodeSelector</Button>
                        </Space>
                      </Form.Item>
                    )}
                  </Form.List>
                  <Form.Item name="priority_class_name" label="PriorityClassName">
                    <Input placeholder="例如：system-cluster-critical" />
                  </Form.Item>
                  <Divider style={{ margin: "8px 0" }} />
                  <Typography.Text strong>Affinity</Typography.Text>
                  <Typography.Paragraph className="inline-muted" style={{ margin: "6px 0 0" }}>
                    以表单方式配置 NodeAffinity / PodAffinity / PodAntiAffinity；未填写的块不会下发到 PodSpec。
                  </Typography.Paragraph>

                  <Card size="small" title="NodeAffinity（节点亲和）" style={{ marginTop: 10 }}>
                    <Form.List name={["affinity", "node", "required"]}>
                      {(fields, { add, remove }) => (
                        <Form.Item label="Required（必须满足）" style={{ marginBottom: 0 }}>
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Card
                                key={field.key}
                                size="small"
                                type="inner"
                                title={`Term #${field.name + 1}`}
                                extra={<Button danger onClick={() => remove(field.name)}>删除 Term</Button>}
                              >
                                <Form.List name={[field.name, "match_expressions"]}>
                                  {(expFields, expOps) => (
                                    <Space direction="vertical" style={{ width: "100%" }}>
                                      {expFields.map((ef) => (
                                        <Space key={ef.key} style={{ width: "100%" }} align="start" wrap>
                                          <Form.Item
                                            {...ef}
                                            name={[ef.name, "key"]}
                                            rules={[{ required: true, message: "key 必填" }]}
                                            style={{ marginBottom: 0, width: 220 }}
                                          >
                                            <Input placeholder="key" />
                                          </Form.Item>
                                          <Form.Item
                                            {...ef}
                                            name={[ef.name, "operator"]}
                                            rules={[{ required: true, message: "operator 必填" }]}
                                            style={{ marginBottom: 0, width: 180 }}
                                          >
                                            <Select
                                              options={[
                                                { label: "In", value: "In" },
                                                { label: "NotIn", value: "NotIn" },
                                                { label: "Exists", value: "Exists" },
                                                { label: "DoesNotExist", value: "DoesNotExist" },
                                                { label: "Gt", value: "Gt" },
                                                { label: "Lt", value: "Lt" },
                                              ]}
                                            />
                                          </Form.Item>
                                          <Form.Item
                                            {...ef}
                                            name={[ef.name, "values"]}
                                            style={{ marginBottom: 0, width: 320 }}
                                            tooltip="In/NotIn 需要 values；Exists/DoesNotExist 可留空；Gt/Lt 建议填单个数字"
                                          >
                                            <Select mode="tags" placeholder="values" />
                                          </Form.Item>
                                          <Button danger onClick={() => expOps.remove(ef.name)}>删除</Button>
                                        </Space>
                                      ))}
                                      <Button type="dashed" onClick={() => expOps.add({ operator: "In", values: [] })}>
                                        新增 MatchExpression
                                      </Button>
                                    </Space>
                                  )}
                                </Form.List>
                              </Card>
                            ))}
                            <Button type="dashed" onClick={() => add({ match_expressions: [] })}>
                              新增 Required Term
                            </Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>

                    <Divider style={{ margin: "12px 0" }} />
                    <Form.List name={["affinity", "node", "preferred"]}>
                      {(fields, { add, remove }) => (
                        <Form.Item label="Preferred（尽量满足）" style={{ marginBottom: 0 }}>
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Card
                                key={field.key}
                                size="small"
                                type="inner"
                                title={`Preference #${field.name + 1}`}
                                extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}
                              >
                                <Space style={{ width: "100%" }} align="start" wrap>
                                  <Form.Item
                                    {...field}
                                    name={[field.name, "weight"]}
                                    rules={[{ required: true, message: "weight 必填" }]}
                                    style={{ marginBottom: 0, width: 180 }}
                                  >
                                    <InputNumber min={1} max={100} style={{ width: "100%" }} placeholder="weight(1-100)" />
                                  </Form.Item>
                                </Space>
                                <Form.List name={[field.name, "match_expressions"]}>
                                  {(expFields, expOps) => (
                                    <Space direction="vertical" style={{ width: "100%", marginTop: 10 }}>
                                      {expFields.map((ef) => (
                                        <Space key={ef.key} style={{ width: "100%" }} align="start" wrap>
                                          <Form.Item
                                            {...ef}
                                            name={[ef.name, "key"]}
                                            rules={[{ required: true, message: "key 必填" }]}
                                            style={{ marginBottom: 0, width: 220 }}
                                          >
                                            <Input placeholder="key" />
                                          </Form.Item>
                                          <Form.Item
                                            {...ef}
                                            name={[ef.name, "operator"]}
                                            rules={[{ required: true, message: "operator 必填" }]}
                                            style={{ marginBottom: 0, width: 180 }}
                                          >
                                            <Select
                                              options={[
                                                { label: "In", value: "In" },
                                                { label: "NotIn", value: "NotIn" },
                                                { label: "Exists", value: "Exists" },
                                                { label: "DoesNotExist", value: "DoesNotExist" },
                                                { label: "Gt", value: "Gt" },
                                                { label: "Lt", value: "Lt" },
                                              ]}
                                            />
                                          </Form.Item>
                                          <Form.Item
                                            {...ef}
                                            name={[ef.name, "values"]}
                                            style={{ marginBottom: 0, width: 320 }}
                                          >
                                            <Select mode="tags" placeholder="values" />
                                          </Form.Item>
                                          <Button danger onClick={() => expOps.remove(ef.name)}>删除</Button>
                                        </Space>
                                      ))}
                                      <Button type="dashed" onClick={() => expOps.add({ operator: "In", values: [] })}>
                                        新增 MatchExpression
                                      </Button>
                                    </Space>
                                  )}
                                </Form.List>
                              </Card>
                            ))}
                            <Button type="dashed" onClick={() => add({ weight: 50, match_expressions: [] })}>
                              新增 Preferred 规则
                            </Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>
                  </Card>

                  <Card size="small" title="PodAffinity（Pod 亲和）" style={{ marginTop: 12 }}>
                    <Form.List name={["affinity", "pod", "required"]}>
                      {(fields, { add, remove }) => (
                        <Form.Item label="Required" style={{ marginBottom: 0 }}>
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Card
                                key={field.key}
                                size="small"
                                type="inner"
                                title={`Term #${field.name + 1}`}
                                extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}
                              >
                                <Form.Item name={[field.name, "topology_key"]} rules={[{ required: true, message: "topologyKey 必填" }]}>
                                  <Input placeholder="topologyKey，例如：kubernetes.io/hostname" />
                                </Form.Item>
                                <Form.List name={[field.name, "match_labels"]}>
                                  {(kvFields, kvOps) => (
                                    <Space direction="vertical" style={{ width: "100%" }}>
                                      {kvFields.map((kv) => (
                                        <Space key={kv.key} style={{ width: "100%" }} align="start">
                                          <Form.Item {...kv} name={[kv.name, "key"]} rules={[{ required: true, message: "key 必填" }]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="key" />
                                          </Form.Item>
                                          <Form.Item {...kv} name={[kv.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="value" />
                                          </Form.Item>
                                          <Button danger onClick={() => kvOps.remove(kv.name)}>删除</Button>
                                        </Space>
                                      ))}
                                      <Button type="dashed" onClick={() => kvOps.add()}>新增 matchLabel</Button>
                                    </Space>
                                  )}
                                </Form.List>
                              </Card>
                            ))}
                            <Button type="dashed" onClick={() => add({ match_labels: [] })}>新增 Required Term</Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>

                    <Divider style={{ margin: "12px 0" }} />
                    <Form.List name={["affinity", "pod", "preferred"]}>
                      {(fields, { add, remove }) => (
                        <Form.Item label="Preferred" style={{ marginBottom: 0 }}>
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Card
                                key={field.key}
                                size="small"
                                type="inner"
                                title={`Preferred #${field.name + 1}`}
                                extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}
                              >
                                <Space style={{ width: "100%" }} wrap>
                                  <Form.Item name={[field.name, "weight"]} rules={[{ required: true, message: "weight 必填" }]} style={{ width: 180 }}>
                                    <InputNumber min={1} max={100} style={{ width: "100%" }} placeholder="weight(1-100)" />
                                  </Form.Item>
                                  <Form.Item name={[field.name, "topology_key"]} rules={[{ required: true, message: "topologyKey 必填" }]} style={{ flex: 1 }}>
                                    <Input placeholder="topologyKey，例如：kubernetes.io/hostname" />
                                  </Form.Item>
                                </Space>
                                <Form.List name={[field.name, "match_labels"]}>
                                  {(kvFields, kvOps) => (
                                    <Space direction="vertical" style={{ width: "100%" }}>
                                      {kvFields.map((kv) => (
                                        <Space key={kv.key} style={{ width: "100%" }} align="start">
                                          <Form.Item {...kv} name={[kv.name, "key"]} rules={[{ required: true, message: "key 必填" }]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="key" />
                                          </Form.Item>
                                          <Form.Item {...kv} name={[kv.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="value" />
                                          </Form.Item>
                                          <Button danger onClick={() => kvOps.remove(kv.name)}>删除</Button>
                                        </Space>
                                      ))}
                                      <Button type="dashed" onClick={() => kvOps.add()}>新增 matchLabel</Button>
                                    </Space>
                                  )}
                                </Form.List>
                              </Card>
                            ))}
                            <Button type="dashed" onClick={() => add({ weight: 50, match_labels: [] })}>新增 Preferred 规则</Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>
                  </Card>

                  <Card size="small" title="PodAntiAffinity（Pod 反亲和）" style={{ marginTop: 12 }}>
                    <Form.List name={["affinity", "pod_anti", "required"]}>
                      {(fields, { add, remove }) => (
                        <Form.Item label="Required" style={{ marginBottom: 0 }}>
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Card key={field.key} size="small" type="inner" title={`Term #${field.name + 1}`} extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}>
                                <Form.Item name={[field.name, "topology_key"]} rules={[{ required: true, message: "topologyKey 必填" }]}>
                                  <Input placeholder="topologyKey，例如：kubernetes.io/hostname" />
                                </Form.Item>
                                <Form.List name={[field.name, "match_labels"]}>
                                  {(kvFields, kvOps) => (
                                    <Space direction="vertical" style={{ width: "100%" }}>
                                      {kvFields.map((kv) => (
                                        <Space key={kv.key} style={{ width: "100%" }} align="start">
                                          <Form.Item {...kv} name={[kv.name, "key"]} rules={[{ required: true, message: "key 必填" }]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="key" />
                                          </Form.Item>
                                          <Form.Item {...kv} name={[kv.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="value" />
                                          </Form.Item>
                                          <Button danger onClick={() => kvOps.remove(kv.name)}>删除</Button>
                                        </Space>
                                      ))}
                                      <Button type="dashed" onClick={() => kvOps.add()}>新增 matchLabel</Button>
                                    </Space>
                                  )}
                                </Form.List>
                              </Card>
                            ))}
                            <Button type="dashed" onClick={() => add({ match_labels: [] })}>新增 Required Term</Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>

                    <Divider style={{ margin: "12px 0" }} />
                    <Form.List name={["affinity", "pod_anti", "preferred"]}>
                      {(fields, { add, remove }) => (
                        <Form.Item label="Preferred" style={{ marginBottom: 0 }}>
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Card key={field.key} size="small" type="inner" title={`Preferred #${field.name + 1}`} extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}>
                                <Space style={{ width: "100%" }} wrap>
                                  <Form.Item name={[field.name, "weight"]} rules={[{ required: true, message: "weight 必填" }]} style={{ width: 180 }}>
                                    <InputNumber min={1} max={100} style={{ width: "100%" }} placeholder="weight(1-100)" />
                                  </Form.Item>
                                  <Form.Item name={[field.name, "topology_key"]} rules={[{ required: true, message: "topologyKey 必填" }]} style={{ flex: 1 }}>
                                    <Input placeholder="topologyKey，例如：kubernetes.io/hostname" />
                                  </Form.Item>
                                </Space>
                                <Form.List name={[field.name, "match_labels"]}>
                                  {(kvFields, kvOps) => (
                                    <Space direction="vertical" style={{ width: "100%" }}>
                                      {kvFields.map((kv) => (
                                        <Space key={kv.key} style={{ width: "100%" }} align="start">
                                          <Form.Item {...kv} name={[kv.name, "key"]} rules={[{ required: true, message: "key 必填" }]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="key" />
                                          </Form.Item>
                                          <Form.Item {...kv} name={[kv.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="value" />
                                          </Form.Item>
                                          <Button danger onClick={() => kvOps.remove(kv.name)}>删除</Button>
                                        </Space>
                                      ))}
                                      <Button type="dashed" onClick={() => kvOps.add()}>新增 matchLabel</Button>
                                    </Space>
                                  )}
                                </Form.List>
                              </Card>
                            ))}
                            <Button type="dashed" onClick={() => add({ weight: 50, match_labels: [] })}>新增 Preferred 规则</Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>
                  </Card>
                  <Form.List name="label_pairs">
                    {(fields, { add, remove }) => (
                      <Form.Item label="标签" extra="按键值对添加，key 不可重复">
                        <Space direction="vertical" style={{ width: "100%" }}>
                          {fields.map((field) => (
                            <Space key={field.key} style={{ width: "100%" }} align="start">
                              <Form.Item
                                {...field}
                                name={[field.name, "key"]}
                                rules={[
                                  { required: true, message: "请输入标签键" },
                                  {
                                    validator: async (_, value) => {
                                      const key = String(value || "").trim();
                                      if (!key) return;
                                      const list = simpleForm.getFieldValue("label_pairs") || [];
                                      const count = list.filter((it: { key?: string }) => String(it?.key || "").trim() === key).length;
                                      if (count > 1) throw new Error("标签键不能重复");
                                    },
                                  },
                                ]}
                                style={{ marginBottom: 0, flex: 1 }}
                              >
                                <Input placeholder="key" />
                              </Form.Item>
                              <Form.Item
                                {...field}
                                name={[field.name, "value"]}
                                style={{ marginBottom: 0, flex: 1 }}
                              >
                                <Input placeholder="value" />
                              </Form.Item>
                              <Button danger onClick={() => remove(field.name)}>
                                删除
                              </Button>
                            </Space>
                          ))}
                          <Button type="dashed" onClick={() => add()}>
                            新增标签
                          </Button>
                        </Space>
                      </Form.Item>
                    )}
                  </Form.List>
                  <Form.List name="tolerations">
                    {(fields, { add, remove }) => (
                      <Form.Item label="容忍（Tolerations）" extra="用于匹配节点污点；污点(Taints)是节点配置，不在 Pod 内创建">
                        <Space direction="vertical" style={{ width: "100%" }}>
                          {fields.map((field) => (
                            <Space key={field.key} style={{ width: "100%" }} align="start" wrap>
                              <Form.Item
                                {...field}
                                name={[field.name, "key"]}
                                rules={[{ required: true, message: "请输入 key" }]}
                                style={{ marginBottom: 0, width: 150 }}
                              >
                                <Input placeholder="key" />
                              </Form.Item>
                              <Form.Item
                                {...field}
                                name={[field.name, "operator"]}
                                initialValue="Equal"
                                style={{ marginBottom: 0, width: 130 }}
                              >
                                <Select
                                  options={[
                                    { label: "Equal", value: "Equal" },
                                    { label: "Exists", value: "Exists" },
                                  ]}
                                />
                              </Form.Item>
                              <Form.Item
                                {...field}
                                name={[field.name, "value"]}
                                style={{ marginBottom: 0, width: 160 }}
                              >
                                <Input placeholder="value" />
                              </Form.Item>
                              <Form.Item
                                {...field}
                                name={[field.name, "effect"]}
                                style={{ marginBottom: 0, width: 170 }}
                              >
                                <Select
                                  allowClear
                                  placeholder="effect"
                                  options={[
                                    { label: "NoSchedule", value: "NoSchedule" },
                                    { label: "PreferNoSchedule", value: "PreferNoSchedule" },
                                    { label: "NoExecute", value: "NoExecute" },
                                  ]}
                                />
                              </Form.Item>
                              <Form.Item
                                {...field}
                                name={[field.name, "toleration_seconds"]}
                                style={{ marginBottom: 0, width: 160 }}
                              >
                                <InputNumber min={1} style={{ width: "100%" }} placeholder="seconds" />
                              </Form.Item>
                              <Button danger onClick={() => remove(field.name)}>
                                删除
                              </Button>
                            </Space>
                          ))}
                          <Button
                            type="dashed"
                            onClick={() =>
                              add({ operator: "Equal" })
                            }
                          >
                            新增容忍
                          </Button>
                        </Space>
                      </Form.Item>
                    )}
                  </Form.List>
                  <Button type="primary" loading={creating} onClick={() => void submitCreateSimple()}>
                    {simpleMode === "edit" ? "保存并重建" : "创建"}
                  </Button>
                </Form>
              ),
            },
            ...(simpleMode === "create"
              ? [
                  {
                    key: "yaml",
                    label: "YAML 创建",
                    children: (
                      <Form form={yamlForm} layout="vertical" requiredMark="optional" scrollToFirstError initialValues={{ manifest: "" }}>
                        <Space wrap style={{ marginBottom: 8 }}>
                          <Button size="small" type="default" onClick={() => yamlForm.setFieldsValue({ manifest: POD_CREATE_YAML_TEMPLATE })}>
                            填入模板
                          </Button>
                          <Button size="small" onClick={() => yamlForm.setFieldsValue({ manifest: "" })}>
                            清空内容
                          </Button>
                        </Space>
                        <Form.Item name="manifest" label="YAML 内容" rules={[{ required: true, message: "请输入 YAML" }]}>
                          <MonacoYamlEditor height={420} />
                        </Form.Item>
                        <Button type="primary" loading={creating} onClick={() => void submitCreateYAML()}>
                          创建
                        </Button>
                      </Form>
                    ),
                  },
                ]
              : []),
          ]}
        />
      </Drawer>

      <K8sDeleteDialog
        open={deleteDialogOpen}
        resourceName={deleteTarget?.name ?? ""}
        loading={deleteLoading}
        onCancel={() => {
          setDeleteDialogOpen(false);
          setDeleteTarget(null);
        }}
        onConfirm={async (deleteOpts) => {
          if (!deleteTarget) return;
          setDeleteLoading(true);
          try {
            await handleDeletePod(deleteTarget, deleteOpts);
            setDeleteDialogOpen(false);
            setDeleteTarget(null);
          } finally {
            setDeleteLoading(false);
          }
        }}
      />
    </div>
  );
}
