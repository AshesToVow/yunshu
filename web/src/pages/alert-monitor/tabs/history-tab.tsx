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

export function HistoryTab() {
  const ctx = useAlertMonitor();
  return (
<Space direction="vertical" style={{ width: "100%" }} size="middle">
                <Alert
                  type="info"
                  showIcon
                  message="历史记录是统一观测出口"
                  description="无论告警来自 Prometheus + Alertmanager，还是来自平台监控规则，最终都会在这里查看命中订阅、抑制原因码（error_message）、通道结果与错误信息。success=留痕 表示策略拦截未外发。"
                />
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
              </Space>
  );
}
