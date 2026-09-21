import { DownOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Drawer,
  Dropdown,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  message,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import {
  consumeKafkamgmtMessages,
  createKafkamgmtPartitions,
  createKafkamgmtTopic,
  deleteKafkamgmtTopic,
  getKafkamgmtTopicConfig,
  getKafkamgmtTopicDetail,
  listKafkamgmtConnections,
  listKafkamgmtTopics,
  produceKafkamgmtMessage,
  type KafkaConsumedMessage,
  type KafkamgmtConnection,
  type KafkaTopicInfo,
} from "../services/kafkamgmt";
import { extractApiErrorMessage } from "../services/http";

export function KafkamgmtTopicsPage() {
  const [connections, setConnections] = useState<KafkamgmtConnection[]>([]);
  const [connectionId, setConnectionId] = useState<number>();
  const [topics, setTopics] = useState<KafkaTopicInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createForm] = Form.useForm();
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailTopic, setDetailTopic] = useState<string>("");
  const [detailParts, setDetailParts] = useState<Array<{ id: number; leader: number; replicas: number[]; isr: number[] }>>([]);
  const [detailConfig, setDetailConfig] = useState<Record<string, string>>({});
  const [produceOpen, setProduceOpen] = useState(false);
  const [consumeOpen, setConsumeOpen] = useState(false);
  const [activeTopic, setActiveTopic] = useState("");
  const [produceForm] = Form.useForm();
  const [consumeForm] = Form.useForm();
  const [consumed, setConsumed] = useState<KafkaConsumedMessage[]>([]);
  const [partitionOpen, setPartitionOpen] = useState(false);
  const [partitionForm] = Form.useForm();

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
      setTopics((await listKafkamgmtTopics(connectionId)) || []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载 Topic 失败"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (connectionId) void load();
  }, [connectionId]);

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    if (!kw) return topics;
    return topics.filter((t) => t.name.toLowerCase().includes(kw));
  }, [topics, keyword]);

  async function onCreate() {
    const values = await createForm.validateFields();
    setCreating(true);
    try {
      await createKafkamgmtTopic({
        connection_id: connectionId,
        name: values.name,
        num_partitions: values.num_partitions,
        replication_factor: values.replication_factor,
      });
      message.success("已创建");
      setCreateOpen(false);
      void load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建失败"));
    } finally {
      setCreating(false);
    }
  }

  async function openDetail(topic: string) {
    setDetailTopic(topic);
    setDetailOpen(true);
    try {
      const [d, cfg] = await Promise.all([
        getKafkamgmtTopicDetail(topic, connectionId),
        getKafkamgmtTopicConfig(topic, connectionId),
      ]);
      setDetailParts(d?.partitions || []);
      setDetailConfig(cfg || {});
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载详情失败"));
    }
  }

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="Topic 管理"
        description="查看/创建/删除 Topic，扩容分区，受控生产与抽样消费。删除与生产需连接负责人或超管权限。"
        breadcrumbs={[{ title: "Kafka 管理控制台" }, { title: "Topic 管理" }]}
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
            <Input.Search allowClear placeholder="过滤 Topic" onSearch={setKeyword} onChange={(e) => setKeyword(e.target.value)} style={{ width: 200 }} />
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                createForm.resetFields();
                createForm.setFieldsValue({ num_partitions: 1, replication_factor: 1 });
                setCreateOpen(true);
              }}
            >
              创建 Topic
            </Button>
          </Space>
        }
      />
      <Card className="table-card">
        <Table
          rowKey="name"
          loading={loading}
          dataSource={filtered}
          pagination={{ pageSize: 20, showSizeChanger: true }}
          columns={[
            {
              title: "Topic",
              dataIndex: "name",
              render: (v: string, row: KafkaTopicInfo) => (
                <Space>
                  <a onClick={() => void openDetail(v)}>{v}</a>
                  {row.internal ? <Tag>内部</Tag> : null}
                </Space>
              ),
            },
            { title: "分区", dataIndex: "partitions", width: 90 },
            { title: "副本", dataIndex: "replication_factor", width: 90 },
            {
              title: "操作",
              width: 200,
              className: "yunshu-table-actions-cell",
              render: (_: unknown, row?: KafkaTopicInfo) =>
                row && !row.internal ? (
                  <Space size="small" wrap className="yunshu-table-actions">
                    <Button
                      type="link"
                      size="small"
                      onClick={() => {
                        setActiveTopic(row.name);
                        produceForm.resetFields();
                        setProduceOpen(true);
                      }}
                    >
                      生产
                    </Button>
                    <Dropdown
                      menu={{
                        items: [
                          {
                            key: "consume",
                            label: "抽样消费",
                            onClick: () => {
                              setActiveTopic(row.name);
                              consumeForm.setFieldsValue({ max_messages: 10, from: "latest", timeout_ms: 3000 });
                              setConsumed([]);
                              setConsumeOpen(true);
                            },
                          },
                          {
                            key: "partitions",
                            label: "扩容分区",
                            onClick: () => {
                              setActiveTopic(row.name);
                              partitionForm.setFieldsValue({ total_count: row.partitions + 1 });
                              setPartitionOpen(true);
                            },
                          },
                          {
                            key: "delete",
                            label: "删除",
                            danger: true,
                            onClick: () => {
                              Modal.confirm({
                                title: `确认删除 Topic「${row.name}」？`,
                                onOk: async () => {
                                  try {
                                    await deleteKafkamgmtTopic({ connection_id: connectionId, topic: row.name });
                                    message.success("已删除");
                                    void load();
                                  } catch (e) {
                                    message.error(extractApiErrorMessage(e, "删除失败"));
                                  }
                                },
                              });
                            },
                          },
                        ],
                      }}
                    >
                      <Button type="link" size="small">
                        更多 <DownOutlined />
                      </Button>
                    </Dropdown>
                  </Space>
                ) : null,
            },
          ]}
        />
      </Card>

      <Modal title="创建 Topic" open={createOpen} onCancel={() => setCreateOpen(false)} onOk={() => void onCreate()} confirmLoading={creating} destroyOnClose>
        <Form form={createForm} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="num_partitions" label="分区数">
            <InputNumber min={1} max={128} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="replication_factor" label="副本因子">
            <InputNumber min={1} max={10} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer title={`Topic：${detailTopic}`} open={detailOpen} onClose={() => setDetailOpen(false)} width={720}>
        <Table
          size="small"
          rowKey="id"
          pagination={false}
          dataSource={detailParts}
          columns={[
            { title: "分区", dataIndex: "id", width: 80 },
            { title: "Leader", dataIndex: "leader", width: 80 },
            { title: "Replicas", dataIndex: "replicas", render: (v: number[]) => (v || []).join(",") },
            { title: "ISR", dataIndex: "isr", render: (v: number[]) => (v || []).join(",") },
          ]}
          style={{ marginBottom: 16 }}
        />
        <Table
          size="small"
          rowKey="k"
          pagination={{ pageSize: 10 }}
          dataSource={Object.entries(detailConfig).map(([k, v]) => ({ k, v }))}
          columns={[
            { title: "配置项", dataIndex: "k", width: 240 },
            { title: "值", dataIndex: "v", ellipsis: true },
          ]}
        />
      </Drawer>

      <Modal
        title={`生产 → ${activeTopic}`}
        open={produceOpen}
        onCancel={() => setProduceOpen(false)}
        onOk={async () => {
          const values = await produceForm.validateFields();
          try {
            await produceKafkamgmtMessage({
              connection_id: connectionId,
              topic: activeTopic,
              key: values.key,
              value: values.value,
              partition: values.partition,
            });
            message.success("已发送");
            setProduceOpen(false);
          } catch (e) {
            message.error(extractApiErrorMessage(e, "生产失败"));
          }
        }}
        destroyOnClose
      >
        <Form form={produceForm} layout="vertical">
          <Form.Item name="key" label="Key">
            <Input />
          </Form.Item>
          <Form.Item name="value" label="Value" rules={[{ required: true }]}>
            <Input.TextArea rows={6} />
          </Form.Item>
          <Form.Item name="partition" label="分区（可选）">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`抽样消费 ← ${activeTopic}`}
        open={consumeOpen}
        onCancel={() => setConsumeOpen(false)}
        width={800}
        footer={
          <Space>
            <Button onClick={() => setConsumeOpen(false)}>关闭</Button>
            <Button
              type="primary"
              onClick={async () => {
                const values = await consumeForm.validateFields();
                try {
                  const rows = await consumeKafkamgmtMessages({
                    connection_id: connectionId,
                    topic: activeTopic,
                    max_messages: values.max_messages,
                    from: values.from,
                    timeout_ms: values.timeout_ms,
                    partition: values.partition,
                  });
                  setConsumed(rows || []);
                  message.success(`拉取 ${rows?.length || 0} 条`);
                } catch (e) {
                  message.error(extractApiErrorMessage(e, "消费失败"));
                }
              }}
            >
              拉取
            </Button>
          </Space>
        }
        destroyOnClose
      >
        <Form form={consumeForm} layout="inline" style={{ marginBottom: 12 }}>
          <Form.Item name="max_messages" label="条数">
            <InputNumber min={1} max={50} />
          </Form.Item>
          <Form.Item name="from" label="起点">
            <Select style={{ width: 120 }} options={[
              { value: "latest", label: "latest" },
              { value: "earliest", label: "earliest" },
            ]} />
          </Form.Item>
          <Form.Item name="timeout_ms" label="超时ms">
            <InputNumber min={500} max={15000} />
          </Form.Item>
          <Form.Item name="partition" label="分区">
            <InputNumber min={0} />
          </Form.Item>
        </Form>
        <Table
          size="small"
          rowKey={(_, i) => String(i)}
          dataSource={consumed}
          pagination={false}
          scroll={{ y: 360 }}
          columns={[
            { title: "P", dataIndex: "partition", width: 50 },
            { title: "Offset", dataIndex: "offset", width: 90 },
            { title: "Key", dataIndex: "key", width: 120, ellipsis: true },
            { title: "Value", dataIndex: "value", ellipsis: true },
            { title: "Time", dataIndex: "time", width: 170 },
          ]}
        />
      </Modal>

      <Modal
        title={`扩容分区 → ${activeTopic}`}
        open={partitionOpen}
        onCancel={() => setPartitionOpen(false)}
        onOk={async () => {
          const values = await partitionForm.validateFields();
          try {
            await createKafkamgmtPartitions({
              connection_id: connectionId,
              topic: activeTopic,
              total_count: values.total_count,
            });
            message.success("已提交扩容");
            setPartitionOpen(false);
            void load();
          } catch (e) {
            message.error(extractApiErrorMessage(e, "扩容失败"));
          }
        }}
        destroyOnClose
      >
        <Form form={partitionForm} layout="vertical">
          <Form.Item name="total_count" label="目标总分区数（只能增加）" rules={[{ required: true }]}>
            <InputNumber min={1} max={256} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default KafkamgmtTopicsPage;
