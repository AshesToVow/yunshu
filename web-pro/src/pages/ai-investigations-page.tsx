// @ts-nocheck
import { ExperimentOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Descriptions, Select, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { Link, useSearchParams } from '@umijs/max';
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
    case "analyzing":
    case "collecting":
      return "processing";
    case "awaiting_approval":
      return "warning";
    default:
      return "default";
  }
}

export function AiInvestigationsPage() {
  const [searchParams] = useSearchParams();
  const focusId = Number(searchParams.get("id") || 0);
  const [kind, setKind] = useState<string | undefined>();
  const [list, setList] = useState<AIInvestigation[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<AIInvestigation | null>(null);

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

  const report = parseReport(selected);

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="AI 调查"
        description="告警 / Pod / CI 等场景的采集→分析→报告闭环；可从告警历史、Pod 排障入口发起。"
        breadcrumbs={[{ title: "AI" }, { title: "AI 调查" }]}
        extra={
          <Space>
            <Link to="/ai/assistant">运维助手</Link>
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
            ]}
            onChange={(v) => setKind(v)}
          />
          <Typography.Text type="secondary">共 {total} 条</Typography.Text>
        </Space>
        <Table
          rowKey="id"
          loading={loading}
          dataSource={list}
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
              width: 120,
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
          </Descriptions>
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
            <Typography.Text type="secondary">暂无结构化报告</Typography.Text>
          )}
        </Card>
      ) : null}
    </div>
  );
}

export default AiInvestigationsPage;
