import { extractApiErrorMessage } from "../../../services/http";
import {
  Alert,
  Button,
  Card,
  Input,
  Modal,
  Segmented,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { ReloadOutlined, RobotOutlined } from "@ant-design/icons";
import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { useAlertMonitor } from "../context";
import { listCurEvents, listHisEvents, type AlertCurEventItem, type AlertHisEventItem } from "../../../services/alerts";
import { analyzeAlertExplainAI, type AIAlertExplainResult } from "../../../services/ai";
import { formatDateTime } from "../../../utils/format";

const AlertConfigCenterPanel = lazy(async () => {
  const mod = await import("../../alert-config-center-panel");
  return { default: mod.AlertConfigCenterPanel };
});

type EventView = "current" | "lifecycle" | "delivery";

export function HistoryTab() {
  const ctx = useAlertMonitor();
  const [view, setView] = useState<EventView>("current");
  const [keyword, setKeyword] = useState("");
  const [curRows, setCurRows] = useState<AlertCurEventItem[]>([]);
  const [hisRows, setHisRows] = useState<AlertHisEventItem[]>([]);
  const [curTotal, setCurTotal] = useState(0);
  const [hisTotal, setHisTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [aiOpen, setAiOpen] = useState(false);
  const [aiLoading, setAiLoading] = useState(false);
  const [aiTarget, setAiTarget] = useState<AlertCurEventItem | null>(null);
  const [aiResult, setAiResult] = useState<AIAlertExplainResult | null>(null);

  const loadCur = useCallback(async () => {
    setLoading(true);
    try {
      const r = await listCurEvents({
        project_id: ctx.projectContextId || undefined,
        keyword: keyword || undefined,
        page: 1,
        page_size: 50,
      });
      setCurRows(r.list ?? r.items ?? []);
      setCurTotal(r.total ?? 0);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载当前告警失败"));
    } finally {
      setLoading(false);
    }
  }, [ctx.projectContextId, keyword]);

  const loadHis = useCallback(async () => {
    setLoading(true);
    try {
      const r = await listHisEvents({
        project_id: ctx.projectContextId || undefined,
        keyword: keyword || undefined,
        page: 1,
        page_size: 50,
      });
      setHisRows(r.list ?? r.items ?? []);
      setHisTotal(r.total ?? 0);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载历史告警失败"));
    } finally {
      setLoading(false);
    }
  }, [ctx.projectContextId, keyword]);

  useEffect(() => {
    if (view === "current") void loadCur();
    if (view === "lifecycle") void loadHis();
  }, [view, loadCur, loadHis]);

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

  return (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      <Alert
        type="info"
        showIcon
        message="事件台：当前告警 / 生命周期历史 / 投递流水"
        description="当前告警为正在 firing 的实例（屏蔽不入库）。恢复后迁入生命周期历史。投递流水仍为通道发送审计（原历史记录）。"
      />
      <Space wrap>
        <Segmented
          value={view}
          onChange={(v) => setView(v as EventView)}
          options={[
            { label: `当前告警 (${curTotal})`, value: "current" },
            { label: `生命周期 (${hisTotal})`, value: "lifecycle" },
            { label: "投递流水", value: "delivery" },
          ]}
        />
        {view !== "delivery" ? (
          <>
            <Input.Search
              allowClear
              placeholder="搜索告警名/指纹/摘要"
              style={{ width: 260 }}
              onSearch={(v) => setKeyword(v)}
            />
            <Button
              icon={<ReloadOutlined />}
              onClick={() => void (view === "current" ? loadCur() : loadHis())}
            >
              刷新
            </Button>
          </>
        ) : null}
      </Space>

      {view === "current" ? (
        <Table
          rowKey="id"
          loading={loading}
          dataSource={curRows}
          pagination={{ pageSize: 20, showSizeChanger: true }}
          scroll={{ x: 1100 }}
          columns={[
            { title: "告警名", dataIndex: "alertname", width: 180, ellipsis: true },
            {
              title: "级别",
              dataIndex: "severity",
              width: 90,
              render: (v: string) => <Tag color={v === "critical" ? "red" : "orange"}>{v}</Tag>,
            },
            { title: "集群", dataIndex: "cluster", width: 120, ellipsis: true },
            { title: "摘要", dataIndex: "summary", ellipsis: true },
            { title: "值", dataIndex: "value", width: 90 },
            { title: "开始", dataIndex: "starts_at", width: 170, render: (v) => formatDateTime(v) || "-" },
            { title: "更新", dataIndex: "updated_at", width: 170, render: (v) => formatDateTime(v) || "-" },
            { title: "指纹", dataIndex: "fingerprint", width: 140, ellipsis: true },
            {
              title: "操作",
              width: 110,
              fixed: "right",
              render: (_: unknown, r: AlertCurEventItem) => (
                <Button
                  type="link"
                  size="small"
                  icon={<RobotOutlined />}
                  disabled={!r.fingerprint}
                  onClick={() => void runAiExplain(r)}
                >
                  AI 解读
                </Button>
              ),
            },
          ]}
          locale={{ emptyText: "暂无当前告警" }}
        />
      ) : null}

      {view === "lifecycle" ? (
        <Table
          rowKey="id"
          loading={loading}
          dataSource={hisRows}
          pagination={{ pageSize: 20, showSizeChanger: true }}
          scroll={{ x: 1000 }}
          columns={[
            { title: "告警名", dataIndex: "alertname", width: 180, ellipsis: true },
            {
              title: "级别",
              dataIndex: "severity",
              width: 90,
              render: (v: string) => <Tag>{v}</Tag>,
            },
            { title: "摘要", dataIndex: "summary", ellipsis: true },
            { title: "开始", dataIndex: "starts_at", width: 170, render: (v) => formatDateTime(v) || "-" },
            { title: "恢复", dataIndex: "resolved_at", width: 170, render: (v) => formatDateTime(v) || "-" },
            { title: "指纹", dataIndex: "fingerprint", width: 160, ellipsis: true },
          ]}
          locale={{ emptyText: "暂无生命周期历史" }}
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
