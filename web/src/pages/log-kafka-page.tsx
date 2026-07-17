import { ClusterOutlined, LinkOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Col, Row, Space, Statistic, Table, Tag, Typography, message } from "antd";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  getKafkaConfigPreview,
  getKafkaQueueStats,
  type KafkaConfigPreview,
  type KafkaPartitionLag,
  type KafkaQueueStats,
} from "../services/log-platform";
import { formatDateTime } from "../utils/format";

export function LogKafkaPage() {
  const [stats, setStats] = useState<KafkaQueueStats | null>(null);
  const [cfg, setCfg] = useState<KafkaConfigPreview | null>(null);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [s, c] = await Promise.all([getKafkaQueueStats(), getKafkaConfigPreview()]);
      setStats(s);
      setCfg(c);
    } catch (e: any) {
      message.error(e?.message || "加载 Kafka 状态失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
    const t = window.setInterval(() => void load(), 15000);
    return () => window.clearInterval(t);
  }, [load]);

  const partitions: KafkaPartitionLag[] = stats?.partitions ?? [];

  return (
    <div className="log-kafka-page">
      <Space style={{ marginBottom: 16 }} wrap>
        <Typography.Title level={4} style={{ margin: 0 }}>
          Kafka 队列
        </Typography.Title>
        <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>
          刷新
        </Button>
        <Link to="/dict-entries?keyword=kafka_">
          <Button icon={<LinkOutlined />}>数据字典配置</Button>
        </Link>
      </Space>

      {stats?.message ? (
        <Alert style={{ marginBottom: 16 }} type={stats.sink_via_kafka ? "info" : "warning"} showIcon message={stats.message} />
      ) : null}

      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={12} md={6}>
          <Card size="small">
            <Statistic
              title="中转开关"
              value={stats?.sink_via_kafka ? "Kafka → ES" : "直写 ES"}
              prefix={<ClusterOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card size="small">
            <Statistic title="消费积压 (lag)" value={stats?.lag_total ?? 0} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card size="small">
            <Statistic title="已消费" value={stats?.consumed_total ?? 0} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card size="small">
            <Statistic title="已写入 ES" value={stats?.written_total ?? 0} />
          </Card>
        </Col>
      </Row>

      <Card title="连接与消费状态" style={{ marginBottom: 16 }} size="small">
        <Space wrap size={[16, 8]}>
          <span>
            启用：
            {cfg?.enabled ? <Tag color="green">true</Tag> : <Tag>false</Tag>}
          </span>
          <span>
            消费者：
            {stats?.consumer_running ? <Tag color="green">运行中</Tag> : <Tag color="orange">未运行</Tag>}
          </span>
          <span>Topic：{stats?.topic || cfg?.topic || "-"}</span>
          <span>消费组：{stats?.consumer_group || cfg?.consumer_group || "-"}</span>
          <span>Brokers：{(stats?.brokers || cfg?.brokers || []).join(", ") || "-"}</span>
          <span>SASL：{stats?.has_sasl || cfg?.has_password ? "已配置" : "无"}</span>
          <span>最近消费：{stats?.last_consume_at ? formatDateTime(stats.last_consume_at) : "-"}</span>
          <span>错误数：{stats?.error_total ?? 0}</span>
        </Space>
        {stats?.last_error ? (
          <Alert style={{ marginTop: 12 }} type="error" showIcon message={stats.last_error} />
        ) : null}
      </Card>

      <Card title="分区积压" size="small">
        <Table
          rowKey="partition"
          size="small"
          loading={loading}
          pagination={false}
          dataSource={partitions}
          locale={{ emptyText: stats?.sink_via_kafka ? "暂无分区数据" : "未启用 Kafka 中转" }}
          columns={[
            { title: "分区", dataIndex: "partition", width: 100 },
            { title: "高水位", dataIndex: "high_water_mark" },
            { title: "消费位移", dataIndex: "consumer_offset", render: (v: number) => (v < 0 ? "-" : v) },
            {
              title: "Lag",
              dataIndex: "lag",
              render: (v: number) => (v < 0 ? <Tag>-</Tag> : <Tag color={v > 1000 ? "red" : v > 0 ? "orange" : "green"}>{v}</Tag>),
            },
          ]}
        />
      </Card>
    </div>
  );
}
