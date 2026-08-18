import { extractApiErrorMessage } from "../../../services/http";
import {
  Alert,
  Button,
  Card,
  Input,
  Modal,
  Segmented,
  Space,
  Tag,
  Typography,
  message,
} from "antd";
import { BellOutlined, CheckOutlined, ReloadOutlined, RobotOutlined, StopOutlined } from "@ant-design/icons";
import { lazy, Suspense, useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useAlertMonitor } from "../context";
import { AlertEventDetailDrawer, type AlertEventDetailTarget } from "../event-detail-drawer";
import { ResizableTable } from "../../../components/resizable-table";
import {
  acknowledgeAlert,
  clearAlertAck,
  listCurEvents,
  listHisEvents,
  type AlertCurEventItem,
  type AlertHisEventItem,
} from "../../../services/alerts";
import { analyzeAlertExplainAI, type AIAlertExplainResult } from "../../../services/ai";
import { formatDateTime } from "../../../utils/format";
import { DEFAULT_PAGE_SIZE, tablePagination } from "../../../utils/table-pagination";

const AlertConfigCenterPanel = lazy(async () => {
  const mod = await import("../../alert-config-center-panel");
  return { default: mod.AlertConfigCenterPanel };
});

type EventView = "current" | "lifecycle" | "delivery";

