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


export function RulesTab() {
  const ctx = useAlertMonitor();
  return (
<Space direction="vertical" style={{ width: "100%" }} size="middle">
                <Space wrap align="center">
                  <Button type="primary" icon={<PlusOutlined />} onClick={ctx.openRuleCreate}>
                    新建规则
                  </Button>
                  <Button icon={<ReloadOutlined />} onClick={() => void Promise.all([ctx.loadRules(ctx.projectContextId), ctx.loadDatasources(ctx.projectContextId)])}>
                    刷新
                  </Button>
                  <Segmented
                    value={ctx.ruleEnabledFilter}
                    options={[
                      { label: `全部 (${ctx.ruleEnabledStats.total})`, value: "all" },
                      { label: `启用 (${ctx.ruleEnabledStats.enabled})`, value: "enabled" },
                      { label: `停用 (${ctx.ruleEnabledStats.disabled})`, value: "disabled" },
                    ]}
                    onChange={(v) => ctx.setRuleEnabledFilter(v as "all" | "enabled" | "disabled")}
                  />
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    当前显示 {ctx.ruleDisplayList.length} 条
                    {ctx.ruleEnabledFilter !== "all" ? `（共 ${ctx.ruleEnabledStats.total} 条）` : ""}
                    · 停用规则不会参与平台 PromQL 评估
                  </Typography.Text>
                </Space>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  这里配置的是平台内监控规则：平台会定时对数据源执行 PromQL，命中后走与 Webhook 入站相同的通知与历史记录链路。规则级「处理人」与所选「值班表」当前班次通知邮箱会在告警 outgoing 中合并去重；部门选择为根部门子树全员。
                </Typography.Paragraph>
                <Alert
                  type="info"
                  showIcon
                  message="规则配置建议"
                  description={
                    <Space direction="vertical" size={8} style={{ width: "100%" }}>
                      <span>
                        建议先确认四项：1) <Typography.Text code>datasource</Typography.Text> 选正确集群；2){" "}
                        <Typography.Text code>severity</Typography.Text> 与策略匹配（critical/warning/info）；3){" "}
                        <Typography.Text code>for_seconds</Typography.Text> 防抖时长；4) <Typography.Text code>eval_interval_seconds</Typography.Text>{" "}
                        评估频率（常用 30s/60s，只决定多久查一次 PromQL）。
                      </span>
                      <span>
                        <strong>通知重复间隔</strong>不由「间隔(s)」控制，而由配置 <Typography.Text code>alert.repeat_interval_seconds</Typography.Text>（默认 300s）与{" "}
                        <Typography.Text code>group_interval_seconds</Typography.Text>（默认 60s）控制；持续 ctx.firing 时不会每 30s 都发钉钉。
                      </span>
                      <Space wrap>
                        <Button size="small" onClick={ctx.openHistoryTab}>查看规则触发历史</Button>
                      </Space>
                    </Space>
                  }
                />
                <Table rowKey="id" columns={ctx.ruleColumns} dataSource={ctx.ruleDisplayList} pagination={false} scroll={{ x: 1100 }} />
              </Space>
  );
}
