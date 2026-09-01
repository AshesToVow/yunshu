import { Card, Table, Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useMemo } from "react";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import { usePlugins } from "../contexts/plugin-context";
import type { PluginInfo } from "../services/plugins";

export function PluginsPage() {
  const { plugins, enabled, loading } = usePlugins();

  const columns = useMemo<ColumnsType<PluginInfo>>(
    () => [
      {
        title: "插件",
        dataIndex: "name",
        key: "name",
        render: (name: string) => <Tag color="blue">{name}</Tag>,
      },
      {
        title: "说明",
        dataIndex: "description",
        key: "description",
        ellipsis: true,
      },
      {
        title: "状态",
        dataIndex: "enabled",
        key: "enabled",
        width: 120,
        render: (on: boolean) => (on ? <Tag color="success">已启用</Tag> : <Tag>未启用</Tag>),
      },
    ],
    [],
  );

  return (
    <div className="page-stack">
      <PageTelemetryHeader
        label="[ MODULE REGISTRY ]"
        title="业务插件"
        subtitle={`与后端 config.yaml plugins.enabled 同步。当前启用：${enabled.length ? enabled.join(", ") : "NONE"}`}
        meta={[`COUNT / ${plugins.length}`, `ACTIVE / ${enabled.length}`]}
      />
      <Card bordered={false}>
        <Table<PluginInfo>
          rowKey="name"
          loading={loading}
          columns={columns}
          dataSource={plugins}
          pagination={false}
          size="middle"
        />
      </Card>
    </div>
  );
}
