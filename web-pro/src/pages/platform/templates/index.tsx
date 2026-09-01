// @ts-nocheck
import { PageContainer } from "@ant-design/pro-components";
import {
  CloudUploadOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import Editor from "@monaco-editor/react";
import {
  Button,
  Card,
  Drawer,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  createPlatformTemplate,
  deletePlatformTemplate,
  getPlatformTemplateVersion,
  listPlatformTemplates,
  listPlatformTemplateVersions,
  publishPlatformTemplate,
  savePlatformTemplateDraft,
  updatePlatformTemplate,
  type PlatformTemplate,
  type PlatformTemplateVersion,
} from "@/services/platform-templates";
import { formatDateTime } from "@/utils/format";

const CATEGORY_OPTIONS = [
  { label: "全部", value: "" },
  { label: "CI/CD 片段", value: "cicd_snippet" },
  { label: "告警", value: "alert" },
  { label: "巡检", value: "inspect" },
  { label: "Loggie", value: "loggie" },
];

function categoryLabel(c: string) {
  return CATEGORY_OPTIONS.find((o) => o.value === c)?.label || c;
}

function monacoLanguage(format: string) {
  switch (format) {
    case "yaml":
      return "yaml";
    case "html":
      return "html";
    case "shell":
      return "shell";
    case "go_template":
      return "plaintext";
    default:
      return "plaintext";
  }
}

export default function PlatformTemplatesPage() {
  const [category, setCategory] = useState("");
  const [keyword, setKeyword] = useState("");
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<PlatformTemplate[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const [metaOpen, setMetaOpen] = useState(false);
  const [editing, setEditing] = useState<PlatformTemplate | null>(null);
  const [metaForm] = Form.useForm();

  const [editorOpen, setEditorOpen] = useState(false);
  const [editorRow, setEditorRow] = useState<PlatformTemplate | null>(null);
  const [content, setContent] = useState("");
  const [versions, setVersions] = useState<PlatformTemplateVersion[]>([]);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listPlatformTemplates({
        category: category || undefined,
        keyword: keyword || undefined,
        page,
        page_size: pageSize,
      });
      setRows(res.list ?? []);
      setTotal(res.total ?? 0);
    } finally {
      setLoading(false);
    }
  }, [category, keyword, page, pageSize]);

  useEffect(() => {
    void load();
  }, [load]);

  const openCreate = () => {
    setEditing(null);
    metaForm.resetFields();
    metaForm.setFieldsValue({
      category: "cicd_snippet",
      format: "yaml",
      status: 1,
    });
    setMetaOpen(true);
  };

  const openEditMeta = (row: PlatformTemplate) => {
    setEditing(row);
    metaForm.setFieldsValue({
      category: row.category,
      name: row.name,
      format: row.format,
      description: row.description,
      status: row.status,
    });
    setMetaOpen(true);
  };

  const submitMeta = async () => {
    const values = await metaForm.validateFields();
    if (editing) {
      await updatePlatformTemplate(editing.id, values);
      message.success("已更新");
    } else {
      await createPlatformTemplate(values);
      message.success("已创建");
    }
    setMetaOpen(false);
    void load();
  };

  const openEditor = async (row: PlatformTemplate) => {
    setEditorRow(row);
    setEditorOpen(true);
    setContent("");
    const vers = await listPlatformTemplateVersions(row.id);
    setVersions(vers ?? []);
    const targetVer = row.published_version || vers?.[0]?.version;
    if (targetVer) {
      const full = await getPlatformTemplateVersion(row.id, targetVer);
      setContent(full.content_inline ?? "");
    }
  };

  const saveDraft = async () => {
    if (!editorRow) return;
    setSaving(true);
    try {
      const ver = await savePlatformTemplateDraft(editorRow.id, content, "ui draft");
      message.success(`已保存草稿 v${ver.version}`);
      const vers = await listPlatformTemplateVersions(editorRow.id);
      setVersions(vers ?? []);
      void load();
    } finally {
      setSaving(false);
    }
  };

  const publish = async (version?: number) => {
    if (!editorRow) return;
    setSaving(true);
    try {
      if (!version) {
        await savePlatformTemplateDraft(editorRow.id, content, "publish from editor");
      }
      const updated = await publishPlatformTemplate(editorRow.id, version);
      message.success(`已发布 v${updated.published_version}`);
      setEditorRow(updated);
      const vers = await listPlatformTemplateVersions(editorRow.id);
      setVersions(vers ?? []);
      void load();
    } finally {
      setSaving(false);
    }
  };

  const columns: ColumnsType<PlatformTemplate> = useMemo(
    () => [
      { title: "引用键", dataIndex: "template_key", width: 240, ellipsis: true },
      {
        title: "分类",
        dataIndex: "category",
        width: 110,
        render: (v: string) => <Tag>{categoryLabel(v)}</Tag>,
      },
      { title: "名称", dataIndex: "name", ellipsis: true },
      { title: "格式", dataIndex: "format", width: 100 },
      {
        title: "发布版本",
        dataIndex: "published_version",
        width: 100,
        render: (v: number) => (v > 0 ? `v${v}` : "未发布"),
      },
      {
        title: "状态",
        dataIndex: "status",
        width: 80,
        render: (v: number) => (
          <Tag color={v === 1 ? "green" : "default"}>{v === 1 ? "启用" : "停用"}</Tag>
        ),
      },
      {
        title: "更新",
        dataIndex: "updated_at",
        width: 220,
        render: (v: string, row) => (
          <Space size={8} wrap>
            {row.is_builtin ? <Tag>内置</Tag> : <Tag>自定义</Tag>}
            {row.has_minio_mirror ? <Tag color="blue">MinIO</Tag> : null}
            <Typography.Text type="secondary">{formatDateTime(v)}</Typography.Text>
          </Space>
        ),
      },
      {
        title: "操作",
        key: "actions",
        width: 200,
        fixed: "right",
        render: (_, row) => (
          <Space size="small">
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => void openEditor(row)}>
              编辑正文
            </Button>
            <Button type="link" size="small" onClick={() => openEditMeta(row)}>
              元数据
            </Button>
            {!row.is_builtin ? (
              <Popconfirm title="确认删除？" onConfirm={() => void deletePlatformTemplate(row.id).then(load)}>
                <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                  删除
                </Button>
              </Popconfirm>
            ) : null}
          </Space>
        ),
      },
    ],
    [load],
  );

  return (
    <PageContainer
      header={{
        title: "模板中心",
        subTitle: "CI/CD 片段 · 告警 · 巡检 · Loggie（MySQL 权威 + MinIO 镜像）",
      }}
    >
      <Card>
        <Space wrap style={{ marginBottom: 16 }}>
          <Select
            style={{ width: 140 }}
            value={category}
            options={CATEGORY_OPTIONS}
            onChange={(v) => {
              setCategory(v);
              setPage(1);
            }}
          />
          <Input.Search
            allowClear
            placeholder="搜索 key / 名称"
            style={{ width: 240 }}
            onSearch={(v) => {
              setKeyword(v);
              setPage(1);
            }}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建
          </Button>
        </Space>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          业务侧通过稳定 <Typography.Text code>template_key</Typography.Text> 引用；发布后才生效。未发布时解析回退内置种子。
        </Typography.Paragraph>
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: 1100 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>

      <Modal
        title={editing ? "编辑元数据" : "新建模板"}
        open={metaOpen}
        onCancel={() => setMetaOpen(false)}
        onOk={() => void submitMeta()}
        destroyOnClose
      >
        <Form form={metaForm} layout="vertical">
          {!editing ? (
            <Form.Item name="template_key" label="引用键" rules={[{ required: true }]}>
              <Input placeholder="如 cicd.apollo.custom" />
            </Form.Item>
          ) : null}
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="category" label="分类" rules={[{ required: true }]}>
            <Select options={CATEGORY_OPTIONS.filter((o) => o.value)} />
          </Form.Item>
          <Form.Item name="format" label="格式">
            <Select
              options={[
                { label: "YAML", value: "yaml" },
                { label: "Shell", value: "shell" },
                { label: "HTML", value: "html" },
                { label: "Go Template", value: "go_template" },
                { label: "Text", value: "text" },
              ]}
            />
          </Form.Item>
          <Form.Item name="description" label="说明">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select
              options={[
                { label: "启用", value: 1 },
                { label: "停用", value: 0 },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={editorRow ? `编辑 · ${editorRow.template_key}` : "编辑"}
        width="72%"
        open={editorOpen}
        onClose={() => setEditorOpen(false)}
        extra={
          <Space>
            <Button icon={<SaveOutlined />} loading={saving} onClick={() => void saveDraft()}>
              保存草稿
            </Button>
            <Button type="primary" icon={<CloudUploadOutlined />} loading={saving} onClick={() => void publish()}>
              保存并发布
            </Button>
          </Space>
        }
      >
        {editorRow ? (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            <Typography.Text type="secondary">
              当前发布：{editorRow.published_version > 0 ? `v${editorRow.published_version}` : "无"} · 格式{" "}
              {editorRow.format}
            </Typography.Text>
            <div style={{ border: "1px solid #eee", borderRadius: 8, overflow: "hidden" }}>
              <Editor
                height="52vh"
                language={monacoLanguage(editorRow.format)}
                value={content}
                onChange={(v) => setContent(v ?? "")}
                options={{
                  minimap: { enabled: false },
                  wordWrap: "on",
                  fontSize: 13,
                  automaticLayout: true,
                }}
              />
            </div>
            <Typography.Title level={5} style={{ margin: 0 }}>
              历史版本
            </Typography.Title>
            <Table
              size="small"
              rowKey="id"
              pagination={false}
              dataSource={versions}
              columns={[
                { title: "版本", dataIndex: "version", width: 80, render: (v: number) => `v${v}` },
                { title: "校验", dataIndex: "checksum", ellipsis: true, width: 140 },
                { title: "备注", dataIndex: "remark", ellipsis: true },
                { title: "时间", dataIndex: "created_at", width: 168, render: (v: string) => formatDateTime(v) },
                {
                  title: "操作",
                  width: 160,
                  render: (_, ver) => (
                    <Space size="small">
                      <Button
                        type="link"
                        size="small"
                        onClick={() =>
                          void getPlatformTemplateVersion(editorRow.id, ver.version).then((f) =>
                            setContent(f.content_inline ?? ""),
                          )
                        }
                      >
                        加载
                      </Button>
                      <Button type="link" size="small" onClick={() => void publish(ver.version)}>
                        发布此版
                      </Button>
                    </Space>
                  ),
                },
              ]}
            />
          </Space>
        ) : null}
      </Drawer>
    </PageContainer>
  );
}