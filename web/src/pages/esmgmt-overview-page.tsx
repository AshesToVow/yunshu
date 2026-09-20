import { CloudUploadOutlined, DownOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Dropdown, Form, Input, Modal, Select, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  closeEsmgmtIndex,
  createEsmgmtBackup,
  createEsmgmtIndex,
  deleteEsmgmtIndex,
  getEsmgmtClusterHealth,
  listEsmgmtConnections,
  listEsmgmtIndices,
  listEsmgmtNodes,
  openEsmgmtIndex,
  type EsmgmtConnection,
} from "../services/esmgmt";
import { extractApiErrorMessage } from "../services/http";

const DEFAULT_CREATE_BODY = `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 1
  },
  "mappings": {
    "properties": {
      "message": { "type": "text" },
      "@timestamp": { "type": "date" }
    }
  }
}`;

export function EsmgmtOverviewPage() {
  const [connections, setConnections] = useState<EsmgmtConnection[]>([]);
  const [connectionId, setConnectionId] = useState<number>();
  const [health, setHealth] = useState<Record<string, unknown> | null>(null);
  const [indices, setIndices] = useState<Array<{ name: string; store_bytes?: number; docs_count?: number }>>([]);
  const [nodes, setNodes] = useState<Record<string, unknown>[]>([]);
  const [indexFilter, setIndexFilter] = useState("");
  const [indexPage, setIndexPage] = useState(1);
  const [indexPageSize, setIndexPageSize] = useState(20);
  const [loading, setLoading] = useState(false);
  const [backingUp, setBackingUp] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createForm] = Form.useForm();

  useEffect(() => {
    void listEsmgmtConnections()
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
      const [h, idx, nd] = await Promise.all([
        getEsmgmtClusterHealth(connectionId),
        listEsmgmtIndices({ connection_id: connectionId }),
        listEsmgmtNodes(connectionId),
      ]);
      setHealth(h);
      setIndices(idx || []);
      setNodes(nd || []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载集群信息失败"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [connectionId]);

  async function onBackup(indexName: string) {
    setBackingUp(indexName);
    try {
      const job = await createEsmgmtBackup({ connection_id: connectionId, index: indexName });
      message.success(`备份任务已创建 #${job.id}，请到「备份与恢复」查看进度`);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建备份失败"));
    } finally {
      setBackingUp(null);
    }
  }

  async function onCreateIndex() {
    const values = await createForm.validateFields();
    let settings: Record<string, unknown> | undefined;
    let mappings: Record<string, unknown> | undefined;
    const raw = String(values.body_json || "").trim();
    if (raw) {
      try {
        const parsed = JSON.parse(raw) as Record<string, unknown>;
        if (parsed.settings && typeof parsed.settings === "object") {
          settings = parsed.settings as Record<string, unknown>;
        }
        if (parsed.mappings && typeof parsed.mappings === "object") {
          mappings = parsed.mappings as Record<string, unknown>;
        }
        if (!settings && !mappings && (parsed.number_of_shards != null || parsed.index != null)) {
          settings = parsed;
        }
      } catch {
        message.error("Body JSON 无法解析");
        return;
      }
    }
    setCreating(true);
    try {
      await createEsmgmtIndex({
        connection_id: connectionId,
        name: String(values.name).trim(),
        settings,
        mappings,
      });
      message.success("索引已创建");
      setCreateOpen(false);
      void load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建索引失败"));
    } finally {
      setCreating(false);
    }
  }

  const status = String(health?.status || "");
  const statusColor = status === "green" ? "success" : status === "yellow" ? "warning" : status ? "error" : "default";
  const statusLabel =
    status === "green" ? "健康" : status === "yellow" ? "降级" : status === "red" ? "异常" : status || "-";

  return (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      <Card size="small">
        <Space wrap>
          <Select
            style={{ minWidth: 360 }}
            placeholder="选择连接"
            value={connectionId}
            options={connections.map((c) => ({
              value: c.id,
              label: `${c.name}${c.is_default ? "（默认）" : ""}（${c.addresses || "-"}）`,
            }))}
            onChange={setConnectionId}
          />
          <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>
            刷新
          </Button>
          {status ? <Tag color={statusColor}>{statusLabel}</Tag> : null}
          <Link to="/esmgmt/backups">备份与恢复</Link>
        </Space>
      </Card>
      {health ? (
        <Alert
          type={statusColor === "success" ? "success" : statusColor === "warning" ? "warning" : "info"}
          showIcon
          message={`集群 ${String(health.cluster_name || "-")} · 节点 ${String(health.number_of_nodes ?? "-")} · 分片 ${String(health.active_shards ?? "-")}`}
        />
      ) : null}
      <Card
        title="索引"
        size="small"
        extra={
          <Space wrap>
            <Input
              allowClear
              size="small"
              placeholder="过滤索引名，如 yunshu-k8s"
              style={{ width: 220 }}
              value={indexFilter}
              onChange={(e) => {
                setIndexFilter(e.target.value);
                setIndexPage(1);
              }}
            />
            <Button
              size="small"
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                createForm.resetFields();
                createForm.setFieldsValue({ body_json: DEFAULT_CREATE_BODY });
                setCreateOpen(true);
              }}
            >
              新建索引
            </Button>
          </Space>
        }
      >
        <Table
          size="small"
          rowKey="name"
          loading={loading}
          dataSource={indices.filter((i) =>
            !indexFilter.trim() ? true : i.name.toLowerCase().includes(indexFilter.trim().toLowerCase()),
          )}
          pagination={{
            current: indexPage,
            pageSize: indexPageSize,
            showSizeChanger: true,
            pageSizeOptions: ["10", "20", "50", "100"],
            showTotal: (t) => `共 ${t} 个索引`,
            onChange: (p, ps) => {
              setIndexPage(p);
              setIndexPageSize(ps);
            },
          }}
          columns={[
            { title: "索引", dataIndex: "name", ellipsis: true },
            { title: "文档数", dataIndex: "docs_count", width: 100 },
            { title: "存储字节", dataIndex: "store_bytes", width: 120 },
            {
              title: "操作",
              width: 160,
              className: "yunshu-table-actions-cell",
              render: (_: unknown, row?: { name: string }) =>
                row ? (
                  <Space size={0} className="yunshu-table-actions">
                    <Button
                      type="link"
                      size="small"
                      icon={<CloudUploadOutlined />}
                      loading={backingUp === row.name}
                      onClick={() => void onBackup(row.name)}
                    >
                      备份
                    </Button>
                    <Dropdown
                      trigger={["click"]}
                      menu={{
                        items: [
                          { key: "open", label: "打开", onClick: () => void openEsmgmtIndex(row.name, connectionId).then(load) },
                          { key: "close", label: "关闭", onClick: () => void closeEsmgmtIndex(row.name, connectionId).then(load) },
                          {
                            key: "delete",
                            danger: true,
                            label: "删除",
                            onClick: () => {
                              const force = row.name.includes("yunshu-agent") || row.name.includes("yunshu-k8s");
                              Modal.confirm({
                                title: force ? "日志索引，需强制删除确认" : "确认删除索引？",
                                onOk: () => deleteEsmgmtIndex(row.name, force, connectionId).then(load),
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
      <Card title="节点" size="small">
        <Table
          size="small"
          rowKey={(r) => String(r.name || r.ip)}
          loading={loading}
          dataSource={nodes}
          pagination={{ pageSize: 8 }}
          columns={[
            { title: "名称", dataIndex: "name" },
            { title: "IP", dataIndex: "ip", width: 140 },
            { title: "角色", dataIndex: "node.role", width: 100 },
            { title: "CPU", dataIndex: "cpu", width: 80 },
            { title: "Heap%", dataIndex: "heap.percent", width: 90 },
          ]}
        />
        {!nodes.length && !loading ? <Typography.Text type="secondary">暂无节点数据</Typography.Text> : null}
      </Card>
      <Modal
        title="新建索引"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => void onCreateIndex()}
        confirmLoading={creating}
        width={640}
        destroyOnClose
      >
        <Form form={createForm} layout="vertical">
          <Form.Item
            name="name"
            label="索引名"
            rules={[
              { required: true, message: "请输入索引名" },
              {
                validator: async (_, v) => {
                  const name = String(v || "").trim();
                  if (!name) return;
                  if (name.startsWith(".")) throw new Error("禁止系统索引名");
                  if (/[\\/?*"<>|,#\s]/.test(name)) throw new Error("索引名含非法字符");
                },
              },
            ]}
          >
            <Input placeholder="例如 my-app-logs" />
          </Form.Item>
          <Form.Item
            name="body_json"
            label="settings / mappings（JSON，可选）"
            extra="可填 { settings, mappings }；留空则按 ES 默认创建空索引"
          >
            <Input.TextArea rows={14} style={{ fontFamily: "monospace" }} />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}

export default EsmgmtOverviewPage;
