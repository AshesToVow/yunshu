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
                  message="通知与路由：决定规则产生的事件发给谁"
                  description={
                    <Space direction="vertical" size={8} style={{ width: "100%" }}>
                      <span>
                        订阅节点按 labels / regex 命中接收组与通道，并执行节点静默窗口与恢复通知。规则在「规则中心」配置；此处不负责 PromQL 评测。
                      </span>
                      <Space wrap>
                        <Button size="small" onClick={ctx.openHistoryTab}>查看事件台</Button>
                      </Space>
                    </Space>
                  }
                />
                <Collapse
                  size="small"
                  items={[
                    {
                      key: "roles",
                      label: "功能边界：规则 / 路由 / 降噪 / 事件",
                      children: (
                        <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                          <ul style={{ margin: 0, paddingLeft: 18 }}>
                            <li>
                              <strong>规则中心</strong>：对数据源执行 PromQL，是唯一告警产生入口（Telegraf/blackbox/Pushgateway 指标进库后在此配规则）。
                            </li>
                            <li>
                              <strong>通知与路由</strong>：事件产生后如何分发给接收组与通道。
                            </li>
                            <li>
                              <strong>降噪</strong>：静默与抑制，减少打扰。
                            </li>
                            <li>
                              <strong>事件台</strong>：查看命中、抑制原因、发送结果。
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
