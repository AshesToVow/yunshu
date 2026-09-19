import { ClusterOutlined, DeleteOutlined, LinkOutlined, ReloadOutlined } from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Col,
  Popconfirm,
  Row,
  Select,
  Space,
  Statistic,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
} from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import {
  deleteKafkaTopic,
  getKafkaConfigPreview,
  getKafkaQueueStats,
  type KafkaConfigPreview,
  type KafkaPartitionLag,
  type KafkaQueueStats,
} from "../services/log-platform";
import { extractApiErrorMessage } from "../services/http";
import { formatDateTime } from "../utils/format";

export function LogRetentionPage() {
  const [kafkaStats, setKafkaStats] = useState<KafkaQueueStats | null>(null);
  const [kafkaCfg, setKafkaCfg] = useState<KafkaConfigPreview | null>(null);
  const [kafkaLoading, setKafkaLoading] = useState(false);
  const [topicPage, setTopicPage] = useState(1);
  const [topicPageSize, setTopicPageSize] = useState(10);

  const reloadKafka = useCallback(async () => {
    setKafkaLoading(true);
    try {
      const [s, c] = await Promise.all([getKafkaQueueStats(), getKafkaConfigPreview()]);
      setKafkaStats(s);
      setKafkaCfg(c);
    } catch (e: unknown) {
      message.error(extractApiErrorMessage(e));
    } finally {
      setKafkaLoading(false);
    }
  }, []);

  useEffect(() => {
    void reloadKafka();
    const t = window.setInterval(() => void reloadKafka(), 15000);
    return () => window.clearInterval(t);
  }, [reloadKafka]);

  const partitions: KafkaPartitionLag[] = kafkaStats?.partitions ?? [];
  const topicRows = useMemo(() => {
    const topicList = kafkaStats?.topics || [];
    const map = new Map<string, { topic: string; lag_total: number; partitions: KafkaPartitionLag[] }>();
    for (const t of topicList) {
      map.set(t, { topic: t, lag_total: 0, partitions: [] });
    }
    const allowOrphan = topicList.length === 0;
    for (const p of partitions) {
      const topic = p.topic || "-";
      if (!allowOrphan && !map.has(topic)) continue;
      const row = map.get(topic) ?? { topic, lag_total: 0, partitions: [] };
      row.partitions.push(p);
      if (p.lag > 0) row.lag_total += p.lag;
      map.set(topic, row);
    }
    return Array.from(map.values()).sort((a, b) => b.lag_total - a.lag_total || a.topic.localeCompare(b.topic));
  }, [partitions, kafkaStats?.topics]);

  const showEmptyTopicHint = !!kafkaStats?.sink_via_kafka && (kafkaStats?.topics || []).length === 0;

  async function handleDeleteTopic(topic: string) {
    try {
      await deleteKafkaTopic(topic);
      message.success(`已删除 Topic：${topic}`);
      await reloadKafka();
    } catch (e: unknown) {
      message.error(extractApiErrorMessage(e));
    }
  }

  return (
    <div className="log-retention-page page-stack">
      <OpsPageHeader
        title="Kafka 队列"
        description="观测 Loggie → Kafka → ES 中转链路；ES 存储与保留策略已迁移至 ES 管理控制台。"
        breadcrumbs={[{ title: "日志平台" }, { title: "Kafka 队列" }]}
        extra={
          <Space>
            <Link to="/esmgmt/storage">日志存储与保留</Link>
            <Link to="/dict-entries?keyword=kafka_">
              <Button icon={<LinkOutlined />}>Kafka 字典</Button>
            </Link>
            <Button icon={<ReloadOutlined />} loading={kafkaLoading} onClick={() => void reloadKafka()}>
              刷新
            </Button>
          </Space>
        }
      />

      <Alert
        type="info"
        showIcon
        message="Topic 命名：yunshu-agent-{ip}-日期 / yunshu-k8s-{cluster}-p{project}-日期。积压下降需消费者运行并从 earliest 追平历史分区。"
      />

      {showEmptyTopicHint ? (
        <Alert type="warning" showIcon message="暂无 Agent Topic，请先引导/热更 Agent" />
      ) : null}

      {kafkaStats?.message && !showEmptyTopicHint ? (
        <Alert type={kafkaStats.sink_via_kafka ? "info" : "warning"} showIcon message={kafkaStats.message} />
      ) : null}

      <Row gutter={[12, 12]}>
        <Col xs={24} sm={12} md={6}>
          <Card size="small">
            <Statistic
              title="中转开关"
              value={kafkaStats?.sink_via_kafka ? "Kafka → ES" : "直写 ES"}
              prefix={<ClusterOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card size="small">
            <Tooltip title="Broker 消费组未提交位移的积压">
              <Statistic title="消费积压 (lag)" value={kafkaStats?.lag_total ?? 0} />
            </Tooltip>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card size="small">
            <Tooltip title="本进程从 Kafka Fetch 的条数（重启清零）">
              <Statistic title="本进程已拉取" value={kafkaStats?.consumed_total ?? 0} />
            </Tooltip>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card size="small">
            <Tooltip title="本进程成功写入 ES 的条数（重启清零）">
              <Statistic title="本进程已写 ES" value={kafkaStats?.written_total ?? 0} />
            </Tooltip>
          </Card>
        </Col>
      </Row>

      <Card title="连接与消费组" size="small">
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 12 }}
          message="「消费者」= Yunshu 后端 Kafka→ES 消费循环（消费组如下），不是 Topic 列表。"
        />
        <Space wrap size={[16, 8]}>
          <span>
            启用：{kafkaCfg?.enabled ? <Tag color="green">true</Tag> : <Tag>false</Tag>}
          </span>
          <span>
            Yunshu 消费者：
            {kafkaStats?.consumer_running ? (
              <Tag color="green">运行中{kafkaStats.consumer_workers ? ` ×${kafkaStats.consumer_workers}` : ""}</Tag>
            ) : (
              <Tag color="orange">未运行</Tag>
            )}
          </span>
          <span>
            消费组：<Tag color="blue">{kafkaStats?.consumer_group || kafkaCfg?.consumer_group || "-"}</Tag>
          </span>
          <span>Topic 数：{(kafkaStats?.topics || []).length}</span>
          <span>Brokers：{(kafkaStats?.brokers || kafkaCfg?.brokers || []).join(", ") || "-"}</span>
          <span>最近拉取：{kafkaStats?.last_consume_at ? formatDateTime(kafkaStats.last_consume_at) : "-"}</span>
        </Space>
        {kafkaStats?.last_error && !String(kafkaStats.last_error).includes("暂无 Agent Topic") ? (
          <Alert style={{ marginTop: 12 }} type="error" showIcon message={kafkaStats.last_error} />
        ) : null}
      </Card>

      <Card className="table-card" title="Topic 积压（按 lag 降序，展开查看分区）" size="small">
        <Table
          rowKey="topic"
          size="small"
          loading={kafkaLoading}
          dataSource={topicRows}
          scroll={{ x: 560 }}
          pagination={{
            current: topicPage,
            pageSize: topicPageSize,
            showSizeChanger: true,
            pageSizeOptions: ["10", "20", "50"],
            showTotal: (t) => `共 ${t} 个 Topic`,
            onChange: (p, ps) => {
              setTopicPage(p);
              setTopicPageSize(ps);
            },
          }}
          expandable={{
            expandedRowRender: (row) => (
              <Table
                size="small"
                pagination={false}
                rowKey={(r) => `${r.topic ?? ""}-${r.partition}`}
                dataSource={row.partitions}
                columns={[
                  { title: "分区", dataIndex: "partition", width: 80 },
                  { title: "高水位", dataIndex: "high_water_mark", width: 120 },
                  {
                    title: "消费位移",
                    dataIndex: "consumer_offset",
                    width: 120,
                    render: (v: number) => (v < 0 ? "-" : v),
                  },
                  {
                    title: "Lag",
                    dataIndex: "lag",
                    width: 100,
                    render: (v: number) =>
                      v < 0 ? <Tag>-</Tag> : <Tag color={v > 1000 ? "red" : v > 0 ? "orange" : "green"}>{v}</Tag>,
                  },
                ]}
              />
            ),
            rowExpandable: (row) => (row.partitions?.length ?? 0) > 0,
          }}
          columns={[
            { title: "Topic", dataIndex: "topic", ellipsis: true },
            { title: "分区数", width: 90, render: (_, row) => row.partitions.length || "-" },
            {
              title: "Lag 合计",
              dataIndex: "lag_total",
              width: 110,
              render: (v: number) => <Tag color={v > 1000 ? "red" : v > 0 ? "orange" : "green"}>{v}</Tag>,
            },
            {
              title: "操作",
              width: 100,
              render: (_, row) => (
                <Popconfirm title={`删除 Topic ${row.topic}？`} onConfirm={() => void handleDeleteTopic(row.topic)}>
                  <Button type="link" danger size="small" icon={<DeleteOutlined />}>
                    删除
                  </Button>
                </Popconfirm>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
}

export default LogRetentionPage;
