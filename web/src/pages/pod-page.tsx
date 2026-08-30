import { CodeOutlined, DeleteOutlined, DownOutlined, DownloadOutlined, EditOutlined, FileSearchOutlined, FileTextOutlined, FolderOpenOutlined, MedicineBoxOutlined, PlusOutlined, ReloadOutlined, UndoOutlined, UploadOutlined } from "@ant-design/icons";
import type { MenuProps } from "antd";
import { Alert, Button, Card, Checkbox, Divider, Drawer, Dropdown, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Tabs, Tooltip, Typography, message } from "antd";
import { useCallback, useEffect, useMemo, useRef, useState, type MouseEvent } from "react";
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
import { validateYaml } from "../components/k8s/monaco-yaml-editor";
import { RealtimeUsageText } from "../components/k8s/k8s-resource-usage-cells";
import type { K8sDeleteOptions } from "../services/service-factory";
import { createPodByYAML, createPodSimple, deletePod, downloadPodLogs, getPodDetail, getPodDiagnose, getPodEvents, getPodLogs, getPods, listPodFiles, restartPod, updatePodSimple, type PodDetail, type PodDiagnoseResult, type PodEventItem, type PodFileItem, type PodItem, type PodLogsQuery } from "../services/pods";
import { analyzePodDiagnoseAI, type AIPodDiagnoseResult } from "../services/ai";
import { openAuthenticatedWebSocket } from "../services/ws-auth";
import { extractApiErrorMessage } from "../services/http";
import {
  buildPodAffinityPayload,
  buildPodPairs,
  buildPodTolerationsPayload,
  podAffinityToForm,
  type PodSimpleFormValues,
} from "./pod/pod-form-payload";
import { usePodExec } from "./pod/use-pod-exec";
import { PodLogsModal } from "./pod/pod-logs-modal";
import { PodDiagnoseDrawer } from "./pod/pod-diagnose-drawer";
import { PodFilesDrawer } from "./pod/pod-files-drawer";
import { PodFormDrawer } from "./pod/pod-form-drawer";

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
  const [aiDiagnoseLoading, setAiDiagnoseLoading] = useState(false);
  const [aiDiagnoseResult, setAiDiagnoseResult] = useState<AIPodDiagnoseResult | null>(null);
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
  const filterRef = useRef({ clusterId, namespace, keyword });
  filterRef.current = { clusterId, namespace, keyword };
  const [watchLive, setWatchLive] = useState(false);
  const execTermHostRef = useRef<HTMLDivElement | null>(null);
  const { closeExecSocket, execTermRef } = usePodExec({
    execOpen,
    clusterId,
    selected,
    detail,
    execCommand,
    execTermHostRef,
  });
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [simpleMode, setSimpleMode] = useState<"create" | "edit">("create");
  const [editTarget, setEditTarget] = useState<PodItem | null>(null);
  const [simpleForm] = Form.useForm<PodSimpleFormValues>();
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
    setAiDiagnoseResult(null);
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

  async function handleAIDiagnose() {
    if (!clusterId || !selected || aiDiagnoseLoading) return;
    setAiDiagnoseLoading(true);
    try {
      const res = await analyzePodDiagnoseAI({
        cluster_id: clusterId,
        namespace: selected.namespace,
        name: selected.name,
      });
      setAiDiagnoseResult(res);
      if (res.diagnose) setDiagnoseResult(res.diagnose);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "AI 分析失败"));
    } finally {
      setAiDiagnoseLoading(false);
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
        credentials: "include",
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
    const env = buildPodPairs(values.env_pairs);
    const labels = buildPodPairs(values.label_pairs);
    const nodeSelector = buildPodPairs(values.node_selector_pairs);
    const affinityObj = buildPodAffinityPayload(values.affinity);
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
        tolerations: buildPodTolerationsPayload(values.tolerations),
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
      affinity: podAffinityToForm(d.affinity),
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

      <PodLogsModal
        logsOpen={logsOpen}
        logsTitle={logsTitle}
        logsLoading={logsLoading}
        logsText={logsText}
        logsKeyword={logsKeyword}
        logsStartTime={logsStartTime}
        logsEndTime={logsEndTime}
        logsPrevious={logsPrevious}
        logsTimestamps={logsTimestamps}
        logsSinceSeconds={logsSinceSeconds}
        logsSinceTime={logsSinceTime}
        logsContainer={logsContainer}
        logContainerOptions={logContainerOptions}
        streaming={streaming}
        selected={selected}
        setLogsOpen={setLogsOpen}
        setLogsKeyword={setLogsKeyword}
        setLogsStartTime={setLogsStartTime}
        setLogsEndTime={setLogsEndTime}
        setLogsPrevious={setLogsPrevious}
        setLogsTimestamps={setLogsTimestamps}
        setLogsSinceSeconds={setLogsSinceSeconds}
        setLogsSinceTime={setLogsSinceTime}
        setLogsContainer={setLogsContainer}
        stopLogStream={stopLogStream}
        startLogStream={startLogStream}
        handleFilterLogs={handleFilterLogs}
        handleDownloadLogs={handleDownloadLogs}
        handleViewLogs={handleViewLogs}
      />

      <PodDiagnoseDrawer
        diagnoseOpen={diagnoseOpen}
        setDiagnoseOpen={setDiagnoseOpen}
        selected={selected}
        diagnoseLoading={diagnoseLoading}
        diagnoseResult={diagnoseResult}
        aiDiagnoseLoading={aiDiagnoseLoading}
        aiDiagnoseResult={aiDiagnoseResult}
        handleAIDiagnose={handleAIDiagnose}
      />

      <PodFilesDrawer
        fileOpen={fileOpen}
        setFileOpen={setFileOpen}
        selected={selected}
        clusterId={clusterId}
        filePath={filePath}
        setFilePath={setFilePath}
        fileList={fileList}
        fileLoading={fileLoading}
        fileContent={fileContent}
        setFileContent={setFileContent}
        fileInputRef={fileInputRef}
        loadFiles={loadFiles}
      />

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
      <PodFormDrawer
        createOpen={createOpen}
        setCreateOpen={setCreateOpen}
        namespace={namespace}
        simpleMode={simpleMode}
        setSimpleMode={setSimpleMode}
        editTarget={editTarget}
        setEditTarget={setEditTarget}
        creating={creating}
        simpleForm={simpleForm}
        yamlForm={yamlForm}
        rfc1123Subdomain={rfc1123Subdomain}
        rfc1123Label={rfc1123Label}
        submitCreateSimple={submitCreateSimple}
        submitCreateYAML={submitCreateYAML}
      />

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
