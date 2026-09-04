import { BellOutlined, ClockCircleOutlined, FileSearchOutlined, MedicineBoxOutlined, QuestionCircleOutlined, RobotOutlined, StopOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Descriptions, Drawer, Dropdown, Input, Space, Spin, Steps, Tag, Timeline, Typography, message } from "antd";
import dayjs from "dayjs";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
  acknowledgeAlert,
  clearAlertAck,
  collectAlertEvidence,
  createAlertNote,
  explainAlertByFingerprint,
  getActiveAlertAck,
  listAlertEvents,
  listAlertNotes,
  type AlertEventItem,
  type AlertEvidenceResult,
  type AlertProgressNote,
  type FingerprintDeliveryExplain,
} from "../../services/alerts";
import { createAlertSilence } from "../../services/alert-platform";
import { extractApiErrorMessage } from "../../services/http";
import { getPodDiagnose, type PodDiagnoseResult } from "../../services/pods";
import { formatDateTime } from "../../utils/format";
import { formatMatchedPolicyNamesDisplay } from "../../utils/alert-policy-display";
import { describeAlertEvent, summarizeAlertEventHint } from "../../utils/alert-event-reasons";
import { explainAlertRecipients, parseLabelMap } from "../../utils/alert-recipient-reason";
import { buildAlertLogContextUrl } from "../../utils/log-navigation";
import { AlertAckActionButton } from "./ack-action";

export type AlertEventDetailTarget = {
  fingerprint: string;
  alertname: string;
  severity?: string;
  status?: string;
  cluster?: string;
  summary?: string;
  value?: string;
  starts_at?: string;
  resolved_at?: string;
  project_id?: number;
  labels_json?: string;
};

type Props = {
  target: AlertEventDetailTarget | null;
  open: boolean;
  onClose: () => void;
  projectId?: number;
  onAiExplain?: (target: AlertEventDetailTarget) => void;
  onCustomSilence?: (target: AlertEventDetailTarget) => void;
  allowSilence?: boolean;
};

function severityColor(sev?: string) {
  const s = String(sev || "").toLowerCase();
  if (s === "critical") return "red";
  if (s === "warning") return "orange";
  return "blue";
}

function statusLabel(status?: string, resolvedAt?: string) {
  const s = String(status || "").toLowerCase();
  if (s === "resolved" || resolvedAt) return { text: "已恢复", color: "success" as const };
  return { text: "正在告警", color: "error" as const };
}

function silenceMatchers(target: AlertEventDetailTarget) {
  const labels = parseLabelMap(target.labels_json);
  const matchers: Array<{ name: string; value: string; is_regex: boolean }> = [];
  if (target.fingerprint) {
    matchers.push({ name: "fingerprint", value: target.fingerprint, is_regex: false });
  }
  const alertname = labels.alertname || target.alertname;
  if (alertname) matchers.push({ name: "alertname", value: alertname, is_regex: false });
  if (labels.monitor_rule_id) {
    matchers.push({ name: "monitor_rule_id", value: labels.monitor_rule_id, is_regex: false });
  }
  return matchers;
}

