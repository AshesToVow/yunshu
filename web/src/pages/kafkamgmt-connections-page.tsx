import { DeleteOutlined, CloudDownloadOutlined, PlusOutlined, ReloadOutlined, ThunderboltOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, message } from "antd";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import {
  createKafkamgmtConnection,
  deleteKafkamgmtConnection,
  importKafkamgmtConnectionFromDict,
  listKafkamgmtConnections,
  pingKafkamgmtConnection,
  testKafkamgmtConnection,
  updateKafkamgmtConnection,
  type KafkamgmtConnection,
} from "../services/kafkamgmt";
import { extractApiErrorMessage } from "../services/http";

export function KafkamgmtConnectionsPage() {
  const [list, setList] = useState<KafkamgmtConnection[]>([]);
  const [loading, setLoading] = useState(false);
  const [importing, setImporting] = useState(false);
  const [open, setOpen] = useState(false);
  const [testing, setTesting] = useState(false);
  const [current, setCurrent] = useState<KafkamgmtConnection | null>(null);
  const [form] = Form.useForm();

  async function load() {
    setLoading(true);
    try {
      setList((await listKafkamgmtConnections()) || []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载连接失败"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  async function onImportFromDict() {
    setImporting(true);
    try {
      const row = await importKafkamgmtConnectionFromDict();
      message.success(`已从数据字典导入：${row?.name || "连接"}（#${row?.id}）`);
      void load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "从字典导入失败"));
    } finally {
      setImporting(false);
    }
  }

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ timeout_sec: 10, is_default: false, sasl_mechanism: "" });
    setOpen(true);
  }

  function openEdit(row: KafkamgmtConnection) {
    setCurrent(row);
    form.setFieldsValue({
      name: row.name,
      brokers: row.brokers,
      username: row.username,
      sasl_mechanism: row.sasl_mechanism || "",
      timeout_sec: row.timeout_sec || 10,
      is_default: row.is_default,
      remark: row.remark,
      password: "",
    });
    setOpen(true);
  }

  async function onTest() {
    const values = await form.validateFields(["brokers", "username", "password", "sasl_mechanism", "timeout_sec"]);
    setTesting(true);
    try {
      const res = await testKafkamgmtConnection({
        brokers: values.brokers,
        username: values.username,
        password: values.password,
        sasl_mechanism: values.sasl_mechanism,
        timeout_sec: values.timeout_sec,
        connection_id: current?.id,
      });
      if (res?.ok) message.success("连通成功");
      else message.error(res?.message || "连通失败");
    } catch (e) {
      message.error(extractApiErrorMessage(e, "连通失败"));
    } finally {
      setTesting(false);
    }
  }

  async function onSubmit() {
    const values = await form.validateFields();
    try {
      if (current) {
        await updateKafkamgmtConnection(current.id, values);
        message.success("已更新");
      } else {
        await createKafkamgmtConnection(values);
        message.success("已创建");
      }
      setOpen(false);
      void load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "保存失败"));
    }
  }

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="Kafka 连接管理"
        description="维护 Kafka 集群连接（密码加密存储）。可从数据字典 kafka_* 一键导入。与「Kafka 队列」日志管道相互独立。"
        breadcrumbs={[{ title: "Kafka 管理控制台" }, { title: "连接管理" }]}
        extra={
          <Space wrap>
            <Link to="/kafkamgmt/topics">Topic</Link>
            <Link to="/kafkamgmt/groups">消费组</Link>
            <Link to="/kafkamgmt/brokers">Broker</Link>
            <Link to="/log-retention">日志队列</Link>
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
            <Button icon={<CloudDownloadOutlined />} loading={importing} onClick={() => void onImportFromDict()}>
              从数据字典导入
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建连接
            </Button>
          </Space>
        }
      />
      <Card className="table-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={list}
          locale={{ emptyText: "暂无连接，请新建或从数据字典导入" }}
          pagination={{ pageSize: 20, showSizeChanger: true }}
          columns={[
            { title: "名称", dataIndex: "name" },
            { title: "Brokers", dataIndex: "brokers", ellipsis: true },
            { title: "用户", dataIndex: "username", width: 120 },
            { title: "SASL", dataIndex: "sasl_mechanism", width: 120, render: (v?: string) => v || "—" },
            {
              title: "默认",
              dataIndex: "is_default",
              width: 80,
              render: (v: boolean) => (v ? <Tag color="blue">是</Tag> : "—"),
            },
            {
              title: "密码",
              width: 80,
              render: (_: unknown, r?: KafkamgmtConnection) => (r?.has_password ? "已配置" : "—"),
            },
            {
              title: "操作",
              width: 200,
              className: "yunshu-table-actions-cell",
              render: (_: unknown, row?: KafkamgmtConnection) =>
                row ? (
                  <Space size="small" wrap className="yunshu-table-actions">
                    <Button
                      type="link"
                      size="small"
                      icon={<ThunderboltOutlined />}
                      onClick={async () => {
                        try {
                          const res = await pingKafkamgmtConnection(row.id);
                          if (res?.ok) message.success("连通成功");
                          else message.error(res?.message || "连通失败");
                        } catch (e) {
                          message.error(extractApiErrorMessage(e, "连通失败"));
                        }
                      }}
                    >
                      探测
                    </Button>
                    <Button type="link" size="small" onClick={() => openEdit(row)}>
                      编辑
                    </Button>
                    <Popconfirm title="确认删除？" onConfirm={() => void deleteKafkamgmtConnection(row.id).then(load)}>
                      <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                        删除
                      </Button>
                    </Popconfirm>
                  </Space>
                ) : null,
            },
          ]}
        />
      </Card>
      <Modal
        title={current ? "编辑连接" : "新建连接"}
        open={open}
        onCancel={() => setOpen(false)}
        destroyOnClose
        footer={
          <Space>
            <Button onClick={() => setOpen(false)}>取消</Button>
            <Button loading={testing} icon={<ThunderboltOutlined />} onClick={() => void onTest()}>
              连通测试
            </Button>
            <Button type="primary" onClick={() => void onSubmit()}>
              保存
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="brokers" label="Brokers（逗号分隔）" rules={[{ required: true }]}>
            <Input.TextArea rows={2} placeholder="127.0.0.1:9092" />
          </Form.Item>
          <Form.Item name="username" label="用户名">
            <Input autoComplete="username" />
          </Form.Item>
          <Form.Item name="password" label={current ? "密码（留空不改）" : "密码"}>
            <Input.Password autoComplete="new-password" />
          </Form.Item>
          <Form.Item name="sasl_mechanism" label="SASL">
            <Select
              allowClear
              options={[
                { value: "", label: "无 / 自动" },
                { value: "plain", label: "PLAIN" },
                { value: "scram-sha-256", label: "SCRAM-SHA-256" },
                { value: "scram-sha-512", label: "SCRAM-SHA-512" },
              ]}
            />
          </Form.Item>
          <Form.Item name="timeout_sec" label="超时秒">
            <InputNumber min={3} max={60} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="is_default" label="默认连接" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default KafkamgmtConnectionsPage;
