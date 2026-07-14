import { DeleteOutlined, PlayCircleOutlined, ReloadOutlined, SaveOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Col, Form, Input, InputNumber, Row, Select, Space, Switch, Table, Tag, message } from "antd";
import { useCallback, useEffect, useState } from "react";
import {
  deleteProjectLogRetention,
  getESStorageStats,
  getGlobalLogRetention,
  getProjects,
  listLogRetentionPolicies,
  runLogRetentionCleanup,
  upsertGlobalLogRetention,
  upsertProjectLogRetention,
  type ESStorageStats,
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
  const [globalForm] = Form.useForm<GlobalForm>();
  const [stats, setStats] = useState<ESStorageStats | null>(null);
  const [policies, setPolicies] = useState<LogRetentionItem[]>([]);
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [projectId, setProjectId] = useState<number>();
  const [projectDays, setProjectDays] = useState(30);
  const [projectEnabled, setProjectEnabled] = useState(true);

  const reload = useCallback(async () => {
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

  useEffect(() => {
    void reload();
  }, [reload]);

  async function saveGlobal() {
    const values = await globalForm.validateFields();
    await upsertGlobalLogRetention(values);
    message.success("全局保留策略已保存");
    await reload();
  }

  async function saveProjectOverride() {
    if (!projectId) {
      message.warning("请选择项目");
      return;
    }
    await upsertProjectLogRetention(projectId, { retention_days: projectDays, enabled: projectEnabled });
    message.success("项目覆盖策略已保存");
    await reload();
  }

  async function removeProjectOverride() {
    if (!projectId) return;
    await deleteProjectLogRetention(projectId);
    message.success("已删除项目覆盖，将继承全局策略");
    await reload();
  }

  return (
    <div className="log-retention-page">
      <Space direction="vertical" size={16} style={{ width: "100%" }}>
        <Alert
          type="info"
          showIcon
          message="日志保留说明"
          description={
            <div>
              <p>Elasticsearch 默认不会自动删除数据。Yunshu 按「保留天数」定时清理过期日志。</p>
              <p>
                推荐 Loggie 使用<strong>按日滚动索引</strong>（如 <code>yunshu-logs-2026.07.13</code>），清理时直接删除过期索引，效率最高。
              </p>
              <p>Loggie 自身的状态检查、pipeline 解析、实时采集由 Agent 侧配置，与 Yunshu 保留策略互补。</p>
            </div>
          }
        />

        <Card
          title="ES 存储概览"
          extra={
            <Button icon={<ReloadOutlined />} onClick={() => void reload()} loading={loading}>
              刷新
            </Button>
          }
        >
          {stats ? (
            <Space size={24} wrap>
              <span>索引模式：{stats.index_pattern}</span>
              <span>索引数：{stats.index_count}</span>
              <span>文档数：{stats.document_count}</span>
              <span>占用：{stats.store_human}</span>
            </Space>
          ) : (
            <span>无法连接 ES 或未启用 elasticsearch.enabled</span>
          )}
        </Card>

        <Card
          title="全局默认保留策略"
          extra={
            <Space>
              <Button
                icon={<PlayCircleOutlined />}
                onClick={() =>
                  void (async () => {
                    const res = await runLogRetentionCleanup();
                    message.success(res.message || "清理完成");
                    await reload();
                  })()
                }
              >
                立即清理
              </Button>
              <Button type="primary" icon={<SaveOutlined />} onClick={() => void saveGlobal()}>
                保存
              </Button>
            </Space>
          }
        >
          <Form form={globalForm} layout="vertical">
            <Row gutter={16}>
              <Col span={6}>
                <Form.Item label="保留天数" name="retention_days" rules={[{ required: true }]}>
                  <InputNumber min={1} max={3650} style={{ width: "100%" }} addonAfter="天" />
                </Form.Item>
              </Col>
              <Col span={6}>
                <Form.Item label="启用自动清理" name="enabled" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={6}>
                <Form.Item label="索引模式（可选）" name="index_pattern">
                  <Input placeholder="默认 yunshu-logs-*" />
                </Form.Item>
              </Col>
              <Col span={6}>
                <Form.Item label="备注" name="remark">
                  <Input placeholder="可选" />
                </Form.Item>
              </Col>
            </Row>
          </Form>
        </Card>

        <Card title="项目级覆盖（可选）">
          <Space wrap style={{ marginBottom: 16 }}>
            <Select
              style={{ width: 280 }}
              placeholder="选择项目"
              options={projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` }))}
              value={projectId}
              onChange={setProjectId}
              allowClear
            />
            <InputNumber min={1} max={3650} value={projectDays} onChange={(v) => setProjectDays(v ?? 30)} addonAfter="天" />
            <span>
              启用 <Switch checked={projectEnabled} onChange={setProjectEnabled} />
            </span>
            <Button type="primary" onClick={() => void saveProjectOverride()}>
              保存覆盖
            </Button>
            <Button danger icon={<DeleteOutlined />} onClick={() => void removeProjectOverride()}>
              删除覆盖
            </Button>
          </Space>
          <Table
            rowKey={(r) => `${r.project_id}-${r.id}`}
            size="small"
            dataSource={policies}
            pagination={false}
            columns={[
              {
                title: "范围",
                dataIndex: "project_id",
                render: (v: number) => (v === 0 ? <Tag color="blue">全局</Tag> : <Tag>项目 #{v}</Tag>),
              },
              { title: "保留天数", dataIndex: "retention_days" },
              {
                title: "启用",
                dataIndex: "enabled",
                render: (v: boolean) => (v ? <Tag color="success">是</Tag> : <Tag>否</Tag>),
              },
              { title: "索引模式", dataIndex: "index_pattern", render: (v?: string) => v || "-" },
              { title: "更新时间", dataIndex: "updated_at", render: (v?: string) => (v ? formatDateTime(v) : "-") },
            ]}
          />
        </Card>
      </Space>
    </div>
  );
}
