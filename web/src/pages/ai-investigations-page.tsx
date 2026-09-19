import { ExperimentOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Descriptions, Select, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import {
  getAIInvestigation,
  listAIInvestigations,
  type AIInvestigation,
  type AIInvestigationReport,
} from "../services/ai";
import { extractApiErrorMessage } from "../services/http";

function parseReport(row?: AIInvestigation | null): AIInvestigationReport | null {
  if (!row?.report_json) return null;
  try {
    return JSON.parse(row.report_json) as AIInvestigationReport;
  } catch {
    return null;
  }
}

function statusColor(s: string) {
  switch (s) {
    case "done":
      return "success";
    case "failed":
      return "error";
    case "cancelled":
      return "default";
    case "analyzing":
    case "collecting":
    case "awaiting_approval":
      return "processing";
    default:
      return "default";
  }
}

const IN_FLIGHT = new Set(["collecting", "analyzing", "awaiting_approval"]);

export function AiInvestigationsPage() {
  const [searchParams] = useSearchParams();
  const focusId = Number(searchParams.get("id") || 0);
  const [kind, setKind] = useState<string | undefined>();
  const [list, setList] = useState<AIInvestigation[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<AIInvestigation | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  async function refresh(p = page) {
    setLoading(true);
    try {
      const res = await listAIInvestigations({ kind, page: p, page_size: 20 });
      setList(res?.list || []);
      setTotal(res?.total || 0);
      setPage(res?.page || p);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载调查列表失败"));
    } finally {
      setLoading(false);
    }
  }

  async function openDetail(id: number) {
    try {
      const row = await getAIInvestigation(id);
      setSelected(row);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载调查详情失败"));
    }
  }

  useEffect(() => {
    void refresh(1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind]);

  useEffect(() => {
    if (focusId > 0) void openDetail(focusId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [focusId]);

  useEffect(() => {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
    if (!selected?.id || !IN_FLIGHT.has(selected.status)) return;
    pollRef.current = setInterval(() => {
      void getAIInvestigation(selected.id)
        .then((row) => {
          setSelected(row);
          if (!IN_FLIGHT.has(row.status)) {
            void refresh();
          }
        })
        .catch(() => undefined);
    }, 2500);
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selected?.id, selected?.status]);

  const report = parseReport(selected);

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="AI 调查"
        description="告警 / Pod / CI / 对话等场景的采集→分析→报告；写动作会挂接审批（awaiting_approval）。"
        breadcrumbs={[{ title: "AI" }, { title: "AI 调查" }]}
        extra={
          <Space>
            <Link to="/ai/assistant">运维助手</Link>
            <Link to="/ai/approvals">操作审批</Link>
            <Button icon={<ReloadOutlined />} onClick={() => void refresh()}>
              刷新
            </Button>
          </Space>
        }
      />

      <Card className="table-card">
        <Space wrap style={{ marginBottom: 12 }}>
          <Select
            allowClear
            placeholder="类型"
            style={{ width: 140 }}
            value={kind}
            options={[
              { value: "alert", label: "告警" },
              { value: "pod", label: "Pod" },
              { value: "cicd", label: "CI/CD" },
              { value: "chat", label: "对话" },
              { value: "incident", label: "综合" },
            ]}
            onChange={(v) => setKind(v)}
          />
          <Typography.Text type="secondary">共 {total} 条</Typography.Text>
        </Space>
        <Table
          rowKey="id"
          loading={loading}
          dataSource={list}
          locale={{ emptyText: "暂无调查记录" }}
          pagination={{
            current: page,
            pageSize: 20,
            total,
            onChange: (p) => void refresh(p),
          }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "标题", dataIndex: "title", ellipsis: true },
            {
              title: "类型",
              dataIndex: "kind",
              width: 90,
              render: (v: string) => <Tag>{v}</Tag>,
            },
            {
              title: "状态",
              dataIndex: "status",
              width: 140,
              render: (v: string) => <Tag color={statusColor(v)}>{v}</Tag>,
            },
            { title: "更新时间", dataIndex: "updated_at", width: 180 },
            {
              title: "操作",
              width: 100,
              render: (_: unknown, row: AIInvestigation) => (
                <Button type="link" size="small" icon={<ExperimentOutlined />} onClick={() => void openDetail(row.id)}>
                  详情
                </Button>
              ),
            },
          ]}
        />
      </Card>

      {selected ? (
        <Card
          className="table-card"
          title={`调查 #${selected.id} · ${selected.title}`}
          extra={
            <Button type="link" onClick={() => setSelected(null)}>
              关闭
            </Button>
          }
        >
          <Descriptions size="small" column={2} style={{ marginBottom: 12 }}>
            <Descriptions.Item label="状态">
              <Tag color={statusColor(selected.status)}>{selected.status}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="类型">{selected.kind}</Descriptions.Item>
            <Descriptions.Item label="项目">{selected.project_id || "-"}</Descriptions.Item>
            <Descriptions.Item label="集群">{selected.cluster_id || "-"}</Descriptions.Item>
            <Descriptions.Item label="命名空间">{selected.namespace || "-"}</Descriptions.Item>
            <Descriptions.Item label="资源/指纹">{selected.resource || selected.fingerprint || "-"}</Descriptions.Item>
            <Descriptions.Item label="审批单" span={2}>
              {(() => {
                const ids = new Set<number>();
                if (selected.approval_id) ids.add(selected.approval_id);
                try {
                  const analysis = selected.analysis_json ? JSON.parse(selected.analysis_json) : null;
                  const list = (analysis?.approvals || analysis?.actions || []) as Array<{
                    approval_id?: number;
                    action?: string;
                  }>;
                  for (const a of list) {
                    if (a?.approval_id && (a.action === "pending_approval" || !a.action)) {
                      ids.add(Number(a.approval_id));
                    }
                  }
                } catch {
                  /* ignore */
                }
                if (ids.size === 0) return "-";
                return (
                  <Space wrap size="small">
                    {[...ids].map((id) => (
                      <Link key={id} to="/ai/approvals">
                        #{id}
                      </Link>
                    ))}
                    <Typography.Text type="secondary">（去审批）</Typography.Text>
                  </Space>
                );
              })()}
            </Descriptions.Item>
          </Descriptions>
          {IN_FLIGHT.has(selected.status) ? (
            <Alert
              type="info"
              showIcon
              message={
                selected.status === "awaiting_approval"
                  ? "等待审批中，审批完成后将自动刷新状态…"
                  : "调查进行中，自动刷新状态…"
              }
              style={{ marginBottom: 12 }}
            />
          ) : null}
          {selected.error_msg ? <Alert type="error" showIcon message={selected.error_msg} style={{ marginBottom: 12 }} /> : null}
          {report ? (
            <Space direction="vertical" style={{ width: "100%" }} size="middle">
              <Alert type="info" showIcon message="摘要" description={report.summary || "（无）"} />
              {report.root_causes?.length ? (
                <Card size="small" title="可能根因">
                  <pre style={{ margin: 0, whiteSpace: "pre-wrap", fontSize: 12 }}>
                    {JSON.stringify(report.root_causes, null, 2)}
                  </pre>
                </Card>
              ) : null}
              {report.actions?.length ? (
                <Card size="small" title="建议动作">
                  <pre style={{ margin: 0, whiteSpace: "pre-wrap", fontSize: 12 }}>
                    {JSON.stringify(report.actions, null, 2)}
                  </pre>
                </Card>
              ) : null}
              {report.evidence?.length ? (
                <Card size="small" title="证据">
                  <pre style={{ margin: 0, whiteSpace: "pre-wrap", fontSize: 12, maxHeight: 280, overflow: "auto" }}>
                    {JSON.stringify(report.evidence, null, 2)}
                  </pre>
                </Card>
              ) : null}
              {report.raw_reply ? (
                <Card size="small" title="原始回复">
                  <Typography.Paragraph style={{ whiteSpace: "pre-wrap", marginBottom: 0 }}>
                    {report.raw_reply}
                  </Typography.Paragraph>
                </Card>
              ) : null}
            </Space>
          ) : (
            <Typography.Text type="secondary">
              {IN_FLIGHT.has(selected.status) ? "等待报告生成…" : "暂无结构化报告"}
            </Typography.Text>
          )}
        </Card>
      ) : null}
    </div>
  );
}

export default AiInvestigationsPage;