export function AlertEventDetailDrawer({
  target,
  open,
  onClose,
  projectId,
  onAiExplain,
  onCustomSilence,
  allowSilence = true,
}: Props) {
  const [loading, setLoading] = useState(false);
  const [explain, setExplain] = useState<FingerprintDeliveryExplain | null>(null);
  const [events, setEvents] = useState<AlertEventItem[]>([]);
  const [evidence, setEvidence] = useState<AlertEvidenceResult | null>(null);
  const [silencing, setSilencing] = useState(false);
  const [acking, setAcking] = useState(false);
  const [acked, setAcked] = useState(false);
  const [ackBy, setAckBy] = useState("");
  const [ackExpires, setAckExpires] = useState("");
  const [whyOpen, setWhyOpen] = useState(false);
  const [notes, setNotes] = useState<AlertProgressNote[]>([]);
  const [noteDraft, setNoteDraft] = useState("");
  const [noteSaving, setNoteSaving] = useState(false);
  const [podDiagnosing, setPodDiagnosing] = useState(false);
  const [podDiagnose, setPodDiagnose] = useState<PodDiagnoseResult | null>(null);

  const latestEvent = events[0];
  const logContextUrl = useMemo(() => {
    const pid = projectId || target?.project_id;
    if (!pid || !target) return "";
    return buildAlertLogContextUrl({
      project_id: pid,
      alert_id: latestEvent?.id,
      fingerprint: target.fingerprint,
      starts_at: target.starts_at || latestEvent?.createdAt,
      window_minutes: 5,
    });
  }, [projectId, target, latestEvent]);

  useEffect(() => {
    if (!open || !target?.fingerprint) {
      setExplain(null);
      setEvents([]);
      setEvidence(null);
      setPodDiagnose(null);
      setAcked(false);
      setAckBy("");
      setAckExpires("");
      setWhyOpen(false);
      setNotes([]);
      setNoteDraft("");
      return;
    }
    let cancelled = false;
    setLoading(true);
    setPodDiagnose(null);
    void (async () => {
      try {
        const [ex, list, ack, noteRes, ev] = await Promise.all([
          explainAlertByFingerprint(target.fingerprint),
          listAlertEvents({
            page: 1,
            page_size: 50,
            fingerprint: target.fingerprint,
            projectId: projectId || target.project_id || undefined,
          }),
          getActiveAlertAck(target.fingerprint).catch(() => null),
          listAlertNotes(target.fingerprint).catch(() => ({ list: [] as AlertProgressNote[] })),
          collectAlertEvidence(target.fingerprint).catch(() => null),
        ]);
        if (cancelled) return;
        setExplain(ex);
        setEvents(list.list ?? list.items ?? []);
        setAcked(Boolean(ack?.acked));
        setAckBy(ack?.user_name || "");
        setAckExpires(ack?.expires_at || "");
        setNotes(noteRes.list ?? []);
        setEvidence(ev);
      } catch (e) {
        if (!cancelled) message.error(extractApiErrorMessage(e, "加载通知记录失败"));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open, target?.fingerprint, projectId, target?.project_id]);

  const whySteps = useMemo(() => {
    const steps: Array<{ title: string; description: string; status: "wait" | "process" | "finish" | "error" }> = [];
    const skips = explain?.skip_summary ?? [];
    const hasSilence = skips.some((s) => /silence|静默/i.test(`${s.category} ${s.error_message} ${s.hint}`));
    const hasRoute = skips.some((s) => /no_channel|routing|route|策略/i.test(`${s.category} ${s.error_message} ${s.hint}`));
    const hasTiming = skips.some((s) => /timing|group_wait|aggregate|ack/i.test(`${s.category} ${s.error_message} ${s.hint}`));
    const delivered = Boolean(explain?.firing_delivered) || events.some((e) => e.success);

    steps.push({
      title: "是否被静默/认领抑制",
      description: hasSilence || acked
        ? acked
          ? `当前认领中${ackBy ? `（${ackBy}）` : ""}${ackExpires ? `，至 ${formatDateTime(ackExpires)}` : ""}`
          : skips.find((s) => /silence|静默/i.test(`${s.category} ${s.hint}`))?.hint || "命中静默"
        : "未命中静默/认领抑制",
      status: hasSilence || acked ? "error" : "finish",
    });
    steps.push({
      title: "是否匹配到通知策略与通道",
      description: hasRoute
        ? skips.find((s) => /no_channel|routing|route|策略/i.test(`${s.category} ${s.hint}`))?.hint || "未匹配通道"
        : delivered || events.length > 0
          ? "已有路由/投递记录"
          : "暂无明显路由跳过记录",
      status: hasRoute ? "error" : delivered || events.length > 0 ? "finish" : "process",
    });
    steps.push({
      title: "是否在等待合并/节流",
      description: hasTiming
        ? skips.find((s) => /timing|group_wait|aggregate|ack/i.test(`${s.category} ${s.hint}`))?.hint || "被节流抑制"
        : "未见合并等待抑制",
      status: hasTiming ? "error" : "finish",
    });
    steps.push({
      title: "最终是否发出通知",
      description: delivered
        ? "已有成功投递记录"
        : skips.length
          ? skips.map((s) => s.hint || s.error_message).join("；")
          : "尚无成功投递；请核对通道收件人与规则处理人配置",
      status: delivered ? "finish" : "error",
    });
    return steps;
  }, [explain, events, acked, ackBy, ackExpires]);

  async function quickSilence(hours: number, label: string) {
    if (!target) return;
    const matchers = silenceMatchers(target);
    if (!matchers.length) {
      message.warning("缺少指纹，无法静默");
      return;
    }
    setSilencing(true);
    try {
      await createAlertSilence({
        name: `静默 ${target.alertname || target.fingerprint}（${label}）`,
        matchers_json: JSON.stringify(matchers),
        comment: `事件台快捷静默 ${label}`,
        enabled: true,
        starts_at: dayjs().toISOString(),
        ends_at: dayjs().add(hours, "hour").toISOString(),
        project_id: projectId && projectId > 0 ? projectId : target.project_id || undefined,
      });
      message.success(`已静默 ${label}`);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建静默失败"));
    } finally {
      setSilencing(false);
    }
  }

  async function toggleAck(minutes?: number) {
    if (!target?.fingerprint) return;
    setAcking(true);
    try {
      if (acked) {
        await clearAlertAck(target.fingerprint);
        setAcked(false);
        setAckBy("");
        setAckExpires("");
        message.success("已取消认领");
      } else {
        const row = await acknowledgeAlert({
          fingerprint: target.fingerprint,
          ttl_minutes: minutes && minutes > 0 ? minutes : undefined,
        });
        setAcked(true);
        setAckBy(row.user_name || "");
        setAckExpires(row.expires_at || "");
        message.success(minutes && minutes > 0 ? `已认领 ${minutes} 分钟` : "已认领");
      }
    } catch (e) {
      message.error(extractApiErrorMessage(e, "认领操作失败"));
    } finally {
      setAcking(false);
    }
  }

  async function submitNote() {
    if (!target?.fingerprint) return;
    const content = noteDraft.trim();
    if (!content) {
      message.warning("请填写进展");
      return;
    }
    setNoteSaving(true);
    try {
      const row = await createAlertNote({ fingerprint: target.fingerprint, content });
      setNotes((prev) => [...prev, row]);
      setNoteDraft("");
      message.success("已记录进展");
    } catch (e) {
      message.error(extractApiErrorMessage(e, "记录进展失败"));
    } finally {
      setNoteSaving(false);
    }
  }

  async function runPodDiagnose() {
    const hint = evidence?.pod_diagnose_hint;
    if (!hint?.available) return;
    if (!hint.cluster_id || !hint.namespace || !hint.pod) {
      message.warning(hint.reason || "缺少 cluster_id，无法发起 Pod 诊断");
      return;
    }
    setPodDiagnosing(true);
    try {
      const result = await getPodDiagnose({
        cluster_id: hint.cluster_id,
        namespace: hint.namespace,
        name: hint.pod,
      });
      setPodDiagnose(result);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "Pod 诊断失败"));
    } finally {
      setPodDiagnosing(false);
    }
  }

  const st = statusLabel(target?.status, target?.resolved_at);
  const firing = allowSilence && st.text === "正在告警";
  const dimTags = evidence?.dims
    ? [
        evidence.dims.cluster ? { key: "cluster", label: `cluster: ${evidence.dims.cluster}` } : null,
        evidence.dims.namespace ? { key: "ns", label: `ns: ${evidence.dims.namespace}` } : null,
        evidence.dims.pod ? { key: "pod", label: `pod: ${evidence.dims.pod}` } : null,
        evidence.dims.service ? { key: "svc", label: `service: ${evidence.dims.service}` } : null,
      ].filter(Boolean) as Array<{ key: string; label: string }>
    : [];
  const logSamples = (evidence?.log_samples ?? []).slice(0, 5);
  const levelCounts = evidence?.log_overview?.level_counts;

  return (
    <Drawer
      title={target ? target.alertname || "告警详情" : "告警详情"}
      placement="right"
      width={640}
      open={open}
      onClose={onClose}
      destroyOnClose
      extra={
        target ? (
          <Space wrap>
            {firing ? (
              <AlertAckActionButton
                variant="default"
                acked={acked}
                loading={acking}
                onAck={(minutes) => void toggleAck(minutes)}
                onClear={() => void toggleAck()}
              />
            ) : null}
            {firing ? (
              <Dropdown
                menu={{
                  items: [
                    { key: "0.5", label: "静默 30 分钟", onClick: () => void quickSilence(0.5, "30 分钟") },
                    { key: "2", label: "静默 2 小时", onClick: () => void quickSilence(2, "2 小时") },
                    {
                      key: "tonight",
                      label: "静默到今晚 23:59",
                      onClick: () => {
                        const h = Math.max(0.5, dayjs().endOf("day").diff(dayjs(), "minute") / 60);
                        void quickSilence(h, "到今晚");
                      },
                    },
                    { type: "divider" },
                    {
                      key: "custom",
                      label: "自定义匹配条件…",
                      onClick: () => onCustomSilence?.(target),
                    },
                  ],
                }}
              >
                <Button icon={<StopOutlined />} loading={silencing}>
                  静默
                </Button>
              </Dropdown>
            ) : null}
            <Button icon={<QuestionCircleOutlined />} onClick={() => setWhyOpen((v) => !v)}>
              为什么没收到
            </Button>
            {onAiExplain ? (
              <Button icon={<RobotOutlined />} onClick={() => onAiExplain(target)}>
                AI 解读
              </Button>
            ) : null}
            {logContextUrl ? (
              <Link to={logContextUrl}>
                <Button icon={<FileSearchOutlined />}>关联日志</Button>
              </Link>
            ) : null}
          </Space>
        ) : null
      }
    >
      {!target ? null : (
        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          <Space wrap>
            <Tag color={st.color}>{st.text}</Tag>
            {target.severity ? <Tag color={severityColor(target.severity)}>{target.severity}</Tag> : null}
            {target.cluster ? <Tag>{target.cluster}</Tag> : null}
            {acked ? <Tag color="processing">认领中 · {ackBy || "已认领"}</Tag> : null}
          </Space>
          <Typography.Paragraph style={{ marginBottom: 0 }}>{target.summary || "无摘要"}</Typography.Paragraph>
          <Descriptions size="small" column={1} bordered>
            <Descriptions.Item label="当前值">{target.value || "-"}</Descriptions.Item>
            <Descriptions.Item label="开始">{formatDateTime(target.starts_at) || "-"}</Descriptions.Item>
            {target.resolved_at ? (
              <Descriptions.Item label="恢复">{formatDateTime(target.resolved_at)}</Descriptions.Item>
            ) : null}
            <Descriptions.Item label="指纹">
              <Typography.Text copyable style={{ fontSize: 12 }}>
                {target.fingerprint}
              </Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label="深链">
              <Typography.Text copyable style={{ fontSize: 12 }}>
                {`/alert-monitor-platform/history?fingerprint=${target.fingerprint}`}
              </Typography.Text>
            </Descriptions.Item>
          </Descriptions>
          <Typography.Title level={5} style={{ margin: 0 }}>
            备注 / 进展
          </Typography.Title>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            值班与处理人在此留下排查进度；恢复后仍可查看。不会自动发通知。
          </Typography.Paragraph>
          {notes.length === 0 ? (
            <Typography.Text type="secondary">暂无进展</Typography.Text>
          ) : (
            <Timeline
              items={notes.map((n) => ({
                children: (
                  <div>
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      {formatDateTime(n.created_at) || "-"} · {n.user_name || "用户"}
                    </Typography.Text>
                    <div>{n.content}</div>
                  </div>
                ),
              }))}
            />
          )}
          <Input.TextArea
            rows={3}
            maxLength={2000}
            showCount
            value={noteDraft}
            onChange={(e) => setNoteDraft(e.target.value)}
            placeholder="例如：已扩容磁盘，观察 10 分钟"
          />
          <Button type="primary" loading={noteSaving} onClick={() => void submitNote()}>
            记录进展
          </Button>

          {whyOpen ? (
            <Alert
              type="info"
              showIcon
              message="为什么没收到通知"
              description={
                <Steps
                  direction="vertical"
                  size="small"
                  current={whySteps.findIndex((s) => s.status === "error" || s.status === "process")}
                  items={whySteps}
                  style={{ marginTop: 8 }}
                />
              }
            />
          ) : null}

          {loading ? <Spin /> : null}

          {explain?.skip_summary?.length ? (
            <Alert
              type="warning"
              showIcon
              message="有通知被策略跳过"
              description={explain.skip_summary
                .map((s) => `${s.hint || s.error_message}（${s.count} 次）`)
                .join("；")}
            />
          ) : null}

          <Card size="small" title="证据包">
            {dimTags.length ? (
              <Space wrap style={{ marginBottom: 8 }}>
                {dimTags.map((t) => (
                  <Tag key={t.key}>{t.label}</Tag>
                ))}
              </Space>
            ) : (
              <Typography.Text type="secondary" style={{ display: "block", marginBottom: 8 }}>
                暂无维度标签
              </Typography.Text>
            )}

            {evidence?.log_overview ? (
              <Typography.Paragraph style={{ marginBottom: 8 }}>
                日志概览：共 {evidence.log_overview.total ?? 0} 条
                {levelCounts && Object.keys(levelCounts).length
                  ? ` · ${Object.entries(levelCounts)
                      .map(([lv, n]) => `${lv}:${n}`)
                      .join(" / ")}`
                  : ""}
              </Typography.Paragraph>
            ) : evidence?.log_hint ? (
              <Typography.Paragraph type="secondary" style={{ marginBottom: 8 }}>
                {evidence.log_hint}
              </Typography.Paragraph>
            ) : null}

            {logSamples.length ? (
              <Space direction="vertical" size={4} style={{ width: "100%", marginBottom: 8 }}>
                {logSamples.map((s, i) => (
                  <div key={`${s.timestamp}-${i}`} style={{ fontSize: 12 }}>
                    <Typography.Text type="secondary">{formatDateTime(s.timestamp) || s.timestamp}</Typography.Text>
                    {s.level ? (
                      <Tag style={{ marginLeft: 6 }} color={String(s.level).toLowerCase().includes("err") ? "red" : undefined}>
                        {s.level}
                      </Tag>
                    ) : null}
                    <Typography.Text ellipsis style={{ display: "block", maxWidth: "100%" }}>
                      {s.message || "-"}
                    </Typography.Text>
                  </div>
                ))}
              </Space>
            ) : null}

            <Typography.Text strong style={{ display: "block", marginBottom: 4 }}>
              近期变更
            </Typography.Text>
            {(evidence?.recent_changes ?? []).length ? (
              <Timeline
                style={{ marginBottom: 8 }}
                items={(evidence?.recent_changes ?? []).map((c) => ({
                  children: (
                    <div style={{ fontSize: 12 }}>
                      <Typography.Text type="secondary">{formatDateTime(c.started_at) || "-"}</Typography.Text>
                      <div>{c.summary || c.action || c.source || "-"}</div>
                    </div>
                  ),
                }))}
              />
            ) : (
              <Typography.Text type="secondary" style={{ display: "block", marginBottom: 8 }}>
                暂无关联变更
              </Typography.Text>
            )}

            {evidence?.pod_diagnose_hint?.available ? (
              <Space direction="vertical" size={8} style={{ width: "100%" }}>
                <Button
                  size="small"
                  icon={<MedicineBoxOutlined />}
                  loading={podDiagnosing}
                  onClick={() => void runPodDiagnose()}
                >
                  Pod 诊断
                </Button>
                {podDiagnose ? (
                  <Alert
                    type="info"
                    showIcon
                    message={podDiagnose.summary || "诊断结果"}
                    description={
                      (podDiagnose.hints ?? []).length
                        ? (podDiagnose.hints ?? [])
                            .slice(0, 5)
                            .map((h) => `${h.title}${h.detail ? `：${h.detail}` : ""}`)
                            .join("；")
                        : "暂无诊断提示"
                    }
                  />
                ) : null}
              </Space>
            ) : evidence?.pod_diagnose_hint?.cluster_name && !evidence.pod_diagnose_hint.cluster_id ? (
              <Typography.Text type="secondary">
                {evidence.pod_diagnose_hint.reason ||
                  `仅有集群名「${evidence.pod_diagnose_hint.cluster_name}」，缺少 cluster_id，无法发起 Pod 诊断`}
              </Typography.Text>
            ) : null}

            {logContextUrl ? (
              <div style={{ marginTop: 8 }}>
                <Link to={logContextUrl}>
                  <Button size="small" icon={<FileSearchOutlined />}>
                    关联日志
                  </Button>
                </Link>
              </div>
            ) : null}
          </Card>

          <Typography.Text strong>
            <BellOutlined /> 通知记录
          </Typography.Text>
          {!loading && events.length === 0 && !explain?.events?.length ? (
            <Typography.Text type="secondary">
              还没有通知记录。可能仍在等待同组合并，或尚未匹配到通道。
            </Typography.Text>
          ) : events.length ? (
            <Timeline
              items={events.map((ev) => {
                const policy = formatMatchedPolicyNamesDisplay(ev.matchedPolicyNameList ?? ev.matchedPolicyNames);
                const who = explainAlertRecipients(ev.requestPayload);
                const hint = summarizeAlertEventHint(ev);
                const sent = ev.success && !ev.errorMessage;
                return {
                  color: sent ? "green" : ev.success ? "gray" : "red",
                  dot: <ClockCircleOutlined />,
                  children: (
                    <Space direction="vertical" size={2} style={{ width: "100%" }}>
                      <Typography.Text>
                        {formatDateTime(ev.createdAt) || "-"} · {ev.channelName || "未匹配通道"}
                        {sent ? " · 已发送" : hint !== "-" ? ` · ${hint}` : ev.success ? " · 留痕" : " · 失败"}
                      </Typography.Text>
                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                        路由 {policy.text}
                        {who.short !== "-" ? ` · ${who.short}` : ""}
                        {ev.receiverList?.length ? ` · ${ev.receiverList.join("、")}` : ""}
                      </Typography.Text>
                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                        {who.kind === "unknown" ? describeAlertEvent(ev) : who.detail}
                      </Typography.Text>
                    </Space>
                  ),
                };
              })}
            />
          ) : (
            <Timeline
              items={(explain?.events ?? []).map((ev) => ({
                color: ev.success && !ev.error_message ? "green" : ev.success ? "gray" : "red",
                children: (
                  <Typography.Text>
                    {formatDateTime(ev.created_at) || "-"} · {ev.channel_name || "通道"}
                    {ev.reason_hint ? ` · ${ev.reason_hint}` : ""}
                  </Typography.Text>
                ),
              }))}
            />
          )}
        </Space>
      )}
    </Drawer>
  );
}