export function HistoryTab() {
  const ctx = useAlertMonitor();
  const [searchParams] = useSearchParams();
  const deepFingerprint = String(searchParams.get("fingerprint") || "").trim();
  const [view, setView] = useState<EventView>("current");
  const [keyword, setKeyword] = useState(deepFingerprint);
  const [curRows, setCurRows] = useState<AlertCurEventItem[]>([]);
  const [hisRows, setHisRows] = useState<AlertHisEventItem[]>([]);
  const [curTotal, setCurTotal] = useState(0);
  const [hisTotal, setHisTotal] = useState(0);
  const [curPage, setCurPage] = useState(1);
  const [curPageSize, setCurPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [hisPage, setHisPage] = useState(1);
  const [hisPageSize, setHisPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [loading, setLoading] = useState(false);
  const [aiOpen, setAiOpen] = useState(false);
  const [aiLoading, setAiLoading] = useState(false);
  const [aiTarget, setAiTarget] = useState<AlertCurEventItem | null>(null);
  const [aiResult, setAiResult] = useState<AIAlertExplainResult | null>(null);
  const [detail, setDetail] = useState<AlertEventDetailTarget | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const deepOpenedRef = useRef(false);
  const deepHisTriedRef = useRef(false);

  function toDetailFromCur(r: AlertCurEventItem): AlertEventDetailTarget {
    return {
      fingerprint: r.fingerprint,
      alertname: r.alertname,
      severity: r.severity,
      status: r.status || "firing",
      cluster: r.cluster,
      summary: r.summary,
      value: r.value,
      starts_at: r.starts_at,
      project_id: r.project_id,
      labels_json: r.labels_json,
    };
  }

  function toDetailFromHis(r: AlertHisEventItem): AlertEventDetailTarget {
    return {
      fingerprint: r.fingerprint,
      alertname: r.alertname,
      severity: r.severity,
      status: r.status || "resolved",
      cluster: r.cluster,
      summary: r.summary,
      starts_at: r.starts_at,
      resolved_at: r.resolved_at,
      project_id: r.project_id,
      labels_json: r.labels_json,
    };
  }

  function openDetail(t: AlertEventDetailTarget) {
    setDetail(t);
    setDetailOpen(true);
  }

  const loadCur = useCallback(
    async (page: number, pageSize: number, kw: string) => {
      setLoading(true);
      try {
        const r = await listCurEvents({
          project_id: ctx.projectContextId || undefined,
          keyword: kw || undefined,
          page,
          page_size: pageSize,
        });
        setCurRows(r.list ?? r.items ?? []);
        setCurTotal(r.total ?? 0);
        setCurPage(r.page ?? page);
        setCurPageSize(r.page_size ?? pageSize);
      } catch (e) {
        message.error(extractApiErrorMessage(e, "加载当前告警失败"));
      } finally {
        setLoading(false);
      }
    },
    [ctx.projectContextId],
  );

  const loadHis = useCallback(
    async (page: number, pageSize: number, kw: string) => {
      setLoading(true);
      try {
        const r = await listHisEvents({
          project_id: ctx.projectContextId || undefined,
          keyword: kw || undefined,
          page,
          page_size: pageSize,
        });
        setHisRows(r.list ?? r.items ?? []);
        setHisTotal(r.total ?? 0);
        setHisPage(r.page ?? page);
        setHisPageSize(r.page_size ?? pageSize);
      } catch (e) {
        message.error(extractApiErrorMessage(e, "加载历史告警失败"));
      } finally {
        setLoading(false);
      }
    },
    [ctx.projectContextId],
  );

  useEffect(() => {
    if (view === "current") void loadCur(1, DEFAULT_PAGE_SIZE, keyword);
    if (view === "lifecycle") void loadHis(1, DEFAULT_PAGE_SIZE, keyword);
    // 切换视图 / 项目时从第 1 页加载；keyword 由搜索触发
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [view, ctx.projectContextId, loadCur, loadHis]);

  useEffect(() => {
    if (!deepFingerprint || deepOpenedRef.current) return;
    const cur = curRows.find((r) => r.fingerprint === deepFingerprint);
    if (cur) {
      deepOpenedRef.current = true;
      openDetail(toDetailFromCur(cur));
      return;
    }
    const his = hisRows.find((r) => r.fingerprint === deepFingerprint);
    if (his) {
      deepOpenedRef.current = true;
      setView("lifecycle");
      openDetail(toDetailFromHis(his));
      return;
    }
    if (!loading && !deepHisTriedRef.current) {
      deepHisTriedRef.current = true;
      void loadHis(1, DEFAULT_PAGE_SIZE, deepFingerprint);
      return;
    }
    if (!loading && deepHisTriedRef.current && !deepOpenedRef.current) {
      deepOpenedRef.current = true;
      openDetail({ fingerprint: deepFingerprint, alertname: deepFingerprint, status: "firing" });
    }
  }, [deepFingerprint, curRows, hisRows, loading, loadHis]);

  async function runAiExplain(row: AlertCurEventItem) {
    if (!row.fingerprint) {
      message.warning("该告警无指纹，无法 AI 解读");
      return;
    }
    setAiTarget(row);
    setAiResult(null);
    setAiOpen(true);
    setAiLoading(true);
    try {
      const res = await analyzeAlertExplainAI({
        fingerprint: row.fingerprint,
        project_id: ctx.projectContextId || row.project_id || undefined,
        window_hours: 24,
      });
      setAiResult(res);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "AI 解读失败"));
    } finally {
      setAiLoading(false);
    }
  }

  async function toggleAck(row: AlertCurEventItem) {
    if (!row.fingerprint) {
      message.warning("缺少指纹，无法认领");
      return;
    }
    try {
      if (row.acked) {
        await clearAlertAck(row.fingerprint);
        message.success("已取消认领，将恢复通知");
      } else {
        await acknowledgeAlert({ fingerprint: row.fingerprint, ttl_minutes: 15 });
        message.success("已认领 15 分钟：同指纹通知将暂停");
      }
      await loadCur(curPage, curPageSize, keyword);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "认领操作失败"));
    }
  }

  return (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      <Alert
        type="info"
        showIcon
        message="正在告警 / 已恢复 / 通知记录"
        description="正在告警：还没恢复的实例。已恢复：曾经响过并恢复的记录（没恢复过的不会出现在这里）。通知记录：发给谁、为何没发。点一行可看时间线。"
      />
      <Space wrap>
        <Segmented
          value={view}
          onChange={(v) => setView(v as EventView)}
          options={[
            { label: `正在告警 (${curTotal})`, value: "current" },
            { label: `已恢复 (${hisTotal})`, value: "lifecycle" },
            { label: "通知记录", value: "delivery" },
          ]}
        />
        {view !== "delivery" ? (
          <>
            <Input.Search
              allowClear
              placeholder="搜索告警名/指纹/摘要"
              style={{ width: 260 }}
              onSearch={(v) => {
                setKeyword(v);
                if (view === "current") void loadCur(1, curPageSize, v);
                else void loadHis(1, hisPageSize, v);
              }}
            />
            <Button
              icon={<ReloadOutlined />}
              onClick={() =>
                void (view === "current"
                  ? loadCur(curPage, curPageSize, keyword)
                  : loadHis(hisPage, hisPageSize, keyword))
              }
            >
              刷新
            </Button>
          </>
        ) : null}
      </Space>

      {view === "current" ? (
        <ResizableTable
          rowKey="id"
          loading={loading}
          dataSource={curRows}
          pagination={tablePagination({
            current: curPage,
            pageSize: curPageSize,
            total: curTotal,
            onChange: (page, pageSize) => void loadCur(page, pageSize, keyword),
          })}
          scroll={{ x: 1380 }}
          onRow={(r) => ({
            onClick: () => openDetail(toDetailFromCur(r)),
            style: { cursor: "pointer" },
          })}
          columns={[
            { title: "告警名", dataIndex: "alertname", width: 180, ellipsis: true },
            {
              title: "级别",
              dataIndex: "severity",
              width: 90,
              render: (v: string) => <Tag color={v === "critical" ? "red" : "orange"}>{v}</Tag>,
            },
            { title: "集群", dataIndex: "cluster", width: 120, ellipsis: true },
            { title: "摘要", dataIndex: "summary", width: 220, ellipsis: true },
            {
              title: "处理人",
              dataIndex: "handler_summary",
              width: 160,
              ellipsis: true,
              render: (v: string) =>
                v ? (
                  <Typography.Text ellipsis title={v}>
                    {v}
                  </Typography.Text>
                ) : (
                  <Typography.Text type="secondary">通道默认</Typography.Text>
                ),
            },
            {
              title: "认领",
              width: 110,
              render: (_: unknown, r: AlertCurEventItem) =>
                r.acked ? (
                  <Tag color="processing" title={r.ack_expires_at ? `至 ${formatDateTime(r.ack_expires_at)}` : undefined}>
                    {r.ack_by || "已认领"}
                  </Tag>
                ) : (
                  <Typography.Text type="secondary">-</Typography.Text>
                ),
            },
            { title: "值", dataIndex: "value", width: 90 },
            { title: "开始", dataIndex: "starts_at", width: 170, render: (v) => formatDateTime(v) || "-" },
            { title: "更新", dataIndex: "updated_at", width: 170, render: (v) => formatDateTime(v) || "-" },
            {
              title: "操作",
              width: 280,
              fixed: "right",
              render: (_: unknown, r: AlertCurEventItem) => (
                <Space size={0} onClick={(e) => e.stopPropagation()}>
                  <Button type="link" size="small" icon={<CheckOutlined />} onClick={() => void toggleAck(r)}>
                    {r.acked ? "取消认领" : "认领"}
                  </Button>
                  <Button type="link" size="small" icon={<BellOutlined />} onClick={() => openDetail(toDetailFromCur(r))}>
                    通知
                  </Button>
                  <Button
                    type="link"
                    size="small"
                    icon={<StopOutlined />}
                    onClick={() => ctx.openSilenceForEvent?.(toDetailFromCur(r))}
                  >
                    静默
                  </Button>
                  <Button
                    type="link"
                    size="small"
                    icon={<RobotOutlined />}
                    disabled={!r.fingerprint}
                    onClick={() => void runAiExplain(r)}
                  >
                    AI
                  </Button>
                </Space>
              ),
            },
          ]}
          locale={{ emptyText: "当前没有正在告警的实例" }}
        />
      ) : null}

      {view === "lifecycle" ? (
        <ResizableTable
          rowKey="id"
          loading={loading}
          dataSource={hisRows}
          pagination={tablePagination({
            current: hisPage,
            pageSize: hisPageSize,
            total: hisTotal,
            onChange: (page, pageSize) => void loadHis(page, pageSize, keyword),
          })}
          scroll={{ x: 1080 }}
          onRow={(r) => ({
            onClick: () => openDetail(toDetailFromHis(r)),
            style: { cursor: "pointer" },
          })}
          columns={[
            { title: "告警名", dataIndex: "alertname", width: 180, ellipsis: true },
            {
              title: "级别",
              dataIndex: "severity",
              width: 90,
              render: (v: string) => <Tag>{v}</Tag>,
            },
            { title: "摘要", dataIndex: "summary", width: 280, ellipsis: true },
            { title: "开始", dataIndex: "starts_at", width: 170, render: (v) => formatDateTime(v) || "-" },
            { title: "恢复", dataIndex: "resolved_at", width: 170, render: (v) => formatDateTime(v) || "-" },
            {
              title: "操作",
              width: 90,
              fixed: "right",
              render: (_: unknown, r: AlertHisEventItem) => (
                <Button type="link" size="small" icon={<BellOutlined />} onClick={(e) => { e.stopPropagation(); openDetail(toDetailFromHis(r)); }}>
                  通知
                </Button>
              ),
            },
          ]}
          locale={{ emptyText: "还没有已恢复的记录。正在告警的实例不会出现在这里。" }}
        />
      ) : null}

      {view === "delivery" ? (
        <Suspense fallback={<Card loading />}>
          <AlertConfigCenterPanel
            embedded
            hideTabs
            activeTab="history"
            onTabChange={() => undefined}
            initialEventCategory={ctx.historyEventCategory}
            projectContextId={ctx.projectContextId}
          />
        </Suspense>
      ) : null}

      <AlertEventDetailDrawer
        open={detailOpen}
        target={detail}
        onClose={() => setDetailOpen(false)}
        projectId={ctx.projectContextId}
        allowSilence={String(detail?.status || "").toLowerCase() !== "resolved"}
        onCustomSilence={(t) => ctx.openSilenceForEvent?.(t)}
        onAiExplain={(t) => {
          void runAiExplain({
            id: 0,
            fingerprint: t.fingerprint,
            alertname: t.alertname,
            severity: t.severity || "",
            status: t.status || "firing",
            summary: t.summary,
            project_id: t.project_id,
          });
        }}
      />

      <Modal
        title={aiTarget ? `AI 解读：${aiTarget.alertname || aiTarget.fingerprint}` : "AI 解读"}
        open={aiOpen}
        onCancel={() => {
          setAiOpen(false);
          setAiResult(null);
          setAiTarget(null);
        }}
        footer={
          <Space>
            <Button
              type="primary"
              loading={aiLoading}
              disabled={!aiTarget?.fingerprint}
              onClick={() => aiTarget && void runAiExplain(aiTarget)}
            >
              重新解读
            </Button>
            <Button onClick={() => setAiOpen(false)}>关闭</Button>
          </Space>
        }
        width={720}
        destroyOnClose
      >
        {aiTarget ? (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            <Typography.Text type="secondary">
              指纹 <Typography.Text code>{aiTarget.fingerprint}</Typography.Text>
              {aiTarget.summary ? ` · ${aiTarget.summary}` : ""}
            </Typography.Text>
            {aiLoading && !aiResult ? <Card loading /> : null}
            {aiResult ? (
              <>
                <Alert
                  type="info"
                  showIcon
                  message={aiResult.ai_summary || "（无摘要）"}
                  description={
                    aiResult.provider
                      ? `模型：${aiResult.provider}${aiResult.model ? ` / ${aiResult.model}` : ""}`
                      : undefined
                  }
                />
                {aiResult.root_causes?.length ? (
                  <Card size="small" title="可能原因">
                    <ul style={{ margin: 0, paddingLeft: 18 }}>
                      {aiResult.root_causes.map((c, i) => (
                        <li key={i}>
                          {typeof c === "object" && c && "title" in c
                            ? String((c as { title?: string }).title || JSON.stringify(c))
                            : JSON.stringify(c)}
                        </li>
                      ))}
                    </ul>
                  </Card>
                ) : null}
                {aiResult.actions?.length ? (
                  <Card size="small" title="建议动作">
                    <ul style={{ margin: 0, paddingLeft: 18 }}>
                      {aiResult.actions.map((a, i) => (
                        <li key={i}>
                          {typeof a === "object" && a && "title" in a
                            ? String((a as { title?: string }).title || JSON.stringify(a))
                            : JSON.stringify(a)}
                        </li>
                      ))}
                    </ul>
                  </Card>
                ) : null}
                {aiResult.raw_reply ? (
                  <Typography.Paragraph>
                    <pre style={{ whiteSpace: "pre-wrap", margin: 0, fontSize: 12 }}>{aiResult.raw_reply}</pre>
                  </Typography.Paragraph>
                ) : null}
              </>
            ) : null}
          </Space>
        ) : null}
      </Modal>
    </Space>
  );
}
