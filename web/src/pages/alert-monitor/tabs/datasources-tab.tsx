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
                  message="数据源是告警入口与巡检基础"
                  description="这里维护 Prometheus / Alertmanager 的访问地址，供平台规则巡检、活跃告警快照与 PromQL 调试复用。"
                />
                <Space>
                  <Button type="primary" icon={<PlusOutlined />} onClick={ctx.openDsCreate}>
                    新建数据源
                  </Button>
                  <Button icon={<ReloadOutlined />} onClick={() => void ctx.loadDatasources(ctx.projectContextId)}>
                    刷新
                  </Button>
                </Space>
                <Table rowKey="id" columns={ctx.dsColumns} dataSource={ctx.dsList} pagination={false} scroll={{ x: 900 }} />
              </Space>
  );
}
