import { PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Drawer, Form, Input, Modal, Select, Space, Table, message } from "antd";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import {
  deleteEsmgmtDoc,
  listEsmgmtConnections,
  searchEsmgmtDocs,
  upsertEsmgmtDoc,
  type EsmgmtConnection,
  type EsmgmtDocHit,
} from "../services/esmgmt";
import { extractApiErrorMessage } from "../services/http";

export function EsmgmtDocsPage() {
  const [connections, setConnections] = useState<EsmgmtConnection[]>([]);
  const [connectionId, setConnectionId] = useState<number>();
  const [index, setIndex] = useState("");
  const [query, setQuery] = useState("");
  const [hits, setHits] = useState<EsmgmtDocHit[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editId, setEditId] = useState("");
  const [editIndex, setEditIndex] = useState("");
  const [editBody, setEditBody] = useState("{}");
  const [createOpen, setCreateOpen] = useState(false);
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
    if (!index.trim()) {
      message.warning("请填写索引名");
      return;
    }
    setLoading(true);
    try {
      const res = await searchEsmgmtDocs({
        connection_id: connectionId,
        index: index.trim(),
        query: query.trim() || undefined,
        size: 20,
      });
      setHits(res?.hits || []);
      setTotal(res?.total || 0);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "检索失败"));
    } finally {
      setLoading(false);
    }
  }

  function openEdit(row: EsmgmtDocHit) {
    setEditIndex(row.index || index.trim());
    setEditId(row.id);
    setEditBody(JSON.stringify(row.source ?? {}, null, 2));
    setEditOpen(true);
  }

  async function saveEdit() {
    let source: unknown;
    try {
      source = JSON.parse(editBody);
    } catch {
      message.error("JSON 无法解析");
      return;
    }
    if (!source || typeof source !== "object" || Array.isArray(source)) {
      message.error("文档须为 JSON 对象");
      return;
    }
    try {
      await upsertEsmgmtDoc({ connection_id: connectionId, index: editIndex, id: editId, source });
      message.success("已保存");
      setEditOpen(false);
      void load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "保存失败"));
    }
  }

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="文档检索"
        description="按索引检索文档并编辑单行。写入 yunshu-* 须超级管理员。不允许脚本。"
        breadcrumbs={[{ title: "ES 管理控制台" }, { title: "文档检索" }]}
        extra={
          <Space wrap>
            <Link to="/esmgmt/connections">连接管理</Link>
            <Select
              style={{ minWidth: 200 }}
              placeholder="连接"
              value={connectionId}
              options={connections.map((c) => ({ value: c.id, label: c.name }))}
              onChange={setConnectionId}
            />
            <Input style={{ width: 220 }} placeholder="索引，如 app-logs-*" value={index} onChange={(e) => setIndex(e.target.value)} />
            <Input style={{ width: 220 }} placeholder="query_string，空则 match_all" value={query} onChange={(e) => setQuery(e.target.value)} />
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              检索
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                createForm.resetFields();
                createForm.setFieldsValue({ index: index.trim(), source: "{\n  \n}" });
                setCreateOpen(true);
              }}
            >
              新建文档
            </Button>
          </Space>
        }
      />
      <Card className="table-card">
        <Table
          rowKey={(r) => `${r.index}-${r.id}`}
          loading={loading}
          dataSource={hits}
          pagination={false}
          locale={{ emptyText: total ? "无命中" : "输入索引后检索" }}
          columns={[
            { title: "索引", dataIndex: "index", width: 220, ellipsis: true },
            { title: "ID", dataIndex: "id", width: 180, ellipsis: true },
            {
              title: "内容",
              dataIndex: "source",
              ellipsis: true,
              render: (v: unknown) => JSON.stringify(v),
            },
            {
              title: "操作",
              width: 140,
              className: "yunshu-table-actions-cell",
              render: (_: unknown, row?: EsmgmtDocHit) =>
                row ? (
                  <Space size="small" className="yunshu-table-actions">
                    <Button type="link" size="small" onClick={() => openEdit(row)}>
                      编辑
                    </Button>
                    <Button
                      type="link"
                      size="small"
                      danger
                      onClick={() => {
                        Modal.confirm({
                          title: `删除文档 ${row.id}？`,
                          onOk: async () => {
                            try {
                              await deleteEsmgmtDoc({ connection_id: connectionId, index: row.index, id: row.id });
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
      <Drawer
        title={`${editIndex} / ${editId}`}
        open={editOpen}
        onClose={() => setEditOpen(false)}
        width={640}
        footer={
          <Space>
            <Button onClick={() => setEditOpen(false)}>取消</Button>
            <Button type="primary" onClick={() => void saveEdit()}>
              保存
            </Button>
          </Space>
        }
      >
        <Input.TextArea rows={22} value={editBody} onChange={(e) => setEditBody(e.target.value)} />
      </Drawer>
      <Modal
        title="新建文档"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={async () => {
          const values = await createForm.validateFields();
          let source: unknown;
          try {
            source = JSON.parse(values.source);
          } catch {
            message.error("JSON 无法解析");
            return;
          }
          try {
            await upsertEsmgmtDoc({
              connection_id: connectionId,
              index: values.index,
              id: values.id,
              source,
            });
            message.success("已写入");
            setCreateOpen(false);
            setIndex(values.index);
          } catch (e) {
            message.error(extractApiErrorMessage(e, "写入失败"));
          }
        }}
        destroyOnClose
      >
        <Form form={createForm} layout="vertical">
          <Form.Item name="index" label="索引" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="id" label="文档 ID" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="source" label="JSON" rules={[{ required: true }]}>
            <Input.TextArea rows={8} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default EsmgmtDocsPage;
