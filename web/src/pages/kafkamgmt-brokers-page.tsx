import { ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Select, Space, Table, message } from "antd";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import {
  listKafkamgmtBrokers,
  listKafkamgmtConnections,
  type KafkaBrokerInfo,
  type KafkamgmtConnection,
} from "../services/kafkamgmt";
import { extractApiErrorMessage } from "../services/http";

export function KafkamgmtBrokersPage() {
  const [connections, setConnections] = useState<KafkamgmtConnection[]>([]);
  const [connectionId, setConnectionId] = useState<number>();
  const [brokers, setBrokers] = useState<KafkaBrokerInfo[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    void listKafkamgmtConnections()
      .then((list) => {
        setConnections(list || []);
        const def = list?.find((c) => c.is_default) || list?.[0];
        if (def) setConnectionId(def.id);
      })
      .catch((e) => message.error(extractApiErrorMessage(e, "加载连接失败")));
  }, []);

  async function load() {
    setLoading(true);
    try {
      setBrokers((await listKafkamgmtBrokers(connectionId)) || []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载 Broker 失败"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (connectionId) void load();
  }, [connectionId]);

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="Broker"
        description="查看集群 Broker 节点列表。"
        breadcrumbs={[{ title: "Kafka 管理控制台" }, { title: "Broker" }]}
        extra={
          <Space wrap>
            <Link to="/kafkamgmt/connections">连接管理</Link>
            <Select
              style={{ minWidth: 220 }}
              placeholder="选择连接"
              value={connectionId}
              options={connections.map((c) => ({ value: c.id, label: c.name }))}
              onChange={setConnectionId}
            />
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
          </Space>
        }
      />
      <Card className="table-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={brokers}
          pagination={false}
          columns={[
            { title: "ID", dataIndex: "id", width: 80 },
            { title: "Host", dataIndex: "host" },
            { title: "Port", dataIndex: "port", width: 100 },
            { title: "Rack", dataIndex: "rack", width: 120, render: (v?: string) => v || "—" },
          ]}
        />
      </Card>
    </div>
  );
}

export default KafkamgmtBrokersPage;
