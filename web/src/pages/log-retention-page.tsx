import {
  ClusterOutlined,
  DeleteOutlined,
  LinkOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Row,
  Select,
  Space,
  Statistic,
  Switch,
  Table,
  Tabs,
  Tag,
  message,
} from "antd";
import { useCallback, useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  deleteProjectLogRetention,
  getESStorageStats,
  getGlobalLogRetention,
  getKafkaConfigPreview,
  getKafkaQueueStats,
  getProjects,
  listLogRetentionPolicies,
  runLogRetentionCleanup,
  upsertGlobalLogRetention,
  upsertProjectLogRetention,
  type ESStorageStats,
  type KafkaConfigPreview,
  type KafkaPartitionLag,
  type KafkaQueueStats,
  type LogRetentionItem,
  type ProjectItem,
} from "../services/log-platform";
import { formatDateTime } from "../utils/format";

type GlobalForm = {
  retention_days: number;
  enabled: boolean;
  index_pattern?: string;
  remark?: string;
};

export function LogRetentionPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const tab = searchParams.get("tab") === "kafka" ? "kafka" : "es";

  const [globalForm] = Form.useForm<GlobalForm>();
  const [stats, setStats] = useState<ESStorageStats | null>(null);
  const [policies, setPolicies] = useState<LogRetentionItem[]>([]);
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [projectId, setProjectId] = useState<number>();
  const [projectDays, setProjectDays] = useState(30);
  const [projectEnabled, setProjectEnabled] = useState(true);

  const [kafkaStats, setKafkaStats] = useState<KafkaQueueStats | null>(null);
  const [kafkaCfg, setKafkaCfg] = useState<KafkaConfigPreview | null>(null);
  const [kafkaLoading, setKafkaLoading] = useState(false);

  const reloadES = useCallback(async () => {
    setLoading(true);
    try {
      const [global, list, storage, proj] = await Promise.all([
        getGlobalLogRetention(),
        listLogRetentionPolicies(),
        getESStorageStats().catch(() => null),
        getProjects({ page: 1, page_size: 1000 }),
      ]);
      globalForm.setFieldsValue({
        retention_days: global.retention_days,
        enabled: global.enabled,
        index_pattern: global.index_pattern || undefined,
        remark: global.remark || undefined,
      });
      setPolicies(list.list);
      setStats(storage);
      setProjects(proj.list);
    } catch (e: unknown) {
      message.error(String((e as Error)?.message ?? e));
    } finally {
      setLoading(false);
    }
  }, [globalForm]);

  const reloadKafka = useCallback(async () => {
    setKafkaLoading(true);
    try {
      const [s, c] = await Promise.all([getKafkaQueueStats(), getKafkaConfigPreview()]);
      setKafkaStats(s);
      setKafkaCfg(c);
    } catch (e: unknown) {
      message.error(String((e as Error)?.message ?? e));
    } finally {
      setKafkaLoading(false);
    }
  }, []);

  useEffect(() => {
    void reloadES();
  }, [reloadES]);

  useEffect(() => {
    if (tab !== "kafka") return;
    void reloadKafka();
    const t = window.setInterval(() => void reloadKafka(), 15000);
    return () => window.clearInterval(t);
  }, [tab, reloadKafka]);

  async function saveGlobal() {
    const values = await globalForm.validateFields();
    await upsertGlobalLogRetention(values);
    message.success("全局保留策略已保存");
    await reloadES();
  }

  async function saveProjectOverride() {
    if (!projectId) {
      message.warning("请选择项目");
      return;
    }
    await upsertProjectLogRetention(projectId, { retention_days: projectDays, enabled: projectEnabled });
    message.success("项目覆盖策略已保存");
    await reloadES();
  }

  async function removeProjectOverride() {
    if (!projectId) return;
    await deleteProjectLogRetention(projectId);
    message.success("已删除项目覆盖，将继承全局策略");
    await reloadES();
  }

  const projectOverrides = policies.filter((p) => p.project_id !== 0);
  const partitions: KafkaPartitionLag[] = kafkaStats?.partitions ?? [];

  return (
    <div className="log-retention-page">
      <Tabs
        activeKey={tab}
        onChange={(key) => {
          if (key === "kafka") setSearchParams({ tab: "kafka" });
          else setSearchParams({});
        }}
        items={[
          {
            key: "es",
            label: "ES 索引保留",
            children: (
              <Space direction="vertical" size={12} style={{ width: "100%" }}>
                <Alert
                  type="info"
                  showIcon
                  message="按保留天数定时清理过期 ES 索引；Agent 启停与热更请到「Agent 管理」。推荐索引：yunshu-agent-{server_id}-YYYY.MM.DD。"
                />

                <Card
                  size="small"
                  title="ES 存储概览"
                  extra={
                    <Button size="small" icon={<ReloadOutlined />} onClick={() => void reloadES()} loading={loading}>
                      刷新
                    </Button>
                  }
                >
                  {stats ? (
                    <Space direction="vertical" size={12} style={{ width: "100%" }}>
                      <Row gutter={[16, 12]}>
                        <Col xs={12} sm={6}>
                          <Statistic title="全部索引" value={stats.index_count} suffix="个" />
                        </Col>
                        <Col xs={12} sm={6}>
                          <Statistic title="全部文档" value={stats.document_count} />
                        </Col>
                        <Col xs={12} sm={6}>
                          <Statistic title="全部占用" value={stats.store_human || "-"} />
                        </Col>
                        <Col xs={12} sm={6}>
                          <Statistic
                            title={`匹配 ${stats.index_pattern || "yunshu-agent-*"}`}
                            value={stats.pattern_index_count ?? 0}
                            suffix={`/ ${stats.pattern_store_human || "-"}`}
                          />
                        </Col>
                      </Row>
                      <Table
                        size="small"
                        rowKey="name"
                        pagination={{ pageSize: 8, showSizeChanger: true, size: "small" }}
                        dataSource={stats.indices ?? []}
                        columns={[
                          {
                            title: "索引名",
                            dataIndex: "name",
                            ellipsis: true,
                            render: (name: string, r: { matched_pattern?: boolean }) => (
                              <Space size={4}>
                                <span>{name}</span>
                                {r.matched_pattern ? <Tag color="blue">平台</Tag> : null}
                              </Space>
                            ),
                          },
                          {
                            title: "文档",
                            dataIndex: "docs_count",
                            width: 100,
                            render: (n: number) => (n ?? 0).toLocaleString(),
                          },
                          { title: "占用", dataIndex: "store_human", width: 100 },
                        ]}
                      />
                    </Space>
                  ) : (
                    <span>无法连接 ES 或未启用 elasticsearch.enabled</span>
                  )}
                </Card>

                <Row gutter={[12, 12]}>
                  <Col xs={24} lg={10}>
                    <Card
                      size="small"
                      title="全局默认策略"
                      extra={
                        <Space size={8}>
                          <Button
                            size="small"
                            icon={<PlayCircleOutlined />}
                            onClick={() =>
                              void (async () => {
                                const res = await runLogRetentionCleanup();
                                message.success(res.message || "清理完成");
                                await reloadES();
                              })()
                            }
                          >
                            立即清理
                          </Button>
                          <Button size="small" type="primary" icon={<SaveOutlined />} onClick={() => void saveGlobal()}>
                            保存
                          </Button>
                        </Space>
                      }
                    >
                      <Form form={globalForm} layout="vertical" size="small">
                        <Row gutter={12}>
                          <Col span={12}>
                            <Form.Item label="保留天数" name="retention_days" rules={[{ required: true }]} style={{ marginBottom: 12 }}>
                              <InputNumber min={1} max={3650} style={{ width: "100%" }} addonAfter="天" />
                            </Form.Item>
                          </Col>
                          <Col span={12}>
                            <Form.Item label="启用自动清理" name="enabled" valuePropName="checked" style={{ marginBottom: 12 }}>
                              <Switch />
                            </Form.Item>
                          </Col>
                          <Col span={24}>
                            <Form.Item label="索引模式" name="index_pattern" style={{ marginBottom: 12 }}>
                              <Input placeholder="默认 yunshu-agent-*" />
                            </Form.Item>
                          </Col>
                          <Col span={24}>
                            <Form.Item label="备注" name="remark" style={{ marginBottom: 0 }}>
                              <Input placeholder="可选" />
                            </Form.Item>
                          </Col>
                        </Row>
                      </Form>
                    </Card>
                  </Col>

                  <Col xs={24} lg={14}>
                    <Card size="small" title="项目级覆盖">
                      <Space wrap size={8} style={{ marginBottom: 12, width: "100%" }}>
                        <Select
                          style={{ minWidth: 220, flex: 1 }}
                          placeholder="选择项目"
                          options={projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` }))}
                          value={projectId}
                          onChange={setProjectId}
                          allowClear
                          size="small"
                        />
                        <InputNumber
                          size="small"
                          min={1}
                          max={3650}
                          value={projectDays}
                          onChange={(v) => setProjectDays(v ?? 30)}
                          addonAfter="天"
                        />
                        <span>
                          启用 <Switch size="small" checked={projectEnabled} onChange={setProjectEnabled} />
                        </span>
                        <Button size="small" type="primary" onClick={() => void saveProjectOverride()}>
                          保存覆盖
                        </Button>
                        <Button size="small" danger icon={<DeleteOutlined />} onClick={() => void removeProjectOverride()}>
                          删除
                        </Button>
                      </Space>
                      <Table
                        rowKey={(r) => `${r.project_id}-${r.id}`}
                        size="small"
                        dataSource={projectOverrides}
                        pagination={false}
                        locale={{ emptyText: "无项目覆盖，均继承全局策略" }}
                        columns={[
                          {
                            title: "项目",
                            dataIndex: "project_id",
                            render: (id: number) => {
                              const p = projects.find((x) => x.id === id);
                              return p ? `${p.name} (${p.code})` : id;
                            },
                          },
                          { title: "天数", dataIndex: "retention_days", width: 80 },
                          {
                            title: "启用",
                            dataIndex: "enabled",
                            width: 80,
                            render: (v: boolean) => (v ? <Tag color="green">是</Tag> : <Tag>否</Tag>),
                          },
                          { title: "索引模式", dataIndex: "index_pattern", ellipsis: true, render: (v?: string) => v || "-" },
                          {
                            title: "更新",
                            dataIndex: "updated_at",
                            width: 150,
                            render: (v?: string) => (v ? formatDateTime(v) : "-"),
                          },
                        ]}
                      />
                    </Card>
                  </Col>
                </Row>
              </Space>
            ),
          },
          {
            key: "kafka",
            label: "Kafka 队列",
            children: (
              <Space direction="vertical" size={12} style={{ width: "100%" }}>
                <Space wrap>
                  <Button icon={<ReloadOutlined />} loading={kafkaLoading} onClick={() => void reloadKafka()}>
                    刷新
                  </Button>
                  <Link to="/dict-entries?keyword=kafka_">
                    <Button icon={<LinkOutlined />}>数据字典配置</Button>
                  </Link>
                </Space>

                <Alert
                  type="info"
                  showIcon
                  message="开启 Kafka 后：每个 Agent 独立 Topic（yunshu-agent-{server_id}），Yunshu 使用消费组统一消费写 ES。引导/热更 Agent 时会自动建 Topic。"
                />

                {kafkaStats?.message ? (
                  <Alert type={kafkaStats.sink_via_kafka ? "info" : "warning"} showIcon message={kafkaStats.message} />
                ) : null}

                <Row gutter={[16, 16]}>
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
                      <Statistic title="消费积压 (lag)" value={kafkaStats?.lag_total ?? 0} />
                    </Card>
                  </Col>
                  <Col xs={24} sm={12} md={6}>
                    <Card size="small">
                      <Statistic title="已消费" value={kafkaStats?.consumed_total ?? 0} />
                    </Card>
                  </Col>
                  <Col xs={24} sm={12} md={6}>
                    <Card size="small">
                      <Statistic title="已写入 ES" value={kafkaStats?.written_total ?? 0} />
                    </Card>
                  </Col>
                </Row>

                <Card title="连接与消费组" size="small">
                  <Space wrap size={[16, 8]}>
                    <span>
                      启用：
                      {kafkaCfg?.enabled ? <Tag color="green">true</Tag> : <Tag>false</Tag>}
                    </span>
                    <span>
                      消费者：
                      {kafkaStats?.consumer_running ? <Tag color="green">运行中</Tag> : <Tag color="orange">未运行</Tag>}
                    </span>
                    <span>
                      消费组：
                      <Tag color="blue">{kafkaStats?.consumer_group || kafkaCfg?.consumer_group || "-"}</Tag>
                    </span>
                    <span>Topic 前缀：{kafkaStats?.topic_prefix || kafkaCfg?.topic_prefix || "yunshu-agent"}</span>
                    <span>示例 Topic：{kafkaCfg?.topic_example || "yunshu-agent-1"}</span>
                    <span>订阅 Topic 数：{(kafkaStats?.topics || []).length}</span>
                    <span>Brokers：{(kafkaStats?.brokers || kafkaCfg?.brokers || []).join(", ") || "-"}</span>
                    <span>最近消费：{kafkaStats?.last_consume_at ? formatDateTime(kafkaStats.last_consume_at) : "-"}</span>
                    <span>错误数：{kafkaStats?.error_total ?? 0}</span>
                  </Space>
                  {(kafkaStats?.topics || []).length > 0 ? (
                    <div style={{ marginTop: 8 }}>
                      <Space wrap size={[4, 4]}>
                        {kafkaStats!.topics!.map((t) => (
                          <Tag key={t}>{t}</Tag>
                        ))}
                      </Space>
                    </div>
                  ) : null}
                  {kafkaStats?.last_error ? (
                    <Alert style={{ marginTop: 12 }} type="error" showIcon message={kafkaStats.last_error} />
                  ) : null}
                </Card>

                <Card title="分区积压（按 Topic）" size="small">
                  <Table
                    rowKey={(r) => `${r.topic ?? ""}-${r.partition}`}
                    size="small"
                    loading={kafkaLoading}
                    pagination={{ pageSize: 10, size: "small" }}
                    dataSource={partitions}
                    locale={{ emptyText: kafkaStats?.sink_via_kafka ? "暂无分区数据" : "未启用 Kafka 中转" }}
                    columns={[
                      { title: "Topic", dataIndex: "topic", ellipsis: true },
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
                </Card>
              </Space>
            ),
          },
        ]}
      />
    </div>
  );
}
