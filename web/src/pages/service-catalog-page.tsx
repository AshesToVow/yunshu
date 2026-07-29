import { DeleteOutlined, LinkOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { getProjects, type ProjectItem } from "../services/projects";
import {
  addServiceCatalogLink,
  deleteServiceCatalog,
  deleteServiceCatalogLink,
  listServiceCatalog,
  upsertServiceCatalog,
  type ServiceCatalogItem,
  type ServiceLinkItem,
} from "../services/service-catalog";

const LINK_TYPE_OPTIONS = [
  { value: "cicd_service", label: "CI/CD 服务" },
  { value: "cmdb_service", label: "CMDB 服务" },
  { value: "log_source", label: "日志源" },
  { value: "k8s_workload", label: "K8s Workload" },
  { value: "alert_monitor_rule", label: "告警规则" },
  { value: "db_instance", label: "数据库实例" },
];

export function ServiceCatalogPage() {
  const navigate = useNavigate();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<ServiceCatalogItem[]>([]);
  const [editorOpen, setEditorOpen] = useState(false);
  const [linkOpen, setLinkOpen] = useState(false);
  const [current, setCurrent] = useState<ServiceCatalogItem | null>(null);
  const [form] = Form.useForm();
  const [linkForm] = Form.useForm();

  const projectOptions = useMemo(
    () => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })),
    [projects],
  );

  useEffect(() => {
    void (async () => {
      const p = await getProjects({ page: 1, page_size: 1000 });
      setProjects(p.list);
      if (p.list[0]) setProjectId(p.list[0].id);
    })();
  }, []);

  useEffect(() => {
    if (!projectId) return;
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  async function load() {
    if (!projectId) return;
    setLoading(true);
    try {
      const res = await listServiceCatalog(projectId, { page: 1, page_size: 200 });
      setList(res.list);
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ status: 1, criticality: "normal" });
    setEditorOpen(true);
  }

  function openEdit(row: ServiceCatalogItem) {
    setCurrent(row);
    form.setFieldsValue(row);
    setEditorOpen(true);
  }

  async function onSubmit() {
    if (!projectId) return;
    const v = await form.validateFields();
    await upsertServiceCatalog(projectId, { ...v, id: current?.id });
    message.success(current ? "已更新" : "已创建");
    setEditorOpen(false);
    void load();
  }

  function openLink(row: ServiceCatalogItem) {
    setCurrent(row);
    linkForm.resetFields();
    linkForm.setFieldsValue({ link_type: "cicd_service" });
    setLinkOpen(true);
  }

  async function onLinkSubmit() {
    if (!projectId || !current) return;
    const v = await linkForm.validateFields();
    const refId = v.ref_id != null && v.ref_id !== "" ? Number(v.ref_id) : undefined;
    await addServiceCatalogLink(projectId, current.id, {
      link_type: v.link_type,
      ref_id: Number.isFinite(refId) ? refId : undefined,
      ref_key: v.ref_key || undefined,
    });
    message.success("已绑定");
    setLinkOpen(false);
    void load();
  }

  async function onDeleteLink(row: ServiceCatalogItem, link: ServiceLinkItem) {
    if (!projectId) return;
    await deleteServiceCatalogLink(projectId, row.id, link.id);
    message.success("已解绑");
    void load();
  }

  return (
    <Card
      title="服务目录"
      extra={
        <Space wrap>
          <Select style={{ width: 260 }} value={projectId} onChange={setProjectId} options={projectOptions} placeholder="选择项目" />
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建
          </Button>
        </Space>
      }
    >
      <Table
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={false}
        columns={[
          { title: "标识", dataIndex: "identifier", width: 160 },
          { title: "名称", dataIndex: "name", width: 160 },
          { title: "负责人", dataIndex: "owner", width: 120 },
          { title: "产品线", dataIndex: "product_line", width: 120 },
          {
            title: "等级",
            dataIndex: "criticality",
            width: 100,
            render: (v: string) => <Tag>{v || "normal"}</Tag>,
          },
          {
            title: "绑定",
            dataIndex: "links",
            render: (links: ServiceLinkItem[] = [], row: ServiceCatalogItem) => (
              <Space wrap size={[4, 4]}>
                {(links || []).map((l) => (
                  <Tag
                    key={l.id}
                    closable
                    onClose={(e) => {
                      e.preventDefault();
                      void onDeleteLink(row, l);
                    }}
                  >
                    {l.link_type}:{l.ref_id ?? l.ref_key}
                  </Tag>
                ))}
              </Space>
            ),
          },
          {
            title: "操作",
            width: 200,
            render: (_: unknown, row: ServiceCatalogItem) => (
              <Space>
                <Button type="link" onClick={() => openEdit(row)}>
                  编辑
                </Button>
                <Button type="link" onClick={() => navigate(`/service-portrait?project_id=${projectId}&catalog_id=${row.id}`)}>
                  画像
                </Button>
                <Button type="link" icon={<LinkOutlined />} onClick={() => openLink(row)}>
                  绑定
                </Button>
                <Popconfirm title="确认删除？" onConfirm={() => projectId && deleteServiceCatalog(projectId, row.id).then(() => { message.success("已删除"); void load(); })}>
                  <Button type="link" danger icon={<DeleteOutlined />}>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal title={current ? "编辑服务" : "新建服务"} open={editorOpen} onOk={() => void onSubmit()} onCancel={() => setEditorOpen(false)} destroyOnClose>
        <Form form={form} layout="vertical">
          <Form.Item name="identifier" label="标识" rules={[{ required: true }]}>
            <Input disabled={!!current} placeholder="与 CI/CD identifier 对齐" />
          </Form.Item>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="owner" label="负责人">
            <Input />
          </Form.Item>
          <Form.Item name="product_line" label="产品线">
            <Input />
          </Form.Item>
          <Form.Item name="criticality" label="关键等级">
            <Select options={[{ value: "critical" }, { value: "high" }, { value: "normal" }, { value: "low" }]} />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={`绑定关联：${current?.name || ""}`} open={linkOpen} onOk={() => void onLinkSubmit()} onCancel={() => setLinkOpen(false)} destroyOnClose>
        <Form form={linkForm} layout="vertical">
          <Form.Item name="link_type" label="类型" rules={[{ required: true }]}>
            <Select options={LINK_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="ref_id" label="引用 ID">
            <Input placeholder="数值型外键，如 cicd_service.id" />
          </Form.Item>
          <Form.Item name="ref_key" label="引用 Key">
            <Input placeholder="如 clusterId/ns/kind/name" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
