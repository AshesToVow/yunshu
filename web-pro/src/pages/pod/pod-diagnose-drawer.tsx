// @ts-nocheck
/**
 * Pod 排障诊断抽屉（RF-04 拆分产物）
 * 从 pod-page.tsx 原地搬迁 JSX。
 */
import { Alert, Button, Card, Drawer, Space, Table, Tag, Typography } from "antd";
import type { AIPodDiagnoseResult } from "../../services/ai";
import type { PodDiagnoseResult, PodItem } from "../../services/pods";

export type PodDiagnoseDrawerProps = {
  diagnoseOpen: boolean;
  setDiagnoseOpen: (v: boolean) => void;
  selected: PodItem | null;
  diagnoseLoading: boolean;
  diagnoseResult: PodDiagnoseResult | null;
  aiDiagnoseLoading: boolean;
  aiDiagnoseResult: AIPodDiagnoseResult | null;
  handleAIDiagnose: () => void | Promise<void>;
  aiInvestigateLoading?: boolean;
  handleAIInvestigate?: () => void | Promise<void>;
};

export function PodDiagnoseDrawer({
  diagnoseOpen,
  setDiagnoseOpen,
  selected,
  diagnoseLoading,
  diagnoseResult,
  aiDiagnoseLoading,
  aiDiagnoseResult,
  handleAIDiagnose,
  aiInvestigateLoading,
  handleAIInvestigate,
}: PodDiagnoseDrawerProps) {
  return (
        <Drawer
          title={selected ? `Pod 排障 - ${selected.namespace}/${selected.name}` : "Pod 排障"}
          open={diagnoseOpen}
          onClose={() => setDiagnoseOpen(false)}
          width={920}
          extra={
            <Space>
              <Button
                type="primary"
                loading={aiDiagnoseLoading}
                disabled={!diagnoseResult || diagnoseLoading}
                onClick={() => void handleAIDiagnose()}
              >
                AI 分析
              </Button>
              {handleAIInvestigate ? (
                <Button
                  loading={aiInvestigateLoading}
                  disabled={!selected || diagnoseLoading}
                  onClick={() => void handleAIInvestigate()}
                >
                  AI 调查
                </Button>
              ) : null}
            </Space>
          }
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
              {aiDiagnoseResult ? (
                <Card size="small" title={`AI 分析（${aiDiagnoseResult.provider} / ${aiDiagnoseResult.model}）`}>
                  <Space direction="vertical" style={{ width: "100%" }} size="small">
                    <Typography.Paragraph style={{ marginBottom: 0 }}>
                      {aiDiagnoseResult.ai_summary || "（无摘要）"}
                    </Typography.Paragraph>
                    {(aiDiagnoseResult.root_causes || []).length > 0 ? (
                      <>
                        <Typography.Text strong>可能根因</Typography.Text>
                        {(aiDiagnoseResult.root_causes || []).map((c, i) => (
                          <Alert
                            key={`cause-${i}`}
                            type="warning"
                            showIcon
                            message={String(c.title || c.cause || `根因 ${i + 1}`)}
                            description={String(c.detail || c.description || JSON.stringify(c))}
                          />
                        ))}
                      </>
                    ) : null}
                    {(aiDiagnoseResult.actions || []).length > 0 ? (
                      <>
                        <Typography.Text strong>建议动作</Typography.Text>
                        {(aiDiagnoseResult.actions || []).map((a, i) => (
                          <Alert
                            key={`action-${i}`}
                            type="info"
                            showIcon
                            message={String(a.title || a.action || `建议 ${i + 1}`)}
                            description={String(a.detail || a.description || JSON.stringify(a))}
                          />
                        ))}
                      </>
                    ) : null}
                    {!aiDiagnoseResult.root_causes?.length && !aiDiagnoseResult.actions?.length && aiDiagnoseResult.raw_reply ? (
                      <pre className="code-block-panel" style={{ maxHeight: 240, fontSize: 12 }}>
                        {aiDiagnoseResult.raw_reply}
                      </pre>
                    ) : null}
                  </Space>
                </Card>
              ) : null}
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
  );
}
