import { ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, Select, Space, Table, Tag, message } from "antd";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import {
  cancelEsmgmtReindex,
  createEsmgmtReindex,
  listEsmgmtConnections,
  listEsmgmtReindex,
  type EsmgmtConnection,
  type EsmgmtReindexJob,
} from "../services/esmgmt";
import { extractApiErrorMessage } from "../services/http";

const STATUS_COLOR: Record<string, string> = {
  running: "processing",
  pending: "default",
  success: "success",
  failed: "error",
  cancelled: "warning",
};

export function EsmgmtReindexPage() {
  const [connections, setConnections] = useState<EsmgmtConnection[]>([]);
  const [connectionId, setConnectionId] = useState<number>();
  const [jobs, setJobs] = useState<EsmgmtReindexJob[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm();

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
      setJobs((await listEsmgmtReindex({ connection_id: connectionId })) || []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载任务失败"));
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
        title="Reindex"
        description="异步复制索引数据。任务可取消，并记录操作人。目标为 yunshu-* 时须超级管理员。不开放任意 REST 代理。"
        breadcrumbs={[{ title: "ES 管理控制台" }, { title: "Reindex" }]}
        extra={
          <Space wrap>
            <Link to="/esmgmt/connections">连接管理</Link>
            <Select
              style={{ minWidth: 200 }}
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
      <Card title="提交任务" style={{ marginBottom: 16 }}>
        <Form
          form={form}
          layout="inline"
          onFinish={async (values) => {
            let query: unknown;
            const raw = String(values.query || "").trim();
            if (raw) {
              try {
                query = JSON.parse(raw);
              } catch {
                message.error("query JSON 无法解析");
                return;
              }
            }
            setSubmitting(true);
            try {
              const job = await createEsmgmtReindex({
                connection_id: connectionId,
                source_index: values.source_index,
                dest_index: values.dest_index,
                query,
              });
              message.success(`已提交 #${job.id}`);
              void load();
            } catch (e) {
              message.error(extractApiErrorMessage(e, "提交失败"));
            } finally {
              setSubmitting(false);
            }
          }}
        >
          <Form.Item name="source_index" rules={[{ required: true, message: "源索引" }]}>
            <Input placeholder="源索引" style={{ width: 200 }} />
          </Form.Item>
          <Form.Item name="dest_index" rules={[{ required: true, message: "目标索引" }]}>
            <Input placeholder="目标索引" style={{ width: 200 }} />
          </Form.Item>
          <Form.Item name="query">
            <Input placeholder='可选 query JSON，如 {"match_all":{}}' style={{ width: 280 }} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={submitting}>
              提交
            </Button>
          </Form.Item>
        </Form>
      </Card>
      <Card className="table-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={jobs}
          pagination={{ pageSize: 20 }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "源", dataIndex: "source_index", ellipsis: true },
            { title: "目标", dataIndex: "dest_index", ellipsis: true },
            {
              title: "状态",
              dataIndex: "status",
              width: 110,
              render: (v: string) => <Tag color={STATUS_COLOR[v] || "default"}>{v}</Tag>,
            },
            { title: "进度", width: 120, render: (_: unknown, r: EsmgmtReindexJob) => `${r.created_docs || 0}/${r.total || 0}` },
            { title: "错误", dataIndex: "error_message", ellipsis: true },
            {
              title: "操作",
              width: 100,
              render: (_: unknown, row?: EsmgmtReindexJob) =>
                row && (row.status === "running" || row.status === "pending") ? (
                  <Button
                    type="link"
                    size="small"
                    onClick={async () => {
                      try {
                        await cancelEsmgmtReindex(row.id);
                        message.success("已取消");
                        void load();
                      } catch (e) {
                        message.error(extractApiErrorMessage(e, "取消失败"));
                      }
                    }}
                  >
                    取消
                  </Button>
                ) : null,
            },
          ]}
        />
      </Card>
    </div>
  );
}

export default EsmgmtReindexPage;
