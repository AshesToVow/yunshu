import { ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Drawer, Modal, Select, Space, Table, message } from "antd";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import {
  deleteKafkamgmtGroup,
  getKafkamgmtGroupLag,
  getKafkamgmtGroupMembers,
  listKafkamgmtConnections,
  listKafkamgmtGroups,
  type KafkaGroupInfo,
  type KafkaGroupLag,
  type KafkaGroupMember,
  type KafkamgmtConnection,
} from "../services/kafkamgmt";
import { extractApiErrorMessage } from "../services/http";

export function KafkamgmtGroupsPage() {
  const [connections, setConnections] = useState<KafkamgmtConnection[]>([]);
  const [connectionId, setConnectionId] = useState<number>();
  const [groups, setGroups] = useState<KafkaGroupInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [activeGroup, setActiveGroup] = useState("");
  const [members, setMembers] = useState<KafkaGroupMember[]>([]);
  const [lag, setLag] = useState<KafkaGroupLag | null>(null);

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
      setGroups((await listKafkamgmtGroups(connectionId)) || []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载消费组失败"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (connectionId) void load();
  }, [connectionId]);

  async function openDetail(groupId: string) {
    setActiveGroup(groupId);
    setOpen(true);
    setMembers([]);
    setLag(null);
    try {
      const [m, l] = await Promise.all([
        getKafkamgmtGroupMembers(groupId, connectionId),
        getKafkamgmtGroupLag(groupId, connectionId),
      ]);
      setMembers(m || []);
      setLag(l);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载详情失败"));
    }
  }

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="消费组"
        description="查看消费组成员与 Lag。删除消费组需连接负责人或超管；平台保留组（如 yunshu-log-es）不可删。"
        breadcrumbs={[{ title: "Kafka 管理控制台" }, { title: "消费组" }]}
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
          rowKey="group_id"
          loading={loading}
          dataSource={groups}
          pagination={{ pageSize: 20, showSizeChanger: true }}
          columns={[
            {
              title: "Group ID",
              dataIndex: "group_id",
              render: (v: string) => <a onClick={() => void openDetail(v)}>{v}</a>,
            },
            {
              title: "操作",
              width: 160,
              className: "yunshu-table-actions-cell",
              render: (_: unknown, row?: KafkaGroupInfo) =>
                row ? (
                  <Space size="small" className="yunshu-table-actions">
                    <Button type="link" size="small" onClick={() => void openDetail(row.group_id)}>
                      详情
                    </Button>
                    <Button
                      type="link"
                      size="small"
                      danger
                      onClick={() => {
                        Modal.confirm({
                          title: `确认删除消费组「${row.group_id}」？`,
                          onOk: async () => {
                            try {
                              await deleteKafkamgmtGroup({ connection_id: connectionId, group_id: row.group_id });
                              message.success("已删除");
                              void load();
                            } catch (e) {
                              message.error(extractApiErrorMessage(e, "删除失败"));
                            }
                          },
                        });
                      }}
                    >
                      删除
                    </Button>
                  </Space>
                ) : null,
            },
          ]}
        />
      </Card>

      <Drawer title={`消费组：${activeGroup}`} open={open} onClose={() => setOpen(false)} width={800}>
        <Card size="small" title={`Lag 合计：${lag?.lag_total ?? "—"}`} style={{ marginBottom: 12 }}>
          <Table
            size="small"
            rowKey={(r) => `${r.topic}-${r.partition}`}
            pagination={{ pageSize: 10 }}
            dataSource={lag?.partitions || []}
            columns={[
              { title: "Topic", dataIndex: "topic", ellipsis: true },
              { title: "分区", dataIndex: "partition", width: 70 },
              { title: "Committed", dataIndex: "committed", width: 110 },
              { title: "End", dataIndex: "end", width: 110 },
              { title: "Lag", dataIndex: "lag", width: 90 },
            ]}
          />
        </Card>
        <Card size="small" title="成员">
          <Table
            size="small"
            rowKey="member_id"
            pagination={false}
            dataSource={members}
            columns={[
              { title: "Member", dataIndex: "member_id", ellipsis: true },
              { title: "Client", dataIndex: "client_id", width: 140 },
              { title: "Host", dataIndex: "client_host", width: 140 },
              {
                title: "Topics",
                dataIndex: "topics",
                render: (v?: string[]) => (v || []).join(", ") || "—",
              },
            ]}
          />
        </Card>
      </Drawer>
    </div>
  );
}

export default KafkamgmtGroupsPage;
