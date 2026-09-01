// @ts-nocheck
/**
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
  reloadEvents: (page: number, pageSize: number) => void | Promise<void>;
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
    <>
    <Alert
      type="info"
      showIcon
      style={{ marginBottom: 12 }}
      message="历史告警：通知投递审计与抑制原因码"
      description={
        <>
          <p style={{ marginBottom: 8 }}>
            每一行对应一次<strong>外发尝试或策略留痕</strong>（<Typography.Text code>alert_events</Typography.Text>
            ）。<Typography.Text code>success=true</Typography.Text> 且带 <Typography.Text code>error_message</Typography.Text>{" "}
            时，通常表示「未外发但已记录原因」，并非通道 HTTP 失败。
          </p>
          <ul style={{ marginBottom: 8, paddingLeft: 18 }}>
            {ALERT_HISTORY_PIPELINE_HELP.map((item) => (
              <li key={item.title}>
                <strong>{item.title}</strong>：{item.body}
              </li>
            ))}
          </ul>
          <p style={{ marginBottom: 0 }}>
            <strong>Prometheus 活跃告警</strong>（/api/v1/alerts）请在「告警监控平台 → PromQL / 平台静默」查看；与本表「是否已进 Webhook
            链路」不是同一数据源。抑制规则配置见「告警监控平台 → 告警抑制」。
          </p>
        </>
      }
    />
    {embedded && projectContextId ? (
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        message={`已按顶栏项目筛选历史记录（项目 #${projectContextId}）`}
      />
    ) : null}
    <Space className="ops-filter-bar" style={{ width: "100%", marginBottom: 12 }} wrap>
      <Radio.Group
        value={eventHistoryMode}
        onChange={(e) => setEventHistoryMode(e.target.value as "list" | "grouped")}
        optionType="button"
        buttonStyle="solid"
        options={[
          { label: "明细列表", value: "list" },
          { label: "按 GroupKey 聚合", value: "grouped" },
        ]}
      />
      <Input
        style={{ width: 260 }}
        placeholder="关键词（标题/错误/通道）"
        value={eventKeyword}
        onChange={(e) => setEventKeyword(e.target.value)}
        allowClear
      />
      <Select
        style={{ width: 220 }}
        placeholder="告警IP（labels.instance / pod_ip）"
        value={eventAlertIP || undefined}
        options={alertIPOptions}
        showSearch
        allowClear
        onSearch={(v) => setEventAlertIP(v)}
        onChange={(v) => setEventAlertIP((v as string) || "")}
        filterOption={(input, option) => String(option?.value ?? "").toLowerCase().includes(input.toLowerCase())}
      />
      <Select
        style={{ width: 280 }}
        placeholder={ALERT_ROUTING_TERMS.historySourceFilter}
        value={eventSourceFilter || undefined}
        options={sourceFilterOptions}
        showSearch
        allowClear
        onChange={(v) => setEventSourceFilter((v as string) || "")}
        filterOption={(input, option) => String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
      />
      <Select
        style={{ width: 160 }}
        placeholder="状态"
        value={eventStatus || undefined}
        options={[
          { label: "firing", value: "firing" },
          { label: "resolved", value: "resolved" },
        ]}
        allowClear
        onChange={(v) => setEventStatus((v as string) || "")}
      />
      <Select
        style={{ width: 160 }}
        placeholder="策略分类"
        value={eventCategory || undefined}
        options={ALERT_EVENT_CATEGORY_OPTIONS}
        allowClear
        onChange={(v) => setEventCategory((v as AlertEventCategory) || "")}
      />
      <Input
        style={{ width: 220 }}
        placeholder="groupKey"
        value={eventGroupKey}
        onChange={(e) => setEventGroupKey(e.target.value)}
        allowClear
      />
      <Input
        style={{ width: 240 }}
        placeholder="fingerprint"
        value={eventFingerprint}
        onChange={(e) => setEventFingerprint(e.target.value)}
        allowClear
        onPressEnter={() => {
          const fp = eventFingerprint.trim();
          if (!fp) return;
          void (async () => {
            setFpExplainLoading(true);
            try {
              const r = await explainAlertByFingerprint(fp);
              setFpExplain(r);
              setFpAiResult(null);
              setFpExplainOpen(true);
            } catch {
              message.error("指纹追溯失败");
            } finally {
              setFpExplainLoading(false);
            }
          })();
        }}
      />
      <Button
        loading={fpExplainLoading}
        disabled={!eventFingerprint.trim()}
        onClick={() => {
          const fp = eventFingerprint.trim();
          if (!fp) return;
          void (async () => {
            setFpExplainLoading(true);
            try {
              const r = await explainAlertByFingerprint(fp);
              setFpExplain(r);
              setFpAiResult(null);
              setFpExplainOpen(true);
            } catch {
              message.error("指纹追溯失败");
            } finally {
              setFpExplainLoading(false);
            }
          })();
        }}
      >
        指纹追溯
      </Button>
      <Button icon={<ReloadOutlined />} onClick={() => void reloadEvents(eventsPage, eventsPageSize)}>
        刷新
      </Button>
    </Space>
    {eventHistoryMode === "grouped" ? (
      <ResizableTable
        rowKey="group_key"
        loading={eventsLoading}
        dataSource={groupedEvents}
        scroll={{ x: 1100 }}
        pagination={tablePagination({
          current: eventsPage,
          pageSize: eventsPageSize,
          total: eventsTotal,
          onChange: (p, ps) => void reloadEvents(p, ps),
        })}
        columns={[
          {
            title: "GroupKey",
            dataIndex: "group_key",
            width: 200,
            ellipsis: true,
            render: (v: string) => (
              <Button
                type="link"
                size="small"
                onClick={() => {
                  setEventGroupKey(v);
                  setEventHistoryMode("list");
                }}
              >
                {v || "-"}
              </Button>
            ),
          },
          { title: "标题", dataIndex: "title", width: 320, ellipsis: true },
          { title: "次数", dataIndex: "count", width: 80 },
          { title: "最近时间", dataIndex: "last_at", width: 170, render: (v: string) => formatDateTime(v) },
          {
            title: "级别",
            dataIndex: "severity",
            width: 100,
            render: (v: string) => (
              <Tag color={v === "critical" ? "red" : v === "warning" ? "orange" : "blue"}>{v || "-"}</Tag>
            ),
          },
          { title: "状态", dataIndex: "status", width: 90, render: (v: string) => <Tag>{v || "-"}</Tag> },
          { title: "集群", dataIndex: "cluster", width: 140, ellipsis: true, render: (v: string) => v || "-" },
        ]}
      />
    ) : (
    <ResizableTable
      rowKey="id"
      loading={eventsLoading}
      dataSource={events}
      scroll={{ x: 2460 }}
      pagination={tablePagination({
        current: eventsPage,
        pageSize: eventsPageSize,
        total: eventsTotal,
        onChange: (p, ps) => void reloadEvents(p, ps),
      })}
      columns={[
        { title: "ID", dataIndex: "id", width: 80 },
        {
          title: "标题",
          dataIndex: "title",
          width: 360,
          render: (v: string) => (
            <Typography.Text style={{ whiteSpace: "normal", wordBreak: "break-word" }}>
              {v || "-"}
            </Typography.Text>
          ),
        },
        { title: "告警IP", dataIndex: "alertIP", width: 160, ellipsis: true, render: (v: string) => v || "-" },
        {
          title: "数据源 / 来源",
          key: "datasourceDisplay",
          width: 200,
          render: (_: unknown, row: AlertEventItem) => {
            const name = String(row.datasourceName ?? "").trim();
            const typ = String(row.datasourceType ?? "").trim();
            const slug = String(row.monitorPipeline ?? "").trim();
            if (name) {
              return (
                <Space align="start" size={4} wrap>
                  <Typography.Text style={{ maxWidth: 160 }} ellipsis={{ tooltip: name }}>
                    {name}
                  </Typography.Text>
                  {typ ? <Tag>{typ}</Tag> : null}
                </Space>
              );
            }
            if (slug === "alertmanager") return <Tag color="blue">Alertmanager</Tag>;
            if (slug === "cloud_expiry") return <Tag color="volcano">云到期</Tag>;
            if (slug === "platform_monitor") return <Tag color="purple">平台规则</Tag>;
            if (slug === "platform") return <Tag color="purple">platform（历史）</Tag>;
            if (slug === "prometheus") return <Tag color="blue">prometheus（历史）</Tag>;
            if (slug.startsWith("ds:")) return <Tag>{slug}</Tag>;
            return slug ? <Tag>{slug}</Tag> : <span>-</span>;
          },
        },
        { title: "GroupKey", dataIndex: "groupKey", width: 140, ellipsis: true, render: (v: string) => v || "-" },
        {
          title: "Fingerprint",
          dataIndex: "fingerprint",
          width: 160,
          ellipsis: true,
          render: (v: string) =>
            v ? (
              <Button
                type="link"
                size="small"
                onClick={() => {
                  setEventFingerprint(v);
                  void (async () => {
                    setFpExplainLoading(true);
                    try {
                      const r = await explainAlertByFingerprint(v);
                      setFpExplain(r);
                      setFpAiResult(null);
                      setFpExplainOpen(true);
                    } catch {
                      message.error("指纹追溯失败");
                    } finally {
                      setFpExplainLoading(false);
                    }
                  })();
                }}
              >
                {v}
              </Button>
            ) : (
              "-"
            ),
        },
        {
          title: "级别",
          dataIndex: "severity",
          width: 100,
          render: (v: string) => (
            <Tag color={v === "critical" ? "red" : v === "warning" ? "orange" : "blue"}>{v || "-"}</Tag>
          ),
        },
        { title: "状态", dataIndex: "status", width: 90, render: (v: string) => <Tag>{v || "-"}</Tag> },
        {
          title: "告警值 / 恢复时值",
          key: "metric_values",
          width: 200,
          ellipsis: true,
          render: (_: unknown, row: AlertEventItem) => {
            const a = String(row.metricCurrent ?? "").trim();
            const b = String(row.metricResolved ?? "").trim();
            if (!a && !b) return <span className="inline-muted">-</span>;
            if (String(row.status).toLowerCase() === "resolved" && b) {
              return (
                <Typography.Text ellipsis={{ tooltip: `触发侧快照: ${a || "—"}\n恢复时再查: ${b}` }} style={{ fontSize: 12 }}>
                  触发: {a || "—"} / 恢复: {b}
                </Typography.Text>
              );
            }
            return (
              <Typography.Text ellipsis={{ tooltip: a }} style={{ fontSize: 12 }}>
                {a || "—"}
              </Typography.Text>
            );
          },
        },
        {
          title: "命中路由",
          dataIndex: "matchedPolicyNames",
          width: 200,
          ellipsis: true,
          render: (_: string, row: AlertEventItem) => {
            const d = formatMatchedPolicyNamesDisplay(row.matchedPolicyNameList ?? row.matchedPolicyNames);
            return (
              <Typography.Text ellipsis={{ tooltip: d.title }} style={{ fontSize: 12 }}>
                {d.text}
              </Typography.Text>
            );
          },
        },
        { title: "通道", dataIndex: "channelName", width: 160, ellipsis: true },
        {
          title: "接收人",
          dataIndex: "receiverList",
          width: 220,
          ellipsis: true,
          render: (_: unknown, row: AlertEventItem) => {
            if (!row.receiverList?.length) return "-";
            return (
              <Space size={[4, 4]} wrap>
                {row.receiverList.map((one) => (
                  <Tag key={`${row.id}-${one}`}>{one}</Tag>
                ))}
              </Space>
            );
          },
        },
        {
          title: "收件原因",
          key: "recipient_reason",
          width: 150,
          ellipsis: true,
          render: (_: unknown, row: AlertEventItem) => {
            const r = explainAlertRecipients(row.requestPayload);
            return (
              <Typography.Text ellipsis={{ tooltip: r.detail }} style={{ fontSize: 12 }}>
                {r.short}
              </Typography.Text>
            );
          },
        },
        {
          title: "发送结果",
          dataIndex: "success",
          width: 100,
          render: (v: boolean, row: AlertEventItem) => {
            const reason = String(row.errorMessage || "").trim();
            if (v && reason) {
              return <Tag color="default">留痕</Tag>;
            }
            return v ? <Tag color="success">成功</Tag> : <Tag color="error">失败</Tag>;
          },
        },
        {
          title: "策略摘要",
          key: "reason_hint",
          width: 120,
          render: (_: unknown, row: AlertEventItem) => {
            const hint = summarizeAlertEventHint(row);
            if (hint === "-") return <span>-</span>;
            return (
              <Typography.Text ellipsis={{ tooltip: describeAlertEvent(row) }} style={{ fontSize: 12 }}>
                {hint}
              </Typography.Text>
            );
          },
        },
        {
          title: "标签组",
          key: "labels_group",
          width: 110,
          render: (_: unknown, row: AlertEventItem) => {
            const labels = parseLabelsFromAlertEventRequestPayload(row.requestPayload);
            const entries = Object.entries(labels);
            if (!entries.length) return <span>-</span>;
            return (
              <Popover
                title="标签组（labels）"
                trigger={["click"]}
                overlayStyle={{ maxWidth: 560 }}
                content={
                  <div style={{ maxHeight: 400, overflow: "auto" }}>
                    <Space size={[4, 8]} wrap>
                      {entries.map(([k, v]) => (
                        <Tag key={`${row.id}-${k}`} style={{ marginInlineEnd: 0 }}>
                          {k}={v}
                        </Tag>
                      ))}
                    </Space>
                  </div>
                }
              >
                <Button size="small">查看标签</Button>
              </Popover>
            );
          },
        },
        {
          title: "告警数据原始 JSON",
          key: "raw_request_payload",
          width: 110,
          render: (_: unknown, row: AlertEventItem) => {
            const raw = String(row.requestPayload || "").trim();
            if (!raw) return <span>-</span>;
            const pretty = prettifyAlertRequestPayload(raw);
            const http = row.httpStatusCode;
            return (
              <Popover
                title={`入库 requestPayload · HTTP ${http ?? "-"}`}
                trigger={["click"]}
                overlayStyle={{ maxWidth: 760 }}
                content={
                  <pre
                    style={{
                      maxHeight: 480,
                      overflow: "auto",
                      margin: 0,
                      fontSize: 12,
                      whiteSpace: "pre-wrap",
                      wordBreak: "break-word",
                    }}
                  >
                    {pretty}
                  </pre>
                }
              >
                <Button size="small">查看 JSON</Button>
              </Popover>
            );
          },
        },
        {
          title: "告警产生时间",
          dataIndex: "alertStartedAt",
          width: 170,
          render: (v: string) => (v ? formatDateTime(v) : "-"),
        },
        {
          title: "链路说明",
          key: "flow_explain",
          width: 340,
          ellipsis: true,
          render: (_: unknown, row: AlertEventItem) => {
            const msg = describeAlertEvent(row);
            return (
              <Typography.Text type="secondary" ellipsis={{ tooltip: msg }}>
                {msg}
              </Typography.Text>
            );
          },
        },
        { title: "发送/记录时间", dataIndex: "createdAt", width: 170, render: (v: string) => formatDateTime(v) },
      ]}
    />
    )}
    </>
  );
}
