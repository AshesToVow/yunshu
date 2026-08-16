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


export function DatasourcesTab() {
  const ctx = useAlertMonitor();
  return (
<Space direction="vertical" style={{ width: "100%" }} size="middle">
                <Alert
                  type="info"
                  showIcon
                  message="数据源 = 时序库（Prometheus / VictoriaMetrics）"
                  description="平台规则中心只查询此处配置的 PromQL API。采集由 Telegraf / Pushgateway / blackbox 完成，推荐经 Consul 服务发现由 Prometheus scrape；无需配置 Alertmanager。"
                />
                <Space>
                  <Button type="primary" icon={<PlusOutlined />} onClick={ctx.openDsCreate}>
                    新建数据源
                  </Button>
                  <Button icon={<ReloadOutlined />} onClick={() => void ctx.loadDatasources(ctx.projectContextId)}>
                    刷新
                  </Button>
                </Space>
                <Table rowKey="id" columns={ctx.dsColumns} dataSource={ctx.dsList} pagination={{ pageSize: 20, showSizeChanger: true }} scroll={{ x: 900 }} />
              </Space>
  );
}
