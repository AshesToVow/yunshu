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

const AlertConfigCenterPanel = lazy(async () => {
  const mod = await import("../../alert-config-center-panel");
  return { default: mod.AlertConfigCenterPanel };
});

export function PoliciesTab() {
  const ctx = useAlertMonitor();
  return (
<Space direction="vertical" style={{ width: "100%" }} size="middle">
                <Alert
                  type="info"
                  showIcon
                  message="告警路由树统一处理 Webhook 入站告警与平台规则告警"
                  description={
                    <Space direction="vertical" size={8} style={{ width: "100%" }}>
                      <span>订阅节点按 labels / regex 命中接收组与通道，并执行节点静默窗口与恢复通知；它不等于 Prometheus 规则，也不直接改 Alertmanager 的静默状态。</span>
                      <Space wrap>
                        <Button size="small" onClick={ctx.openHistoryTab}>查看历史记录</Button>
                      </Space>
                    </Space>
                  }
                />
                <Collapse
                  size="small"
                  items={[
                    {
                      key: "roles",
                      label: "功能边界：策略 / 监控规则 / 历史记录",
                      children: (
                        <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                          <ul style={{ margin: 0, paddingLeft: 18 }}>
                            <li>
                              <strong>告警路由树</strong>：对 Alertmanager Webhook 入站告警和平台规则告警统一生效，决定命中路由节点、通知接收组与通道，并执行路由静默窗口。
                            </li>
                            <li>
                              <strong>监控规则与值班</strong>：平台定时向已登记数据源执行 PromQL，命中后走同一套通知链路。
                            </li>
                            <li>
                              <strong>告警抑制</strong>：源告警触发时抑制目标告警（类似 Alertmanager inhibit_rules），见「告警抑制」Tab。
                            </li>
                            <li>
                              <strong>历史记录</strong>：统一查看命中、抑制原因码、发送成功/失败的结果证据。
                            </li>
                          </ul>
                        </Typography.Paragraph>
                      ),
                    },
                  ]}
                />
                <Suspense fallback={<Card loading />}>
                  <AlertConfigCenterPanel
                    embedded
                    hideTabs
                    activeTab="subscriptions"
                    onTabChange={() => undefined}
                    projectContextId={ctx.projectContextId}
                  />
                </Suspense>
              </Space>
  );
}
