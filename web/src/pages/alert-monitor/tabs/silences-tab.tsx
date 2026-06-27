// @ts-nocheck
import {
  Alert,
  Button,
  Card,
  Collapse,
  Input,
  Radio,
  Segmented,
  Select,
  Space,
  Table,
  Typography,
} from "antd";
import { PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { lazy, Suspense } from "react";
import { useAlertMonitor } from "../context";


export function SilencesTab() {
  const ctx = useAlertMonitor();
  return (
<Space direction="vertical" style={{ width: "100%" }} size="middle">
                <Alert
                  type="info"
                  showIcon
                  message="平台静默 ≠ Alertmanager 静默"
                  description={
                    <Space direction="vertical" size={8} style={{ width: "100%" }}>
                      <span>
                        下方「平台静默规则」在<strong>服务端入站分发前</strong>按 ctx.matchers 与告警 labels 比对（含<strong>平台内置监控规则</strong>与 Webhook），命中则<strong>不再向通道发送</strong>。不会调用 Alertmanager API；Prometheus 活跃告警列表仅用于对照 Prom 侧规则，其「静默」按钮与平台规则不是同一批告警名。静默平台规则请用「监控规则与策略」表格中的<strong>静默</strong>，或手动填写 <Typography.Text code>monitor_rule_id</Typography.Text> + <Typography.Text code>alertname</Typography.Text>。
                      </span>
                      <Space wrap>
                        <Button size="small" onClick={ctx.openHistoryTab}>查看静默后的历史记录</Button>
                      </Space>
                    </Space>
                  }
                />
                <Typography.Title level={5} style={{ margin: 0 }}>
                  Prometheus 活跃告警（只读快照）
                </Typography.Title>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  与「历史告警记录」不同：此处直接查询当前项目下数据源的 <Typography.Text code>/api/v1/alerts</Typography.Text>
                  （跟随顶栏「全局项目上下文」），用于对照 Prometheus UI 中 Firing 的条目是否已进 Webhook 链路。
                  {ctx.silenceDatasource ? (
                    <>
                      {" "}
                      当前使用数据源：<Typography.Text strong>{ctx.silenceDatasource.name}</Typography.Text>
                      {ctx.dsList.length > 1 ? `（本项目共 ${ctx.dsList.length} 个，默认首个已启用）` : null}。
                    </>
                  ) : (
                    <> 当前项目暂无可用数据源。</>
                  )}
                </Typography.Paragraph>
                <Space wrap>
                  <Button type="primary" loading={ctx.nativeAlertsLoading} onClick={() => void ctx.loadNativeSilAlerts()}>
                    拉取活跃告警
                  </Button>
                  <Button
                    onClick={() => {
                      const rows = ctx.nativeAlertsRows.filter((r) => ctx.selectedNativeAlertKeys.includes(r.key));
                      ctx.openQuickSilence(rows);
                    }}
                    disabled={ctx.selectedNativeAlertKeys.length === 0}
                  >
                    批量静默
                  </Button>
                </Space>
                <Table
                  rowKey="key"
                  size="small"
                  loading={ctx.nativeAlertsLoading}
                  columns={ctx.nativeAlertsColumns}
                  dataSource={ctx.nativeAlertsRows}
                  rowSelection={{
                    type: "checkbox",
                    selectedRowKeys: ctx.selectedNativeAlertKeys,
                    onChange: (keys) => ctx.setSelectedNativeAlertKeys(keys.map((k) => String(k))),
                  }}
                  pagination={{ pageSize: 8 }}
                  locale={{
                    emptyText: ctx.silenceDatasourceId
                      ? "暂无数据，请点击「拉取活跃告警」"
                      : "请先在「数据源」Tab 为当前项目创建 Prometheus 数据源",
                  }}
                />
                <Typography.Title level={5} style={{ margin: 0 }}>
                  静默列表
                </Typography.Title>
                {ctx.projectContextId ? (
                  <Typography.Text type="secondary">
                    已按顶栏项目 #{ctx.projectContextId} 筛选；新建静默将写入该项目
                  </Typography.Text>
                ) : (
                  <Typography.Text type="secondary">未选顶栏项目时显示全部静默（含历史全局记录）</Typography.Text>
                )}
                <Space>
                  <Button type="primary" onClick={() => void ctx.releaseSelectedSilences()} disabled={ctx.selectedSilenceIds.length === 0}>
                    批量解除静默
                  </Button>
                  <Button icon={<ReloadOutlined />} onClick={() => void ctx.loadSilences()}>
                    刷新
                  </Button>
                </Space>
                <Table
                  rowKey="id"
                  rowSelection={{
                    type: "checkbox",
                    selectedRowKeys: ctx.selectedSilenceIds,
                    onChange: (keys) => ctx.setSelectedSilenceIds(keys.map((k) => Number(k)).filter((n) => Number.isFinite(n))),
                  }}
                  columns={ctx.silColumns}
                  dataSource={ctx.silenceList}
                  pagination={false}
                  scroll={{ x: 960 }}
                />
              </Space>
  );
}
