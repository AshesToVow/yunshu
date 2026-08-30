/**
 * 告警监控平台：静默（silences Tab）状态（RF-03 第三步拆分产物）
 *
 * 从 `use-alert-monitor-platform-state.tsx` 原地搬迁，逐字保留语义：
 * - 平台静默固定 page_size=200，按顶栏项目过滤；Alertmanager 已下线，
 *   `loadAmSilences` 保留为空实现（列表仍保留 source 分列渲染，便于回滚）
 * - Prometheus 活跃告警跟随顶栏项目，数据源取「首个已启用」，无启用项时取首条
 * - 「解除静默」= 以原字段回写 enabled:false（不删除记录）
 * - 批量静默共用一条 comment，逐条落库
 *
 * 注意：`loadSilences` / `loadAmSilences` 仍由主 Hook 的 Tab 副作用调用，
 * 这里不自建 Tab 级 effect，避免同一 Tab 触发两次请求。
 */
import { DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { Badge, Button, Form, Popconfirm, Space, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import { useCallback, useMemo, useState } from "react";

import {
  createAlertSilence,
  createAlertSilencesBatch,
  deleteAlertSilence,
  listAlertSilences,
  promActiveAlerts,
  updateAlertSilence,
  type AlertDatasourceItem,
  type AlertMonitorRuleItem,
  type AlertSilenceItem,
} from "../../../services/alert-platform";
import { parseLabelMap } from "../../../utils/alert-recipient-reason";
import { formatDateTime } from "../../../utils/format";
import type {
  AlertmanagerSilenceRow,
  PromNativeAlertRow,
  QuickSilenceTarget,
  SilenceDisplayRow,
  SilenceMatcherForm,
} from "../platform-provider-types";
import { parsePrometheusActiveAlertsTable } from "../prom-parse";
import { buildMatchersByLabels, parseSilenceMatchersForForm, toQuickSilenceTarget } from "../silence-parse";

export function useAlertMonitorSilenceState(params: {
  /** 顶栏项目上下文：列表过滤与新建静默的 project_id 都以它为准 */
  projectContextId?: number;
  /** 数据源列表（由数据源 Hook 加载）：用于推导活跃告警查询用的数据源 */
  dsList: AlertDatasourceItem[];
  /**
   * `alert_promql_label_key` 字典项。
   *
   * 由主 Hook 传入而非在此重复 `useDictOptions`：规则弹窗的
   * `commonLabelKeyOptions` 用的是同一份字典，重复调用会多发一次请求。
   */
  promqlLabelKeyOpts: Array<{ label: string; value: string | number }>;
}) {
  const { projectContextId, dsList, promqlLabelKeyOpts } = params;

  const [silenceList, setSilenceList] = useState<AlertSilenceItem[]>([]);
  const [silModalOpen, setSilModalOpen] = useState(false);
  const [silCurrent, setSilCurrent] = useState<AlertSilenceItem | null>(null);
  const [silForm] = Form.useForm();
  const [silSubmitting, setSilSubmitting] = useState(false);

  const [nativeAlertsLoading, setNativeAlertsLoading] = useState(false);
  const [nativeAlertsRows, setNativeAlertsRows] = useState<PromNativeAlertRow[]>([]);
  const [selectedNativeAlertKeys, setSelectedNativeAlertKeys] = useState<string[]>([]);
  const [selectedSilenceIds, setSelectedSilenceIds] = useState<number[]>([]);
  const [amSilenceRows, setAmSilenceRows] = useState<AlertmanagerSilenceRow[]>([]);
  const [amSilencesLoading, setAmSilencesLoading] = useState(false);
  const [quickSilenceOpen, setQuickSilenceOpen] = useState(false);
  const [quickSilenceSubmitting, setQuickSilenceSubmitting] = useState(false);
  const [quickSilenceTargets, setQuickSilenceTargets] = useState<QuickSilenceTarget[]>([]);
  /** 批量静默（从活跃告警勾选）时共用的说明，写入每条 alert_silences.comment */
  const [quickSilenceComment, setQuickSilenceComment] = useState("");

  const silenceMatcherNameOptions = useMemo(() => {
    const platformKeys = [
      { label: "monitor_rule_id（平台监控规则 ID）", value: "monitor_rule_id" },
      { label: "alertname（规则/告警名）", value: "alertname" },
      { label: "project_id（项目）", value: "project_id" },
      { label: "source（来源，如 prometheus_monitor）", value: "source" },
    ];
    const fromProm = promqlLabelKeyOpts
      .map((o) => {
        const value = String(o.value || "").trim();
        const label = String(o.label || "").trim() || value;
        return { label: `${label} (${value})`, value };
      })
      .filter((o) => o.value);
    const seen = new Set<string>();
    const out = [...platformKeys, ...fromProm].filter((o) => {
      if (seen.has(o.value)) return false;
      seen.add(o.value);
      return true;
    });
    out.sort((a, b) => a.value.localeCompare(b.value, "zh-CN"));
    return out;
  }, [promqlLabelKeyOpts]);

  /** 平台静默：Prometheus 活跃告警跟随顶栏项目，默认取首个已启用数据源 */
  const silenceDatasource = useMemo(() => {
    const enabled = dsList.filter((d) => d.enabled !== false);
    return (enabled.length ? enabled : dsList)[0];
  }, [dsList]);
  const silenceDatasourceId = silenceDatasource?.id;

  const loadSilences = useCallback(async () => {
    const r = await listAlertSilences({
      page: 1,
      page_size: 200,
      project_id: projectContextId && projectContextId > 0 ? projectContextId : undefined,
    });
    setSilenceList(r.list ?? []);
  }, [projectContextId]);

  const loadAmSilences = useCallback(async () => {
    // Alertmanager 已下线：不再拉取 /api/v2/silences
    setAmSilenceRows([]);
    setAmSilencesLoading(false);
  }, []);

  const silenceDisplayList = useMemo((): SilenceDisplayRow[] => {
    const platformRows = (silenceList ?? []).map((r) => ({
      ...r,
      source: "platform" as const,
      rowKey: String(r.id),
    }));
    return [...platformRows, ...amSilenceRows];
  }, [silenceList, amSilenceRows]);

  const loadNativeSilAlerts = useCallback(async () => {
    if (!silenceDatasourceId) {
      message.warning(
        projectContextId ? "当前项目下暂无 Prometheus 数据源，请先在「数据源」Tab 创建并启用" : "请先在顶栏选择项目",
      );
      return;
    }
    setNativeAlertsLoading(true);
    try {
      const raw = await promActiveAlerts(silenceDatasourceId);
      const rows = parsePrometheusActiveAlertsTable(raw);
      setNativeAlertsRows(rows);
      setSelectedNativeAlertKeys((prev) => prev.filter((k) => rows.some((r) => r.key === k)));
    } catch {
      setNativeAlertsRows([]);
      setSelectedNativeAlertKeys([]);
    } finally {
      setNativeAlertsLoading(false);
    }
  }, [silenceDatasourceId, projectContextId]);
  const nativeAlertsColumns: ColumnsType<PromNativeAlertRow> = useMemo(
    () => [
      { title: "告警名", dataIndex: "alertname", width: 160, ellipsis: true },
      {
        title: "状态",
        dataIndex: "state",
        width: 120,
        render: (v: string) => {
          const s = String(v || "").toLowerCase();
          const firing = s === "firing";
          const resolved = s === "resolved";
          return (
            <Space size={6}>
              <Badge status={firing ? "error" : resolved ? "success" : "default"} />
              <Typography.Text>{firing ? "触发中" : resolved ? "已恢复" : v || "-"}</Typography.Text>
            </Space>
          );
        },
      },
      { title: "标签", dataIndex: "labelsShort", ellipsis: true },
      { title: "开始时间", dataIndex: "activeAt", width: 180, ellipsis: true },
      {
        title: "操作",
        width: 110,
        render: (_: unknown, r: PromNativeAlertRow) => (
          <Button type="link" size="small" onClick={() => openQuickSilence([r])}>
            静默
          </Button>
        ),
      },
    ],
    [],
  );

  const silColumns = [
      {
        title: "来源",
        key: "source",
        width: 120,
        render: (_: unknown, r: SilenceDisplayRow) =>
          r.source === "alertmanager" ? <Tag color="blue">Alertmanager</Tag> : <Tag color="green">平台</Tag>,
      },
      {
        title: "ID",
        key: "id",
        width: 120,
        render: (_: unknown, r: SilenceDisplayRow) => (r.source === "alertmanager" ? r.amId : r.id),
      },
      { title: "名称", dataIndex: "name" },
      {
        title: "说明",
        dataIndex: "comment",
        width: 140,
        ellipsis: true,
        render: (c: string) => (c && String(c).trim() ? c : "—"),
      },
      {
        title: "匹配摘要",
        key: "m",
        width: 200,
        ellipsis: true,
        render: (_: unknown, r: SilenceDisplayRow) => {
          if (r.matchers?.length) {
            return r.matchers.map((x) => `${x.name ?? ""}=${x.value ?? ""}`).join(", ");
          }
          if (r.source === "platform") {
            return r.matchers_json?.slice(0, 80) ?? "—";
          }
          return "—";
        },
      },
      { title: "开始", dataIndex: "starts_at", width: 170, render: (t: string) => formatDateTime(t) },
      { title: "结束", dataIndex: "ends_at", width: 170, render: (t: string) => formatDateTime(t) },
      {
        title: "状态",
        key: "status",
        width: 100,
        render: (_: unknown, r: SilenceDisplayRow) => {
          const expired = dayjs(r.ends_at).isBefore(dayjs());
          if (expired) return <Tag color="red">已过期</Tag>;
          if (r.source === "alertmanager") {
            const st = String(r.state || "").toLowerCase();
            const label = st === "active" ? "生效中" : st === "pending" ? "待生效" : st === "expired" ? "已过期" : st || "停用";
            return r.enabled ? <Tag color="green">{label}</Tag> : <Tag>{label}</Tag>;
          }
          return r.enabled ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>;
        },
      },
      {
        title: "操作",
        width: 230,
        render: (_: unknown, r: SilenceDisplayRow) =>
          r.source === "alertmanager" ? (
            <Typography.Text type="secondary">在 Alertmanager UI 管理</Typography.Text>
          ) : (
            <Space>
              <Button type="link" size="small" disabled={!r.enabled} onClick={() => void releaseSingleSilence(r)}>
                解除静默
              </Button>
              <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openSilEdit(r)}>
                编辑
              </Button>
              <Popconfirm title="删除静默？" onConfirm={() => void removeSil(r.id)}>
                <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                  删除
                </Button>
              </Popconfirm>
            </Space>
          ),
      },
    ];

  function openSilCreate() {
    setSilCurrent(null);
    silForm.resetFields();
    silForm.setFieldsValue({
      name: "",
      matchers: [{ name: "alertname", value: "", is_regex: false }],
      comment: "",
      enabled: true,
      starts_at: dayjs(),
      ends_at: dayjs().add(2, "hour"),
    });
    setSilModalOpen(true);
  }


  /** 为平台内置监控规则预填静默（monitor_rule_id + alertname），与 Prometheus 活跃告警列表无关。 */
  function openSilenceForMonitorRule(r: AlertMonitorRuleItem) {
    setSilCurrent(null);
    silForm.resetFields();
    const ruleName = String(r.name || "").trim();
    silForm.setFieldsValue({
      name: ruleName ? `静默规则 ${ruleName}` : `静默 monitor_rule ${r.id}`,
      matchers: [
        { name: "monitor_rule_id", value: String(r.id), is_regex: false },
        ...(ruleName ? [{ name: "alertname", value: ruleName, is_regex: false }] : []),
      ],
      comment: "平台监控规则一键静默",
      enabled: true,
      starts_at: dayjs(),
      ends_at: dayjs().add(2, "hour"),
    });
    setSilModalOpen(true);
  }

  /** 事件台：按指纹 + 告警名预填静默（可再改匹配器）。 */
  function openSilenceForEvent(row: {
    fingerprint?: string;
    alertname?: string;
    labels_json?: string;
    project_id?: number;
  }) {
    setSilCurrent(null);
    silForm.resetFields();
    const labels = parseLabelMap(row.labels_json);
    const alertname = String(labels.alertname || row.alertname || "").trim();
    const matchers: SilenceMatcherForm[] = [];
    if (row.fingerprint) matchers.push({ name: "fingerprint", value: row.fingerprint, is_regex: false });
    if (alertname) matchers.push({ name: "alertname", value: alertname, is_regex: false });
    if (labels.monitor_rule_id) {
      matchers.push({ name: "monitor_rule_id", value: labels.monitor_rule_id, is_regex: false });
    }
    silForm.setFieldsValue({
      name: alertname ? `静默 ${alertname}` : `静默 ${row.fingerprint || "告警"}`,
      matchers: matchers.length ? matchers : [{ name: "fingerprint", value: row.fingerprint || "", is_regex: false }],
      comment: "事件台自定义静默",
      enabled: true,
      starts_at: dayjs(),
      ends_at: dayjs().add(2, "hour"),
    });
    setSilModalOpen(true);
  }

  function openQuickSilence(rows: PromNativeAlertRow[]) {
    setQuickSilenceComment("");
    const targets = rows.map(toQuickSilenceTarget);
    if (!targets.length) {
      message.warning("请先选择需要静默的告警");
      return;
    }
    setQuickSilenceTargets(targets);
    setQuickSilenceOpen(true);
  }

  async function submitQuickSilence() {
    if (!quickSilenceTargets.length) {
      setQuickSilenceOpen(false);
      return;
    }
    for (const it of quickSilenceTargets) {
      if (!it.endsAt.isAfter(it.startsAt)) {
        message.error(`「${it.name}」结束时间必须晚于开始时间`);
        return;
      }
    }
    setQuickSilenceSubmitting(true);
    try {
      const comment = quickSilenceComment.trim();
      const items = quickSilenceTargets.map((it) => ({
        name: it.name,
        matchers_json: JSON.stringify(buildMatchersByLabels(it.labels)),
        comment,
        enabled: true,
        starts_at: it.startsAt.toISOString(),
        ends_at: it.endsAt.toISOString(),
      }));
      const { created } = await createAlertSilencesBatch(items);
      message.success(`已创建 ${created} 条静默`);
      setQuickSilenceOpen(false);
      await loadSilences();
    } finally {
      setQuickSilenceSubmitting(false);
    }
  }

  function openSilEdit(r: AlertSilenceItem) {
    setSilCurrent(r);
    silForm.setFieldsValue({
      name: r.name,
      matchers: r.matchers?.length ? r.matchers : parseSilenceMatchersForForm(r.matchers_json),
      comment: r.comment,
      enabled: r.enabled,
      starts_at: dayjs(r.starts_at),
      ends_at: dayjs(r.ends_at),
    });
    setSilModalOpen(true);
  }

  async function submitSil() {
    setSilSubmitting(true);
    try {
      const v = await silForm.validateFields();
      const rawMatchers = (v.matchers ?? []) as SilenceMatcherForm[];
      const matchers = rawMatchers
        .map((m) => ({
          name: String(m?.name ?? "").trim(),
          value: String(m?.value ?? "").trim(),
          is_regex: Boolean(m?.is_regex),
        }))
        .filter((m) => m.name !== "");
      if (matchers.length === 0) {
        message.error("至少添加一条匹配器，并填写名称（如 alertname）");
        return;
      }
      const payload = {
        name: v.name,
        matchers_json: JSON.stringify(matchers),
        comment: v.comment,
        enabled: v.enabled,
        starts_at: (v.starts_at as Dayjs).toISOString(),
        ends_at: (v.ends_at as Dayjs).toISOString(),
        project_id: projectContextId && projectContextId > 0 ? projectContextId : undefined,
      };
      if (silCurrent) {
        await updateAlertSilence(silCurrent.id, payload);
        message.success("已更新");
      } else {
        await createAlertSilence(payload);
        message.success("已创建");
      }
      setSilModalOpen(false);
      await loadSilences();
    } finally {
      setSilSubmitting(false);
    }
  }

  async function removeSil(id: number) {
    await deleteAlertSilence(id);
    message.success("已删除");
    await loadSilences();
  }

  async function releaseSilenceNow(row: AlertSilenceItem) {
    await updateAlertSilence(row.id, {
      name: row.name,
      matchers_json: row.matchers_json,
      comment: row.comment ?? "",
      enabled: false,
      starts_at: row.starts_at,
      ends_at: row.ends_at,
    });
  }

  async function releaseSingleSilence(row: AlertSilenceItem) {
    await releaseSilenceNow(row);
    message.success("已解除静默");
    await loadSilences();
  }

  async function releaseSelectedSilences() {
    const rows = silenceList.filter((it) => selectedSilenceIds.includes(it.id) && it.enabled);
    if (!rows.length) {
      message.warning("请选择需要解除的启用静默");
      return;
    }
    const results = await Promise.allSettled(rows.map((r) => releaseSilenceNow(r)));
    const ok = results.filter((r) => r.status === "fulfilled").length;
    const fail = rows.length - ok;
    if (ok > 0) message.success(`已解除 ${ok} 条静默`);
    if (fail > 0) message.warning(`${fail} 条静默解除失败`);
    setSelectedSilenceIds([]);
    await loadSilences();
  }

  return {
    amSilencesLoading,
    loadAmSilences,
    loadNativeSilAlerts,
    loadSilences,
    nativeAlertsColumns,
    nativeAlertsLoading,
    nativeAlertsRows,
    openQuickSilence,
    openSilCreate,
    openSilenceForEvent,
    openSilenceForMonitorRule,
    quickSilenceComment,
    quickSilenceOpen,
    quickSilenceSubmitting,
    quickSilenceTargets,
    releaseSelectedSilences,
    selectedNativeAlertKeys,
    selectedSilenceIds,
    setQuickSilenceComment,
    setQuickSilenceOpen,
    setQuickSilenceTargets,
    setSelectedNativeAlertKeys,
    setSelectedSilenceIds,
    setSilModalOpen,
    silColumns,
    silCurrent,
    silForm,
    silModalOpen,
    silSubmitting,
    silenceDatasource,
    silenceDatasourceId,
    silenceDisplayList,
    silenceList,
    silenceMatcherNameOptions,
    submitQuickSilence,
    submitSil,
  };
}


