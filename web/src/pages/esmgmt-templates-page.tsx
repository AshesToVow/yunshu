import { PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, Modal, Select, Space, Table, Tag, message } from "antd";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import {
  deleteEsmgmtTemplate,
  listEsmgmtConnections,
  listEsmgmtTemplates,
  putEsmgmtTemplate,
  type EsmgmtConnection,
  type EsmgmtIndexTemplate,
} from "../services/esmgmt";
import { extractApiErrorMessage } from "../services/http";

export function EsmgmtTemplatesPage() {
  const [connections, setConnections] = useState<EsmgmtConnection[]>([]);
  const [connectionId, setConnectionId] = useState<number>();
  const [kind, setKind] = useState<string>("");
  const [rows, setRows] = useState<EsmgmtIndexTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
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
      setRows((await listEsmgmtTemplates({ connection_id: connectionId, kind: kind || undefined })) || []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载模板失败"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (connectionId) void load();
  }, [connectionId, kind]);

  function openEdit(row?: EsmgmtIndexTemplate) {
    form.setFieldsValue({
      kind: row?.kind || "legacy",
      name: row?.name || "",
      body: JSON.stringify(row?.body ?? { index_patterns: ["app-*"], mappings: { properties: {} } }, null, 2),
    });
    setOpen(true);
  }

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="索引模板"
        description="管理 ES 7 旧版 _template 与可组合 _index_template。不允许写入 painless 脚本。"
        breadcrumbs={[{ title: "ES 管理控制台" }, { title: "索引模板" }]}
        extra={
          <Space wrap>
            <Link to="/esmgmt/connections">连接管理</Link>
            <Select
              style={{ minWidth: 200 }}
              value={connectionId}
              options={connections.map((c) => ({ value: c.id, label: c.name }))}
              onChange={setConnectionId}
            />
            <Select
              style={{ width: 160 }}
              value={kind}
              onChange={setKind}
              options={[
                { value: "", label: "全部" },
                { value: "legacy", label: "旧版 _template" },
                { value: "composable", label: "可组合" },
              ]}
            />
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openEdit()}>
              新建
            </Button>
          </Space>
        }
      />
      <Card className="table-card">
        <Table
          rowKey={(r) => `${r.kind}-${r.name}`}
          loading={loading}
          dataSource={rows}
          pagination={{ pageSize: 20 }}
          columns={[
            {
              title: "类型",
              dataIndex: "kind",
              width: 120,
              render: (v: string) => <Tag>{v === "composable" ? "可组合" : "旧版"}</Tag>,
            },
            { title: "名称", dataIndex: "name" },
            {
              title: "操作",
              width: 140,
              className: "yunshu-table-actions-cell",
              render: (_: unknown, row?: EsmgmtIndexTemplate) =>
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
                          title: `删除模板 ${row.name}？`,
                          onOk: async () => {
                            try {
                              await deleteEsmgmtTemplate({ connection_id: connectionId, kind: row.kind, name: row.name });
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
      <Modal
        title="索引模板"
        open={open}
        width={720}
        onCancel={() => setOpen(false)}
        onOk={async () => {
          const values = await form.validateFields();
          let body: unknown;
          try {
            body = JSON.parse(values.body);
          } catch {
            message.error("JSON 无法解析");
            return;
          }
          try {
            await putEsmgmtTemplate({
              connection_id: connectionId,
              kind: values.kind,
              name: values.name,
              body,
            });
            message.success("已保存");
            setOpen(false);
            void load();
          } catch (e) {
            message.error(extractApiErrorMessage(e, "保存失败"));
          }
        }}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="kind" label="类型" rules={[{ required: true }]}>
            <Select
              options={[
                { value: "legacy", label: "旧版 _template" },
                { value: "composable", label: "可组合 _index_template" },
              ]}
            />
          </Form.Item>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="body" label="JSON" rules={[{ required: true }]}>
            <Input.TextArea rows={16} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default EsmgmtTemplatesPage;
